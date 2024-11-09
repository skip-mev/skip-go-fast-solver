package hyperlane

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/skip-mev/go-fast-solver/hyperlane/cosmos"
	"github.com/skip-mev/go-fast-solver/hyperlane/ethereum"
	"github.com/skip-mev/go-fast-solver/hyperlane/types"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/keys"
)

type Client interface {
	HasBeenDelivered(ctx context.Context, destinationDomain string, messageID string) (bool, error)
	ISMType(ctx context.Context, domain string, recipient string) (uint8, error)
	ValidatorsAndThreshold(ctx context.Context, domain string, recipient string, message string) ([]common.Address, uint8, error)
	ValidatorStorageLocations(ctx context.Context, domain string, validators []common.Address) ([]*types.ValidatorStorageLocation, error)
	MerkleTreeLeafCount(ctx context.Context, domain string) (uint64, error)
	Process(ctx context.Context, domain string, message []byte, metadata []byte) ([]byte, error)
	IsContract(ctx context.Context, domain, address string) (bool, error)
	GetHyperlaneDispatch(ctx context.Context, domain, originChainID, initiateTxHash string) (*types.MailboxDispatchEvent, *types.MailboxMerkleHookPostDispatchEvent, error)
}

type MultiClient struct {
	clients map[string]Client
}

// NewMultiClientFromConfig creates a MultiClient that is configured for every
// chain specific in the config that has a HyperlaneDomain set
func NewMultiClientFromConfig(ctx context.Context, manager evmrpc.EVMRPCClientManager, keystore keys.KeyStore) (*MultiClient, error) {
	clients := make(map[string]Client)
	for _, cfg := range config.GetConfigReader(ctx).Config().Chains {
		if cfg.HyperlaneDomain == "" {
			continue
		}

		switch cfg.Type {
		case config.ChainType_COSMOS:
			client, err := cosmos.NewHyperlaneClient(ctx, cfg.HyperlaneDomain)
			if err != nil {
				return nil, fmt.Errorf("creating cosmos hyperlane client for domain %s: %w", cfg.HyperlaneDomain, err)
			}
			clients[cfg.HyperlaneDomain] = client
		case config.ChainType_EVM:
			client, err := ethereum.NewHyperlaneClient(ctx, cfg.HyperlaneDomain, manager, keystore)
			if err != nil {
				return nil, fmt.Errorf("creating cosmos hyperlane client for domain %s: %w", cfg.HyperlaneDomain, err)
			}
			clients[cfg.HyperlaneDomain] = client
		}
	}
	return &MultiClient{clients: clients}, nil
}

type ClientConfig struct {
	HyperlaneDomain string
	Client          Client
}

func NewMultiClient(ctx context.Context, configs ...ClientConfig) (*MultiClient, error) {
	clients := make(map[string]Client)
	for _, cfg := range configs {
		clients[cfg.HyperlaneDomain] = cfg.Client
	}

	return &MultiClient{clients: clients}, nil
}

func (c *MultiClient) HasBeenDelivered(ctx context.Context, destinationDomain string, messageID string) (bool, error) {
	client, ok := c.clients[destinationDomain]
	if !ok {
		return false, fmt.Errorf("no configured client for domain %s", destinationDomain)
	}
	return client.HasBeenDelivered(ctx, destinationDomain, messageID)
}

func (c *MultiClient) ISMType(ctx context.Context, domain string, recipient string) (uint8, error) {
	client, ok := c.clients[domain]
	if !ok {
		return 0, fmt.Errorf("no configured client for domain %s", domain)
	}
	return client.ISMType(ctx, domain, recipient)
}

func (c *MultiClient) ValidatorsAndThreshold(ctx context.Context, domain string, recipient string, message string) ([]common.Address, uint8, error) {
	client, ok := c.clients[domain]
	if !ok {
		return nil, 0, fmt.Errorf("no configured client for domain %s", domain)
	}
	return client.ValidatorsAndThreshold(ctx, domain, recipient, message)
}

func (c *MultiClient) ValidatorStorageLocations(
	ctx context.Context,
	domain string,
	validators []common.Address,
) ([]*types.ValidatorStorageLocation, error) {
	client, ok := c.clients[domain]
	if !ok {
		return nil, fmt.Errorf("no configured client for domain %s", domain)
	}
	return client.ValidatorStorageLocations(ctx, domain, validators)
}

func (c *MultiClient) MerkleTreeLeafCount(ctx context.Context, domain string) (uint64, error) {
	client, ok := c.clients[domain]
	if !ok {
		return 0, fmt.Errorf("no configured client for domain %s", domain)
	}
	return client.MerkleTreeLeafCount(ctx, domain)
}

func (c *MultiClient) Process(ctx context.Context, domain string, message []byte, metadata []byte) ([]byte, error) {
	client, ok := c.clients[domain]
	if !ok {
		return nil, fmt.Errorf("no configured client for domain %s", domain)
	}
	return client.Process(ctx, domain, message, metadata)
}

func (c *MultiClient) IsContract(ctx context.Context, domain, address string) (bool, error) {
	client, ok := c.clients[domain]
	if !ok {
		return false, fmt.Errorf("no configured client for domain %s", domain)
	}
	return client.IsContract(ctx, domain, address)
}

func (c *MultiClient) GetHyperlaneDispatch(ctx context.Context, domain, originChainID, initiateTxHash string) (*types.MailboxDispatchEvent, *types.MailboxMerkleHookPostDispatchEvent, error) {
	client, ok := c.clients[domain]
	if !ok {
		return nil, nil, fmt.Errorf("no configured client for domain %s", domain)
	}
	return client.GetHyperlaneDispatch(ctx, domain, originChainID, initiateTxHash)
}
