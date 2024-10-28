package db

const (
	OrderStatusPending              string = "PENDING"
	OrderStatusFilled               string = "FILLED"
	OrderStatusExpiredPendingRefund string = "EXPIRED_PENDING_REFUND"
	OrderStatusRefunded             string = "REFUNDED"
	OrderStatusAbandoned            string = "ABANDONED"

	SettlementStatusPending             string = "PENDING"
	SettlementStatusSettlementInitiated string = "SETTLEMENT_INITIATED"
	SettlementStatusComplete            string = "COMPLETE"
	SettlementStatusFailed              string = "FAILED"

	TxStatusPending   string = "PENDING"
	TxStatusSuccess   string = "SUCCESS"
	TxStatusFailed    string = "FAILED"
	TxStatusAbandoned string = "ABANDONED"

	TxTypeOrderFill                string = "ORDER_FILL"
	TxTypeSettlement               string = "SETTLEMENT"
	TxTypeHyperlaneMessageDelivery string = "HYPERLANE_MESSAGE_DELIVERY"
	TxTypeInitiateTimeout          string = "INITIATE_TIMEOUT"

	RebalanceTransactionStatusPending string = "PENDING"
	RebalanceTransactionStatusSuccess string = "SUCCESS"
	RebalanceTransactionStatusFailed  string = "FAILED"

	TransferStatusPending   string = "PENDING"
	TransferStatusSuccess   string = "SUCCESS"
	TransferStatusAbandoned string = "ABANDONED"

	GET    string = "GET"
	INSERT string = "INSERT"
	UPDATE string = "UPDATE"
)
