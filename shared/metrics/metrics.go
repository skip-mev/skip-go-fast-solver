package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/metrics"
	prom "github.com/go-kit/kit/metrics/prometheus"
	stdprom "github.com/prometheus/client_golang/prometheus"
)

const (
	chainIDLabel            = "chain_id"
	sourceChainIDLabel      = "source_chain_id"
	destinationChainIDLabel = "destination_chain_id"
	successLabel            = "success"
	orderStatusLabel        = "order_status"
	transferStatusLabel     = "transfer_status"
	settlementStatusLabel   = "settlement_status"
	transactionTypeLabel    = "transaction_type"
)

type Metrics interface {
	IncTransactionSubmitted(success bool, chainID, transactionType string)
	IncTransactionVerified(success bool, chainID string)

	IncFillOrderStatusChange(sourceChainID, destinationChainID, orderStatus string)
	ObserveFillLatency(sourceChainID, destinationChainID string, orderStatus string, latency time.Duration)

	IncOrderSettlementStatusChange(sourceChainID, destinationChainID, settlementStatus string)
	ObserveSettlementLatency(sourceChainID, destinationChainID string, settlementStatus string, latency time.Duration)

	IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID string, transferStatus string)

	IncHyperlaneCheckpointingErrors()
	IncHyperlaneMessages(sourceChainID, destinationChainID string, messageStatus string)
	DecHyperlaneMessages(sourceChainID, destinationChainID string, messageStatus string)
	ObserveHyperlaneLatency(sourceChainID, destinationChainID, transferStatus string, latency time.Duration)

	ObserveTransferSizeOutOfRange(sourceChainID, destinationChainID string, amountExceededBy int64)
	ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBpsExceededBy int64)
	ObserveInsufficientBalanceError(chainID string, amountInsufficientBy uint64)
}

type metricsContextKey struct{}

func ContextWithMetrics(ctx context.Context, metrics Metrics) context.Context {
	return context.WithValue(ctx, metricsContextKey{}, metrics)
}

func FromContext(ctx context.Context) Metrics {
	metricsFromContext := ctx.Value(metricsContextKey{})
	if metricsFromContext == nil {
		return NewNoOpMetrics()
	} else {
		return metricsFromContext.(Metrics)
	}
}

var _ Metrics = (*PromMetrics)(nil)

type PromMetrics struct {
	totalTransactionSubmitted metrics.Counter
	totalTransactionsVerified metrics.Counter

	fillOrderStatusChange metrics.Counter
	fillLatency           metrics.Histogram

	orderSettlementStatusChange metrics.Counter
	settlementLatency           metrics.Histogram

	fundRebalanceTransferStatusChange metrics.Counter

	hplMessages            metrics.Gauge
	hplCheckpointingErrors metrics.Counter
	hplLatency             metrics.Histogram

	transferSizeOutOfRange    metrics.Histogram
	feeBpsRejections          metrics.Histogram
	insufficientBalanceErrors metrics.Histogram
}

func NewPromMetrics() Metrics {
	return &PromMetrics{
		fillOrderStatusChange: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "fill_order_status_change_counter",
			Help:      "numbers of fill order status changes, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, orderStatusLabel}),
		orderSettlementStatusChange: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "order_settlement_status_change_counter",
			Help:      "numbers of order settlement status changes, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, settlementStatusLabel}),
		fundRebalanceTransferStatusChange: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "funds_rebalance_transfer_status_change_counter",
			Help:      "numbers of funds rebalance transfer status changes, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, transferStatusLabel}),
		totalTransactionSubmitted: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "total_transactions_submitted_counter",
			Help:      "number of transactions submitted, paginated by success status and source and destination chain id",
		}, []string{successLabel, chainIDLabel, transactionTypeLabel}),
		totalTransactionsVerified: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "total_transactions_verified_counter",
			Help:      "number of transactions verified, paginated by success status and chain id",
		}, []string{successLabel, chainIDLabel}),
		fillLatency: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "latency_per_fill_minutes",
			Help:      "latency from source transaction to fill completion, paginated by source and destination chain id (in minutes)",
			Buckets:   []float64{5, 15, 30, 60, 120, 180},
		}, []string{sourceChainIDLabel, destinationChainIDLabel, orderStatusLabel}),
		settlementLatency: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "latency_per_settlement_minutes",
			Help:      "latency from source transaction to fill completion, paginated by source and destination chain id (in minutes)",
			Buckets:   []float64{5, 15, 30, 60, 120, 180},
		}, []string{sourceChainIDLabel, destinationChainIDLabel, settlementStatusLabel}),
		hplMessages: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "hyperlane_messages",
			Help:      "number of hyperlane messages, paginated by source and destination chain, and message status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, transferStatusLabel}),

		hplCheckpointingErrors: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "hyperlane_checkpointing_errors",
			Help:      "number of hyperlane checkpointing errors",
		}, []string{}),
		hplLatency: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "latency_per_hpl_message_seconds",
			Help:      "latency for hyperlane message relaying, paginated by status, source and destination chain id (in seconds)",
			Buckets:   []float64{30, 60, 300, 600, 900, 1200, 1500, 1800, 2400, 3000, 3600},
		}, []string{sourceChainIDLabel, destinationChainIDLabel, transferStatusLabel}),
		transferSizeOutOfRange: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "transfer_size_out_of_range",
			Help:      "histogram of transfer sizes that were out of min/max fill size constraints",
			Buckets: []float64{
				-1000000000,   // -1,000 USDC
				-100000000,    // -100 USDC
				-10000000,     // -10 USDC
				100000000,     // 100 USDC
				1000000000,    // 1,000 USDC
				10000000000,   // 10,000 USDC
				100000000000,  // 100,000 USDC
				1000000000000, // 1,000,000 USDC
			},
		}, []string{sourceChainIDLabel, destinationChainIDLabel}),
		feeBpsRejections: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "fee_bps_rejections",
			Help:      "histogram of fee bps that were rejected for being too low",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 200, 500, 1000},
		}, []string{sourceChainIDLabel, destinationChainIDLabel}),
		insufficientBalanceErrors: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "insufficient_balance_errors",
			Help:      "histogram of fill orders that exceeded available balance",
			Buckets: []float64{
				100000000,     // 100 USDC
				1000000000,    // 1,000 USDC
				10000000000,   // 10,000 USDC
				100000000000,  // 100,000 USDC
				1000000000000, // 1,000,000 USDC
			},
		}, []string{chainIDLabel}),
	}
}

func (m *PromMetrics) IncTransactionSubmitted(success bool, chainID, transactionType string) {
	m.totalTransactionSubmitted.With(successLabel, fmt.Sprint(success), chainIDLabel, chainID, transactionTypeLabel, transactionType).Add(1)
}

func (m *PromMetrics) IncTransactionVerified(success bool, chainID string) {
	m.totalTransactionsVerified.With(successLabel, fmt.Sprint(success), chainIDLabel, chainID).Add(1)
}

func (m *PromMetrics) ObserveFillLatency(sourceChainID, destinationChainID, orderStatus string, latency time.Duration) {
	m.fillLatency.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, orderStatusLabel, orderStatus).Observe(latency.Minutes())
}

func (m *PromMetrics) ObserveSettlementLatency(sourceChainID, destinationChainID, settlementStatus string, latency time.Duration) {
	m.settlementLatency.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, settlementStatusLabel, settlementStatus).Observe(latency.Minutes())
}

func (m *PromMetrics) ObserveHyperlaneLatency(sourceChainID, destinationChainID, transferStatus string, latency time.Duration) {
	m.hplLatency.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, transferStatus).Observe(latency.Seconds())
}

func (m *PromMetrics) IncFillOrderStatusChange(sourceChainID, destinationChainID, orderStatus string) {
	m.fillOrderStatusChange.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, orderStatusLabel, orderStatus).Add(1)
}

func (m *PromMetrics) IncOrderSettlementStatusChange(sourceChainID, destinationChainID, settlementStatus string) {
	m.orderSettlementStatusChange.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, settlementStatusLabel, settlementStatus).Add(1)
}

func (m *PromMetrics) IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID, transferStatus string) {
	m.fundRebalanceTransferStatusChange.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, transferStatus).Add(1)
}

func (m *PromMetrics) IncHyperlaneCheckpointingErrors() {
	m.hplCheckpointingErrors.Add(1)
}
func (m *PromMetrics) IncHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {
	m.hplMessages.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, messageStatus).Add(1)
}
func (m *PromMetrics) DecHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {
	m.hplMessages.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, messageStatus).Add(-1)
}

func (m *PromMetrics) ObserveTransferSizeOutOfRange(sourceChainID, destinationChainID string, transferSize int64) {
	m.transferSizeOutOfRange.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
	).Observe(float64(transferSize))
}

func (m *PromMetrics) ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBps int64) {
	m.feeBpsRejections.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
	).Observe(float64(feeBps))
}

func (m *PromMetrics) ObserveInsufficientBalanceError(chainID string, transferSizeOutOfRangeBy uint64) {
	m.insufficientBalanceErrors.With(
		chainIDLabel, chainID,
	).Observe(float64(transferSizeOutOfRangeBy))
}

type NoOpMetrics struct{}

func (n NoOpMetrics) ObserveInsufficientBalanceError(chainID string, amountInsufficientBy uint64) {
}

func (n NoOpMetrics) IncTransactionSubmitted(success bool, chainID, transactionType string) {
}
func (n NoOpMetrics) IncTransactionVerified(success bool, chainID string) {
}
func (n NoOpMetrics) ObserveFillLatency(sourceChainID, destinationChainID, orderStatus string, latency time.Duration) {
}
func (n NoOpMetrics) ObserveSettlementLatency(sourceChainID, destinationChainID, settlementStatus string, latency time.Duration) {
}
func (n NoOpMetrics) ObserveHyperlaneLatency(sourceChainID, destinationChainID, orderstatus string, latency time.Duration) {
}
func (n NoOpMetrics) IncFillOrderStatusChange(sourceChainID, destinationChainID, orderStatus string) {
}
func (n NoOpMetrics) IncOrderSettlementStatusChange(sourceChainID, destinationChainID, settlementStatus string) {
}
func (n NoOpMetrics) IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID, transferStatus string) {
}
func (n NoOpMetrics) IncHyperlaneCheckpointingErrors()                                             {}
func (n NoOpMetrics) IncHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {}
func (n NoOpMetrics) DecHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {}
func (n NoOpMetrics) ObserveTransferSizeOutOfRange(sourceChainID, destinationChainID string, amountExceededBy int64) {
}
func (n NoOpMetrics) ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBps int64) {}
func NewNoOpMetrics() Metrics {
	return &NoOpMetrics{}
}
