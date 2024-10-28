package order_fulfillment_handler_test

import (
	"context"
	"testing"

	handler "github.com/skip-mev/go-fast-solver/orderfulfiller/order_fulfillment_handler"
	"github.com/stretchr/testify/assert"
)

func Test_CheckFeeAmount(t *testing.T) {
	tests := []struct {
		Name         string
		MinFeeBps    int64
		AmountIn     string
		AmountOut    string
		ShouldFill   bool
		ExpectedDiff int64
	}{
		{
			Name:         "100 bps min, 100 in 99 out",
			MinFeeBps:    100,
			AmountIn:     "100",
			AmountOut:    "99",
			ShouldFill:   true,
			ExpectedDiff: 0,
		},
		{
			Name:         "200 bps min, 100 in 99 out",
			MinFeeBps:    200,
			AmountIn:     "100",
			AmountOut:    "99",
			ShouldFill:   false,
			ExpectedDiff: 100,
		},
		{
			Name:         "100 bps min, 100 in 50 out",
			MinFeeBps:    100,
			AmountIn:     "100",
			AmountOut:    "50",
			ShouldFill:   true,
			ExpectedDiff: -4900,
		},
		{
			Name:         "1 bps min, 2 in 1 out",
			MinFeeBps:    1,
			AmountIn:     "2",
			AmountOut:    "1",
			ShouldFill:   true,
			ExpectedDiff: -4999,
		},
		{
			Name:         "100 bps min, 5mil in 4.99mil out",
			MinFeeBps:    100,
			AmountIn:     "5000000",
			AmountOut:    "4990000",
			ShouldFill:   false,
			ExpectedDiff: 80,
		},
		{
			Name:         "100 bps min, 5mil in 4.95mil out",
			MinFeeBps:    100,
			AmountIn:     "5000000",
			AmountOut:    "4950000",
			ShouldFill:   true,
			ExpectedDiff: 0,
		},
		{
			Name:         "200 bps min, 5mil in 4.95mil out",
			MinFeeBps:    200,
			AmountIn:     "5000000",
			AmountOut:    "4950000",
			ShouldFill:   false,
			ExpectedDiff: 100,
		},
		{
			Name:         "200 bps min, 5mil in 4.9mil out",
			MinFeeBps:    200,
			AmountIn:     "5000000",
			AmountOut:    "4900000",
			ShouldFill:   true,
			ExpectedDiff: 0,
		},
		{
			Name:         "0 bps min, 100 in 100 out",
			MinFeeBps:    0,
			AmountIn:     "100",
			AmountOut:    "100",
			ShouldFill:   true,
			ExpectedDiff: 0,
		},
		{
			Name:         "0 bps min, 100 in 99 out",
			MinFeeBps:    0,
			AmountIn:     "100",
			AmountOut:    "99",
			ShouldFill:   true,
			ExpectedDiff: -100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			shouldFill, bpsDiff, err := handler.IsWithinBpsRange(context.Background(), tt.MinFeeBps, tt.AmountIn, tt.AmountOut)
			assert.NoError(t, err)
			assert.Equal(t, tt.ShouldFill, shouldFill)
			assert.Equal(t, tt.ExpectedDiff, bpsDiff, "BPS difference mismatch")
		})
	}
}
