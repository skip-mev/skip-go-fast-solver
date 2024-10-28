package evmrpc

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
)

const (
	coingeckoEthID       = "ethereum"
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
	return o.gasCostUUSDC(ctx, estimatedPricePerGas, big.NewInt(int64(estimatedGasUsed)))
}

// gasCostUUSDC converts an amount of gas and the price per gas in gwei to
// uusdc based on the current CoinGecko price of ethereum in usd.
func (o *Oracle) gasCostUUSDC(ctx context.Context, pricePerGasGwei *big.Int, gasUsed *big.Int) (*big.Int, error) {
	txFeeGwei := new(big.Float).SetInt(new(big.Int).Mul(gasUsed, pricePerGasGwei))
	fmt.Println("tx fee gwei", txFeeGwei.String())

	// get the price of eth in usd from coin gecko and convert to gwei
	ethPriceUSD, err := o.coingecko.GetSimplePrice(ctx, coingeckoEthID, coingeckoUSDCurrency)
	if err != nil {
		return nil, fmt.Errorf("getting coin gecko price of ethereum in USD: %w", err)
	}
	const GWEI_PER_ETH = 1000000000
	gweiPriceUSD := new(big.Float).Quo(big.NewFloat(ethPriceUSD), big.NewFloat(GWEI_PER_ETH))

	// get the tx fee in usd and convert to uusdc
	txFeeUSD := new(big.Float).Mul(txFeeGwei, gweiPriceUSD)

	// assuming 1usd == 1usdc
	const UUSDC_PER_USDC = 1000000
	txFeeUUSDC := new(big.Float).Mul(txFeeUSD, big.NewFloat(UUSDC_PER_USDC))

	// we may be off by 1 uusdc in either direction here due to floating point
	// numbers being annoying
	uusdcFee, _ := txFeeUUSDC.Int(nil)
	return uusdcFee, nil
}
