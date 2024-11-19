package evmrpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
)

const (
	coingeckoUSDCurrency = "usd"
)

// Oracle is a evm uusdc tx execution price oracle that determines the price of
// executing a tx on chain in uusdc.
type Oracle struct {
	coingecko coingecko.PriceClient
}

// NewOracle creates a new evm uusdc tx execution price oracle.
func NewOracle(coingecko coingecko.PriceClient) *Oracle {
	return &Oracle{coingecko: coingecko}
}

// TxFeeUUSDC estimates what the cost in uusdc would be to execute a tx. The
// tx's gas fee cap and gas limit must be set.
func (o *Oracle) TxFeeUUSDC(ctx context.Context, tx *types.Transaction, gasTokenCoingeckoID string) (*big.Int, error) {
	if tx.Type() != types.DynamicFeeTxType {
		return nil, fmt.Errorf("tx type must be dynamic fee tx, got %d", tx.Type())
	}

	// for a dry ran tx, GasFeeCap() will be the suggested gas tip cap + base
	// fee of current chain head
	estimatedPricePerGas := tx.GasFeeCap()
	if estimatedPricePerGas == nil {
		return nil, fmt.Errorf("tx's gas fee cap must be set")
	}

	// for a dry ran tx, Gas() will be the result of calling eth_estimateGas
	estimatedGasUsed := tx.Gas()
	return o.gasCostUUSDC(ctx, estimatedPricePerGas, big.NewInt(int64(estimatedGasUsed)), gasTokenCoingeckoID)
}

// gasCostUUSDC converts an amount of gas and the price per gas in gwei to
// uusdc based on the current CoinGecko price of ethereum in usd.
func (o *Oracle) gasCostUUSDC(ctx context.Context, pricePerGasWei *big.Int, gasUsed *big.Int, gasTokenCoingeckoID string) (*big.Int, error) {
	// Calculate transaction fee in Wei
	txFeeWei := new(big.Int).Mul(gasUsed, pricePerGasWei)

	// Get the ETH price in USD cents from CoinGecko
	ethPriceUSD, err := o.coingecko.GetSimplePrice(ctx, gasTokenCoingeckoID, coingeckoUSDCurrency)
	if err != nil {
		return nil, fmt.Errorf("getting CoinGecko price of Ethereum in USD: %w", err)
	}

	// Convert ETH price to microunits of USDC (uusdc) per Wei
	// WEI_PER_ETH = 1_000_000_000_000_000_000 (1 ETH = 10^18 Wei)
	// UUSDC_PER_USD = 1_000_000 (1 USD = 10^6 UUSDC)
	const WEI_PER_ETH = 1_000_000_000_000_000_000
	const UUSDC_PER_USD = 1_000_000

	// convert eth price in usd to eth price in uusdc
	ethPriceUUSDC := new(big.Float).Mul(big.NewFloat(ethPriceUSD), new(big.Float).SetInt64(UUSDC_PER_USD))

	// eth price in usd comes back from coin gecko with two decimals. Since we
	// just converted to uusdc, shifting the decimal place right by 6, we can
	// safely turn this into an int now
	ethPriceUUSDCInt, ok := new(big.Int).SetString(ethPriceUUSDC.String(), 10)
	if !ok {
		return nil, fmt.Errorf("converting eth price in uusdc %s to *big.Int", ethPriceUUSDC.String())
	}

	// What we are really trying to do is:
	//   eth price uusdc / wei per eth = wei price in uusdc
	//   wei price in uusdc * tx fee wei = tx fee uusdc
	// However we are choosing to first multiply eth price uusdc by tx fee wei
	// so that we can do integer division when converting to wei, since if we
	// first do integer division (before multiplying), we are going to cut off
	// necessary decimals. there are limits of this, if eth price uusdc * tx
	// fee wei has less than 9 digits, then we will just return 0. However,
	// this is unlikely in practice and the tx fee would be very small if this
	// is the case.
	tmp := new(big.Int).Mul(ethPriceUUSDCInt, txFeeWei)
	txFeeUUSDC := new(big.Int).Div(tmp, big.NewInt(WEI_PER_ETH))
	return txFeeUUSDC, nil
}
