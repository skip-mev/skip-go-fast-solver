package clientmanager

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"
	"sync"

	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/keys"

	"math/big"
	"net/http"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	rpcclienthttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	ethereumrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	"github.com/skip-mev/go-fast-solver/shared/utils"
)

type ClientManager struct {
	keyStore         keys.KeyStore
	clients          map[string]cctp.BridgeClient
	mu               sync.RWMutex
	cosmosTxExecutor cosmos.CosmosTxExecutor
}

func NewClientManager(chainIDToPrivateKey keys.KeyStore, cosmosTxExecutor cosmos.CosmosTxExecutor) *ClientManager {
	return &ClientManager{
		keyStore:         chainIDToPrivateKey,
		clients:          make(map[string]cctp.BridgeClient),
		cosmosTxExecutor: cosmosTxExecutor,
	}
}

func (cm *ClientManager) GetClient(
	ctx context.Context,
	chainID string,
) (cctp.BridgeClient, error) {
	chainCfg, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, err
	}

	cm.mu.RLock()
	client, ok := cm.clients[chainID]
	cm.mu.RUnlock()

	if ok {
		return client, nil
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check again in case another goroutine created the client while we were waiting for the lock
	if client, ok := cm.clients[chainID]; ok {
		return client, nil
	}

	var newClient cctp.BridgeClient
	switch chainCfg.Type {
	case config.ChainType_COSMOS:
		newClient, err = cm.createCosmosClient(ctx, chainID)
	case config.ChainType_EVM:
		newClient, err = cm.createEVMClient(ctx, chainID)
	default:
		return nil, errors.New("unsupported cctp domain")
	}

	if err != nil {
		return nil, err
	}

	cm.clients[chainID] = newClient
	return newClient, nil
}

func (cm *ClientManager) createCosmosClient(
	ctx context.Context,
	chainID string,
) (cctp.BridgeClient, error) {

	chainCfg, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, err
	}

	rpc, err := config.GetConfigReader(ctx).GetRPCEndpoint(chainID)
	if err != nil {
		return nil, err
	}

	basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(chainID)
	if err != nil {
		return nil, err
	}

	rpcClient, err := rpcclienthttp.NewWithClient(rpc, "/websocket", &http.Client{
		Transport: utils.NewBasicAuthTransport(basicAuth, http.DefaultTransport),
	})
	if err != nil {
		return nil, err
	}

	creds := insecure.NewCredentials()
	if chainCfg.Cosmos.GRPCTLSEnabled {
		creds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
	}
	grpcClient, err := grpc.Dial(chainCfg.Cosmos.GRPC, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	privateKeyStr, ok := cm.keyStore.GetPrivateKey(chainID)
	if !ok {
		return nil, fmt.Errorf("solver private key not found for chainID %s", chainID)
	}

	if privateKeyStr[:2] == "0x" {
		privateKeyStr = privateKeyStr[2:]
	}
	privateKeyBytes, err := hex.DecodeString(privateKeyStr)
	if err != nil {
		return nil, err
	}

	privateKey := &secp256k1.PrivKey{}
	if err := privateKey.UnmarshalAmino(privateKeyBytes); err != nil {
		return nil, err
	}

	bech32Address, err := bech32.ConvertAndEncode(chainCfg.Cosmos.AddressPrefix, privateKey.PubKey().Address())
	if err != nil {
		lmt.Logger(ctx).Error("error converting address to bech32", zap.String("chainID", chainID), zap.Error(err))
		return nil, err
	}

	bridgeClient, err := cctp.NewCosmosBridgeClient(
		rpcClient,
		grpcClient,
		chainID,
		chainCfg.Cosmos.AddressPrefix,
		signing.NewLocalCosmosSigner(privateKey, bech32Address),
		chainCfg.Cosmos.GasPrice,
		chainCfg.Cosmos.GasDenom,
		cm.cosmosTxExecutor,
	)

	return bridgeClient, err
}

func (cm *ClientManager) createEVMClient(
	ctx context.Context,
	chainID string,
) (cctp.BridgeClient, error) {
	chainCfg, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, err
	}

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

	privateKeyStr, ok := cm.keyStore.GetPrivateKey(chainID)
	if !ok {
		return nil, fmt.Errorf("solver private key not found for chainID %s", chainID)
	}

	if privateKeyStr[:2] == "0x" {
		privateKeyStr = privateKeyStr[2:]
	}

	privateKey, err := crypto.HexToECDSA(string(privateKeyStr))
	if err != nil {
		return nil, err
	}
	var minGasTip *big.Int
	if chainCfg.EVM.MinGasTipCap != nil {
		minGasTip = big.NewInt(*chainCfg.EVM.MinGasTipCap)
	}

	bridgeClient, err := cctp.NewEVMBridgeClient(
		client,
		chainID,
		signing.NewLocalEthereumSigner(privateKey),
		minGasTip,
	)

	return bridgeClient, err
}
