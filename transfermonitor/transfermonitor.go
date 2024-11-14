package transfermonitor

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	dbtypes "github.com/skip-mev/go-fast-solver/db"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	ethereumrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/tmrpc"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	maxBlocksProcessedPerIteration = 100000
	destinationChainID             = "osmosis-1"
	orderSubmittedEventSignature   = "0x59f858504f8d8ad967dd7453df850e265270474e364b7e2fbd3333e06efdbfc0"
)

type MonitorDBQueries interface {
	InsertTransferMonitorMetadata(ctx context.Context, arg db.InsertTransferMonitorMetadataParams) (db.TransferMonitorMetadatum, error)
	GetTransferMonitorMetadata(ctx context.Context, chainID string) (db.TransferMonitorMetadatum, error)
	InsertOrder(ctx context.Context, arg db.InsertOrderParams) (db.Order, error)
}

type TransferMonitor struct {
	db           MonitorDBQueries
	clients      map[string]*ethclient.Client
	tmRPCManager tmrpc.TendermintRPCClientManager
	quickStart   bool
}

func NewTransferMonitor(db MonitorDBQueries, quickStart bool) *TransferMonitor {
	return &TransferMonitor{
		db:           db,
		clients:      make(map[string]*ethclient.Client),
		tmRPCManager: tmrpc.NewTendermintRPCClientManager(),
		quickStart:   quickStart,
	}
}

func (t *TransferMonitor) Start(ctx context.Context) error {
	lmt.Logger(ctx).Info("Starting transfer monitor")

	evmChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_EVM)
	if err != nil {
		return fmt.Errorf("error getting EVM chains: %w", err)
	}

	for _, chain := range evmChains {
		if chain.FastTransferContractAddress == "" {
			continue
		}

		if err := t.subscribeToTransferIntents(ctx, chain); err != nil {
			lmt.Logger(ctx).Error("failed to subscribe to chain logs",
				zap.String("chainID", chain.ChainID),
				zap.Error(err))
			continue
		}
	}

	<-ctx.Done()
	return nil
}

func (t *TransferMonitor) findNewTransferIntentsOnEVMChain(ctx context.Context, chain config.ChainConfig, startBlockHeight uint64) ([]Order, uint64, error) {
	client, err := t.getClient(ctx, chain.ChainID)
	if err != nil {
		lmt.Logger(ctx).Error("Error getting client", zap.Error(err))
		return nil, 0, err
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		lmt.Logger(ctx).Error("Error fetching latest block", zap.Error(err))
		return nil, 0, err
	}

	endBlockHeight := math.Min(header.Number.Uint64(), startBlockHeight+maxBlocksProcessedPerIteration)

	fastTransferContractAddress := chain.FastTransferContractAddress
	fastTransferGateway, err := fast_transfer_gateway.NewFastTransferGateway(
		common.HexToAddress(fastTransferContractAddress),
		client,
	)
	if err != nil {
		lmt.Logger(ctx).Error("Error creating MessageTransmitter object", zap.Error(err))
		return nil, 0, err
	}

	orders, err := t.findTransferIntents(ctx, startBlockHeight, endBlockHeight, fastTransferGateway, client, chain.Environment, chain.ChainID)
	if err != nil {
		lmt.Logger(ctx).Error("Error finding burn transactions", zap.Error(err))
		return nil, 0, err
	}

	if orders != nil {
		orderCounts := make(map[string]int)
		for _, order := range orders {
			key := fmt.Sprintf("%s->%s", order.ChainID, order.DestinationChainID)
			orderCounts[key]++
		}

		for chainPair, numOfOrders := range orderCounts {
			lmt.Logger(ctx).Info("Fast transfer orders found",
				zap.String("source->destination", chainPair),
				zap.Int("numOfOrders", numOfOrders))
		}
	}
	return orders, endBlockHeight, nil
}

func (t *TransferMonitor) getClient(ctx context.Context, chainID string) (*ethclient.Client, error) {
	if _, ok := t.clients[chainID]; !ok {
		rpc, err := config.GetConfigReader(ctx).GetRPCEndpoint(chainID)
		if err != nil {
			return nil, err
		}

		basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(chainID)
		if err != nil {
			return nil, err
		}

		conn, err := ethereumrpc.DialContext(ctx, rpc)
		if err != nil {
			return nil, err
		}
		if basicAuth != nil {
			conn.SetHeader("Authorization", fmt.Sprintf("Basic %s", *basicAuth))
		}

		client := ethclient.NewClient(conn)
		t.clients[chainID] = client
	}

	return t.clients[chainID], nil
}

type Order struct {
	TxHash             string                                  `json:"tx_hash"`
	TxBlockHeight      uint64                                  `json:"tx_block_height"`
	ChainID            string                                  `json:"chain_id"`
	DestinationChainID string                                  `json:"destination_chain_id"`
	ChainEnvironment   config.ChainEnvironment                 `json:"chain_environment"`
	OrderEvent         fast_transfer_gateway.FastTransferOrder `json:"order_event"`
	OrderID            string                                  `json:"order_id"`
	TimeoutTimestamp   int64                                   `json:"timeout_timestamp"`
}

func (t *TransferMonitor) findTransferIntents(
	ctx context.Context,
	startBlock,
	endBlock uint64,
	fastTransferGateway *fast_transfer_gateway.FastTransferGateway,
	client *ethclient.Client,
	chainEnvironment config.ChainEnvironment,
	chainID string,
) (orders []Order, err error) {
	offset := uint64(0)
	limit := uint64(1000)
	m := sync.Mutex{}
	eg, egctx := errgroup.WithContext(ctx)
	eg.SetLimit(20)
OuterLoop:
	for {
		select {
		case <-egctx.Done():
			return nil, nil
		default:
			start := startBlock + offset
			end := startBlock + offset + limit
			if start > endBlock {
				break OuterLoop
			}
			if end > endBlock {
				end = endBlock
			}
			eg.Go(func() error {
				var iter *fast_transfer_gateway.FastTransferGatewayOrderSubmittedIterator
				for i := 0; i < 5; i++ {
					iter, err = fastTransferGateway.FilterOrderSubmitted(&bind.FilterOpts{
						Context: ctx,
						Start:   start,
						End:     &[]uint64{end}[0],
					}, nil)
					if err != nil && i == 4 { // TODO dont retry on context cancellation
						return err
					}
					if err == nil {
						break
					}
					time.Sleep(1 * time.Second)
				}
				if iter == nil {
					return nil
				}

				for iter.Next() {
					m.Lock()
					orderData := decodeOrder(iter.Event.Order)
					orders = append(orders, Order{
						TxHash:             iter.Event.Raw.TxHash.Hex(),
						TxBlockHeight:      iter.Event.Raw.BlockNumber,
						ChainID:            chainID,
						DestinationChainID: destinationChainID,
						OrderEvent:         orderData,
						ChainEnvironment:   chainEnvironment,
						OrderID:            hex.EncodeToString(iter.Event.OrderID[:]),
						TimeoutTimestamp:   int64(orderData.TimeoutTimestamp),
					})
					m.Unlock()
				}

				if err := iter.Error(); err != nil {
					return err
				}

				return nil
			})
			offset += limit
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err := eg.Wait(); err != nil {
		lmt.Logger(egctx).Error("Error encountered while searching for transfers", zap.Error(err))
		return nil, err
	}
	return orders, nil
}

func decodeOrder(bytes []byte) fast_transfer_gateway.FastTransferOrder {
	var order fast_transfer_gateway.FastTransferOrder
	order.Sender = [32]byte(bytes[0:32])
	order.Recipient = [32]byte(bytes[32:64])
	order.AmountIn = new(big.Int).SetBytes(bytes[64:96])
	order.AmountOut = new(big.Int).SetBytes(bytes[96:128])
	order.Nonce = uint32(new(big.Int).SetBytes(bytes[128:132]).Uint64())
	order.SourceDomain = uint32(new(big.Int).SetBytes(bytes[132:136]).Uint64())
	order.DestinationDomain = uint32(new(big.Int).SetBytes(bytes[136:140]).Uint64())
	order.TimeoutTimestamp = new(big.Int).SetBytes(bytes[140:148]).Uint64()
	order.Data = bytes[148:]
	return order
}

func getChainID(chain config.ChainConfig) (string, error) {
	switch chain.Type {
	case config.ChainType_COSMOS:
		return chain.ChainID, nil
	case config.ChainType_EVM:
		return chain.ChainID, nil
	default:
		return "", fmt.Errorf("unknown chain type")
	}
}

func (t *TransferMonitor) getLatestBlockHeight(ctx context.Context, chain config.ChainConfig) (uint64, error) {
	switch chain.Type {
	case config.ChainType_EVM:
		client, err := t.getClient(ctx, chain.ChainID)
		if err != nil {
			return 0, err
		}
		header, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return 0, err
		}
		return header.Number.Uint64(), nil
	case config.ChainType_COSMOS:
		client, err := t.tmRPCManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			return 0, err
		}
		status, err := client.Status(ctx)
		if err != nil {
			return 0, err
		}
		return uint64(status.SyncInfo.LatestBlockHeight), nil
	default:
		return 0, fmt.Errorf("unsupported chain type: %s", chain.Type)
	}
}

func (t *TransferMonitor) getWsClient(ctx context.Context, chainID string) (*ethclient.Client, error) {
	if _, ok := t.clients[chainID]; !ok {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
		if err != nil {
			return nil, err
		}

		rpc := chainConfig.EVM.WsRPC
		if rpc == "" {
			return nil, fmt.Errorf("WebSocket RPC not configured for chainID %s", chainID)
		}

		conn, err := ethereumrpc.DialContext(ctx, rpc)
		if err != nil {
			return nil, err
		}

		basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(chainID)
		if err != nil {
			return nil, err
		}
		if basicAuth != nil {
			conn.SetHeader("Authorization", fmt.Sprintf("Basic %s", *basicAuth))
		}

		client := ethclient.NewClient(conn)
		t.clients[chainID] = client
	}

	return t.clients[chainID], nil
}

func (t *TransferMonitor) subscribeToTransferIntents(
	ctx context.Context,
	chain config.ChainConfig,
) error {
	client, err := t.getWsClient(ctx, chain.ChainID)
	if err != nil {
		return fmt.Errorf("getting rpc client: %w", err)
	}

	contractAddr := common.HexToAddress(chain.FastTransferContractAddress)
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddr},
		Topics: [][]common.Hash{{
			common.HexToHash(orderSubmittedEventSignature), // OrderSubmitted event signature
		}},
	}

	logs := make(chan types.Log)
	sub, err := client.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("subscribing to logs: %w", err)
	}

	go func() {
		for {
			select {
			case err := <-sub.Err():
				lmt.Logger(ctx).Error("subscription error", zap.Error(err))
				return
			case vLog := <-logs:
				fastTransferGateway, err := fast_transfer_gateway.NewFastTransferGateway(contractAddr, client)
				if err != nil {
					lmt.Logger(ctx).Error("creating contract instance", zap.Error(err))
					continue
				}

				event, err := fastTransferGateway.ParseOrderSubmitted(vLog)
				if err != nil {
					lmt.Logger(ctx).Error("parsing log", zap.Error(err))
					continue
				}

				order := Order{
					TxHash:             vLog.TxHash.Hex(),
					TxBlockHeight:      vLog.BlockNumber,
					ChainID:            chain.ChainID,
					DestinationChainID: destinationChainID,
					OrderEvent:         decodeOrder(event.Order),
					ChainEnvironment:   chain.Environment,
					OrderID:            hex.EncodeToString(event.OrderID[:]),
					TimeoutTimestamp:   int64(decodeOrder(event.Order).TimeoutTimestamp),
				}

				if err := t.insertOrder(ctx, order, chain.FastTransferContractAddress); err != nil {
					lmt.Logger(ctx).Error("inserting order", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (t *TransferMonitor) insertOrder(ctx context.Context, order Order, contractAddress string) error {
	toInsert := db.InsertOrderParams{
		SourceChainID:                     order.ChainID,
		DestinationChainID:                order.DestinationChainID,
		SourceChainGatewayContractAddress: contractAddress,
		Sender:                            order.OrderEvent.Sender[:],
		Recipient:                         order.OrderEvent.Recipient[:],
		AmountIn:                          order.OrderEvent.AmountIn.String(),
		AmountOut:                         order.OrderEvent.AmountOut.String(),
		Nonce:                             int64(order.OrderEvent.Nonce),
		OrderCreationTx:                   order.TxHash,
		OrderCreationTxBlockHeight:        int64(order.TxBlockHeight),
		OrderID:                           order.OrderID,
		OrderStatus:                       dbtypes.OrderStatusPending,
		TimeoutTimestamp:                  time.Unix(order.TimeoutTimestamp, 0).UTC(),
	}
	if len(order.OrderEvent.Data) > 0 {
		toInsert.Data = sql.NullString{String: hex.EncodeToString(order.OrderEvent.Data), Valid: true}
	}

	_, err := t.db.InsertOrder(ctx, toInsert)
	if err != nil {
		return fmt.Errorf("error inserting order: %w", err)
	}
	return nil
}
