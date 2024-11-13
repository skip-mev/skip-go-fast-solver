package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	rpcclienthttp "github.com/cometbft/cometbft/rpc/client/http"

	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/utils"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var loadTestCmd = &cobra.Command{
	Use:   "load-test",
	Short: "Execute concurrent fast transfers from EVM chains to Osmosis",
	Long: `Execute multiple concurrent fast transfers from each EVM chain to Osmosis.
Example:
  ./build/solvercli load-test \
  --config ./config/local/config.yml \
  --recipient osmo13c9seh3vgvtfvdufz4eh2zhp0cepq4wj0egc02 \
  --amount 1000000 \
  --private-key 0xf6079d30f832f998c86e5841385a4be06b6ca2b0875b90dcab8e167eba4dcab1`,
	Run: runLoadTest,
}

type loadTestFlags struct {
	configPath string
	recipient  string
	amount     string
	privateKey string
}

type OrderStatus struct {
	OrderID string
	ChainID string
	Status  string
}

func runLoadTest(cmd *cobra.Command, args []string) {
	flags, err := parseLoadTestFlags(cmd)
	if err != nil {
		fmt.Printf("Failed to parse flags: %v\n", err)
		return
	}

	cfg, err := config.LoadConfig(flags.configPath)
	if err != nil {
		fmt.Printf("Unable to load config: %v\n", err)
		return
	}

	var evmChains []string
	for chainID, chain := range cfg.Chains {
		if chain.Type == config.ChainType_EVM {
			if chain.EVM == nil {
				continue
			}
			evmChains = append(evmChains, chainID)
		}
	}

	orderChan := make(chan OrderStatus, len(evmChains)*6)
	errorChan := make(chan error, len(evmChains)*6)

	totalTransfers := len(evmChains) * 6
	var wg sync.WaitGroup
	wg.Add(totalTransfers)

	for _, sourceChain := range evmChains {
		chainCfg := cfg.Chains[sourceChain]
		for i := 0; i < 6; i++ {
			go func(chainID string, iteration int) {
				defer wg.Done()

				args := []string{
					"submit-transfer",
					"--config", flags.configPath,
					"--token", chainCfg.EVM.Contracts.USDCERC20Address,
					"--recipient", flags.recipient,
					"--amount", flags.amount,
					"--source-chain-id", chainID,
					"--destination-chain-id", "osmosis-1",
					"--gateway", chainCfg.EVM.FastTransferContractAddress,
					"--private-key", flags.privateKey,
				}

				fmt.Printf("Executing transfer %d from chain %s\n", iteration, chainID)
				submitCmd.SetArgs(args)
				result, err := submitTransfer(submitCmd, args)
				if err != nil {
					errorChan <- fmt.Errorf("Error executing transfer %d from chain %s: %v", iteration, chainID, err)
					return
				}

				orderChan <- OrderStatus{
					OrderID: result.OrderID,
					ChainID: chainID,
					Status:  "pending",
				}
			}(sourceChain, i)
		}
	}

	go func() {
		wg.Wait()
		close(orderChan)
		close(errorChan)
	}()

	var orders []OrderStatus
	for order := range orderChan {
		orders = append(orders, order)
	}

	for err := range errorChan {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
	checkOrderStatus(orders, &cfg)
}

func checkOrderStatus(orders []OrderStatus, cfg *config.Config) {
	ctx := context.Background()

	osmosisChainCfg := cfg.Chains["osmosis-1"]

	rpc, err := config.GetConfigReader(ctx).GetRPCEndpoint(osmosisChainCfg.ChainID)
	if err != nil {
		return
	}

	basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(osmosisChainCfg.ChainID)
	if err != nil {
		return
	}

	rpcClient, err := rpcclienthttp.NewWithClient(rpc, "/websocket", &http.Client{
		Transport: utils.NewBasicAuthTransport(basicAuth, http.DefaultTransport),
	})
	if err != nil {
		return
	}

	creds := insecure.NewCredentials()
	if osmosisChainCfg.Cosmos.GRPCTLSEnabled {
		creds = credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
	}
	grpcClient, err := grpc.Dial(osmosisChainCfg.Cosmos.GRPC, grpc.WithTransportCredentials(creds))
	if err != nil {
		return
	}

	client, err := cctp.NewCosmosBridgeClient(
		rpcClient,
		grpcClient,
		osmosisChainCfg.ChainID,
		osmosisChainCfg.Cosmos.AddressPrefix,
		nil, // we don't need a signer for querying
		osmosisChainCfg.Cosmos.GasPrice,
		osmosisChainCfg.Cosmos.GasDenom,
	)
	if err != nil {
		fmt.Printf("Failed to create Osmosis client: %v\n", err)
		return
	}
	defer client.Close()

	// Wait 30 seconds before checking order statuses
	time.Sleep(30)
	fmt.Printf("\nChecking status for %d orders on Osmosis:\n", len(orders))

	for _, order := range orders {
		fillTx, filler, timestamp, err := client.QueryOrderFillEvent(ctx, osmosisChainCfg.FastTransferContractAddress, order.OrderID)
		if err != nil {
			fmt.Printf("❌ Error checking fill status: %v\n", err)
			continue
		}

		if fillTx != nil && filler != nil {
			fmt.Printf("✅ Filled successfully!\n")
			fmt.Printf("Fill tx: %s\n", *fillTx)
			fmt.Printf("Filled by: %s\n", *filler)
			fmt.Printf("Timestamp: %s\n", timestamp)
			continue
		}
	}
}

func parseLoadTestFlags(cmd *cobra.Command) (*loadTestFlags, error) {
	flags := &loadTestFlags{}
	var err error

	if flags.configPath, err = cmd.Flags().GetString("config"); err != nil {
		return nil, err
	}
	if flags.recipient, err = cmd.Flags().GetString("recipient"); err != nil {
		return nil, err
	}
	if flags.amount, err = cmd.Flags().GetString("amount"); err != nil {
		return nil, err
	}
	if flags.privateKey, err = cmd.Flags().GetString("private-key"); err != nil {
		return nil, err
	}

	return flags, nil
}

func init() {
	rootCmd.AddCommand(loadTestCmd)

	loadTestCmd.Flags().String("recipient", "", "Recipient address on Osmosis")
	loadTestCmd.Flags().String("amount", "", "Amount to transfer (in token decimals)")
	loadTestCmd.Flags().String("private-key", "", "Sender wallet private key to sign the transactions")

	requiredFlags := []string{
		"recipient",
		"amount",
		"private-key",
	}

	for _, flag := range requiredFlags {
		if err := loadTestCmd.MarkFlagRequired(flag); err != nil {
			panic(fmt.Sprintf("failed to mark %s flag as required: %v", flag, err))
		}
	}
}
