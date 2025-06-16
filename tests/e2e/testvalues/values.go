package testvalues

import (
	"cosmossdk.io/math"

	"github.com/strangelove-ventures/interchaintest/v8/chain/ethereum"
)

const (
	// InitialBalance is the amount of tokens to give to each user at the start of the test.
	InitialBalance int64 = 1_000_000_000_000

	// EnvKeyTendermintRPC Tendermint RPC URL.
	EnvKeyTendermintRPC = "TENDERMINT_RPC_URL"
	// EnvKeyEthRPC Ethereum RPC URL.
	EnvKeyEthRPC = "RPC_URL"
	// The log level for the Rust logger.
	EnvKeyRustLog = "RUST_LOG"

	// Log level for the Rust logger.
	EnvValueRustLog_Info = "info"

	// FaucetPrivateKey is the private key of the faucet account.
	// '0x' prefix is trimmed.
	FaucetPrivateKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

var (
	// StartingEthBalance is the amount of ETH to give to each user at the start of the test.
	StartingEthBalance = math.NewInt(2 * ethereum.ETHER)
)
