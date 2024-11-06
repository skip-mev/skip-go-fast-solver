package coingecko

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

//go:generate mockery --name PriceClient --filename mock_price_client.go
type PriceClient interface {
	GetSimplePrice(ctx context.Context, coingeckoID string, currency string) (float64, error)
}

type CachedPriceClient struct {
	mu                   sync.RWMutex
	internalClient       PriceClient
	cache                map[string]PriceResponse
	cacheRefreshInterval time.Duration
}

type PriceResponse struct {
	price       float64
	lastUpdated time.Time
}

func NewCachedPriceClient(client PriceClient, cacheRefreshInterval time.Duration) *CachedPriceClient {
	return &CachedPriceClient{
		internalClient:       client,
		cache:                make(map[string]PriceResponse),
		cacheRefreshInterval: cacheRefreshInterval,
	}
}

func cacheKey(coingeckoID string, currency string) string {
	return fmt.Sprintf("%s/%s", strings.ReplaceAll(coingeckoID, "/", "//"), strings.ReplaceAll(currency, "/", "//"))
}

func (c *CachedPriceClient) GetSimplePrice(ctx context.Context, coingeckoID string, currency string) (float64, error) {
	key := cacheKey(coingeckoID, currency)
	c.mu.RLock()
	priceResponse, ok := c.cache[key]
	c.mu.RUnlock()
	if !ok || time.Since(priceResponse.lastUpdated) > c.cacheRefreshInterval {
		var err error
		price, err := c.internalClient.GetSimplePrice(ctx, coingeckoID, currency)
		if err != nil {
			return 0, err
		}
		priceResponse = PriceResponse{
			price:       price,
			lastUpdated: time.Now(),
		}
		c.mu.Lock()
		c.cache[key] = priceResponse
		c.mu.Unlock()
	}
	return priceResponse.price, nil
}

type NoOpPriceClient struct{}

func NewNoOpPriceClient() PriceClient {
	return &NoOpPriceClient{}
}

func (m *NoOpPriceClient) GetSimplePrice(ctx context.Context, coingeckoID string, currency string) (float64, error) {
	return 0, nil
}
