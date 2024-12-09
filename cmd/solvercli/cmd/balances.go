package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/skip-mev/go-fast-solver/shared/clients/skipgo"
	"github.com/skip-mev/go-fast-solver/shared/config"
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
	Use:     "balances",
	Short:   "Show current on-chain balances for USDC, gas tokens, and custom assets across all configured chains.",
	Long:    `Show current on-chain balances for USDC, gas tokens, and custom assets across all configured chains.`,
	Example: `solver balances --custom-assets '{"osmosis-1":["uosmo","uion"],"celestia-1":["utia"]}'`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupContext(cmd)

		usdcBalances := make(map[string]*ChainBalance)
		gasBalances := make(map[string]*ChainGasBalance)
		customBalances := make(map[string][]*ChainBalance)
		totalUSDCBalance := new(big.Int)
		totalCustomAssetsUSDValue := new(big.Float)
		err := getBalances(ctx, usdcBalances, gasBalances, customBalances, totalUSDCBalance, totalCustomAssetsUSDValue, cmd)
		if err != nil {
			lmt.Logger(ctx).Fatal("Failed to get balances", zap.Error(err))
		}

		fmt.Println("\nOn-Chain Balances:")
		fmt.Println("------------------")

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
				}
			}
		}

		fmt.Printf("\nTotals Across All Chains:")
		fmt.Printf("\n------------------------\n")
		fmt.Printf("Total USDC Balance: %s USDC\n", normalizeBalance(totalUSDCBalance, CCTP_TOKEN_DECIMALS))
		if totalCustomAssetsUSDValue.Cmp(big.NewFloat(0)) > 0 {
			fmt.Printf("Total Custom Assets USD Value: %.2f USD\n", totalCustomAssetsUSDValue)
		}
	},
}

func init() {
	rootCmd.AddCommand(balancesCmd)
	balancesCmd.Flags().String("custom-assets", "", "JSON map of chain IDs to denom arrays, e.g. '{\"osmosis\":[\"uosmo\",\"uion\"],\"celestia\":[\"utia\"]}'")
	balancesCmd.Flags().String("evm-address", "", "Optional EVM address to check balances for instead of config address")
	balancesCmd.Flags().String("osmosis-address", "", "Optional Osmosis address to check balances for instead of config address")
}

func getBalances(ctx context.Context,
	usdcBalances map[string]*ChainBalance,
	gasBalances map[string]*ChainGasBalance,
	customBalances map[string][]*ChainBalance,
	totalUSDCBalance *big.Int,
	totalCustomAssetsUSDValue *big.Float,
	cmd *cobra.Command) error {
	request, err := buildBalancesRequest(ctx, cmd)
	if err != nil {
		return fmt.Errorf("building balances request: %w", err)
	}

	skipResp, err := fetchBalances(ctx, request)
	if err != nil {
		return fmt.Errorf("fetching balances: %w", err)
	}

	lmt.Logger(ctx).Info("resp", zap.Any("", skipResp))
	for chainID, chainResp := range skipResp.Chains {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
		if err != nil {
			return fmt.Errorf("getting chain config for %s: %w", chainID, err)
		}

		if err := processUSDCBalance(chainID, chainConfig, chainResp, usdcBalances, totalUSDCBalance); err != nil {
			return fmt.Errorf("processing USDC balance: %w", err)
		}

		if err := processGasBalance(ctx, chainID, chainConfig, chainResp, gasBalances); err != nil {
			return fmt.Errorf("processing gas balance: %w", err)
		}

		if err := processCustomBalances(chainID, chainConfig, chainResp, customBalances, totalCustomAssetsUSDValue); err != nil {
			return fmt.Errorf("processing custom balances: %w", err)
		}
	}

	return nil
}

func buildBalancesRequest(ctx context.Context, cmd *cobra.Command) (*skipgo.BalancesRequest, error) {
	request := &skipgo.BalancesRequest{
		Chains: make(map[string]skipgo.ChainRequest),
	}

	customAssetMap := make(map[string][]string)
	if customAssetsFlag := cmd.Flags().Lookup("custom-assets").Value.String(); customAssetsFlag != "" {
		if err := json.Unmarshal([]byte(customAssetsFlag), &customAssetMap); err != nil {
			return nil, fmt.Errorf("parsing custom-assets JSON: %w", err)
		}
	}

	evmAddress := cmd.Flags().Lookup("evm-address").Value.String()
	osmosisAddress := cmd.Flags().Lookup("osmosis-address").Value.String()

	chains := config.GetConfigReader(ctx).Config().Chains
	for _, chain := range chains {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chain.ChainID)
		if err != nil {
			return nil, fmt.Errorf("getting chain config for %s: %w", chain.ChainID, err)
		}

		address := chainConfig.SolverAddress
		if chainConfig.Type == config.ChainType_EVM && evmAddress != "" {
			address = evmAddress
		} else if chainConfig.Type == config.ChainType_COSMOS && osmosisAddress != "" {
			address = osmosisAddress
		}

		var gasTokenDenom string
		if chainConfig.Type == config.ChainType_COSMOS {
			gasTokenDenom = chainConfig.Cosmos.GasDenom
		} else if chainConfig.Type == config.ChainType_EVM {
			gasTokenDenom = chainConfig.ChainName + "-native"
		}

		denoms := []string{chainConfig.USDCDenom, gasTokenDenom}
		if customDenoms, ok := customAssetMap[chain.ChainID]; ok {
			denoms = append(denoms, customDenoms...)
		}

		request.Chains[chain.ChainID] = skipgo.ChainRequest{
			Address: address,
			Denoms:  denoms,
		}
	}

	lmt.Logger(ctx).Info("", zap.Any("request", request))

	return request, nil
}

func fetchBalances(ctx context.Context, request *skipgo.BalancesRequest) (*skipgo.BalancesResponse, error) {
	skipClient, err := skipgo.NewSkipGoClient("https://api.skip.build")
	if err != nil {
		return nil, fmt.Errorf("creating skip client: %w", err)
	}

	return skipClient.Balance(ctx, request)
}

func processUSDCBalance(
	chainID string,
	chainConfig config.ChainConfig,
	chainResp skipgo.ChainResponse,
	usdcBalances map[string]*ChainBalance,
	totalUSDCBalance *big.Int,
) error {
	if usdcDetail, ok := chainResp.Denoms[chainConfig.USDCDenom]; ok {
		balance, ok := new(big.Int).SetString(usdcDetail.Amount, 10)
		if !ok {
			return fmt.Errorf("invalid USDC amount for chain %s", chainID)
		}

		usdcBalances[chainID] = &ChainBalance{
			ChainID:    chainID,
			AssetDenom: chainConfig.USDCDenom,
			Balance:    balance,
			Symbol:     "USDC",
			Decimals:   CCTP_TOKEN_DECIMALS,
		}
		totalUSDCBalance.Add(totalUSDCBalance, balance)
	}
	return nil
}

func processGasBalance(
	ctx context.Context,
	chainID string,
	chainConfig config.ChainConfig,
	chainResp skipgo.ChainResponse,
	gasBalances map[string]*ChainGasBalance,
) error {
	lmt.Logger(ctx).Info("", zap.Any("", chainResp))

	var gasTokenDenom string
	if chainConfig.Type == config.ChainType_COSMOS {
		gasTokenDenom = chainConfig.Cosmos.GasDenom
	} else if chainConfig.Type == config.ChainType_EVM {
		gasTokenDenom = chainConfig.ChainName + "-native"
	}

	if gasDetail, ok := chainResp.Denoms[gasTokenDenom]; ok {
		balance, ok := new(big.Int).SetString(gasDetail.Amount, 10)
		if !ok {
			return fmt.Errorf("invalid gas token amount for chain %s", chainID)
		}

		warningThreshold, criticalThreshold, err := config.GetConfigReader(ctx).GetGasAlertThresholds(chainID)
		if err != nil {
			return fmt.Errorf("getting gas alert thresholds for %s: %w", chainID, err)
		}

		gasBalances[chainID] = &ChainGasBalance{
			ChainID:           chainID,
			Balance:           balance,
			Symbol:            chainConfig.GasTokenSymbol,
			Decimals:          chainConfig.GasTokenDecimals,
			WarningThreshold:  warningThreshold,
			CriticalThreshold: criticalThreshold,
		}
	}
	return nil
}

func processCustomBalances(
	chainID string,
	chainConfig config.ChainConfig,
	chainResp skipgo.ChainResponse,
	customBalances map[string][]*ChainBalance,
	totalCustomAssetsUSDValue *big.Float,
) error {
	for denom, detail := range chainResp.Denoms {
		// Skip USDC and gas token
		if denom == chainConfig.USDCDenom || denom == chainConfig.GasTokenSymbol {
			continue
		}

		balance, ok := new(big.Int).SetString(detail.Amount, 10)
		if !ok {
			return fmt.Errorf("invalid amount for chain %s denom %s", chainID, denom)
		}

		valueUSD, ok := new(big.Float).SetString(detail.ValueUSD)
		if !ok {
			return fmt.Errorf("invalid USD value for chain %s denom %s", chainID, denom)
		}

		price, ok := new(big.Float).SetString(detail.Price)
		if !ok {
			return fmt.Errorf("invalid price for chain %s denom %s", chainID, denom)
		}

		if _, exists := customBalances[chainID]; !exists {
			customBalances[chainID] = make([]*ChainBalance, 0)
		}

		customBalances[chainID] = append(customBalances[chainID], &ChainBalance{
			ChainID:    chainID,
			AssetDenom: denom,
			Balance:    balance,
			Decimals:   detail.Decimals,
			PriceUSD:   price,
			ValueUSD:   valueUSD,
		})

		totalCustomAssetsUSDValue.Add(totalCustomAssetsUSDValue, valueUSD)
	}
	return nil
}
