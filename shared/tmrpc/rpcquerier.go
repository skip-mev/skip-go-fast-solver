package tmrpc

import (
	"context"
	"net/http"

	"github.com/cometbft/cometbft/rpc/client"
	rpcclienthttp "github.com/cometbft/cometbft/rpc/client/http"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/utils"
)

var (
	reg codectypes.InterfaceRegistry
)

func init() {
	reg = codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(reg)
	cryptocodec.RegisterInterfaces(reg)
	banktypes.RegisterInterfaces(reg)
}

func DefaultTendermintRPCClient(ctx context.Context, chainID string) (client.Client, error) {
	rpc, err := config.GetConfigReader(ctx).GetRPCEndpoint(chainID)
	if err != nil {
		return nil, err
	}

	basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(chainID)
	if err != nil {
		return nil, err
	}

	client, err := rpcclienthttp.NewWithClient(rpc, "/websocket", &http.Client{
		Transport: utils.NewBasicAuthTransport(basicAuth, http.DefaultTransport),
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}
