package coingecko_test

import (
	"context"
	"errors"
	coingeckomocks "github.com/skip-mev/go-fast-solver/mocks/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type CachedPriceClientTestSuite struct {
	suite.Suite
}

func (s *CachedPriceClientTestSuite) TestGetSimplePrice() {
	// Given
	ctx := context.Background()
	mockInternalClient := coingeckomocks.NewMockPriceClient(s.T())
	// only expect one call since second call will be cached
	mockInternalClient.On("GetSimplePrice", mock.Anything, "huahua", "usd").Once().Return(5.0, nil)

	// When
	client := coingecko.NewCachedPriceClient(mockInternalClient, time.Duration(60)*time.Second)
	price, err := client.GetSimplePrice(ctx, "huahua", "usd")

	s.Require().Nil(err)
	s.Require().Equal(price, 5.0)

	price, err = client.GetSimplePrice(ctx, "huahua", "usd")

	// Then
	s.Require().Nil(err)
	s.Require().Equal(price, 5.0)
}

func (s *CachedPriceClientTestSuite) TestGetSimplePrice_Error() {
	// Given
	ctx := context.Background()
	mockInternalClient := coingeckomocks.NewMockPriceClient(s.T())
	mockInternalClient.On("GetSimplePrice", mock.Anything, "huahua", "usd").Once().Return(0.0, errors.New("failed"))

	// When
	client := coingecko.NewCachedPriceClient(mockInternalClient, time.Duration(60)*time.Second)
	_, err := client.GetSimplePrice(ctx, "huahua", "usd")

	// Then
	s.Require().ErrorContains(err, "failed")
}

func TestCachedPriceClientTestSuite(t *testing.T) {
	suite.Run(t, new(CachedPriceClientTestSuite))
}
