package keys

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"os"
)

const (
	KeyStoreTypePlaintextFile = "plaintext-file"
	KeyStoreTypeEncryptedFile = "encrypted-file"
	KeyStoreTypeEnv           = "env"
)

type GetKeyStoreOpts struct {
	KeyFilePath string
	AESKeyHex   string
}

func GetKeyStore(keyStoreType string, opts GetKeyStoreOpts) (map[string]string, error) {
	switch keyStoreType {
	case KeyStoreTypePlaintextFile:
		if opts.KeyFilePath == "" {
			return nil, errors.New("key file path is required")
		}
		return LoadKeyStoreFromPlaintextFile(opts.KeyFilePath)
	case KeyStoreTypeEncryptedFile:
		if opts.KeyFilePath == "" || opts.AESKeyHex == "" {
			return nil, errors.New("key file path and aes key are required")
		}
		return LoadKeyStoreFromEncryptedFile(opts.KeyFilePath, opts.AESKeyHex)
	case KeyStoreTypeEnv:
		return LoadKeyStoreFromEnv()
	default:
		return nil, fmt.Errorf("key store type must be one of %v", []string{
			KeyStoreTypePlaintextFile,
			KeyStoreTypeEncryptedFile,
			KeyStoreTypeEnv,
		})
	}
}

func LoadKeyStoreFromPlaintextFile(keysPath string) (map[string]string, error) {
	keysBytes, err := os.ReadFile(keysPath)
	if err != nil {
		return nil, err
	}

	rawKeysMap := make(map[string]map[string]string)
	if err := json.Unmarshal(keysBytes, &rawKeysMap); err != nil {
		return nil, err
	}

	keysMap := make(map[string]string)
	for key, value := range rawKeysMap {
		keysMap[key] = value["private_key"]
	}

	return keysMap, nil
}

func LoadKeyStoreFromEnv() (map[string]string, error) {
	keysEnv := os.Getenv("SOLVER_KEYS")
	if keysEnv == "" {
		return nil, nil
	}

	rawKeysMap := make(map[string]map[string]string)
	if err := json.Unmarshal([]byte(keysEnv), &rawKeysMap); err != nil {
		return nil, err
	}

	keysMap := make(map[string]string)
	for key, value := range rawKeysMap {
		keysMap[key] = value["private_key"]
	}

	return keysMap, nil
}

func LoadKeyStoreFromEncryptedFile(keysPath, aesKeyHex string) (map[string]string, error) {
	passphrase, err := hex.DecodeString(aesKeyHex)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(passphrase[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	keysBytesEncrypted, err := os.ReadFile(keysPath)
	if err != nil {
		return nil, err
	}

	keysBytesEncryptedHexString := string(keysBytesEncrypted)
	nonce := keysBytesEncryptedHexString[:gcm.NonceSize()*2]
	ciphertext := keysBytesEncryptedHexString[gcm.NonceSize()*2:]
	nonceBytes, err := hex.DecodeString(nonce)
	if err != nil {
		return nil, err
	}
	ciphertextBytes, err := hex.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	keysBytesDecrypted, err := gcm.Open(nil,
		nonceBytes,
		ciphertextBytes,
		nil,
	)

	if err != nil {
		return nil, err
	}

	rawKeysMap := make(map[string]map[string]string)
	if err := json.Unmarshal(keysBytesDecrypted, &rawKeysMap); err != nil {
		return nil, err
	}

	keysMap := make(map[string]string)
	for key, value := range rawKeysMap {
		keysMap[key] = value["private_key"]
	}

	return keysMap, nil
}

func LogSolverAddresses(ctx context.Context, chainIDToPrivateKey KeyStore) {
	for chainID, privateKeyHex := range chainIDToPrivateKey {
		chainCfg, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
		if err != nil {
			lmt.Logger(ctx).Error("error getting chain config", zap.String("chainID", chainID), zap.Error(err))
			continue
		}

		if privateKeyHex[:2] == "0x" {
			privateKeyHex = privateKeyHex[2:]
		}

		if chainCfg.Type == config.ChainType_COSMOS {
			privateKeyBytes, err := hex.DecodeString(privateKeyHex)
			if err != nil {
				lmt.Logger(ctx).Error("error decoding private key", zap.String("chainID", chainID), zap.Error(err))
				continue
			}

			privateKey := &secp256k1.PrivKey{}
			if err := privateKey.UnmarshalAmino(privateKeyBytes); err != nil {
				lmt.Logger(ctx).Error("error unmarshaling private key", zap.String("chainID", chainID), zap.Error(err))
				continue
			}

			bech32Address, err := bech32.ConvertAndEncode(chainCfg.Cosmos.AddressPrefix, privateKey.PubKey().Address())
			if err != nil {
				lmt.Logger(ctx).Error("error converting address to bech32", zap.String("chainID", chainID), zap.Error(err))
				continue
			}

			lmt.Logger(ctx).Info("solver address", zap.String("chainID", chainID), zap.String("address", bech32Address))
		} else if chainCfg.Type == config.ChainType_EVM {
			privateKey, err := crypto.HexToECDSA(privateKeyHex)
			if err != nil {
				lmt.Logger(ctx).Error("error decoding private key", zap.String("chainID", chainID), zap.Error(err))
				continue
			}

			address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
			lmt.Logger(ctx).Info("solver address", zap.String("chainID", chainID), zap.String("address", address))
		} else {
			lmt.Logger(ctx).Error("unknown chain type", zap.String("chainID", chainID), zap.String("chainType", string(chainCfg.Type)))
		}
	}
}
