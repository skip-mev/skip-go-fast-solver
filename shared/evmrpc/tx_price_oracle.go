package evmrpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/config"
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
func (o *Oracle) TxFeeUUSDC(ctx context.Context, tx *types.Transaction) (*big.Int, error) {
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
	return o.GasCostUUSDC(ctx, estimatedPricePerGas, big.NewInt(int64(estimatedGasUsed)), tx.ChainId().String())
}

// gasCostUUSDC converts an amount of gas and the price per gas in gwei to
// uusdc based on the current CoinGecko price of ethereum in usd.
func (o *Oracle) GasCostUUSDC(ctx context.Context, pricePerGas *big.Int, gasUsed *big.Int, chainID string) (*big.Int, error) {
	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("getting config for chain %s: %w", chainID, err)
	}

	// Calculate transaction fee
	txFee := new(big.Int).Mul(gasUsed, pricePerGas)

	// Get the gas token price in USD cents from CoinGecko
	gasTokenPriceUSD, err := o.coingecko.GetSimplePrice(ctx, chainConfig.GasTokenCoingeckoID, coingeckoUSDCurrency)
	if err != nil {
		return nil, fmt.Errorf("getting CoinGecko price of %s in USD: %w", chainConfig.GasTokenCoingeckoID, err)
	}

	// conversion to bring gas token to its smallest representation
	// UUSDC_PER_USD = 1_000_000 (1 USD = 10^6 UUSDC)
	smallestGasTokenConversion := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(chainConfig.GasTokenDecimals)), nil)
	const UUSDC_PER_USD = 1_000_000

	// convert eth price in usd to eth price in uusdc
	gasTokenPriceUUSDC := new(big.Float).Mul(big.NewFloat(gasTokenPriceUSD), new(big.Float).SetInt64(UUSDC_PER_USD))

	// gas token price in usd comes back from coin gecko with two decimals.
	// Since we just converted to uusdc, shifting the decimal place right by 6,
	// we can safely turn this into an int now
	gasTokenPriceUUSDCInt, ok := new(big.Int).SetString(gasTokenPriceUUSDC.String(), 10)
	if !ok {
		return nil, fmt.Errorf("converting %s price in uusdc %s to *big.Int", chainConfig.GasTokenCoingeckoID, gasTokenPriceUUSDC.String())
	}

	// What we are really trying to do is:
	//   gas token price uusdc / smallest gas token conversion = smallest gas token representation price in uusdc
	//   smallest gas token representation price in uusdc * tx fee = tx fee uusdc
	// However we are choosing to first multiply gas token price uusdc by tx
	// fee so that we can do integer division when converting to smallest
	// representation, since if we first do integer division (before
	// multiplying), we are going to cut off necessary decimals. there are
	// limits of this, if gas token price uusdc * tx fee has less than 9
	// digits, then we will just return 0. However, this is unlikely in
	// practice and the tx fee would be very small if this is the case.
	tmp := new(big.Int).Mul(gasTokenPriceUUSDCInt, txFee)
	txFeeUUSDC := new(big.Int).Div(tmp, smallestGasTokenConversion)
	return txFeeUUSDC, nil
}
