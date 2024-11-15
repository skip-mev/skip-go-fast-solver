package cmd

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	rpcclienthttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

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
	configPath  string
	recipient   string
	amount      string
	privateKey  string
	nonceMutex  sync.Mutex
	chainNonces map[string]uint64
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

	ctx := context.Background()
	ctx = config.ConfigReaderContext(ctx, config.NewConfigReader(cfg))

	// Pass the context to submitCmd
	submitCmd.SetContext(ctx)

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

	flags.chainNonces = make(map[string]uint64)

	for _, sourceChain := range evmChains {
		chainCfg := cfg.Chains[sourceChain]
		chainConfig := chainCfg
		client, err := ethclient.Dial(chainConfig.EVM.RPC)
		defer client.Close()
		if err != nil {
			fmt.Printf("Failed to connect to network: %v\n", err)
			return
		}

		privateKey := flags.privateKey
		if privateKey[:2] == "0x" {
			privateKey = privateKey[2:]
		}
		key, err := crypto.HexToECDSA(privateKey)
		if err != nil {
			fmt.Printf("Failed to parse private key: %v\n", err)
			return
		}

		address := crypto.PubkeyToAddress(key.PublicKey)
		startingNonce, err := client.PendingNonceAt(ctx, address)
		if err != nil {
			fmt.Printf("Failed to get starting nonce: %v\n", err)
			return
		}

		flags.chainNonces[sourceChain] = startingNonce

		for i := 0; i < 6; i++ {
			chainID := sourceChain
			iteration := i

			go func() {
				defer wg.Done()

				// This assumes all transactions pass
				flags.nonceMutex.Lock()
				currentNonce := flags.chainNonces[chainID]
				flags.chainNonces[chainID]++
				flags.nonceMutex.Unlock()

				localCmd := &cobra.Command{}
				localCmd.SetContext(ctx)

				// Add all required flags from submitCmd
				localCmd.PersistentFlags().String("config", "./config/local/config.yml", "Path to config file")
				localCmd.Flags().String("token", "", "Token address to transfer")
				localCmd.Flags().String("recipient", "", "Recipient address")
				localCmd.Flags().String("amount", "", "Amount to transfer")
				localCmd.Flags().String("source-chain-id", "", "Source chain ID")
				localCmd.Flags().String("destination-chain-id", "", "Destination chain ID")
				localCmd.Flags().String("gateway", "", "Gateway contract address")
				localCmd.Flags().String("private-key", "", "Private key to sign the transaction")
				localCmd.Flags().Uint32("deadline-hours", 1, "Deadline in hours")
				localCmd.Flags().Uint64("nonce", currentNonce, "Transaction nonce")

				if err := localCmd.PersistentFlags().Set("config", flags.configPath); err != nil {
					errorChan <- fmt.Errorf("setting config flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("token", chainConfig.EVM.Contracts.USDCERC20Address); err != nil {
					errorChan <- fmt.Errorf("setting token flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("recipient", flags.recipient); err != nil {
					errorChan <- fmt.Errorf("setting recipient flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("amount", flags.amount); err != nil {
					errorChan <- fmt.Errorf("setting amount flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("source-chain-id", chainID); err != nil {
					errorChan <- fmt.Errorf("setting source-chain-id flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("destination-chain-id", "osmosis-1"); err != nil {
					errorChan <- fmt.Errorf("setting destination-chain-id flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("gateway", chainConfig.FastTransferContractAddress); err != nil {
					errorChan <- fmt.Errorf("setting gateway flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("private-key", flags.privateKey); err != nil {
					errorChan <- fmt.Errorf("setting private-key flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("deadline-hours", "1"); err != nil {
					errorChan <- fmt.Errorf("setting deadline-hours flag: %v", err)
					return
				}
				if err := localCmd.Flags().Set("nonce", fmt.Sprintf("%d", currentNonce)); err != nil {
					errorChan <- fmt.Errorf("setting nonce flag: %v", err)
					return
				}

				fmt.Printf("Executing transfer %d from chain %s\n", iteration, chainID)
				result, err := submitTransfer(localCmd, []string{})
				if err != nil {
					errorChan <- fmt.Errorf("executing transfer %d from chain %s: %v", iteration, chainID, err)
					return
				}

				orderChan <- OrderStatus{
					OrderID: result.OrderID,
					ChainID: chainID,
					Status:  "pending",
				}
			}()
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

	checkOrderStatus(orders, cfg, ctx)
}

func checkOrderStatus(orders []OrderStatus, cfg config.Config, ctx context.Context) {
	osmosisChainCfg := cfg.Chains["osmosis-1"]

	rpc, err := config.GetConfigReader(ctx).GetRPCEndpoint(osmosisChainCfg.ChainID)
	if err != nil {
		fmt.Printf("Error getting RPC endpoint: %v\n", err)
		return
	}

	basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(osmosisChainCfg.ChainID)
	if err != nil {
		fmt.Printf("Error getting basic auth: %v\n", err)
		return
	}

	rpcClient, err := rpcclienthttp.NewWithClient(rpc, "/websocket", &http.Client{
		Transport: utils.NewBasicAuthTransport(basicAuth, http.DefaultTransport),
	})
	if err != nil {
		fmt.Printf("Error creating RPC client: %v\n", err)
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
		fmt.Printf("Error creating gRPC client: %v\n", err)
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

	fmt.Println("Sleeping for 30 seconds before querying fill orders status")
	time.Sleep(30 * time.Second)
	fmt.Printf("\nChecking status for %d orders on Osmosis:\n", len(orders))

	for _, order := range orders {
		fillTx, filler, timestamp, err := client.QueryOrderFillEvent(ctx, osmosisChainCfg.FastTransferContractAddress, order.OrderID)
		if err != nil {
			fmt.Printf("❌ Error checking fill status for order %s: %v\n", order.OrderID, err)
			continue
		}

		if fillTx != nil && filler != nil {
			fmt.Printf("✅ Order %s filled successfully!\n", order.OrderID)
			fmt.Printf("Fill tx: %s\n", *fillTx)
			fmt.Printf("Filled by: %s\n", *filler)
			fmt.Printf("Timestamp: %s\n", timestamp)
		} else {
			fmt.Printf("⏳ Order %s is still pending\n", order.OrderID)
		}
	}
}

func parseLoadTestFlags(cmd *cobra.Command) (*loadTestFlags, error) {
	flags := &loadTestFlags{}
	var err error

	if flags.configPath, err = cmd.Root().PersistentFlags().GetString("config"); err != nil {
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
