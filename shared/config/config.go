package config

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types/bech32"
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
	Chains            map[string]ChainConfig          `yaml:"chains"`
	Metrics           MetricsConfig                   `yaml:"metrics"`
	Coingecko         CoingeckoConfig                 `yaml:"coingecko,omitempty"`
	OrderFillerConfig OrderFillerConfig               `yaml:"order_filler_config"`
	FundRebalancer    map[string]FundRebalancerConfig `yaml:"fund_rebalancer"`
}

type OrderFillerConfig struct {
	// OrderFillWorkerCount specifies the number of concurrent workers that will
	// process order fills. Each worker handles filling orders independently to
	// increase throughput.
	OrderFillWorkerCount int `yaml:"order_fill_worker_count"`

	// MinFeeBps is the min fee amount the solver is willing to fill in bps.
	// For example, if an order has an amount in of 100usdc and an amount out
	// of 99usdc, that is an implied fee to the solver of 1usdc, or a 1%/100bps
	// fee. Thus, if MinFeeBps is set to 200, and an order comes in with the
	// above amount in and out, then the solver will ignore it.
	MinFeeBps int `yaml:"min_fee_bps"`
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

type OrderSettlerConfig struct {
	UUSDCSettleUpThreshold string `yaml:"uusdc_settle_up_threshold"`
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
	// NumBlockConfirmationsBeforeFill is the number of block confirmations required
	// before the solver will attempt to fill an order
	NumBlockConfirmationsBeforeFill int64 `yaml:"num_block_confirmations_before_fill"`
	// HyperlaneDomain is the unique identifier for this chain in the Hyperlane
	// cross-chain messaging system
	HyperlaneDomain string `yaml:"hyperlane_domain"`
	// QuickStartNumBlocksBack specifies how many blocks back to start scanning
	// from when the solver is initialized
	QuickStartNumBlocksBack uint64 `yaml:"quick_start_num_blocks_back"`
	// MinFillSize is the minimum amount of USDC that can be processed in a single
	// order fill. Orders below this size will be abandoned
	MinFillSize big.Int `yaml:"min_fill_size"`
	// MaxFillSize is the maximum amount of USDC that can be processed in a single
	// order fill. Orders exceeding this size will be abandoned
	MaxFillSize big.Int `yaml:"max_fill_size"`
	// Maximum total gas cost for rebalancing txs per chain, fails if gas sum of rebalancing txs exceeds this threshold
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
	// BatchUUSDCSettleUpThreshold is the amount of uusdc that needs to
	// accumulate in filled (but not settled) orders before the solver will
	// initiate a batch settlement. A settlement batch is per (source chain,
	// destination chain).
	BatchUUSDCSettleUpThreshold string `yaml:"batch_uusdc_settle_up_threshold"`
	// MinFeeBps is the min fee amount the solver is willing to fill in bps.
	// For example, if an order has an amount in of 100usdc and an amount out
	// of 99usdc, that is an implied fee to the solver of 1usdc, or a 1%/100bps
	// fee. Thus, if MinFeeBps is set to 200, and an order comes in with the
	// above amount in and out, then the solver will ignore it.
	MinFeeBps int `yaml:"min_fee_bps"`
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

	MaxGasPricePct *uint8 `yaml:"max_gas_price_pct"`
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
	// USDCDenom is the denomination identifier for USDC on this chain
	// (typically an IBC denom hash for Cosmos chains)
	USDCDenom string `yaml:"usdc_denom"`
	// GasPrice is the amount of native tokens to pay per unit of gas
	GasPrice float64 `yaml:"gas_price"`
	// GasDenom is the denomination of the token used to pay for gas
	// (e.g., "uosmo" for Osmosis)
	GasDenom string `yaml:"gas_denom"`
}

type EVMConfig struct {
	// MinGasTipCap is the minimum tip to include for EIP-1559 transactions
	MinGasTipCap *int64 `yaml:"min_gas_tip_cap"`
	// FastTransferContractAddress is the address of Skip Go Fast
	// gateway contract on this chain
	FastTransferContractAddress string `yaml:"fast_transfer_contract_address"`
	// RPC is the HTTP endpoint for the EVM chain's RPC server
	RPC string `yaml:"rpc"`
	// RPCBasicAuthVar is the environment variable name containing the basic auth
	// credentials for the RPC endpoint if required
	RPCBasicAuthVar string `yaml:"rpc_basic_auth_var"`
	// GRPC is the endpoint for the chain's gRPC server
	GRPC string `yaml:"grpc"`
	// GRPCTLSEnabled indicates whether TLS should be used for gRPC connections
	GRPCTLSEnabled bool `yaml:"grpc_tls_enabled"`
	// GasBalance contains thresholds for monitoring the solver's gas balance
	SignerGasBalance SignerGasBalanceConfig `yaml:"signer_gas_balance"`
	// SolverAddress is the address of the solver wallet on this chain
	SolverAddress string `yaml:"solver_address"`
	// USDCDenom is the contract address of the USDC token on this chain
	USDCDenom string `yaml:"usdc_denom"`
	// Contracts contains addresses of various protocol contracts deployed on this chain
	Contracts ContractsConfig `yaml:"contracts"`
}

type ContractsConfig struct {
	// USDCERC20Address is the contract address of the USDC ERC20 token
	// deployed on this EVM chain. This is used for token transfers and
	// balance checks.
	USDCERC20Address string `yaml:"usdc_erc20_address"`
}

type CoingeckoConfig struct {
	BaseURL              string        `yaml:"base_url"`
	RequestsPerMinute    int           `yaml:"requests_per_minute"`
	APIKey               string        `yaml:"api_key"`
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

func (r configReader) GetSolverAddress(domain uint32, environment ChainEnvironment) (string, []byte, error) {
	domainIndex, ok := r.cctpDomainIndex[environment]
	if !ok {
		return "", nil, fmt.Errorf("cctp domain index not found for environment %s", environment)
	}

	chain, ok := domainIndex[domain]
	if !ok {
		return "", nil, fmt.Errorf("cctp domain %d not found for environment %s", domain, environment)
	}
	switch chain.Type {
	case ChainType_COSMOS:
		_, addressBytes, err := bech32.DecodeAndConvert(chain.SolverAddress)
		if err != nil {
			return "", nil, err
		}
		return chain.SolverAddress, addressBytes, nil
	case ChainType_EVM:
		addressBytes, err := hex.DecodeString(strings.TrimPrefix(chain.SolverAddress, "0x"))
		if err != nil {
			return "", nil, err
		}
		return chain.SolverAddress, addressBytes, nil
	default:
		return "", nil, fmt.Errorf("unknown chain type")
	}
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

	switch chainConfig.Type {
	case ChainType_COSMOS:
		if chainConfig.Cosmos == nil {
			return "", fmt.Errorf("cosmos config is nil for chain %s", chainID)
		}
		if chainConfig.Cosmos.USDCDenom == "" {
			return "", fmt.Errorf("usdc denom not configured for cosmos chain %s", chainID)
		}
		return chainConfig.Cosmos.USDCDenom, nil
	case ChainType_EVM:
		if chainConfig.EVM == nil || chainConfig.EVM.Contracts.USDCERC20Address == "" {
			return "", fmt.Errorf("usdc contract address not configured for evm chain %s", chainID)
		}
		return chainConfig.EVM.Contracts.USDCERC20Address, nil
	default:
		return "", fmt.Errorf("unsupported chain type %s for chain %s", chainConfig.Type, chainID)
	}
}
