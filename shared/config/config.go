package config

import (
	"context"
	"encoding/hex"
	"fmt"
	bech322 "github.com/cosmos/cosmos-sdk/types/bech32"
	"gopkg.in/yaml.v3"
	"math/big"
	"os"
	"strings"
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
	OrderFillerConfig OrderFillerConfig               `yaml:"order_filler_config"`
	FundRebalancer    map[string]FundRebalancerConfig `yaml:"fund_rebalancer"`
}

type OrderFillerConfig struct {
	OrderFillWorkerCount int `yaml:"order_fill_worker_count"`
}

type MetricsConfig struct {
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
	ChainName                       string           `yaml:"chain_name"`
	ChainID                         string           `yaml:"chain_id"`
	Type                            ChainType        `yaml:"type"`
	Environment                     ChainEnvironment `yaml:"environment"`
	Cosmos                          *CosmosConfig    `yaml:"cosmos,omitempty"`
	EVM                             *EVMConfig       `yaml:"evm,omitempty"`
	GasTokenSymbol                  string           `yaml:"gas_token_symbol"`
	GasTokenDecimals                uint8            `yaml:"gas_token_decimals"`
	NumBlockConfirmationsBeforeFill int64            `yaml:"num_block_confirmations_before_fill"`
	HyperlaneDomain                 string           `yaml:"hyperlane_domain"`
	QuickStartNumBlocksBack         uint64           `yaml:"quick_start_num_blocks_back"`
	MaxRebalancingGasThreshold      uint64           `yaml:"max_rebalancing_gas_threshold"` // Maximum total gas cost for rebalancing txs per chain
	MinFillSize                     *big.Int         `yaml:"min_fill_size"`
	MaxFillSize                     *big.Int         `yaml:"max_fill_size"`
	FastTransferContractAddress     string           `yaml:"fast_transfer_contract_address"`
	SolverAddress                   string           `yaml:"solver_address"`
	USDCDenom                       string           `yaml:"usdc_denom"`
	Relayer                         RelayerConfig    `yaml:"relayer"`

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
	ValidatorAnnounceContractAddress string `yaml:"validator_announce_contract_address"`
	MerkleHookContractAddress        string `yaml:"merkle_hook_contract_address"`
	MailboxAddress                   string `yaml:"mailbox_address"`
}

type SignerGasBalanceConfig struct {
	WarningThresholdWei  string `yaml:"warning_threshold_wei"`
	CriticalThresholdWei string `yaml:"critical_threshold_wei"`
}

type CosmosConfig struct {
	RPC              string                 `yaml:"rpc"`
	RPCBasicAuthVar  string                 `yaml:"rpc_basic_auth_var"`
	GRPC             string                 `yaml:"grpc"`
	GRPCTLSEnabled   bool                   `yaml:"grpc_tls_enabled"`
	AddressPrefix    string                 `yaml:"address_prefix"`
	SignerGasBalance SignerGasBalanceConfig `yaml:"signer_gas_balance"`
	USDCDenom        string                 `yaml:"usdc_denom"`
	GasPrice         float64                `yaml:"gas_price"`
	GasDenom         string                 `yaml:"gas_denom"`
}

type EVMConfig struct {
	MinGasTipCap                *int64                 `yaml:"min_gas_tip_cap"`
	ChainID                     string                 `yaml:"chain_id"`
	FastTransferContractAddress string                 `yaml:"fast_transfer_contract_address"`
	RPC                         string                 `yaml:"rpc"`
	RPCBasicAuthVar             string                 `yaml:"rpc_basic_auth_var"`
	GRPC                        string                 `yaml:"grpc"`
	GRPCTLSEnabled              bool                   `yaml:"grpc_tls_enabled"`
	AddressPrefix               string                 `yaml:"address_prefix"`
	SignerGasBalance            SignerGasBalanceConfig `yaml:"signer_gas_balance"`
	SolverAddress               string                 `yaml:"solver_address"`
	USDCDenom                   string                 `yaml:"usdc_denom"`
	Contracts                   ContractsConfig        `yaml:"contracts"`
}

type ContractsConfig struct {
	USDCERC20Address string `yaml:"usdc_erc20_address"`
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
		r.chainIDIndex[chain.ChainID] = chain
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
		_, addressBytes, err := bech322.DecodeAndConvert(chain.SolverAddress)
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
		return chainConfig.Cosmos.USDCDenom, nil
	case ChainType_EVM:
		return chainConfig.EVM.Contracts.USDCERC20Address, nil
	default:
		return "", fmt.Errorf("no usdc denom available for chain type %s", chainConfig.Type)
	}
}
