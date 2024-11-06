package cmd

import (
	"fmt"
	"math/big"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/contracts/usdc"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a new fast transfer order",
	Long: `Submit a new fast transfer order through the FastTransferGateway contract.
Example:
  ./build/solvercli submit \
  --config ./config/local/config.yml \
  --token 0xaf88d065e77c8cC2239327C5EDb3A432268e5831 \
  --recipient osmo13c9seh3vgvtfvdufz4eh2zhp0cepq4wj0egc02 \
  --amount 1000000 \
  --source-chain-id 42161 \
  --destination-chain-id osmosis-1 \
  --gateway 0x9c75534d7d6670a3ddd69a55b4067460f3e8744b \
  --private-key 0xf6079d30f832f998c86e5841385a4be06b6ca2b0875b90dcab8e167eba4dcab1 \
  --deadline-hours 24`,
	Run: submitOrder,
}

func submitOrder(cmd *cobra.Command, args []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lmt.ConfigureLogger()
	ctx = lmt.LoggerContext(ctx)

	flags, err := parseFlags(cmd)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to parse flags", zap.Error(err))
		return
	}

	cfg, err := config.LoadConfig(flags.configPath)
	if err != nil {
		lmt.Logger(ctx).Error("Unable to load config", zap.Error(err))
		return
	}

	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

	client, err := ethclient.Dial(cfg.Chains[flags.sourceChainID].EVM.RPC)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to connect to the network", zap.Error(err))
		return
	}

	gateway, auth, err := setupGatewayAndAuth(ctx, client, flags)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to setup gateway and auth", zap.Error(err))
		return
	}

	usdc, err := usdc.NewUsdc(common.HexToAddress(flags.token), client)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to create USDC contract instance", zap.Error(err))
		return
	}

	amountBig := new(big.Int)
	amountBig.SetString(flags.amount, 10)

	tx, err := usdc.Approve(auth, common.HexToAddress(flags.gatewayAddr), amountBig)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to approve USDC spending", zap.Error(err))
		return
	}

	lmt.Logger(ctx).Info("USDC approval submitted",
		zap.String("tx_hash", tx.Hash().Hex()),
		zap.String("amount", flags.amount),
	)

	// Wait for approval transaction to be mined
	_, err = bind.WaitMined(ctx, client, tx)
	if err != nil {
		lmt.Logger(ctx).Error("Failed waiting for USDC approval to be mined", zap.Error(err))
		return
	}

	destChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(flags.destinationChainId)
	if err != nil {
		lmt.Logger(ctx).Error("Failed to get destination chain config: %w", zap.Error(err))
		return
	}

	destDomain, err := strconv.ParseUint(destChainConfig.HyperlaneDomain, 10, 32)
	if err != nil {
		lmt.Logger(ctx).Error("parsing destination hyperlane domain: %w", zap.Error(err))
		return
	}

	tx, err = submitTransferOrder(gateway, auth, flags, uint32(destDomain))
	if err != nil {
		lmt.Logger(ctx).Error("Failed to submit order", zap.Error(err))
		return
	}

	lmt.Logger(ctx).Info("Order submitted successfully",
		zap.String("tx_hash", tx.Hash().Hex()),
		zap.String("token", flags.token),
		zap.String("recipient", flags.recipient),
		zap.String("amount", flags.amount),
		zap.String("source_chain_id", flags.sourceChainID),
		zap.String("destination_chain_id", flags.destinationChainId),
		zap.Uint32("destination_domain", uint32(destDomain)),
		zap.Uint32("deadline_hours", flags.deadlineHours),
	)
}

type submitFlags struct {
	token              string
	recipient          string
	amount             string
	destinationChainId string
	deadlineHours      uint32
	gatewayAddr        string
	configPath         string
	sourceChainID      string
	privateKey         string
}

func parseFlags(cmd *cobra.Command) (*submitFlags, error) {
	flags := &submitFlags{}
	var err error

	if flags.token, err = cmd.Flags().GetString("token"); err != nil {
		return nil, err
	}
	if flags.recipient, err = cmd.Flags().GetString("recipient"); err != nil {
		return nil, err
	}
	if flags.amount, err = cmd.Flags().GetString("amount"); err != nil {
		return nil, err
	}
	if flags.destinationChainId, err = cmd.Flags().GetString("destination-chain-id"); err != nil {
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
	if flags.privateKey, err = cmd.Flags().GetString("private-key"); err != nil {
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

	privateKeyStr := flags.privateKey
	if privateKeyStr[:2] == "0x" {
		privateKeyStr = privateKeyStr[2:]
	}

	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		return nil, nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, nil, err
	}

	return gateway, auth, nil
}

func submitTransferOrder(gateway *fast_transfer_gateway.FastTransferGateway, auth *bind.TransactOpts, flags *submitFlags, destDomain uint32) (*types.Transaction, error) {
	amountBig := new(big.Int)
	amountBig.SetString(flags.amount, 10)
	deadline := time.Now().Add(time.Duration(flags.deadlineHours) * time.Hour)

	senderBytes, err := addressTo32Bytes(auth.From.Hex())
	if err != nil {
		return nil, fmt.Errorf("converting sender address: %w", err)
	}

	recipientBytes, err := addressTo32Bytes(flags.recipient)
	if err != nil {
		return nil, fmt.Errorf("converting recipient address: %w", err)
	}

	return gateway.SubmitOrder(
		auth,
		senderBytes,
		recipientBytes,
		amountBig,
		amountBig,
		destDomain,
		big.NewInt(deadline.Unix()),
		[]byte{},
	)
}

func init() {
	rootCmd.AddCommand(submitCmd)

	submitCmd.Flags().String("token", "", "Token address to transfer")
	submitCmd.Flags().String("recipient", "", "Recipient address")
	submitCmd.Flags().String("amount", "", "Amount to transfer (in token decimals)")
	submitCmd.Flags().String("source-chain-id", "", "Source chain ID")
	submitCmd.Flags().String("destination-chain-id", "", "Destination chain ID")
	submitCmd.Flags().Uint32("deadline-hours", 24, "Deadline in hours (default of 24 hours, after which the order expires)")
	submitCmd.Flags().String("gateway", "", "Gateway contract address")
	submitCmd.Flags().String("private-key", "", "Private key to sign the transaction")

	submitCmd.MarkFlagRequired("token")
	submitCmd.MarkFlagRequired("recipient")
	submitCmd.MarkFlagRequired("amount")
	submitCmd.MarkFlagRequired("source-chain-id")
	submitCmd.MarkFlagRequired("destination-chain-id")
	submitCmd.MarkFlagRequired("gateway")
	submitCmd.MarkFlagRequired("private-key")
}

func addressTo32Bytes(addr string) ([32]byte, error) {
	var result [32]byte

	// EVM address
	if strings.HasPrefix(addr, "0x") {
		addr = addr[2:]
		ethAddr := common.HexToAddress(addr)
		copy(result[12:], ethAddr.Bytes())
		return result, nil
	} else {
		// Bech32 address
		_, bz, err := bech32.DecodeAndConvert(addr)
		if err != nil {
			return result, err
		}
		copy(result[12:], bz)
		return result, nil
	}
}
