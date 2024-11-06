package coingecko

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	utils "github.com/skip-mev/go-fast-solver/shared/clients/utils"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"io"
	"net/http"
	"time"
)

type CoingeckoClient struct {
	client  utils.HTTPClient
	baseURL string
	apiKey  string
}

func NewCoingeckoClient(client utils.HTTPClient, baseURL string, apiKey string) *CoingeckoClient {
	return &CoingeckoClient{client, baseURL, apiKey}
}

func DefaultCoingeckoClient(config config.CoingeckoConfig) *CoingeckoClient {
	client := utils.DefaultRateLimitedHTTPClient(config.RequestsPerMinute)
	return NewCoingeckoClient(client, config.BaseURL, config.APIKey)
}

func (c *CoingeckoClient) GetSimplePrice(ctx context.Context, coingeckoID string, currency string) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/simple/price", nil)
	if err != nil {
		return 0, err
	}

	query := req.URL.Query()
	query.Add("ids", coingeckoID)
	query.Add("vs_currencies", currency)
	if c.apiKey != "" {
		query.Add("x_cg_pro_api_key", c.apiKey)
	}
	req.URL.RawQuery = query.Encode()

	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != 200 {
		return 0, errors.New("error requesting resource from server")
	}

	jsonBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	prices := map[string]struct {
		USD float64 `json:"usd"`
	}{}
	if err := json.Unmarshal(jsonBytes, &prices); err != nil {
		return 0, err
	}

	price, ok := prices[coingeckoID]
	if !ok {
		return 0, fmt.Errorf("failed to return price for coingeckoId %s, response %s", coingeckoID, string(jsonBytes))
	}

	return price.USD, nil
}

type HistoricalPriceResponse struct {
	MarketData MarketData `json:"market_data"`
}

type MarketData struct {
	CurrentPrice map[string]float64 `json:"current_price"`
}

func (c *CoingeckoClient) GetHistoricalPrice(ctx context.Context, coingeckoID string, currency string, date time.Time) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/coins/"+coingeckoID+"/history", nil)
	if err != nil {
		return 0, err
	}

	query := req.URL.Query()
	query.Add("date", date.Format("02-01-2006"))
	if c.apiKey != "" {
		query.Add("x_cg_pro_api_key", c.apiKey)
	}
	req.URL.RawQuery = query.Encode()

	res, err := c.client.Do(req)
	if err != nil {
		return 0, err
	}

	if res.StatusCode != 200 {
		return 0, errors.New("error requesting resource from server")
	}

	jsonBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}

	var historicalPrice HistoricalPriceResponse
	if err := json.Unmarshal(jsonBytes, &historicalPrice); err != nil {
		return 0, err
	}

	price, ok := historicalPrice.MarketData.CurrentPrice[currency]
	if !ok {
		return 0, fmt.Errorf("failed to return price for coingeckoId %s, response %s", coingeckoID, string(jsonBytes))
	}

	return price, nil
}
