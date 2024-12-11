package cosmosgrpc

import (
	"fmt"
	cosmosgrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
)

type CosmosGRPCClientConnManager interface {
	GetClient(ctx context.Context, chainID string) (cosmosgrpc.ClientConn, error)
}

type tendermintRPCClientManagerImpl struct {
	clientConns map[string]cosmosgrpc.ClientConn
	m           sync.Mutex
}

func NewCosmosGRPCClientConnManager() CosmosGRPCClientConnManager {
	m := &tendermintRPCClientManagerImpl{
		clientConns: make(map[string]cosmosgrpc.ClientConn),
	}
	return m
}

func (m *tendermintRPCClientManagerImpl) GetClient(ctx context.Context, chainID string) (cosmosgrpc.ClientConn, error) {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.clientConns[chainID]; !ok {
		q, err := DefaultCosmosGRPCCLientConn(ctx, chainID)
		if err != nil {
			return nil, err
		}
		m.clientConns[chainID] = q
	}
	return m.clientConns[chainID], nil
}

func DefaultCosmosGRPCCLientConn(ctx context.Context, chainID string) (cosmosgrpc.ClientConn, error) {
	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("getting config for chain %s: %w", chainID, err)
	}

	return grpc.DialContext(ctx, chainConfig.Cosmos.GRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
}
