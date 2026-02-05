package utils

import (
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type RateLimitedHTTPClient struct {
	client  HTTPClient
	limiter RateLimiter
}

func NewRateLimitedHTTPClient(client HTTPClient, limiter RateLimiter) *RateLimitedHTTPClient {
	return &RateLimitedHTTPClient{client, limiter}
}

func DefaultRateLimitedHTTPClient(requestsPerMinute int) *RateLimitedHTTPClient {
	if requestsPerMinute == 0 {
		requestsPerMinute = 10
	}
	return NewRateLimitedHTTPClient(
		http.DefaultClient,
		rate.NewLimiter(rate.Every(time.Minute/time.Duration(requestsPerMinute)), requestsPerMinute/2),
	)
}

func (c *RateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if err := c.limiter.Wait(req.Context()); err != nil {
		lmt.Logger(req.Context()).Warn("rate limiter wait failed",
			zap.String("url", req.URL.String()),
			zap.Error(err),
			zap.NamedError("ctxErr", req.Context().Err()),
		)
		return nil, fmt.Errorf("waiting for rate limiter: %w", err)
	}

	res, err := c.client.Do(req)
	if err != nil {
		lmt.Logger(req.Context()).Warn("http request failed",
			zap.String("url", req.URL.String()),
			zap.Error(err),
			zap.NamedError("ctxErr", req.Context().Err()),
		)
		return nil, fmt.Errorf("executing http request: %w", err)
	}
	if res.StatusCode == 429 {
		retryAfterRaw := res.Header.Get("retry-after")
		retryAfter, parseErr := strconv.Atoi(retryAfterRaw)
		lmt.Logger(req.Context()).Warn("rate limited by server",
			zap.String("url", req.URL.String()),
			zap.String("retry-after", retryAfterRaw),
			zap.Bool("retry-after-parseable", parseErr == nil),
		)

		if parseErr != nil {
			return res, nil
		}

		select {
		case <-req.Context().Done():
			lmt.Logger(req.Context()).Warn("context canceled while waiting for rate limit retry",
				zap.String("url", req.URL.String()),
				zap.Int("retry-after-seconds", retryAfter),
			)
		case <-time.After(time.Duration(retryAfter) * time.Second):
		}

		res, err = c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing http request after rate limit retry: %w", err)
		}
	}

	return res, nil
}
