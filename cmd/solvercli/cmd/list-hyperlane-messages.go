/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/skip-mev/go-fast-solver/hyperlane"
	"github.com/skip-mev/go-fast-solver/hyperlane/types"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/clients/utils"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/txexecutor/evm"
	"github.com/spf13/cobra"
	"os/signal"
	"syscall"
	"time"

	cfg "github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// relayCmd represents the relay command
var listHyperlaneMessagesCmd = &cobra.Command{
	Use:   "list-hyperlane-messages",
	Short: "list hyperlane messages",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		lmt.ConfigureLogger()
		ctx = lmt.LoggerContext(ctx)

		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading config command line argument", zap.Error(err))
			return
		}
		chainID, err := cmd.Flags().GetString("chain-id")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading chain-id command line argument", zap.Error(err))
			return
		}
		config, err := cfg.LoadConfig(configPath)
		if err != nil {
			lmt.Logger(ctx).Error("Unable to load config", zap.Error(err))
			return
		}
		ctx = cfg.ConfigReaderContext(ctx, cfg.NewConfigReader(config))
		chainCfg, err := cfg.GetConfigReader(ctx).GetChainConfig(chainID)
		if err != nil {
			lmt.Logger(ctx).Error("Unable to read chain config", zap.Error(err))
			return
		}

		rateLimitedClient := utils.DefaultRateLimitedHTTPClient(3)
		coingeckoClient := coingecko.NewCoingeckoClient(rateLimitedClient, "https://api.coingecko.com/api/v3/", "")
		cachedCoinGeckoClient := coingecko.NewCachedPriceClient(coingeckoClient, 15*time.Minute)
		evmTxPriceOracle := evmrpc.NewOracle(cachedCoinGeckoClient)
		evmTxExecutor := evm.DefaultEVMTxExecutor()
		hype, err := hyperlane.NewMultiClientFromConfig(ctx, evmrpc.NewEVMRPCClientManager(), nil, evmTxPriceOracle, evmTxExecutor)
		if err != nil {
			lmt.Logger(ctx).Error("Error creating hyperlane multi client from config", zap.Error(err))
			return
		}
		messages, err := hype.ListHyperlaneMessageSentTxs(ctx, chainCfg.HyperlaneDomain, 2430975)
		if err != nil {
			lmt.Logger(ctx).Error("Error listing hyperlane messages", zap.Error(err))
			return
		}
		var deliveredMessages []types.HyperlaneMessage
		var undeliveredMessages []types.HyperlaneMessage
		for _, message := range messages {
			isDelivered, err := hype.HasBeenDelivered(ctx, message.DestinationDomain, message.MessageID)
			if err != nil {
				lmt.Logger(ctx).Error("Error checking if message has been delivered", zap.Error(err))
				return
			}
			if isDelivered {
				deliveredMessages = append(deliveredMessages, message)
			} else {
				undeliveredMessages = append(undeliveredMessages, message)
			}
		}
		lmt.Logger(ctx).Info("Delivered hyperlane messages", zap.Any("messages", deliveredMessages))
		lmt.Logger(ctx).Info("Undelivered hyperlane messages", zap.Any("messages", undeliveredMessages))
	},
}

func init() {
	rootCmd.AddCommand(listHyperlaneMessagesCmd)

	listHyperlaneMessagesCmd.Flags().String("chain-id", "", "chain the message is emitted from")
}
