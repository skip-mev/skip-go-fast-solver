package e2e

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/skip-mev/fast-transfer-solver/shared/contracts/fast_transfer_gateway"
	"github.com/strangelove-ventures/interchaintest/v8/chain/ethereum"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/skip-mev/fast-transfer-solver/e2e/testvalues"
	"github.com/skip-mev/fast-transfer-solver/e2e/types/erc20"
	"github.com/skip-mev/fast-transfer-solver/e2e/types/hyperlane"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/suite"
)

// SolverTestSuite is a suite of tests that wraps TestSuite
// and can provide additional functionality
type SolverTestSuite struct {
	TestSuite

	// Whether to generate fixtures for the solidity tests
	generateFixtures bool

	// The private key of a test account
	key *ecdsa.PrivateKey

	// The private key of the faucet account of interchaintest
	faucet   *ecdsa.PrivateKey
	deployer ibc.Wallet

	simdClientID string
	ethClientID  string

	evmContractsAddresses DeployedEVMContracts

	usdcERC20Contract       *erc20.Contract
	ftgContract             *fast_transfer_gateway.FastTransferGateway
	fastTransferGatewayAddr ethcommon.Address

	cleanup func()

	mockMailbox           *hyperlane.TestMailbox
	mockIsm               *hyperlane.TestIsm
	mockMerkleHook        *hyperlane.MerkleTreeHook
	mockValidatorAnnounce *hyperlane.ValidatorAnnounce
}

func (s *SolverTestSuite) SetupSuite(ctx context.Context) {
	s.TestSuite.SetupSuite(ctx)

	eth, simd := s.ChainA, s.ChainB
	s.Require().NotNil(eth, "Ethereum chain (ChainA) is nil")
	s.Require().NotNil(simd, "Cosmos chain (ChainB) is nil")

	s.Require().True(s.Run("Set up EVM environment", func() {
		err := os.Chdir("../..")
		s.Require().NoError(err)

		s.key, err = crypto.GenerateKey()
		s.Require().NoError(err)
		testKeyAddress := crypto.PubkeyToAddress(s.key.PublicKey).Hex()

		s.deployer, err = eth.BuildWallet(ctx, "deployer", "")
		s.Require().NoError(err)
		s.Require().NotNil(s.deployer, "Deployer wallet is nil")

		s.faucet, err = crypto.HexToECDSA(testvalues.FaucetPrivateKey)
		s.Require().NoError(err)

		os.Setenv(testvalues.EnvKeyRustLog, testvalues.EnvValueRustLog_Info)
		os.Setenv(testvalues.EnvKeyEthRPC, eth.GetHostRPCAddress())
		os.Setenv(testvalues.EnvKeyTendermintRPC, simd.GetHostRPCAddress())

		s.Require().NoError(eth.SendFunds(ctx, "faucet", ibc.WalletAmount{
			Amount:  testvalues.StartingEthBalance,
			Address: testKeyAddress,
		}))

		s.Require().NoError(eth.SendFunds(ctx, "faucet", ibc.WalletAmount{
			Amount:  testvalues.StartingEthBalance,
			Address: s.deployer.FormattedAddress(),
		}))
	}))

	s.Require().True(s.Run("Deploy required EVM contracts", func() {
		var (
			stdout []byte
			stderr []byte
			err    error
		)

		s.T().Logf("Deploying EVM contracts with sender: %s", s.deployer.FormattedAddress())

		// First deploy base contracts
		stdout, stderr, err = eth.ForgeScript(ctx, s.deployer.KeyName(), ethereum.ForgeScriptOpts{
			ContractRootDir:  "./tests/e2e",
			SolidityContract: "contracts/solidity/E2EContractsDeploy.s.sol:E2EContractsDeploy",
			RawOptions: []string{
				"--json",
				"--force",
				"-vvvv",
				"--sender", s.deployer.FormattedAddress(),
				"--lib-paths", "lib",
			},
		})

		s.Require().NoError(err, fmt.Sprintf("error deploying contracts: \nstderr: %s\nstdout: %s\nerr: %s", stderr, stdout, err))

		s.T().Logf("Deploying EVM hyperlane contracts with sender: %s", s.deployer.FormattedAddress())

		// deploy hyperlane contracts with a different set of remappings
		hyperlaneDeployOutput, stderr, err := eth.ForgeScript(ctx, s.deployer.KeyName(), ethereum.ForgeScriptOpts{
			ContractRootDir:  "./tests/e2e",
			SolidityContract: "contracts/solidity/HyperlaneTestDeploy.s.sol:HyperlaneTestDeploy",
			RawOptions: []string{
				"--json",
				"--force",
				"-vvvv",
				"--sender", s.deployer.FormattedAddress(),
				"--lib-paths", "lib",
				"--remappings", "@openzeppelin/contracts-upgradeable=lib/hyperlane-monorepo/node_modules/@openzeppelin/contracts-upgradeable",
				"--remappings", "@openzeppelin=lib/hyperlane-monorepo/node_modules/@openzeppelin",
				"--remappings", "@eth-optimism=lib/hyperlane-monorepo/node_modules/@eth-optimism",
				"--remappings", "@hyperlane-xyz/=lib/hyperlane-monorepo/solidity/contracts/",
				"--remappings", "forge-std/=lib/forge-std/src/",
				"--remappings", "ds-test/=lib/openzeppelin-contracts/lib/forge-std/lib/ds-test/src/",
				"--remappings", "hyperlane-monorepo/=lib/hyperlane-monorepo/",
			},
		})
		s.Require().NoError(err, fmt.Sprintf("error deploying hyperlane contracts: \nstderr: %s\nstdout: %s\nerr: %s", stderr, stdout, err))

		s.evmContractsAddresses = s.GetEthContractsFromDeployOutput(string(stdout), string(hyperlaneDeployOutput))
		ethClient, err := ethclient.Dial(eth.GetHostRPCAddress())
		s.Require().NoError(err)
		s.Require().NotNil(ethClient)

		s.usdcERC20Contract, err = erc20.NewContract(ethcommon.HexToAddress(s.evmContractsAddresses.USDCERC20), ethClient)
		s.Require().NoError(err)

		s.ftgContract, err = fast_transfer_gateway.NewFastTransferGateway(ethcommon.HexToAddress(s.evmContractsAddresses.FastTransferGateway), ethClient)
		s.Require().NoError(err)

		s.mockMailbox, err = hyperlane.NewTestMailbox(ethcommon.HexToAddress(s.evmContractsAddresses.Mailbox), ethClient)
		s.Require().NoError(err)
		s.mockIsm, err = hyperlane.NewTestIsm(ethcommon.HexToAddress(s.evmContractsAddresses.Ism), ethClient)
		s.Require().NoError(err)
		s.mockMerkleHook, err = hyperlane.NewMerkleTreeHook(ethcommon.HexToAddress(s.evmContractsAddresses.MerkleHook), ethClient)
		s.Require().NoError(err)
		s.mockValidatorAnnounce, err = hyperlane.NewValidatorAnnounce(ethcommon.HexToAddress(s.evmContractsAddresses.ValidatorAnnounce), ethClient)
		s.Require().NoError(err)
	}))

	s.Require().True(s.Run("Fund evm user address with USDC", func() {
		tx, err := s.usdcERC20Contract.Transfer(s.GetTransactOpts(s.faucet), crypto.PubkeyToAddress(s.key.PublicKey), big.NewInt(testvalues.InitialBalance))
		s.Require().NoError(err)

		_ = s.GetTxReciept(ctx, eth, tx.Hash()) // wait for the tx to be mined
	}))
}

func TestWithSolverTestSuite(t *testing.T) {
	s := new(SolverTestSuite)
	suite.Run(t, s)
	if s.cleanup != nil {
		s.cleanup()
	}
}

func (s *SolverTestSuite) TestDeploy() {
	ctx := context.Background()

	s.SetupSuite(ctx)

	s.Require().True(s.Run("Verify deployment", func() {
		s.Require().True(s.Run("Verify USDC ERC20 Genesis", func() {
			userBalance, err := s.usdcERC20Contract.BalanceOf(nil, crypto.PubkeyToAddress(s.key.PublicKey))
			s.Require().NoError(err)
			s.Require().Equal(testvalues.InitialBalance, userBalance.Int64())
		}))
	}))
}
