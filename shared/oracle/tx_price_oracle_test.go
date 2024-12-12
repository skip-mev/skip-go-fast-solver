package oracle_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/mocks/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/oracle"
	"github.com/stretchr/testify/assert"
)

func Test_Oracle_TxFeeUUSDC(t *testing.T) {
	tests := []struct {
		Name               string
		MaxPricePerGas     uint64
		GasUsed            uint64
		ETHPriceUSD        float64
		ExpectedUUSDCPrice int64
	}{
		{
			Name: "1mil gas used, 21940000wei per gas, 2000usd per eth",
			// 21940000 * 1mil = 21,940,000,000,000 wei fee
			MaxPricePerGas: 21940000,
			GasUsed:        1000000,
			// price per wei in usd = 0.000000000000002000
			ETHPriceUSD: 2000,
			// price per gwei in usd * gwei fee = 0.04388
			// 0.04388 * 10000000 = 43880 uusdc
			ExpectedUUSDCPrice: 43880,
		},
		{
			// NOTE: this test is for very small numbers that are not
			// realistic. However, this is to test the limits of this function,
			// the assumptions we have about how many decimals numbers have
			// break down when the gas fee in gwei and eth price are this low.
			Name:               "1 gas used, 5wei per gas, 1.21usd per eth",
			MaxPricePerGas:     5,
			GasUsed:            1,
			ETHPriceUSD:        1.21,
			ExpectedUUSDCPrice: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx := context.Background()
			cfg := config.Config{
				Chains: map[string]config.ChainConfig{
					"ethereum": {ChainID: "1", GasTokenSymbol: "ETH", GasTokenDecimals: 18, GasTokenCoingeckoID: "ethereum"},
				},
			}
			ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID: big.NewInt(1),
				// max wei paid per gas
				GasFeeCap: big.NewInt(int64(tt.MaxPricePerGas)),
				// total gas used
				Gas: tt.GasUsed,
			})

			mockcoingecko := coingecko.NewMockPriceClient(t)
			mockcoingecko.EXPECT().GetSimplePrice(ctx, "ethereum", "usd").Return(tt.ETHPriceUSD, nil)

			oracle := oracle.NewOracle(mockcoingecko)
			uusdcPrice, err := oracle.TxFeeUUSDC(ctx, tx)
			assert.NoError(t, err)
			assert.Equal(t, tt.ExpectedUUSDCPrice, uusdcPrice.Int64())
		})
	}
}
