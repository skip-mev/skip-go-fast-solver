package config

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Config Enum Types
type ChainType string

const (
	ChainType_COSMOS ChainType = "cosmos"
	ChainType_EVM    ChainType = "evm"
)

type ChainEnvironment string

const (
	ChainEnvironment_MAINNET ChainEnvironment = "mainnet"
	ChainEnvironment_TESTNET ChainEnvironment = "testnet"
)

// Config Schema
type Config struct {
	Chains                map[string]ChainConfig `yaml:"chains"`
	Metrics               MetricsConfig          `yaml:"metrics"`
	OrderFillerConfig     OrderFillerConfig      `yaml:"order_filler_config"`
	TransferMonitorConfig TransferMonitorConfig  `yaml:"transfer_monitor_config"`
	Coingecko             CoingeckoConfig
	// FundRebalancer is an optional configuration to aid in inventory
	// management. You can set per chain target amounts and min allowed
	// amounts, and the FundRebalancer will use skip go to move funds between
	// chains to maintain these values.
	FundRebalancer map[string]FundRebalancerConfig `yaml:"fund_rebalancer"`
}

type OrderFillerConfig struct {
	// OrderFillWorkerCount specifies the number of concurrent workers that will
	// process order fills. Each worker handles filling orders independently to
	// increase throughput.
	OrderFillWorkerCount int `yaml:"order_fill_worker_count"`
}

type MetricsConfig struct {
	// PrometheusAddress is the address where the Prometheus metrics server will
	// listen for scrape requests. This enables monitoring of solver performance
	// and order processing statistics.
	PrometheusAddress string `yaml:"prometheus_address"`
}

type FundRebalancerConfig struct {
	// TargetAmount is the amount of uusdc that you would like ot maintain on
	// this chain. The fund rebalancer will take uusdc from configured chains
	// that are above their target amount and move the uusdc to other chains
	// that are below their MinAllowedAmount.
	TargetAmount string `yaml:"target_amount"`
	// MinAllowedAmount is the minimum amount of uusdc that this chain can hold
	// before a rebalance is triggered to move uusdc from other chains to this
	// chain.
	MinAllowedAmount string `yaml:"min_allowed_amount"`
}

type TransferMonitorConfig struct {
	// PollInterval controls how often the transfer monitor will query the chain for new orders
	PollInterval *time.Duration `yaml:"poll_interval"`
}

type ChainConfig struct {
	// e.g. osmosis
	ChainName string `yaml:"chain_name"`
	// e.g. osmosis-1
	ChainID string `yaml:"chain_id"`
	// (cosmos, evm)
	Type ChainType `yaml:"type"`
	// Environment specifies whether this is a mainnet or testnet configuration
	Environment ChainEnvironment `yaml:"environment"`
	// Cosmos contains specific configuration for Cosmos-based chains
	Cosmos *CosmosConfig `yaml:"cosmos,omitempty"`
	// EVM contains specific configuration for Ethereum Virtual Machine based chains
	EVM *EVMConfig `yaml:"evm,omitempty"`
	// GasTokenSymbol is the symbol of the native gas token (e.g., "ETH", "MATIC")
	GasTokenSymbol string `yaml:"gas_token_symbol"`
	// GasTokenDecimals specifies the number of decimal places for the gas token
	GasTokenDecimals uint8 `yaml:"gas_token_decimals"`
	// GasTokenCoingeckoID is the coingecko ID of the chain's gas token
	GasTokenCoingeckoID string `yaml:"gas_token_coingecko_id"`
	// NumBlockConfirmationsBeforeFill is the number of block confirmations required
	// before the solver will attempt to fill an order
	NumBlockConfirmationsBeforeFill int64 `yaml:"num_block_confirmations_before_fill"`
	// HyperlaneDomain is the unique identifier for this chain in the Hyperlane
	// cross-chain messaging system
	HyperlaneDomain string `yaml:"hyperlane_domain"`
	// QuickStartNumBlocksBack specifies how many blocks back to start scanning
	// from when the solver is initialized
	QuickStartNumBlocksBack uint64 `yaml:"quick_start_num_blocks_back"`
	// Maximum total gas cost for rebalancing txs per chain, fails if gas sum
	// of rebalancing txs exceeds this threshold
	MaxRebalancingGasThreshold uint64 `yaml:"max_rebalancing_gas_threshold"`
	// FastTransferContractAddress is the address of the Skip Go Fast Transfer
	// Protocol contract deployed on this chain
	FastTransferContractAddress string `yaml:"fast_transfer_contract_address"`
	// SolverAddress is the address of the solver wallet on this chain that will
	// be used to fulfill orders and receive fees
	SolverAddress string `yaml:"solver_address"`
	// USDCDenom is the denomination or contract address for USDC on this chain
	// (ERC20 contract address for EVM chains or IBC denom for Cosmos chains)
	USDCDenom string `yaml:"usdc_denom"`
	// Relayer contains configuration for the Hyperlane relayer service
	// used for cross-chain message passing during settlement
	Relayer RelayerConfig `yaml:"relayer"`

	/* *** SETTING THE FOLLOWING CONFIG VALUES ARE VERY IMPORTANT FOR SOLVER PROFITABILITY *** */

	// MinFeeBps is the min fee amount the solver is willing to fill in bps.
	// For example, if an order has an amount in of 100usdc and an amount out
	// of 99usdc, that is an implied fee to the solver of 1usdc, or a 1%/100bps
	// fee. Thus, if MinFeeBps is set to 200, and an order comes in with the
	// above amount in and out, then the solver will ignore it.
	MinFeeBps int `yaml:"min_fee_bps"`

	// BatchUUSDCSettleUpThreshold is the amount of uusdc that needs to
	// accumulate in filled (but not settled) orders before the solver will
	// initiate a batch settlement. A settlement batch is per source chain and
	// destination chain pair. Note that this amount is for the total amount
	// being settled up, not just the profit that will be made.
	BatchUUSDCSettleUpThreshold string `yaml:"batch_uusdc_settle_up_threshold"`

	// MinProfitMarginBPS is the minimum amount of bps that the solver should
	// make when settling order batches. This value should be set carefully as
	// it is used to determine what the max tx fee that should be paid to
	// settle a batch of orders in order to maintain your set profit margin.
	// Thus, this value should always be set to a lower value than the
	// MinFeeBps, since your profit margin must be less than the actual profit
	// (you have to pay some tx fee). Below is an equation that shows how this
	// value will be used when settling up.
	//
	// (NetSettlementProfit - TxFee) / TotalSettlementValue = MinProfitMargin
	//
	// Where:
	// NetSettlementProfit = total amount in of orders in settlement batch -
	//   total amount out of orders in settlement batch.
	// and,
	// TotalSettlementValue = total amount in of orders in settlement batch.
	//
	// To determine the TxFee, we can rearrange the equation as follows.
	//
	// NetSettlementProfit - (TotalSettlementValue * MinProfitMargin) = TxFee
	//
	// Here you can see the relationship between how MinProfitMarginBPS,
	// BatchUUSDCSettleUpThreshold, and MinFeeBps all relate to each other. As
	// you increase BatchUUSDCSettleUpThreshold, the TotalSettlementValue of
	// each batch will increase. As you increase the MinFeeBps, the
	// NetSettlementProfit will increase, and as you increase
	// MinProfitMarginBPS, the max TxFee you are willing to pay to get your
	// settlement landed on chain will decrease. So, all three of these values
	// should be set with care for each chain, based on solver fund reserves on
	// this chain, typical gas costs, and expected minimum fees to be paid by
	// users to submit orders on this chain.
	//
	// As an example, lets say MinFeeBps is set to 20bps,
	// BatchUUSDCSettleUpThreshold is set to 5000000000uusdc (5 usdc), and
	// MinProfitMarginBPS is set to 15bps. When a settlement happens, you can
	// expect a typical batch to have a total value of 5000000000 uusdc, and a
	// profit of 10000000 uusdc (5000usdc and 10usdc, respectively). Using the
	// above formula, we can calculate the max TxFee that we can pay to land
	// the settlement on chain in order to maintain the MinProfitMarginBPS of
	// 15bps.
	//
	// 10000000uusdc - (5000000000uusdc * (20bps / 10000)) = 2500000uusdc
	//
	// Thus, the solver will not submit the settlement on chain if simulating
	// the submission and converting the gas cost to uusdc is > 2500000uusdc.
	// So, if these were you actual numbers, you should be sure that the gas
	// cost will be lower than 2500000uusdc on this chain to land the
	// settlement. This number may be OK for a cheap L2 like Arbitrum, however
	// it would likely be impossible to land a settlement tx on Ethereum
	// mainnet for only 2.5usdc paid in tx fees (you would never receive your
	// profit!).
	//
	// As an extreme example, lets say you keep the above values but set
	// MinProfitMarginBPS to 0bps. Applying the same formula to determine the
	// max TxFee that we can pay to land the settlement on chain in order to
	// maintain the MinProfitMarginBPS of 0bps.
	//
	// 10000000uusdc - (5000000000uusdc * (0bps / 10000)) = 10000000uusdc
	//
	// This means that the solver is willing to (potentially) use all of its
	// profit on the TxFee to settle up (you most likely do not want this).
	//
	// As a final example, if you set the MinProfitMarginBPS higher than your
	// MinFeeBps. For exmaple if MinProfitMarginBPS is 25bps and MinFeeBps is
	// 20bps. Then applying the same formula to determine the max TxFee that we
	// can pay to land the settlement on chain in order to maintain the
	// MinProfitMarginBPS of 25bps.
	//
	// 10000000uusdc - (5000000000uusdc * (25bps / 10000)) = -2500000uusdc
	//
	// The result is now a negative tx fee. This means that chain would need to
	// pay the solver in order to land the settlement tx on chain to maintain
	// the profit margin of 25bps, this is obviously impossible and the tx will
	// never land on chain. The solver will log an error if it sees this
	// occurring.
	MinProfitMarginBPS int `yaml:"min_profit_margin_bps"`
}

type RelayerConfig struct {
	// ValidatorAnnounceContractAddress is the address of the Hyperlane validator
	// announce contract used for cross-chain message validation
	ValidatorAnnounceContractAddress string `yaml:"validator_announce_contract_address"`
	// MerkleHookContractAddress is the address of the Hyperlane merkle hook
	// contract used for verifying cross-chain message proofs
	MerkleHookContractAddress string `yaml:"merkle_hook_contract_address"`
	// MailboxAddress is the address of the Hyperlane mailbox contract used
	// for sending and receiving cross-chain messages
	MailboxAddress string `yaml:"mailbox_address"`

	// ProfitableRelayTimeout is the maximum amount of time delay relaying a
	// transaction waiting for it to be profitable. Currently this only applies
	// to settlement relays. For example, if you have your MinProfitMarginBPS
	// set too high relative to current gas fees on the settle up chain, then
	// the relay will be delayed indefinitely until the gas fees reach a
	// certain level (which they may never reach). Once a tx has been attempted
	// to be relayed for ProfitableRelayTimeout duration, but it hasnt been
	// sent because it is not profitable, then it will be sent regardless of
	// profitability. This can be set to -1 for no timeout.
	ProfitableRelayTimeout *time.Duration `yaml:"profitable_relay_timeout"`

	// RelayCostCapUUSDC is the maximum amount of uusdc to pay to relay a tx,
	// regardless of profitability checking, i.e. if the ProfitableRelayTimeout
	// expires for a tx and it is going to be sent without ensuring it is
	// profitable for the solver to do so, this is a final check to ensure that
	// the tx is not relayed in an extremely expensive block.
	RelayCostCapUUSDC string `yaml:"relay_cost_cap_uusdc"`
}

// Used to monitor gas balance prometheus metric per chain for the solver addresses
type SignerGasBalanceConfig struct {
	// WarningThresholdWei specifies the gas balance threshold in Wei below which the solver
	// gas balance metric for this chain will be set to Warning level
	WarningThresholdWei string `yaml:"warning_threshold_wei"`
	// CriticalThresholdWei specifies the gas balance threshold in Wei
	// below which solver operations may be impacted
	CriticalThresholdWei string `yaml:"critical_threshold_wei"`
}

type CosmosConfig struct {
	// RPC is the HTTP endpoint for the Cosmos chain's RPC server
	RPC string `yaml:"rpc"`
	// RPCBasicAuthVar is the environment variable name containing the basic auth
	// credentials for the RPC endpoint if required
	RPCBasicAuthVar string `yaml:"rpc_basic_auth_var"`
	// GRPC is the endpoint for the chain's gRPC server
	GRPC string `yaml:"grpc"`
	// GRPCTLSEnabled indicates whether TLS should be used for gRPC connections
	GRPCTLSEnabled bool `yaml:"grpc_tls_enabled"`
	// AddressPrefix is the bech32 prefix used for addresses on this chain
	// (e.g., "osmo" for Osmosis addresses)
	AddressPrefix string `yaml:"address_prefix"`
	// GasBalance contains thresholds for monitoring the solver's gas balance
	SignerGasBalance SignerGasBalanceConfig `yaml:"signer_gas_balance"`
	// GasPrice is the amount of native tokens to pay per unit of gas
	GasPrice float64 `yaml:"gas_price"`
	// GasDenom is the denomination of the token used to pay for gas
	// (e.g., "uosmo" for Osmosis)
	GasDenom string `yaml:"gas_denom"`
	// MinFillSize is the minimum amount of USDC that can be processed in a single
	// order fill. Orders below this size will be abandoned
	MinFillSize big.Int `yaml:"min_fill_size"`
	// MaxFillSize is the maximum amount of USDC that can be processed in a single
	// order fill. Orders exceeding this size will be abandoned
	MaxFillSize big.Int `yaml:"max_fill_size"`
}

type EVMConfig struct {
	// MinGasTipCap is the minimum tip to include for EIP-1559 transactions
	// If the gas price oracle price returns a lower tip than MinGasTipCap, MinGasTipCap is used
	// Used mainly for Polygon where there is a network gas tip cap minimum and nodes frequently return values lower
	// than it
	MinGasTipCap *int64 `yaml:"min_gas_tip_cap"`
	// RPC is the HTTP endpoint for the EVM chain's RPC server
	RPC string `yaml:"rpc"`
	// RPCBasicAuthVar is the environment variable name containing the basic auth
	// credentials for the RPC endpoint if required
	RPCBasicAuthVar string `yaml:"rpc_basic_auth_var"`
	// GasBalance contains thresholds for monitoring the solver's gas balance
	SignerGasBalance SignerGasBalanceConfig `yaml:"signer_gas_balance"`
	// SolverAddress is the address of the solver wallet on this chain
	SolverAddress string `yaml:"solver_address"`
}

type CoingeckoConfig struct {
	// BaseURL is the coingecko api url used to fetch token prices
	BaseURL string `yaml:"base_url"`
	// RequestsPerMinute is the max amount of requests allowed to be made to
	// the coin gecko api per minute
	RequestsPerMinute int `yaml:"requests_per_minute"`
	// APIKey is optional. If you do not have an API key, you can remove the
	// APIKey option all together. If you have a coin gecko API key, we will
	// use it to get more up to date gas costs. If you specify an API key, you
	// should reduce the requests per minute and cache refresh interval
	// according to your keys limits.
	APIKey string `yaml:"api_key"`
	// CacheRefreshInterval is how long the internal coin gecko client will
	// cache prices for. Set this accoridng to your coin gecko's plans rate
	// limits (if you have one).
	CacheRefreshInterval time.Duration `yaml:"cache_refresh_interval"`
}

// Config Helpers
func LoadConfig(path string) (Config, error) {
	cfgBytes, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	if err := yaml.Unmarshal(cfgBytes, &config); err != nil {
		return Config{}, err
	}

	for chainID, chainConfig := range config.Chains {
		if err := ValidateChainConfig(chainConfig); err != nil {
			return Config{}, fmt.Errorf("invalid configuration for chain %s: %w", chainID, err)
		}
	}

	return config, nil
}

// ConfigReader Context Helpers

type configContextKey struct{}

func ConfigReaderContext(ctx context.Context, reader ConfigReader) context.Context {
	return context.WithValue(ctx, configContextKey{}, reader)
}

func GetConfigReader(ctx context.Context) ConfigReader {
	return ctx.Value(configContextKey{}).(ConfigReader)
}

// Complex Config Queries

type ConfigReader interface {
	Config() Config

	GetChainEnvironment(chainID string) (ChainEnvironment, error)
	GetRPCEndpoint(chainID string) (string, error)
	GetBasicAuth(chainID string) (*string, error)

	GetChainConfig(chainID string) (ChainConfig, error)
	GetAllChainConfigsOfType(chainType ChainType) ([]ChainConfig, error)

	GetCoingeckoConfig() CoingeckoConfig

	GetGatewayContractAddress(chainID string) (string, error)
	GetChainIDByHyperlaneDomain(domain string) (string, error)

	GetUSDCDenom(chainID string) (string, error)

	GetGasAlertThresholds(chainID string) (warningThreshold, criticalThreshold *big.Int, err error)
}

type configReader struct {
	config          Config
	cctpDomainIndex map[ChainEnvironment]map[uint32]ChainConfig
	chainIDIndex    map[string]ChainConfig
}

func NewConfigReader(config Config) ConfigReader {
	r := &configReader{
		config: config,
	}
	r.createIndexes()
	return r
}

func (r *configReader) createIndexes() {
	r.cctpDomainIndex = make(map[ChainEnvironment]map[uint32]ChainConfig)
	r.chainIDIndex = make(map[string]ChainConfig)

	for _, chain := range r.config.Chains {
		if _, ok := r.cctpDomainIndex[chain.Environment]; !ok {
			r.cctpDomainIndex[chain.Environment] = make(map[uint32]ChainConfig)
		}

		// Validate chain configuration
		if chain.Type == ChainType_COSMOS && chain.Cosmos == nil {
			lmt.Logger(context.Background()).Error(
				"invalid chain configuration",
				zap.String("chainID", chain.ChainID),
				zap.String("type", string(chain.Type)),
				zap.Bool("hasCosmosConfig", chain.Cosmos != nil),
			)
		}

		if chain.Type == ChainType_EVM && chain.EVM == nil {
			lmt.Logger(context.Background()).Error(
				"invalid chain configuration",
				zap.String("chainID", chain.ChainID),
				zap.String("type", string(chain.Type)),
				zap.Bool("hasEVMConfig", chain.EVM != nil),
			)
		}

		r.chainIDIndex[chain.ChainID] = chain

		lmt.Logger(context.Background()).Debug(
			"indexed chain configuration",
			zap.String("chainID", chain.ChainID),
			zap.Any("chainConfig", chain))
	}
}

func (r configReader) Config() Config {
	return r.config
}

func (r configReader) GetChainEnvironment(chainID string) (ChainEnvironment, error) {
	chain, ok := r.chainIDIndex[chainID]
	if !ok {
		return "", fmt.Errorf("chain id %s not found", chainID)
	}

	return chain.Environment, nil
}

func (r configReader) GetRPCEndpoint(chainID string) (string, error) {
	chain, ok := r.chainIDIndex[chainID]
	if !ok {
		return "", fmt.Errorf("chain id %s not found", chainID)
	}

	switch chain.Type {
	case ChainType_COSMOS:
		return chain.Cosmos.RPC, nil
	case ChainType_EVM:
		return chain.EVM.RPC, nil
	}

	return "", fmt.Errorf("unknown chain type")
}

func (r configReader) GetBasicAuth(chainID string) (*string, error) {
	chain, ok := r.chainIDIndex[chainID]
	if !ok {
		return nil, fmt.Errorf("chain id %s not found", chainID)
	}

	var basicAuthVar string
	switch chain.Type {
	case ChainType_COSMOS:
		basicAuthVar = chain.Cosmos.RPCBasicAuthVar
	case ChainType_EVM:
		basicAuthVar = chain.EVM.RPCBasicAuthVar
	}

	if basicAuth, ok := os.LookupEnv(basicAuthVar); ok {
		return &basicAuth, nil
	}

	return nil, nil
}

func (r configReader) GetChainConfig(chainID string) (ChainConfig, error) {
	chain, ok := r.chainIDIndex[chainID]
	if !ok {
		return ChainConfig{}, fmt.Errorf("chain id %s not found", chainID)
	}

	return chain, nil
}

func (r configReader) GetAllChainConfigsOfType(chainType ChainType) ([]ChainConfig, error) {
	var chains []ChainConfig
	for _, chain := range r.config.Chains {
		if chain.Type == chainType {
			chains = append(chains, chain)
		}
	}
	return chains, nil
}

func (r configReader) GetCoingeckoConfig() CoingeckoConfig {
	return r.config.Coingecko
}

func (r configReader) GetGatewayContractAddress(chainID string) (string, error) {
	chain, ok := r.chainIDIndex[chainID]
	if !ok {
		return "", fmt.Errorf("chain id %s not found", chainID)
	}
	switch chain.Type {
	case ChainType_COSMOS:
		return chain.FastTransferContractAddress, nil
	case ChainType_EVM:
		return chain.FastTransferContractAddress, nil
	default:
		return "", fmt.Errorf("unknown chain type")
	}
}

func (r configReader) GetChainIDByHyperlaneDomain(domain string) (string, error) {
	for chainID, cfg := range r.chainIDIndex {
		if cfg.HyperlaneDomain == domain {
			return chainID, nil
		}
	}
	return "", fmt.Errorf("no chain found for Hyperlane domain %s", domain)
}

// GetUSDCDenom gets the configured denom for USDC on a given chain (usdc erc20
// contract address for evm or ibc denom hash for cosmos).
func (r configReader) GetUSDCDenom(chainID string) (string, error) {
	chainConfig, ok := r.chainIDIndex[chainID]
	if !ok {
		return "", fmt.Errorf("chain id %s not found", chainID)
	}

	return chainConfig.USDCDenom, nil
}

func ValidateChainConfig(chain ChainConfig) error {
	if chain.ChainName == "" {
		return fmt.Errorf("chain_name is required")
	}
	if chain.ChainID == "" {
		return fmt.Errorf("chain_id is required")
	}
	if chain.Type == "" {
		return fmt.Errorf("type is required")
	}
	if chain.Environment == "" {
		return fmt.Errorf("environment is required")
	}
	if chain.GasTokenSymbol == "" {
		return fmt.Errorf("gas_token_symbol is required")
	}
	if chain.GasTokenDecimals == 0 {
		return fmt.Errorf("gas_token_decimals is required")
	}
	if chain.NumBlockConfirmationsBeforeFill == 0 {
		return fmt.Errorf("num_block_confirmations_before_fill is required")
	}
	if chain.HyperlaneDomain == "" {
		return fmt.Errorf("hyperlane_domain is required")
	}
	if chain.QuickStartNumBlocksBack == 0 {
		return fmt.Errorf("quick_start_num_blocks_back is required")
	}
	if chain.MaxRebalancingGasThreshold == 0 {
		return fmt.Errorf("max_rebalancing_gas_threshold is required")
	}
	if chain.FastTransferContractAddress == "" {
		return fmt.Errorf("fast_transfer_contract_address is required")
	}
	if chain.SolverAddress == "" {
		return fmt.Errorf("solver_address is required")
	}
	if chain.USDCDenom == "" {
		return fmt.Errorf("usdc_denom is required")
	}
	if chain.MinProfitMarginBPS > chain.MinFeeBps {
		return fmt.Errorf("min_profit_margin_bps can not be > min_fee_bps")
	}
	if chain.Relayer.ProfitableRelayTimeout == nil {
		return fmt.Errorf("relayer.profitable_relay_timeout is required")
	}
	if chain.Relayer.RelayCostCapUUSDC == "" {
		return fmt.Errorf("relayer.relay_cost_cap_u_usdc is required")
	}
	if chain.Relayer.MailboxAddress == "" {
		return fmt.Errorf("relayer.mailbox_address is required")
	}

	switch chain.Type {
	case ChainType_COSMOS:
		if chain.Cosmos == nil {
			return fmt.Errorf("cosmos config is required for cosmos chain type")
		}
		return validateCosmosConfig(chain.Cosmos, &chain.Relayer)
	case ChainType_EVM:
		if chain.BatchUUSDCSettleUpThreshold == "" {
			return fmt.Errorf("batch_uusdc_settle_up_threshold is required")
		}
		if chain.MinFeeBps == 0 {
			return fmt.Errorf("min_fee_bps is required")
		}
		if chain.MinProfitMarginBPS == 0 {
			return fmt.Errorf("min_profit_margin_bps is required")
		}

		if chain.EVM == nil {
			return fmt.Errorf("evm config is required for evm chain type")
		}
		return validateEVMConfig(chain.EVM)
	default:
		return fmt.Errorf("invalid chain type: %s", chain.Type)
	}
}

func validateCosmosConfig(config *CosmosConfig, relayerConfig *RelayerConfig) error {
	if config.RPC == "" {
		return fmt.Errorf("cosmos.rpc is required")
	}
	if config.GRPC == "" {
		return fmt.Errorf("cosmos.grpc is required")
	}
	if config.AddressPrefix == "" {
		return fmt.Errorf("cosmos.address_prefix is required")
	}
	if config.GasPrice == 0 {
		return fmt.Errorf("cosmos.gas_price is required")
	}
	if config.GasDenom == "" {
		return fmt.Errorf("cosmos.gas_denom is required")
	}
	if config.MinFillSize.Sign() < 0 {
		return fmt.Errorf("cosmos.min_fill_size is required")
	}
	if config.MaxFillSize.Sign() < 0 {
		return fmt.Errorf("cosmos.max_fill_size is required")
	}
	if config.MaxFillSize.Cmp(&config.MinFillSize) < 0 {
		return fmt.Errorf("cosmos.max_fill_size must be greater than min_fill_size")
	}

	if config.SignerGasBalance.WarningThresholdWei == "" {
		return fmt.Errorf("cosmos.signer_gas_balance.warning_threshold_wei is required")
	}
	if config.SignerGasBalance.CriticalThresholdWei == "" {
		return fmt.Errorf("cosmos.signer_gas_balance.critical_threshold_wei is required")
	}

	if relayerConfig.ValidatorAnnounceContractAddress == "" {
		return fmt.Errorf("relayer.validator_announce_contract_address is required")
	}
	if relayerConfig.MerkleHookContractAddress == "" {
		return fmt.Errorf("relayer.merkle_hook_contract_address is required")
	}

	return nil
}

func validateEVMConfig(config *EVMConfig) error {
	if config.RPC == "" {
		return fmt.Errorf("evm.rpc is required")
	}

	if config.SignerGasBalance.WarningThresholdWei == "" {
		return fmt.Errorf("evm.signer_gas_balance.warning_threshold_wei is required")
	}
	if config.SignerGasBalance.CriticalThresholdWei == "" {
		return fmt.Errorf("evm.signer_gas_balance.critical_threshold_wei is required")
	}

	return nil
}

func (r configReader) GetGasAlertThresholds(chainID string) (warningThreshold, criticalThreshold *big.Int, err error) {
	var warningThresholdString, criticalThresholdString string

	chain, err := r.GetChainConfig(chainID)
	if err != nil {
		return nil, nil, err
	}
	switch chain.Type {
	case ChainType_COSMOS:
		warningThresholdString = chain.Cosmos.SignerGasBalance.WarningThresholdWei
		criticalThresholdString = chain.Cosmos.SignerGasBalance.CriticalThresholdWei
	case ChainType_EVM:
		warningThresholdString = chain.EVM.SignerGasBalance.WarningThresholdWei
		criticalThresholdString = chain.EVM.SignerGasBalance.CriticalThresholdWei
	default:
		return nil, nil, fmt.Errorf("unknown chain type")
	}

	warningThreshold, ok := new(big.Int).SetString(warningThresholdString, 10)
	if !ok {
		return nil, nil, fmt.Errorf("failed to parse gas balance threshold amount")
	}
	criticalThreshold, ok = new(big.Int).SetString(criticalThresholdString, 10)
	if !ok {
		return nil, nil, fmt.Errorf("failed to parse gas balance threshold amount")
	}

	return warningThreshold, criticalThreshold, nil
}
