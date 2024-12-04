package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type SkipBalancesRequest struct {
	Chains map[string]ChainRequest `json:"chains"`
}

type ChainRequest struct {
	Address string   `json:"address"`
	Denoms  []string `json:"denoms"`
}

type SkipBalancesResponse struct {
	Chains map[string]ChainResponse `json:"chains"`
}

type ChainResponse struct {
	Address string                 `json:"address"`
	Denoms  map[string]DenomDetail `json:"denoms"`
}

type DenomDetail struct {
	Amount          string `json:"amount"`
	Decimals        uint8  `json:"decimals"`
	FormattedAmount string `json:"formatted_amount"`
	Price           string `json:"price"`
	ValueUSD        string `json:"value_usd"`
}

type ChainBalance struct {
	ChainID    string
	AssetDenom string
	Balance    *big.Int
	Symbol     string
	Decimals   uint8
	PriceUSD   *big.Float
	ValueUSD   *big.Float
}

type ChainGasBalance struct {
	ChainID           string
	Balance           *big.Int
	Symbol            string
	Decimals          uint8
	WarningThreshold  *big.Int
	CriticalThreshold *big.Int
}

var balancesCmd = &cobra.Command{
	Use:   "balances",
	Short: "Show current on-chain balances (USDC and gas tokens)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupContext(cmd)
		evmClientManager, cctpClientManager := setupClients(ctx, cmd)

		usdcBalances, gasBalances, err := getChainBalances(ctx, evmClientManager, cctpClientManager)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get USDC balances", zap.Error(err))
		}

		customBalances, totalUSDValue, err := getCustomAssetUSDTotalValue(ctx, cmd, evmClientManager, cctpClientManager)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get custom asset balances", zap.Error(err))
		}

		fmt.Println("\nOn-Chain Balances:")
		fmt.Println("------------------")

		totalUSDCBalance := new(big.Int)
		for chainID, usdc := range usdcBalances {
			gas := gasBalances[chainID]
			fmt.Printf("\nChain: %s\n", chainID)
			fmt.Printf("  USDC Balance: %s USDC\n", normalizeBalance(usdc.Balance, CCTP_TOKEN_DECIMALS))
			fmt.Printf("  Gas Balance: %s %s\n", normalizeBalance(gas.Balance, gas.Decimals), gas.Symbol)

			if gas.Balance.Cmp(gas.CriticalThreshold) < 0 {
				fmt.Printf("  ⚠️  Gas balance below critical threshold!\n")
			} else if gas.Balance.Cmp(gas.WarningThreshold) < 0 {
				fmt.Printf("  ⚠️  Gas balance below warning threshold\n")
			}

			// Print custom assets if available
			if assets, ok := customBalances[chainID]; ok {
				for _, asset := range assets {
					fmt.Printf("  %s Balance: %s %s (%.2f USD)\n",
						asset.AssetDenom,
						normalizeBalance(asset.Balance, asset.Decimals),
						asset.AssetDenom,
						asset.ValueUSD)
					totalUSDValue.Add(totalUSDValue, asset.ValueUSD)
				}
			}

			totalUSDCBalance.Add(totalUSDCBalance, usdc.Balance)
		}

		fmt.Printf("\nTotals Across All Chains:")
		fmt.Printf("\n------------------------\n")
		fmt.Printf("Total USDC Balance: %s USDC\n", normalizeBalance(totalUSDCBalance, CCTP_TOKEN_DECIMALS))
		if totalUSDValue.Cmp(big.NewFloat(0)) > 0 {
			fmt.Printf("Total Custom Assets USD Value: %.2f USD\n", totalUSDValue)
		}
	},
}

func init() {
	rootCmd.AddCommand(balancesCmd)
	balancesCmd.Flags().String("custom-assets", "", "JSON map of chain IDs to denom arrays, e.g. '{\"osmosis\":[\"uosmo\",\"uion\"],\"celestia\":[\"utia\"]}'")
}

// returns USDC and gas balances for all configured chains
func getChainBalances(ctx context.Context, evmClientManager evmrpc.EVMRPCClientManager,
	cctpClientManager *clientmanager.ClientManager) (map[string]*ChainBalance, map[string]*ChainGasBalance, error) {
	balances := make(map[string]*ChainBalance)
	gasBalances := make(map[string]*ChainGasBalance)
	chains := config.GetConfigReader(ctx).Config().Chains

	for _, chain := range chains {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chain.ChainID)
		if err != nil {
			return nil, nil, fmt.Errorf("getting chain config for %s: %w", chain.ChainID, err)
		}

		cctpClient, err := cctpClientManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			return nil, nil, fmt.Errorf("getting cctpClient for %s: %w", chain.ChainID, err)
		}

		var balance *big.Int
		switch chainConfig.Type {
		case config.ChainType_EVM:
			client, err := evmClientManager.GetClient(ctx, chain.ChainID)
			if err != nil {
				return nil, nil, fmt.Errorf("getting evm client for chain %s: %w", chain.ChainID, err)
			}
			balance, err = client.GetUSDCBalance(ctx, chainConfig.USDCDenom, chainConfig.SolverAddress)
			if err != nil {
				return nil, nil, fmt.Errorf("getting balance for %s: %w", chain.ChainID, err)
			}

		case config.ChainType_COSMOS:
			balance, err = cctpClient.Balance(ctx, chainConfig.SolverAddress, chainConfig.USDCDenom)
			if err != nil {
				return nil, nil, fmt.Errorf("getting balance for %s: %w", chain.ChainID, err)
			}
		}

		gasBalance, err := cctpClient.SignerGasTokenBalance(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("getting gas balance for %s: %w", chain.ChainID, err)
		}

		warningThreshold, criticalThreshold, err := config.GetConfigReader(ctx).GetGasAlertThresholds(chain.ChainID)
		if err != nil {
			return nil, nil, fmt.Errorf("getting gas alert thresholds for %s: %w", chain.ChainID, err)
		}

		gasBalances[chain.ChainID] = &ChainGasBalance{
			ChainID:           chain.ChainID,
			Balance:           gasBalance,
			Symbol:            chainConfig.GasTokenSymbol,
			Decimals:          chainConfig.GasTokenDecimals,
			WarningThreshold:  warningThreshold,
			CriticalThreshold: criticalThreshold,
		}

		balances[chain.ChainID] = &ChainBalance{
			ChainID:    chain.ChainID,
			AssetDenom: chainConfig.USDCDenom,
			Balance:    balance,
			Symbol:     "USDC",
			Decimals:   CCTP_TOKEN_DECIMALS,
		}
	}

	return balances, gasBalances, nil
}

func getCustomAssetsBalances(ctx context.Context,
	evmClientManager evmrpc.EVMRPCClientManager,
	cctpClientManager *clientmanager.ClientManager,
	requestMap map[string][]string) (map[string][]*ChainBalance, error) {

	request := &skipgo.BalancesRequest{
		Chains: make(map[string]skipgo.ChainRequest),
	}

	for chainID, denoms := range requestMap {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
		if err != nil {
			return nil, fmt.Errorf("getting chain config for %s: %w", chainID, err)
		}

		request.Chains[chainID] = skipgo.ChainRequest{
			Address: chainConfig.SolverAddress,
			Denoms:  denoms,
		}
	}

	skipClient, err := skipgo.NewSkipGoClient("https://api.skip.build")
	if err != nil {
		return nil, fmt.Errorf("creating skip client: %w", err)
	}

	skipResp, err := skipClient.Balance(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("getting balances from Skip API: %w", err)
	}

	balances := make(map[string][]*ChainBalance)
	for chainID, chainResp := range skipResp.Chains {
		balances[chainID] = make([]*ChainBalance, 0)
		for denom, detail := range chainResp.Denoms {
			balance, ok := new(big.Int).SetString(detail.Amount, 10)
			if !ok {
				return nil, fmt.Errorf("invalid amount for chain %s denom %s", chainID, denom)
			}

			price, ok := new(big.Float).SetString(detail.Price)
			if !ok {
				return nil, fmt.Errorf("invalid price for chain %s denom %s", chainID, denom)
			}

			valueUSD, ok := new(big.Float).SetString(detail.ValueUSD)
			if !ok {
				return nil, fmt.Errorf("invalid USD value for chain %s denom %s", chainID, denom)
			}

			balances[chainID] = append(balances[chainID], &ChainBalance{
				ChainID:    chainID,
				AssetDenom: denom,
				Balance:    balance,
				Decimals:   detail.Decimals,
				PriceUSD:   price,
				ValueUSD:   valueUSD,
			})
		}
	}

	return balances, nil
}

func getCustomAssetUSDTotalValue(ctx context.Context, cmd *cobra.Command, evmClientManager evmrpc.EVMRPCClientManager, cctpClientManager *clientmanager.ClientManager) (map[string][]*ChainBalance, *big.Float, error) {
	customAssetsFlag := cmd.Flags().Lookup("custom-assets").Value.String()
	requestMap := make(map[string][]string)
	if customAssetsFlag != "" {
		if err := json.Unmarshal([]byte(customAssetsFlag), &requestMap); err != nil {
			return nil, nil, fmt.Errorf("parsing custom-assets JSON: %w", err)
		}
	}

	var customBalances map[string][]*ChainBalance
	totalUSDValue := new(big.Float)

	if len(requestMap) > 0 {
		var err error
		customBalances, err = getCustomAssetsBalances(ctx, evmClientManager, cctpClientManager, requestMap)
		if err != nil {
			return nil, nil, fmt.Errorf("getting custom asset balances: %w", err)
		}

		// Calculate total USD value
		for _, assets := range customBalances {
			for _, asset := range assets {
				totalUSDValue.Add(totalUSDValue, asset.ValueUSD)
			}
		}
	}

	return customBalances, totalUSDValue, nil
}
