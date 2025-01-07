package cmd

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

var initiateTimeoutCmd = &cobra.Command{
	Use:   "initiate-timeout",
	Short: "Initiate timeout for an expired order",
	Long: `Initiate timeout for an expired order that hasn't been filled. Note that the timeout transaction needs
to be relayed separately.`,
	Example: `solver initiate-timeout \
  --tx-hash <tx_hash> \
  --source-chain-id <chain_id> \
  --destination-chain-id <chain_id>`,
	Run: initiateTimeout,
}

func init() {
	rootCmd.AddCommand(initiateTimeoutCmd)
	initiateTimeoutCmd.Flags().String("source-chain-id", "", "Source chain ID")
	initiateTimeoutCmd.Flags().String("destination-chain-id", "", "Destination chain ID")
	initiateTimeoutCmd.Flags().String("tx-hash", "", "Transaction hash that created the order")

	requiredFlags := []string{"source-chain-id", "destination-chain-id", "tx-hash"}
	for _, flag := range requiredFlags {
		if err := initiateTimeoutCmd.MarkFlagRequired(flag); err != nil {
			panic(fmt.Sprintf("failed to mark %s flag as required: %v", flag, err))
		}
	}
}

func initiateTimeout(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	sourceChainID, destinationChainID, timeoutTxHash, err := getRequiredFlags(cmd)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get required flags", zap.Error(err))
		return
	}

	cfg, keyStore, err := setupConfig(cmd)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to setup config", zap.Error(err))
		return
	}
	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(*cfg))

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(sourceChainID)
	if err != nil {
		lmt.Logger(ctx).Error("Source chain not found in config", zap.String("sourceChainID", sourceChainID))
		return
	}

	if sourceChainConfig.Type != config.ChainType_EVM {
		lmt.Logger(ctx).Error("Source chain must be an EVM chain",
			zap.String("sourceChainID", sourceChainID),
			zap.String("chainType", string(sourceChainConfig.Type)))
		return
	}

	sourceGatewayAddr := sourceChainConfig.FastTransferContractAddress
	if sourceGatewayAddr == "" {
		lmt.Logger(ctx).Error("Gateway contract address not found in config", zap.String("sourceChainID", sourceChainID))
		return
	}

	order, err := getEVMOrderDetails(ctx, sourceChainConfig, sourceChainID, destinationChainID, sourceGatewayAddr, timeoutTxHash)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get order details", zap.Error(err))
		return
	}

	if err := verifyOrderTimeout(ctx, order); err != nil {
		lmt.Logger(ctx).Error("Failed to verify order timeout", zap.Error(err))
		return
	}

	destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(destinationChainID)
	if err != nil {
		lmt.Logger(ctx).Error("Destination chain not found in config", zap.String("destinationChainID", destinationChainID))
		return
	}

	if destinationChainConfig.Type != config.ChainType_COSMOS {
		lmt.Logger(ctx).Error("Destination chain must be a Cosmos chain",
			zap.String("destinationChainID", destinationChainID),
			zap.String("chainType", string(destinationChainConfig.Type)))
		return
	}

	destinationGatewayAddr := destinationChainConfig.FastTransferContractAddress
	if destinationGatewayAddr == "" {
		lmt.Logger(ctx).Error("Gateway contract address not found in config", zap.String("destinationChainID", destinationChainID))
		return
	}

	cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
	clientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

	bridgeClient, err := clientManager.GetClient(ctx, destinationChainID)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to create bridge client", zap.Error(err))
		return
	}

	timeoutTxHash, _, _, err = bridgeClient.InitiateTimeout(ctx, *order, destinationGatewayAddr)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to initiate timeout", zap.Error(err))
		return
	}

	fmt.Printf("Successfully initiated timeout for order %s\n", order.OrderID)
	fmt.Printf("Timeout transaction hash: %s\n", timeoutTxHash)
	fmt.Printf("Note: You must relay this transaction using:\n")
	fmt.Printf("solver relay --origin-chain-id %s --origin-tx-hash %s\n", destinationChainID, timeoutTxHash)
}

func getEVMOrderDetails(ctx context.Context, chainConfig config.ChainConfig, chainID, destinationChainID, gatewayAddr, txHash string) (*db.Order, error) {
	client, err := ethclient.Dial(chainConfig.EVM.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM network: %w", err)
	}
	defer client.Close()

	gateway, err := fast_transfer_gateway.NewFastTransferGateway(common.HexToAddress(gatewayAddr), client)
	if err != nil {
		return nil, fmt.Errorf("failed to create gateway contract: %w", err)
	}

	tx, isPending, err := client.TransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if isPending {
		return nil, fmt.Errorf("transaction is still pending")
	}

	receipt, err := client.TransactionReceipt(ctx, tx.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	contractABI, err := abi.JSON(strings.NewReader(fast_transfer_gateway.FastTransferGatewayABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	var orderIDBytes common.Hash
	for _, log := range receipt.Logs {
		if len(log.Topics) > 0 {
			if log.Topics[0] == contractABI.Events["OrderSubmitted"].ID {
				orderIDBytes = log.Topics[1]
				fmt.Println("DEBUG: Found OrderSubmitted event, raw order ID bytes:", log.Topics[1].Hex())
				fmt.Println("DEBUG: Event Data:", hex.EncodeToString(log.Data))
				break
			}
		}
	}

	if orderIDBytes == (common.Hash{}) {
		return nil, fmt.Errorf("OrderSubmitted event not found in transaction logs")
	}

	orderID := strings.ToUpper(orderIDBytes.Hex()[2:])

	orderStatus, err := gateway.OrderStatuses(nil, orderIDBytes)
	if err != nil {
		fmt.Println("Error checking order status:", err)
	}

	if orderStatus != 0 { // 0 = Pending
		return nil, fmt.Errorf("order is not in pending status (status: %d)", orderStatus)
	}

	method, err := contractABI.MethodById(tx.Data()[:4])
	if err != nil {
		return nil, fmt.Errorf("failed to get method from transaction data: %w", err)
	}

	if method.Name != "submitOrder" {
		return nil, fmt.Errorf("transaction is not a submitOrder transaction (got %s)", method.Name)
	}

	args, err := method.Inputs.Unpack(tx.Data()[4:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack transaction data: %w", err)
	}

	sender := args[0].([32]byte)
	recipient := args[1].([32]byte)
	amountIn := args[2].(*big.Int)
	amountOut := args[3].(*big.Int)
	timeoutTimestamp := args[5].(uint64)

	return &db.Order{
		OrderID:                           orderID,
		SourceChainID:                     chainID,
		DestinationChainID:                destinationChainID,
		TimeoutTimestamp:                  time.Unix(int64(timeoutTimestamp), 0),
		Sender:                            sender[:],
		Recipient:                         recipient[:],
		AmountIn:                          amountIn.String(),
		AmountOut:                         amountOut.String(),
		Nonce:                             int64(tx.Nonce()),
		SourceChainGatewayContractAddress: gatewayAddr,
	}, nil
}

func getRequiredFlags(cmd *cobra.Command) (string, string, string, error) {
	sourceChainId, err := cmd.Flags().GetString("source-chain-id")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get source-chain-id: %w", err)
	}

	destinationChainId, err := cmd.Flags().GetString("destination-chain-id")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get destination-chain-id: %w", err)
	}

	txHash, err := cmd.Flags().GetString("tx-hash")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get tx-hash: %w", err)
	}

	return sourceChainId, destinationChainId, txHash, nil
}

func setupConfig(cmd *cobra.Command) (*config.Config, keys.KeyStore, error) {
	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load config: %w", err)
	}

	keyStoreType, err := cmd.Flags().GetString("key-store-type")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get key-store-type: %w", err)
	}

	keysPath, err := cmd.Flags().GetString("keys")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get keys path: %w", err)
	}

	keyStore, err := keys.GetKeyStore(keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: keysPath})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to load keystore: %w", err)
	}

	return &cfg, keyStore, nil
}

func verifyOrderTimeout(ctx context.Context, order *db.Order) error {
	if !time.Now().UTC().After(order.TimeoutTimestamp) {
		return fmt.Errorf("order has not timed out yet (timeout: %s, current: %s)",
			order.TimeoutTimestamp.UTC(),
			time.Now().UTC())
	}
	return nil
}
