package chainconfig

import (
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/ethereum"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
)

var DefaultChainSpecs = []*interchaintest.ChainSpec{
	// -- Ethereum --
	{ChainConfig: ethereum.DefaultEthereumAnvilChainConfig("ethereum")},
	// -- Osmosis --
	{
		ChainConfig: ibc.ChainConfig{
			Type:    "cosmos",
			Name:    "osmosis",
			ChainID: "osmosis1",
			Images: []ibc.DockerImage{
				{
					Repository: "osmolabs/osmosis",
					Version:    "28.0.0",
					UidGid:     "1025:1025",
				},
			},
			Bin:            "osmosisd",
			Bech32Prefix:   "osmo",
			Denom:          "stake",
			GasPrices:      "0.0025stake",
			GasAdjustment:  1.3,
			EncodingConfig: CosmosEncodingConfig(),
			ModifyGenesis:  defaultModifyGenesis(),
			TrustingPeriod: "508h",
			NoHostMount:    false,
			AdditionalStartArgs: []string{
				"in-place-testnet",
				"localosmosis",
				"osmo12smx2wdlyttvyzvzg54y2vnqwq2qjateuf7thj",
				"trigger-testnet-upgrade",
			},
		},
	},
}
