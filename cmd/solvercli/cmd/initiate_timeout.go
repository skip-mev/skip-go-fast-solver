package cmd

import (
	"fmt"
	"math/big"
	"os/signal"
	"syscall"
	"time"

	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/cosmos"

	"github.com/skip-mev/go-fast-solver/db/gen/db"

	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

var initiateTimeoutCmd = &cobra.Command{
	Use:   "initiate-timeout",
	Short: "Initiate timeout for an expired order",
	Long: `Initiate timeout for an expired order that hasn't been filled.
Example:
  ./build/solvercli initiate-timeout \
  --order-id <order_id> \
  --tx-hash <tx_hash> \
  --chain-id <chain_id>`,
	Run: initiateTimeout,
}

func init() {
	rootCmd.AddCommand(initiateTimeoutCmd)
	initiateTimeoutCmd.Flags().String("order-id", "", "ID of the order to timeout")
	initiateTimeoutCmd.Flags().String("chain-id", "", "Chain ID where the order was created")
	initiateTimeoutCmd.Flags().String("tx-hash", "", "Transaction hash that created the order")

	requiredFlags := []string{"order-id", "chain-id", "tx-hash"}
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

	orderID, chainID, timeoutTxHash, err := getRequiredFlags(cmd)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get required flags", zap.Error(err))
		return
	}

	cfg, keyStore, err := setupConfig(cmd)
	if err != nil {
		return
	}
	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(*cfg))

	sourceChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		lmt.Logger(ctx).Error("source chain not found in config", zap.String("sourceChainID", chainID))
		return
	}
	if sourceChainConfig.Type != config.ChainType_EVM {
		lmt.Logger(ctx).Error(
			"source chain must be of type evm",
			zap.String("sourceChainID", chainID),
			zap.String("sourceChainType", string(sourceChainConfig.Type)),
		)
		return
	}
	gatewayAddr := sourceChainConfig.FastTransferContractAddress

	client, gateway, err := setupGatewayContract(ctx, sourceChainConfig, gatewayAddr)
	if err != nil {
		return
	}

	order, err := getOrderFromContract(ctx, gateway, client, orderID, chainID, gatewayAddr, timeoutTxHash)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get order from contract", zap.Error(err))
		return
	}

	if err := verifyOrderTimeout(ctx, order); err != nil {
		return
	}
	cosmosTxExecutor := cosmos.DefaultSerializedCosmosTxExecutor()
	clientManager := clientmanager.NewClientManager(keyStore, cosmosTxExecutor)

	bridgeClient, err := clientManager.GetClient(ctx, chainID)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to create bridge client", zap.Error(err))
		return
	}

	timeoutTxHash, _, _, err = bridgeClient.InitiateTimeout(ctx, *order, gatewayAddr)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to initiate timeout", zap.Error(err))
		return
	}

	fmt.Printf("Successfully initiated timeout for order %s\n", orderID)
	fmt.Printf("Timeout transaction hash: %s\n", timeoutTxHash)
}

func getRequiredFlags(cmd *cobra.Command) (string, string, string, error) {
	orderID, err := cmd.Flags().GetString("order-id")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get order-id: %w", err)
	}

	chainID, err := cmd.Flags().GetString("chain-id")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get chain-id: %w", err)
	}

	txHash, err := cmd.Flags().GetString("tx-hash")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get tx-hash: %w", err)
	}

	return orderID, chainID, txHash, nil
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

func getOrderFromContract(ctx context.Context, gateway *fast_transfer_gateway.FastTransferGateway, client *ethclient.Client, orderID string, chainID string, gatewayAddr string, txHash string) (*db.Order, error) {
	orderStatus, err := gateway.OrderStatuses(nil, common.HexToHash(orderID))
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	if orderStatus != 0 { // 0 = Pending
		return nil, fmt.Errorf("order is not in pending status (status: %d)", orderStatus)
	}

	tx, isPending, err := client.TransactionByHash(ctx, common.HexToHash(txHash))
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}
	if isPending {
		return nil, fmt.Errorf("transaction is still pending")
	}

	contractABI, err := abi.JSON(strings.NewReader(fast_transfer_gateway.FastTransferGatewayABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
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
	destinationDomain := args[4].(uint32)
	timeoutTimestamp := args[5].(uint64)

	order := &db.Order{
		OrderID:                           orderID,
		SourceChainID:                     chainID,
		DestinationChainID:                fmt.Sprintf("%d", destinationDomain),
		TimeoutTimestamp:                  time.Unix(int64(timeoutTimestamp), 0),
		Sender:                            sender[:],
		Recipient:                         recipient[:],
		AmountIn:                          amountIn.String(),
		AmountOut:                         amountOut.String(),
		Nonce:                             int64(tx.Nonce()),
		SourceChainGatewayContractAddress: gatewayAddr,
	}

	return order, nil
}

func verifyOrderTimeout(ctx context.Context, order *db.Order) error {
	if !time.Now().UTC().After(order.TimeoutTimestamp) {
		return fmt.Errorf("order has not timed out yet (timeout: %s, current: %s)",
			order.TimeoutTimestamp.UTC(),
			time.Now().UTC())
	}
	return nil
}

func setupGatewayContract(ctx context.Context, sourceChainConfig config.ChainConfig, gatewayAddr string) (*ethclient.Client, *fast_transfer_gateway.FastTransferGateway, error) {
	client, err := ethclient.Dial(sourceChainConfig.EVM.RPC)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to connect to the network", zap.Error(err))
		return nil, nil, fmt.Errorf("failed to connect to the network: %w", err)
	}

	gateway, err := fast_transfer_gateway.NewFastTransferGateway(common.HexToAddress(gatewayAddr), client)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gateway contract: %w", err)
	}

	return client, gateway, nil
}
