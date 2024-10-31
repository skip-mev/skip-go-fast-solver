package signing

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/skip-mev/go-fast-solver/shared/config"
)

func NewSigner(ctx context.Context, chainID string, chainIDToPrivateKey map[string]string) (Signer, error) {
	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("getting chain config for %s: %w", chainID, err)
	}

	switch chainConfig.Type {
	case config.ChainType_COSMOS:
		return newCosmosSigner(ctx, chainID, chainIDToPrivateKey)
	case config.ChainType_EVM:
		return newEVMSigner(ctx, chainID, chainIDToPrivateKey)
	default:
		return nil, fmt.Errorf("no signer available for chain type: %s", chainConfig.Type)
	}
}

func newCosmosSigner(ctx context.Context, chainID string, chainIDToPrivateKey map[string]string) (*LocalCosmosSigner, error) {
	privateKeyStr, ok := chainIDToPrivateKey[chainID]
	if !ok {
		return nil, fmt.Errorf("private key not found for chainID %s", chainID)
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

	chainCfg, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, err
	}
	bech32Address, err := bech32.ConvertAndEncode(chainCfg.Cosmos.AddressPrefix, privateKey.PubKey().Address())
	if err != nil {
		return nil, fmt.Errorf("converting address to bech32: %w", err)
	}

	return NewLocalCosmosSigner(privateKey, bech32Address), nil
}

func newEVMSigner(ctx context.Context, chainID string, chainIDToPrivateKey map[string]string) (*LocalEthereumSigner, error) {
	privateKeyStr, ok := chainIDToPrivateKey[chainID]
	if !ok {
		return nil, fmt.Errorf("solver private key not found for chainID %s", chainID)
	}

	if privateKeyStr[:2] == "0x" {
		privateKeyStr = privateKeyStr[2:]
	}

	privateKey, err := crypto.HexToECDSA(string(privateKeyStr))
	if err != nil {
		return nil, fmt.Errorf("converting private key to esdsa key: %w", err)
	}

	return NewLocalEthereumSigner(privateKey), nil
}
