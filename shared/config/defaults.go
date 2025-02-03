package config

import (
	"math/big"
	"net"
	"time"
)

var (
	defaultProfitableRelayTimeout      = time.Hour
	defaultBatchUUSDCSettleUpThreshold = "10000000"

	DefaultOrderFillerConfig = OrderFillerConfig{
		OrderFillWorkerCount: 10,
	}

	trasferMonitorPollInterval   = time.Second * 5
	DefaultTransferMonitorConfig = TransferMonitorConfig{
		PollInterval: &trasferMonitorPollInterval,
	}

	prometheusHost       = "0.0.0.0"
	prometheusPort       = "8001"
	DefaultMetricsConfig = MetricsConfig{
		PrometheusAddress: net.JoinHostPort(prometheusHost, prometheusPort),
	}

	coingeckoURL           = "https://pro-api.coingecko.com/api/v3/"
	DefaultCoinGeckoConfig = CoingeckoConfig{
		BaseURL:              coingeckoURL,
		RequestsPerMinute:    2,
		CacheRefreshInterval: time.Minute * 15,
	}

	DefaultFundRebalancerConfig map[string]FundRebalancerConfig = nil

	DefaultChainsConfig = map[string]ChainConfig{
		"osmosis-1": DefaultOsmosisConfig,
		"42161":     DefaultArbitrumConfig,
		"10":        DefaultOptimismConfig,
		"1":         DefaultEthereumConfig,
		"137":       DefaultPolygonConfig,
		"43114":     DefaultAvalancheConfig,
		"8453":      DefaultBaseConfig,
	}

	osmosisMinFillSizeUUSDC = big.NewInt(500000)
	osmosisMaxFillSizeUUSDC = big.NewInt(100000000)
	DefaultOsmosisConfig    = ChainConfig{
		ChainName:   "osmosis",
		ChainID:     "osmosis-1",
		Type:        ChainType_COSMOS,
		Environment: ChainEnvironment_MAINNET,
		Relayer: RelayerConfig{
			ValidatorAnnounceContractAddress: "osmo147r8mfdsngswujgkr4tln9rhcrzz6yq0xn448ksd96mlcmp9wg6stvznke",
			MerkleHookContractAddress:        "osmo1e765uc5mctl7rz8dzl9decl5ghgxggeqyxutkjp2xkggrg6zma3qgdq2g4",
			MailboxAddress:                   "osmo1r6u37zv47ke4d2k9tkzun72ch466w6594kv8gqgrtmsvf7qxpm9sj95v98",
		},
		Cosmos: &CosmosConfig{
			AddressPrefix: "osmos",
			SignerGasBalance: SignerGasBalanceConfig{
				// 2 osmo
				WarningThresholdWei: "2000000",
				// 0.5 osmo
				CriticalThresholdWei: "500000",
			},
			GasPrice:    0.0025,
			GasDenom:    "uosmo",
			MinFillSize: osmosisMinFillSizeUUSDC,
			MaxFillSize: osmosisMaxFillSizeUUSDC,
		},
		NumBlockConfirmationsBeforeFill: 1,
		QuickStartNumBlocksBack:         30000,
		GasTokenSymbol:                  "OSMO",
		GasTokenDecimals:                6,
		GasTokenCoingeckoID:             "osmosis",
		HyperlaneDomain:                 "875",
		FastTransferContractAddress:     "osmo1vy34lpt5zlj797w7zqdta3qfq834kapx88qtgudy7jgljztj567s73ny82",
		USDCDenom:                       "ibc/498A0751C798A0D9A389AA3691123DADA57DAA4FE165D5C75894505B876BA6E4",
	}

	DefaultEthereumConfig = ChainConfig{
		ChainName:   "ethereum",
		ChainID:     "1",
		Type:        ChainType_EVM,
		Environment: ChainEnvironment_MAINNET,
		// do not fill an order unless it contains 10 bps of profit
		MinFeeBps: 10,
		// settle up on ethereum after filling 1,000 usdc worth of orders from ethereum
		BatchUUSDCSettleUpThreshold: "1000000000",
		// maintain a 5 bps profit margin when settling up
		MinProfitMarginBPS: 5,
		Relayer: RelayerConfig{
			MailboxAddress:         "0xc005dc82818d67AF737725bD4bf75435d065D239",
			RelayCostCapUUSDC:      "40000000",
			ProfitableRelayTimeout: &defaultProfitableRelayTimeout,
		},
		EVM: &EVMConfig{
			SignerGasBalance: SignerGasBalanceConfig{
				WarningThresholdWei:  "1000000",
				CriticalThresholdWei: "1000000",
			},
		},
		NumBlockConfirmationsBeforeFill: 1,
		QuickStartNumBlocksBack:         30000,
		GasTokenSymbol:                  "ETH",
		GasTokenDecimals:                18,
		GasTokenCoingeckoID:             "ethereum",
		HyperlaneDomain:                 "1",
		FastTransferContractAddress:     "0xe7935104c9670015b21c6300e5b95d2f75474cda",
		USDCDenom:                       "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	}

	DefaultArbitrumConfig = ChainConfig{
		ChainName:   "arbitrum",
		ChainID:     "42161",
		Type:        ChainType_EVM,
		Environment: ChainEnvironment_MAINNET,
		// do not fill an order unless it contains 10 bps of profit
		MinFeeBps: 10,
		// settle up on arbitrum after filling 100 usdc worth of orders from arbitrum
		BatchUUSDCSettleUpThreshold: defaultBatchUUSDCSettleUpThreshold,
		// maintain a 8 bps profit margin when settling up
		MinProfitMarginBPS: 8,
		Relayer: RelayerConfig{
			MailboxAddress:         "0x979Ca5202784112f4738403dBec5D0F3B9daabB9",
			RelayCostCapUUSDC:      "1000000",
			ProfitableRelayTimeout: &defaultProfitableRelayTimeout,
		},
		EVM: &EVMConfig{
			SignerGasBalance: SignerGasBalanceConfig{
				// 0.01 eth
				WarningThresholdWei: "10000000000000000",
				// 0.001 eth
				CriticalThresholdWei: "1000000000000000",
			},
		},
		NumBlockConfirmationsBeforeFill: 1,
		QuickStartNumBlocksBack:         30000,
		GasTokenSymbol:                  "ETH",
		GasTokenDecimals:                18,
		GasTokenCoingeckoID:             "ethereum",
		HyperlaneDomain:                 "42161",
		FastTransferContractAddress:     "0x23cb6147e5600c23d1fb5543916d3d5457c9b54c",
		USDCDenom:                       "0xaf88d065e77c8cC2239327C5EDb3A432268e5831",
	}

	DefaultAvalancheConfig = ChainConfig{
		ChainName:   "avalanche",
		ChainID:     "43114",
		Type:        ChainType_EVM,
		Environment: ChainEnvironment_MAINNET,
		// do not fill an order unless it contains 10 bps of profit
		MinFeeBps: 10,
		// settle up on avalanche after filling 100 usdc worth of orders from avalanche
		BatchUUSDCSettleUpThreshold: defaultBatchUUSDCSettleUpThreshold,
		// maintain a 8 bps profit margin when settling up
		MinProfitMarginBPS: 8,
		Relayer: RelayerConfig{
			MailboxAddress:         "0xFf06aFcaABaDDd1fb08371f9ccA15D73D51FeBD6",
			RelayCostCapUUSDC:      "1000000",
			ProfitableRelayTimeout: &defaultProfitableRelayTimeout,
		},
		EVM: &EVMConfig{
			SignerGasBalance: SignerGasBalanceConfig{
				// 1 avax
				WarningThresholdWei: "1000000000000000000",
				// 0.5 avax
				CriticalThresholdWei: "500000000000000000",
			},
		},
		QuickStartNumBlocksBack:         30000,
		NumBlockConfirmationsBeforeFill: 1,
		GasTokenSymbol:                  "AVAX",
		GasTokenDecimals:                18,
		GasTokenCoingeckoID:             "avalanche-2",
		HyperlaneDomain:                 "43114",
		FastTransferContractAddress:     "0xD415B02A7E91dBAf92EAa4721F9289CFB7f4E1cF",
		USDCDenom:                       "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E",
	}

	DefaultOptimismConfig = ChainConfig{
		ChainName:   "optimism",
		ChainID:     "10",
		Type:        ChainType_EVM,
		Environment: ChainEnvironment_MAINNET,
		// do not fill an order unless it contains 10 bps of profit
		MinFeeBps: 10,
		// settle up on optimism after filling 100 usdc worth of orders from optimism
		BatchUUSDCSettleUpThreshold: defaultBatchUUSDCSettleUpThreshold,
		// maintain a 8 bps profit margin when settling up
		MinProfitMarginBPS: 8,
		Relayer: RelayerConfig{
			MailboxAddress:         "0xd4C1905BB1D26BC93DAC913e13CaCC278CdCC80D",
			RelayCostCapUUSDC:      "1000000",
			ProfitableRelayTimeout: &defaultProfitableRelayTimeout,
		},
		EVM: &EVMConfig{
			SignerGasBalance: SignerGasBalanceConfig{
				// 0.01 eth
				WarningThresholdWei: "10000000000000000",
				// 0.001 eth
				CriticalThresholdWei: "1000000000000000",
			},
		},
		NumBlockConfirmationsBeforeFill: 1,
		QuickStartNumBlocksBack:         30000,
		GasTokenSymbol:                  "ETH",
		GasTokenDecimals:                18,
		GasTokenCoingeckoID:             "ethereum",
		HyperlaneDomain:                 "10",
		FastTransferContractAddress:     "0x0f479de4fd3144642f1af88e3797b1821724f703",
		USDCDenom:                       "0x0b2c639c533813f4aa9d7837caf62653d097ff85",
	}

	DefaultBaseConfig = ChainConfig{
		ChainName:   "base",
		ChainID:     "8453",
		Type:        ChainType_EVM,
		Environment: ChainEnvironment_MAINNET,
		// do not fill an order unless it contains 10 bps of profit
		MinFeeBps: 10,
		// settle up on base after filling 100 usdc worth of orders from base
		BatchUUSDCSettleUpThreshold: defaultBatchUUSDCSettleUpThreshold,
		// maintain a 8 bps profit margin when settling up
		MinProfitMarginBPS: 8,
		Relayer: RelayerConfig{
			MailboxAddress:         "0xeA87ae93Fa0019a82A727bfd3eBd1cFCa8f64f1D",
			RelayCostCapUUSDC:      "1000000",
			ProfitableRelayTimeout: &defaultProfitableRelayTimeout,
		},
		EVM: &EVMConfig{
			SignerGasBalance: SignerGasBalanceConfig{
				// 0.01 eth
				WarningThresholdWei: "10000000000000000",
				// 0.001 eth
				CriticalThresholdWei: "1000000000000000",
			},
		},
		NumBlockConfirmationsBeforeFill: 1,
		QuickStartNumBlocksBack:         30000,
		GasTokenSymbol:                  "ETH",
		GasTokenDecimals:                18,
		GasTokenCoingeckoID:             "ethereum",
		HyperlaneDomain:                 "8453",
		FastTransferContractAddress:     "0x43d090025aaa6c8693b71952b910ac55ccb56bbb",
		USDCDenom:                       "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
	}

	polygonMinGasTipCap  = int64(30000000000)
	DefaultPolygonConfig = ChainConfig{
		ChainName:   "polygon",
		ChainID:     "137",
		Type:        ChainType_EVM,
		Environment: ChainEnvironment_MAINNET,
		// do not fill an order unless it contains 10 bps of profit
		MinFeeBps: 10,
		// settle up on base after filling 100 usdc worth of orders from base
		BatchUUSDCSettleUpThreshold: defaultBatchUUSDCSettleUpThreshold,
		// maintain a 8 bps profit margin when settling up
		MinProfitMarginBPS: 8,
		Relayer: RelayerConfig{
			MailboxAddress:         "0x5d934f4e2f797775e53561bB72aca21ba36B96BB",
			RelayCostCapUUSDC:      "1000000",
			ProfitableRelayTimeout: &defaultProfitableRelayTimeout,
		},
		EVM: &EVMConfig{
			MinGasTipCap: &polygonMinGasTipCap,
			SignerGasBalance: SignerGasBalanceConfig{
				// 1 matic
				WarningThresholdWei: "1000000000000000000",
				// 0.5 matic
				CriticalThresholdWei: "500000000000000000",
			},
		},
		NumBlockConfirmationsBeforeFill: 1,
		QuickStartNumBlocksBack:         30000,
		GasTokenSymbol:                  "MATIC",
		GasTokenDecimals:                18,
		GasTokenCoingeckoID:             "matic-network",
		HyperlaneDomain:                 "137",
		FastTransferContractAddress:     "0x3ffaf8d0d33226302e3a0ae48367cf1dd2023b1f",
		USDCDenom:                       "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359",
	}
)
