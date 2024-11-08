package evmrpc_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/skip-mev/go-fast-solver/mocks/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
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
			Name: "1k gas used, 5gwei per gas, 2000usd per eth",
			// 5 * 1000 = 5000 gwei fee
			MaxPricePerGas: 5,
			GasUsed:        1000,
			// price per gwei in usd = 0.000002
			ETHPriceUSD: 2000,
			// price per gwei in usd * gwei fee = 0.01
			// 0.01 * 10000000 = 10000 uusdc
			ExpectedUUSDCPrice: 10000,
		},
		{
			Name: "150k gas used, 20gwei per gas, 2473.59usd per eth",
			// 20 * 150000 = 3000000 gwei fee
			MaxPricePerGas: 20,
			GasUsed:        150000,
			// price per gwei in usd = 0.00000247359
			ETHPriceUSD: 2473.59,
			// price per gwei in usd * gwei fee = 7.42077 usd
			// 7.42077 * 1000000 = 7420770 uusdc
			ExpectedUUSDCPrice: 7420770,
		},
		{
			Name: "10 gas used, 5gwei per gas, 2473.59usd per eth",
			// 5 * 10 = 50 gwei fee
			MaxPricePerGas: 5,
			GasUsed:        10,
			// price per gwei in usd = 0.00000247359
			ETHPriceUSD: 2473.59,
			// price per gwei in usd * gwei fee = 0.00012368 usd
			// 0.00012368 * 1000000 = 2473 uusdc
			ExpectedUUSDCPrice: 123,
		},
		{
			Name: "100 gas used, 5gwei per gas, 2473.59usd per eth",
			// 5 * 20 = 100 gwei fee
			MaxPricePerGas: 5,
			GasUsed:        20,
			// price per gwei in usd = 0.00000247359
			ETHPriceUSD: 2473.59,
			// price per gwei in usd * gwei fee = 0.00024736 usd
			// 0.00012368 * 1000000 = 2473 uusdc
			ExpectedUUSDCPrice: 247,
		},
		{
			Name: "100 gas used, 5gwei per gas, 10.21usd per eth",
			// 5 * 20 = 100 gwei fee
			MaxPricePerGas: 5,
			GasUsed:        20,
			// price per gwei in usd = 0.00000001021
			ETHPriceUSD: 10.21,
			// price per gwei in usd * gwei fee = 0.000001021 usd
			// 0.00012368 * 1000000 = 2473 uusdc
			ExpectedUUSDCPrice: 1,
		},
		{
			// NOTE: this test is for very small numbers that are not
			// realistic. However, this is to test the limits of this function,
			// the assumptions we have about how many decimals numbers have
			// break down when the gas fee in gwei and eth price are this low.
			Name: "1 gas used, 5gwei per gas, 1.21usd per eth",
			// 5 * 1 = 5 gwei fee
			MaxPricePerGas: 5,
			GasUsed:        1,
			// price per gwei in usd = 0.00000001021
			ETHPriceUSD: 1.21,
			// price per gwei in usd * gwei fee = 0.000001021 usd
			// 0.00012368 * 1000000 = 2473 uusdc
			ExpectedUUSDCPrice: 0,
		},
		{
			Name: "100 gas used, 5gwei per gas, 10.21usd per eth",
			// 20000000 * 1014806 = 20,296,120,000,000 gwei fee
			MaxPricePerGas: 20000000,
			GasUsed:        1014806,
			// price per gwei in usd = 0.0000028295
			ETHPriceUSD: 2829.46,
			// price per gwei in usd * gwei fee = 57,427,871.54 usd
			// 0.00012368 * 1000000 = 2473 uusdc
			ExpectedUUSDCPrice: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx := context.Background()
			tx := types.NewTx(&types.DynamicFeeTx{
				// max gwei paid per gas
				GasFeeCap: big.NewInt(int64(tt.MaxPricePerGas)),
				// total gas used
				Gas: tt.GasUsed,
			})

			mockcoingecko := coingecko.NewMockPriceClient(t)
			mockcoingecko.EXPECT().GetSimplePrice(ctx, "ethereum", "usd").Return(tt.ETHPriceUSD, nil)

			oracle := evmrpc.NewOracle(mockcoingecko)
			uusdcPrice, err := oracle.TxFeeUUSDC(ctx, tx)
			assert.NoError(t, err)
			assert.Equal(t, tt.ExpectedUUSDCPrice, uusdcPrice.Int64())
		})
	}
}
