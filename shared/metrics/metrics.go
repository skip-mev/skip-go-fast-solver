package metrics

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/go-kit/kit/metrics"
	prom "github.com/go-kit/kit/metrics/prometheus"
	stdprom "github.com/prometheus/client_golang/prometheus"
)

const (
	chainIDLabel            = "chain_id"
	chainNameLabel          = "chain_name"
	sourceChainIDLabel      = "source_chain_id"
	destinationChainIDLabel = "destination_chain_id"
	successLabel            = "success"
	orderStatusLabel        = "order_status"
	transferStatusLabel     = "transfer_status"
	settlementStatusLabel   = "settlement_status"
	operationLabel          = "operation"
	gasBalanceLevelLabel    = "gas_balance_level"
	gasTokenSymbolLabel     = "gas_token_symbol"
)

type Metrics interface {
	IncTransactionSubmitted(success bool, sourceChainID, destinationChainID string)
	IncTransactionVerified(success bool, chainID string)

	IncFillOrders(sourceChainID, destinationChainID, orderStatus string)
	DecFillOrders(sourceChainID, destinationChainID, orderStatus string)
	ObserveFillLatency(sourceChainID, destinationChainID, orderStatus string, latency time.Duration)

	IncOrderSettlements(sourceChainID, destinationChainID, settlementStatus string)
	DecOrderSettlements(sourceChainID, destinationChainID, settlementStatus string)
	ObserveSettlementLatency(sourceChainID, destinationChainID, settlementStatus string, latency time.Duration)

	IncFundsRebalanceTransfers(sourceChainID, destinationChainID, transferStatus string)
	DecFundsRebalanceTransfers(sourceChainID, destinationChainID, transferStatus string)

	IncHyperlaneCheckpointingErrors()
	IncHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string)
	DecHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string)
	ObserveHyperlaneLatency(sourceChainID, destinationChainID, transferStatus string, latency time.Duration)

	ObserveTransferSizeExceeded(sourceChainID, destinationChainID string, amountExceededBy uint64)
	ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBpsExceededBy int64)

	IncDatabaseErrors(operation string)

	SetGasBalance(chainID, chainName, gasTokenSymbol string, gasBalance, warningThreshold, criticalThreshold big.Int, gasTokenDecimals uint8)
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

	fillOrders  metrics.Gauge
	fillLatency metrics.Histogram

	settlements       metrics.Gauge
	settlementLatency metrics.Histogram

	fundsRebalanceTransfers metrics.Gauge

	hplMessages            metrics.Gauge
	hplCheckpointingErrors metrics.Counter
	hplLatency             metrics.Histogram

	amountTransferSizeExceeded metrics.Histogram
	feeBpsRejections           metrics.Histogram

	databaseErrors metrics.Counter
	gasBalance     metrics.Gauge
}

func NewPromMetrics() Metrics {
	return &PromMetrics{
		fillOrders: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "fill_orders",
			Help:      "numbers of fill orders, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, orderStatusLabel}),
		settlements: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "settlements",
			Help:      "numbers of settlements intitiated, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, settlementStatusLabel}),
		fundsRebalanceTransfers: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "funds_rebalance_transfers",
			Help:      "numbers of funds rebalance transfers, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, transferStatusLabel}),

		totalTransactionSubmitted: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "total_transactions_submitted_counter",
			Help:      "number of transactions submitted, paginated by success status and source and destination chain id",
		}, []string{successLabel, sourceChainIDLabel, destinationChainIDLabel}),
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
		amountTransferSizeExceeded: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "amount_transfer_size_exceeded",
			Help:      "histogram of fill orders that exceeded max fill size",
			Buckets: []float64{
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
		databaseErrors: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "database_errors_total",
			Help:      "number of errors encountered when making database calls",
		}, []string{}),
		gasBalance: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "gas_balance_gauge",
			Help:      "gas balances, paginated by chain id and gas balance level",
		}, []string{chainIDLabel, chainNameLabel, gasTokenSymbolLabel, gasBalanceLevelLabel}),
	}
}

func (m *PromMetrics) IncTransactionSubmitted(success bool, sourceChainID, destinationChainID string) {
	m.totalTransactionSubmitted.With(successLabel, fmt.Sprint(success), sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID).Add(1)
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

func (m *PromMetrics) IncFillOrders(sourceChainID, destinationChainID, orderStatus string) {
	m.fillOrders.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, orderStatusLabel, orderStatus).Add(1)
}

func (m *PromMetrics) DecFillOrders(sourceChainID, destinationChainID, orderStatus string) {
	m.fillOrders.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, orderStatusLabel, orderStatus).Add(-1)
}

func (m *PromMetrics) IncOrderSettlements(sourceChainID, destinationChainID, settlementStatus string) {
	m.settlements.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, settlementStatusLabel, settlementStatus).Add(1)
}

func (m *PromMetrics) DecOrderSettlements(sourceChainID, destinationChainID, settlementStatus string) {
	m.settlements.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, settlementStatusLabel, settlementStatus).Add(-1)
}

func (m *PromMetrics) IncFundsRebalanceTransfers(sourceChainID, destinationChainID, transferStatus string) {
	m.fundsRebalanceTransfers.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, transferStatus).Add(1)
}

func (m *PromMetrics) DecFundsRebalanceTransfers(sourceChainID, destinationChainID, transferStatus string) {
	m.fundsRebalanceTransfers.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, transferStatus).Add(-1)
}

func (m *PromMetrics) IncHyperlaneCheckpointingErrors() {
	m.hplCheckpointingErrors.Add(1)
}
func (m *PromMetrics) IncHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {
	m.hplMessages.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, messageStatus).Add(1)
}
func (m *PromMetrics) DecHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {
	m.hplMessages.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, messageStatus).Add(-1)
}

func (m *PromMetrics) ObserveTransferSizeExceeded(sourceChainID, destinationChainID string, transferSize uint64) {
	m.amountTransferSizeExceeded.With(
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

func (m *PromMetrics) IncDatabaseErrors(operation string) {
	m.databaseErrors.With(operationLabel, operation).Add(1)
}

func (m *PromMetrics) SetGasBalance(chainID, chainName, gasTokenSymbol string, gasBalance, warningThreshold, criticalThreshold big.Int, gasTokenDecimals uint8) {
	gasBalanceLevel := "ok"
	if gasBalance.Cmp(&warningThreshold) < 0 {
		gasBalanceLevel = "warning"
	}
	if gasBalance.Cmp(&criticalThreshold) < 0 {
		gasBalanceLevel = "critical"
	}
	// We compare the gas balance against thresholds locally rather than in the grafana alert definition since
	// the prometheus metric is exported as a float64 and the thresholds reach Wei amounts where precision is lost.
	gasBalanceFloat, _ := gasBalance.Float64()
	gasTokenAmount := gasBalanceFloat / (math.Pow10(int(gasTokenDecimals)))
	m.gasBalance.With(chainIDLabel, chainID, chainNameLabel, chainName, gasTokenSymbolLabel, gasTokenSymbol, gasBalanceLevelLabel, gasBalanceLevel).Set(gasTokenAmount)
}

type NoOpMetrics struct{}

func (n NoOpMetrics) IncTransactionSubmitted(success bool, sourceChainID, destinationChainID string) {
}
func (n NoOpMetrics) IncTransactionVerified(success bool, chainID string) {
}
func (n NoOpMetrics) ObserveFillLatency(sourceChainID, destinationChainID, orderStatus string, latency time.Duration) {
}
func (n NoOpMetrics) ObserveSettlementLatency(sourceChainID, destinationChainID, settlementStatus string, latency time.Duration) {
}
func (n NoOpMetrics) ObserveHyperlaneLatency(sourceChainID, destinationChainID, orderstatus string, latency time.Duration) {
}
func (n NoOpMetrics) IncFillOrders(sourceChainID, destinationChainID, orderStatus string) {}
func (n NoOpMetrics) DecFillOrders(sourceChainID, destinationChainID, orderStatus string) {}
func (n NoOpMetrics) IncOrderSettlements(sourceChainID, destinationChainID, settlementStatus string) {
}
func (n NoOpMetrics) DecOrderSettlements(sourceChainID, destinationChainID, settlementStatus string) {
}
func (n NoOpMetrics) IncFundsRebalanceTransfers(sourceChainID, destinationChainID, transferStatus string) {
}
func (n NoOpMetrics) DecFundsRebalanceTransfers(sourceChainID, destinationChainID, transferStatus string) {
}
func (n NoOpMetrics) IncHyperlaneCheckpointingErrors()                                             {}
func (n NoOpMetrics) IncHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {}
func (n NoOpMetrics) DecHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {}
func (n NoOpMetrics) ObserveTransferSizeExceeded(sourceChainID, destinationChainID string, transferSize uint64) {
}
func (n NoOpMetrics) IncDatabaseErrors(operation string)                                            {}
func (n NoOpMetrics) ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBps int64) {}
func (n *NoOpMetrics) SetGasBalance(chainID, chainName, gasTokenSymbol string, gasBalance, warningThreshold, criticalThreshold big.Int, gasTokenDecimals uint8) {
}

func NewNoOpMetrics() Metrics {
	return &NoOpMetrics{}
}
