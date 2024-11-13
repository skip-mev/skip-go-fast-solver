/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"encoding/json"
	"time"

	"os/signal"
	"syscall"

	"github.com/skip-mev/go-fast-solver/hyperlane"
	"github.com/skip-mev/go-fast-solver/shared/clients/coingecko"
	"github.com/skip-mev/go-fast-solver/shared/clients/utils"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"go.uber.org/zap"
	"golang.org/x/net/context"

	"github.com/spf13/cobra"
)

// relayCmd represents the relay command
var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "manually relay a transaction",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		lmt.ConfigureLogger()
		ctx = lmt.LoggerContext(ctx)

		keysPath, err := cmd.Flags().GetString("keys")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading keys command line argument", zap.Error(err))
			return
		}
		keyStoreType, err := cmd.Flags().GetString("key-store-type")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading key-store-type command line argument", zap.Error(err))
			return
		}
		aesKeyHex, err := cmd.Flags().GetString("aes-key-hex")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading aes-key-hex command line argument", zap.Error(err))
			return
		}
		configPath, err := cmd.Flags().GetString("config")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading config command line argument", zap.Error(err))
			return
		}
		originChainID, err := cmd.Flags().GetString("origin-chain-id")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading origin-chain-id command line argument", zap.Error(err))
			return
		}
		originTxHash, err := cmd.Flags().GetString("origin-tx-hash")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading origin-tx-hash command line argument", zap.Error(err))
			return
		}
		storageOverrideMapJson, err := cmd.Flags().GetString("checkpoint-storage-location-override")
		if err != nil {
			lmt.Logger(ctx).Error("Error reading keys command line argument", zap.Error(err))
			return
		}
		var storageOverrideMap map[string]string
		err = json.Unmarshal([]byte(storageOverrideMapJson), &storageOverrideMap)
		if err != nil {
			lmt.Logger(ctx).Error("Error unmarshalling storage override map", zap.Error(err))
			return
		}
		promMetrics := metrics.NewPromMetrics()
		ctx = metrics.ContextWithMetrics(ctx, promMetrics)

		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			lmt.Logger(ctx).Error("Unable to load config", zap.Error(err))
			return
		}
		ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

		keyStore, err := keys.GetKeyStore(keyStoreType, keys.GetKeyStoreOpts{KeyFilePath: keysPath, AESKeyHex: aesKeyHex})
		if err != nil {
			lmt.Logger(ctx).Error("Unable to load keystore", zap.Error(err))
			return
		}

		rateLimitedClient := utils.DefaultRateLimitedHTTPClient(3)
		coingeckoClient := coingecko.NewCoingeckoClient(rateLimitedClient, "https://api.coingecko.com/api/v3/", "")
		cachedCoinGeckoClient := coingecko.NewCachedPriceClient(coingeckoClient, 15*time.Minute)
		evmTxPriceOracle := evmrpc.NewOracle(cachedCoinGeckoClient)

		hype, err := hyperlane.NewMultiClientFromConfig(ctx, evmrpc.NewEVMRPCClientManager(), keyStore, evmTxPriceOracle)
		if err != nil {
			lmt.Logger(ctx).Error("Error creating hyperlane multi client from config", zap.Error(err))
		}

		destinationTxHash, destinationChainID, err := hyperlane.NewRelayer(hype, storageOverrideMap).Relay(ctx, originChainID, originTxHash, nil)
		if err != nil {
			lmt.Logger(ctx).Error("Error relaying message", zap.Error(err))
			return
		}
		lmt.Logger(ctx).Info(
			"Successfully relayed message",
			zap.String("tx_hash", destinationTxHash),
			zap.String("chain_id", destinationChainID))
	},
}

func init() {
	rootCmd.AddCommand(relayCmd)

	relayCmd.Flags().String("origin-chain-id", "", "chain the message is emitted from")
	relayCmd.Flags().String("origin-tx-hash", "", "transaction the message emitted from")
	relayCmd.Flags().String("checkpoint-storage-location-override", "{}", "map of validator addresses to storage locations")
}
