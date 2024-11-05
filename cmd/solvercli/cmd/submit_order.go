package cmd

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"golang.org/x/net/context"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a new fast transfer order",
	Long: `Submit a new fast transfer order through the FastTransferGateway contract.
Example:
  fast-solver submit \
    --token 0xaf88d065e77c8cC2239327C5EDb3A432268e5831 \
    --recipient osmo1v57fx2l2rt6ehujuu99u2fw05779m5e2ux4z2h \
    --amount 5000000 \
    --source-chain-id 42161 \
    --destination-chain-id osmosis-1 \
    --gateway 0x24a9267cE9e0a8F4467B584FDDa12baf1Df772B5`,
	Run: submitOrder,
}

func submitOrder(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	logger := lmt.Logger(ctx)

	flags, err := parseFlags(cmd)
	if err != nil {
		logger.Error("Failed to parse flags", zap.Error(err))
		return
	}

	cfg, err := config.LoadConfig(flags.configPath)
	if err != nil {
		logger.Error("Unable to load config", zap.Error(err))
		return
	}

	client, err := ethclient.Dial(cfg.Chains[flags.sourceChainID].EVM.RPC)
	if err != nil {
		logger.Error("Failed to connect to the network", zap.Error(err))
		return
	}

	gateway, auth, err := setupGatewayAndAuth(ctx, client, flags)
	if err != nil {
		logger.Error("Failed to setup gateway and auth", zap.Error(err))
		return
	}

	tx, err := submitTransferOrder(gateway, auth, flags)
	if err != nil {
		logger.Error("Failed to submit order", zap.Error(err))
		return
	}

	logSuccess(logger, tx, flags)
}

type submitFlags struct {
	keysPath          string
	keyStoreType      string
	aesKeyHex         string
	token             string
	recipient         string
	amount            string
	destinationDomain uint32
	deadlineHours     uint32
	gatewayAddr       string
	configPath        string
	sourceChainID     string
}

func parseFlags(cmd *cobra.Command) (*submitFlags, error) {
	flags := &submitFlags{}
	var err error

	if flags.keysPath, err = cmd.Flags().GetString("keys"); err != nil {
		return nil, err
	}
	if flags.keyStoreType, err = cmd.Flags().GetString("key-store-type"); err != nil {
		return nil, err
	}
	if flags.aesKeyHex, err = cmd.Flags().GetString("aes-key-hex"); err != nil {
		return nil, err
	}
	if flags.token, err = cmd.Flags().GetString("token"); err != nil {
		return nil, err
	}
	if flags.recipient, err = cmd.Flags().GetString("recipient"); err != nil {
		return nil, err
	}
	if flags.amount, err = cmd.Flags().GetString("amount"); err != nil {
		return nil, err
	}
	if flags.destinationDomain, err = cmd.Flags().GetUint32("destination-chain-id"); err != nil {
		return nil, err
	}
	if flags.deadlineHours, err = cmd.Flags().GetUint32("deadline-hours"); err != nil {
		return nil, err
	}
	if flags.gatewayAddr, err = cmd.Flags().GetString("gateway"); err != nil {
		return nil, err
	}
	if flags.configPath, err = cmd.Flags().GetString("config"); err != nil {
		return nil, err
	}
	if flags.sourceChainID, err = cmd.Flags().GetString("source-chain-id"); err != nil {
		return nil, err
	}

	return flags, nil
}

func setupGatewayAndAuth(ctx context.Context, client *ethclient.Client, flags *submitFlags) (*fast_transfer_gateway.FastTransferGateway, *bind.TransactOpts, error) {
	gateway, err := fast_transfer_gateway.NewFastTransferGateway(common.HexToAddress(flags.gatewayAddr), client)
	if err != nil {
		return nil, nil, err
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, nil, err
	}

	keystore, err := keys.GetKeyStore(flags.keyStoreType, keys.GetKeyStoreOpts{
		KeyFilePath: flags.keysPath,
		AESKeyHex:   flags.aesKeyHex,
	})
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := getPrivateKey(keystore, flags.sourceChainID)
	if err != nil {
		return nil, nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, nil, err
	}

	return gateway, auth, nil
}

func getPrivateKey(keystore map[string]string, chainID string) (*ecdsa.PrivateKey, error) {
	privateKeyStr, ok := keystore[chainID]
	if !ok {
		return nil, fmt.Errorf("private key not found for chain ID: %s", chainID)
	}
	if privateKeyStr[:2] == "0x" {
		privateKeyStr = privateKeyStr[2:]
	}
	return crypto.HexToECDSA(privateKeyStr)
}

func submitTransferOrder(gateway *fast_transfer_gateway.FastTransferGateway, auth *bind.TransactOpts, flags *submitFlags) (*types.Transaction, error) {
	amountBig := new(big.Int)
	amountBig.SetString(flags.amount, 10)
	deadline := time.Now().Add(time.Duration(flags.deadlineHours) * time.Hour)

	return gateway.SubmitOrder(
		auth,
		addressTo32Bytes(auth.From),
		addressTo32Bytes(common.HexToAddress(flags.recipient)),
		amountBig,
		amountBig,
		flags.destinationDomain,
		big.NewInt(deadline.Unix()),
		[]byte{},
	)
}

func logSuccess(logger *zap.Logger, tx *types.Transaction, flags *submitFlags) {
	logger.Info("Order submitted successfully",
		zap.String("tx_hash", tx.Hash().Hex()),
		zap.String("token", flags.token),
		zap.String("recipient", flags.recipient),
		zap.String("amount", flags.amount),
		zap.String("source_chain_id", flags.sourceChainID),
		zap.Uint32("destination_chain_id", flags.destinationDomain),
		zap.Uint32("deadline_hours", flags.deadlineHours),
	)
}

func init() {
	rootCmd.AddCommand(submitCmd)

	submitCmd.Flags().String("token", "", "Token address to transfer")
	submitCmd.Flags().String("recipient", "", "Recipient address")
	submitCmd.Flags().String("amount", "", "Amount to transfer (in token decimals)")
	submitCmd.Flags().String("source-chain-id", "", "Source chain ID")
	submitCmd.Flags().Uint32("destination-chain-id", 0, "Destination chain ID")
	submitCmd.Flags().Uint32("deadline-hours", 24, "Deadline in hours (default of 24 hours, after which the order expires)")
	submitCmd.Flags().String("gateway", "", "Gateway contract address")

	submitCmd.MarkFlagRequired("token")
	submitCmd.MarkFlagRequired("recipient")
	submitCmd.MarkFlagRequired("amount")
	submitCmd.MarkFlagRequired("source-chain-id")
	submitCmd.MarkFlagRequired("destination-chain-id")
	submitCmd.MarkFlagRequired("gateway")
}

func addressTo32Bytes(addr common.Address) [32]byte {
	var result [32]byte
	copy(result[12:], addr.Bytes())
	return result
}
