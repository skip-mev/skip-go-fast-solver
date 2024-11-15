package tmrpc

import (
	"context"
	"github.com/cometbft/cometbft/rpc/client"
	"sync"
)

type TendermintRPCClientManager interface {
	GetClient(ctx context.Context, chainID string) (client.Client, error)
}

type tendermintRPCClientManagerImpl struct {
	clients map[string]client.Client
	m       sync.Mutex
}

func NewTendermintRPCClientManager() TendermintRPCClientManager {
	m := &tendermintRPCClientManagerImpl{
		clients: make(map[string]client.Client),
	}
	return m
}

func (m *tendermintRPCClientManagerImpl) GetClient(ctx context.Context, chainID string) (client.Client, error) {
	m.m.Lock()
	defer m.m.Unlock()
	if _, ok := m.clients[chainID]; !ok {
		q, err := DefaultTendermintRPCClient(ctx, chainID)
		if err != nil {
			return nil, err
		}
		m.clients[chainID] = q
	}
	return m.clients[chainID], nil
}
