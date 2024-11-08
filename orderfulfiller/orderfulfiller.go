package orderfulfiller

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/orderfulfiller/order_fulfillment_handler"
	"github.com/skip-mev/go-fast-solver/orderfulfiller/orderqueue"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/skip-mev/go-fast-solver/shared/lmt"
)

const (
	requeueDelay                 = 30 * time.Second
	orderQueueCapacity           = 100
	pendingOrderDispatchInterval = 1 * time.Second
	timeoutInterval              = 10 * time.Second
)

type OrderFulfillmentHandler interface {
	UpdateFulfillmentStatus(ctx context.Context, order db.Order) (fulfillmentStatus string, err error)
	FillOrder(ctx context.Context, order db.Order) (string, error)
	InitiateTimeout(ctx context.Context, order db.Order, tx order_fulfillment_handler.Database) (string, error)
	SubmitTimeoutForRelay(ctx context.Context, order db.Order, txHash string, tx order_fulfillment_handler.Database) error
}

type Database interface {
	GetAllOrdersWithOrderStatus(ctx context.Context, orderStatus string) ([]db.Order, error)
	InTx(ctx context.Context, fn func(ctx context.Context, q db.Querier) error, opts *sql.TxOptions) error
}

type OrderFulfiller struct {
	db                   Database
	ordersQueue          *orderqueue.OrderQueue
	fillHandler          OrderFulfillmentHandler
	orderFillWorkerCount int
	shouldFillOrders     bool
	shouldRefundOrders   bool
}

func NewOrderFulfiller(ctx context.Context, db Database, orderFulfillmentWorkerCount int, orderFulfillmentHandler OrderFulfillmentHandler, shouldFillOrders, shouldRefundOrders bool) (*OrderFulfiller, error) {
	workerCount := orderFulfillmentWorkerCount
	if workerCount <= 0 {
		workerCount = 1
	}
	return &OrderFulfiller{
		db:                   db,
		ordersQueue:          orderqueue.NewOrderQueue(ctx, requeueDelay, orderQueueCapacity),
		fillHandler:          orderFulfillmentHandler,
		orderFillWorkerCount: workerCount,
		shouldFillOrders:     shouldFillOrders,
		shouldRefundOrders:   shouldRefundOrders,
	}, nil
}

func (r *OrderFulfiller) Run(ctx context.Context) {
	if r.shouldRefundOrders {
		go r.startOrderTimeoutWorker(ctx)
	}
	if r.shouldFillOrders {
		go r.startOrderFillWorkers(ctx)
		r.dispatchOrderFills(ctx)
	}
}

func (r *OrderFulfiller) dispatchOrderFills(ctx context.Context) {
	ticker := time.NewTicker(pendingOrderDispatchInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := r.db.GetAllOrdersWithOrderStatus(ctx, dbtypes.OrderStatusPending)
			if err != nil {
				lmt.Logger(ctx).Error("error getting pending orders", zap.Error(err))
				continue
			}
			for _, order := range orders {
				// we continuously try and push pending orders onto the queue
				// so we don't need to check whether the order was successfully queued
				_ = r.ordersQueue.QueueOrder(order)
			}
		}
	}
}

func (r *OrderFulfiller) startOrderTimeoutWorker(ctx context.Context) {
	ticker := time.NewTicker(timeoutInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			orders, err := r.db.GetAllOrdersWithOrderStatus(ctx, dbtypes.OrderStatusExpiredPendingRefund)
			if err != nil {
				lmt.Logger(ctx).Error("error getting expired orders", zap.Error(err))
				continue
			}

			for _, order := range orders {
				fulfillmentStatus, err := r.fillHandler.UpdateFulfillmentStatus(ctx, order)
				if err != nil {
					lmt.Logger(ctx).Warn(
						"error updating fulfillment status",
						zap.Error(err),
						zap.String("orderID", order.OrderID),
						zap.String("sourceChainID", order.SourceChainID),
						zap.String("destinationChainID", order.DestinationChainID),
					)
					continue
				}

				// do not try and refund this order
				if !r.shouldRefundOrders || fulfillmentStatus != dbtypes.OrderStatusExpiredPendingRefund {
					continue
				}

				err = r.db.InTx(ctx, func(ctx context.Context, q db.Querier) error {
					txHash, err := r.fillHandler.InitiateTimeout(ctx, order, q)
					if err != nil {
						return fmt.Errorf("initiating timeout for order %s with source chain %s and destination chain %s: %w", order.OrderID, order.SourceChainID, order.DestinationChainID, err)
					}
					if txHash == "" {
						return nil
					}

					if err = r.fillHandler.SubmitTimeoutForRelay(ctx, order, txHash, q); err != nil {
						lmt.Logger(ctx).Warn(
							"error submitting timeout to be relayed",
							zap.Error(err),
							zap.String("orderID", order.OrderID),
							zap.String("sourceChainID", order.SourceChainID),
							zap.String("destinationChainID", order.DestinationChainID),
						)
						return fmt.Errorf("submitting timeout to be relayed for order %s with source chain %s and destination chain %s: %w", order.OrderID, order.SourceChainID, order.DestinationChainID, err)
					}

					lmt.Logger(ctx).Info(
						"successfully initiated timeout",
						zap.String("orderID", order.OrderID),
						zap.String("sourceChainID", order.SourceChainID),
						zap.String("destinationChainID", order.DestinationChainID),
					)
					return nil
				}, nil)
				if err != nil {
					lmt.Logger(ctx).Warn("error initiating timeout", zap.Error(err))
				}
			}
		}
	}
}

func (r *OrderFulfiller) startOrderFillWorkers(ctx context.Context) {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(r.orderFillWorkerCount)
	for i := 0; i < r.orderFillWorkerCount; i++ {
		eg.Go(func() error {
			for {
				select {
				case order := <-r.ordersQueue.PopOrder():
					if fulfillmentStatus, err := r.fillHandler.UpdateFulfillmentStatus(egCtx, order); err != nil {
						lmt.Logger(ctx).Warn(
							"error updating fulfillment status",
							zap.Error(err),
							zap.String("orderID", order.OrderID),
							zap.String("sourceChainID", order.SourceChainID),
						)
					} else if fulfillmentStatus == dbtypes.OrderStatusPending && r.shouldFillOrders {
						hash, err := r.fillHandler.FillOrder(ctx, order)
						if err != nil {
							lmt.Logger(ctx).Warn(
								"error filling order",
								zap.Error(err),
								zap.String("orderID", order.OrderID),
								zap.String("sourceChainID", order.SourceChainID),
							)
						} else if hash != "" {
							lmt.Logger(ctx).Info(
								"successfully filled order",
								zap.String("orderID", order.OrderID),
								zap.String("sourceChainID", order.SourceChainID),
								zap.String("txHash", hash),
							)
						}
					}
				case <-egCtx.Done():
					return nil
				}
			}
		})
	}
	if err := eg.Wait(); err != nil {
		lmt.Logger(ctx).Error("error processing orders", zap.Error(err))
	}
}
