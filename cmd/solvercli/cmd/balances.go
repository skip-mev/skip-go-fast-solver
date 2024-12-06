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
	Use:   "balances",
	Short: "Show current on-chain balances (USDC and gas tokens)",
	Long: `Show current on-chain balances for USDC and gas tokens across all configured chains.
    
Example:
    ./build/solvercli balances \
    --custom-assets '{"osmosis-1":["uosmo","uion"],"celestia-1":["utia"]}'`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := setupContext(cmd)
		usdcBalances, gasBalances, customBalances, totalUSDCBalance, totalCustomAssetsUSDValue, err := getBalances(ctx, cmd)
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
}

func getBalances(ctx context.Context, cmd *cobra.Command) (map[string]*ChainBalance, map[string]*ChainGasBalance, map[string][]*ChainBalance, *big.Int, *big.Float, error) {
	usdcBalances := make(map[string]*ChainBalance)
	gasBalances := make(map[string]*ChainGasBalance)
	customBalances := make(map[string][]*ChainBalance)
	totalUSDCBalance := new(big.Int)
	totalCustomAssetsUSDValue := new(big.Float)

	skipClient, err := skipgo.NewSkipGoClient("https://api.skip.build")
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("creating skip client: %w", err)
	}

	request := &skipgo.BalancesRequest{
		Chains: make(map[string]skipgo.ChainRequest),
	}

	customAssetsFlag := cmd.Flags().Lookup("custom-assets").Value.String()
	customAssetMap := make(map[string][]string)
	if customAssetsFlag != "" {
		if err := json.Unmarshal([]byte(customAssetsFlag), &customAssetMap); err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("parsing custom-assets JSON: %w", err)
		}
	}

	chains := config.GetConfigReader(ctx).Config().Chains
	for _, chain := range chains {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chain.ChainID)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("getting chain config for %s: %w", chain.ChainID, err)
		}

		denoms := []string{chainConfig.USDCDenom, chainConfig.GasTokenSymbol}
		if customDenoms, ok := customAssetMap[chain.ChainID]; ok {
			denoms = append(denoms, customDenoms...)
		}

		request.Chains[chain.ChainID] = skipgo.ChainRequest{
			Address: chainConfig.SolverAddress,
			Denoms:  denoms,
		}
	}

	skipResp, err := skipClient.Balance(ctx, request)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("getting balances from Skip API: %w", err)
	}

	for chainID, chainResp := range skipResp.Chains {
		chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
		if err != nil {
			return nil, nil, nil, nil, nil, fmt.Errorf("getting chain config for %s: %w", chainID, err)
		}

		// Process USDC balance
		if usdcDetail, ok := chainResp.Denoms[chainConfig.USDCDenom]; ok {
			balance, ok := new(big.Int).SetString(usdcDetail.Amount, 10)
			if !ok {
				return nil, nil, nil, nil, nil, fmt.Errorf("invalid USDC amount for chain %s", chainID)
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

		// Process gas token balance
		if gasDetail, ok := chainResp.Denoms[chainConfig.GasTokenSymbol]; ok {
			balance, ok := new(big.Int).SetString(gasDetail.Amount, 10)
			if !ok {
				return nil, nil, nil, nil, nil, fmt.Errorf("invalid gas token amount for chain %s", chainID)
			}

			warningThreshold, criticalThreshold, err := config.GetConfigReader(ctx).GetGasAlertThresholds(chainID)
			if err != nil {
				return nil, nil, nil, nil, nil, fmt.Errorf("getting gas alert thresholds for %s: %w", chainID, err)
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

		// Process custom assets
		if customDenoms, ok := customAssetMap[chainID]; ok {
			customBalances[chainID] = make([]*ChainBalance, 0)
			for _, denom := range customDenoms {
				if detail, ok := chainResp.Denoms[denom]; ok {
					balance, ok := new(big.Int).SetString(detail.Amount, 10)
					if !ok {
						return nil, nil, nil, nil, nil, fmt.Errorf("invalid amount for chain %s denom %s", chainID, denom)
					}

					valueUSD, ok := new(big.Float).SetString(detail.ValueUSD)
					if !ok {
						return nil, nil, nil, nil, nil, fmt.Errorf("invalid USD value for chain %s denom %s", chainID, denom)
					}

					price, ok := new(big.Float).SetString(detail.Price)
					if !ok {
						return nil, nil, nil, nil, nil, fmt.Errorf("invalid price for chain %s denom %s", chainID, denom)
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
			}
		}
	}

	return usdcBalances, gasBalances, customBalances, totalUSDCBalance, totalCustomAssetsUSDValue, nil
}
