package e2e

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/skip-mev/fast-transfer-solver/shared/contracts/fast_transfer_gateway"
	"github.com/strangelove-ventures/interchaintest/v8/chain/ethereum"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/skip-mev/fast-transfer-solver/e2e/testvalues"
	"github.com/skip-mev/fast-transfer-solver/e2e/types/erc20"
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

	contractAddresses DeployedContracts

	erc20Contract           *erc20.Contract
	ftgContract             *fast_transfer_gateway.FastTransferGateway
	fastTransferGatewayAddr ethcommon.Address

	cleanup func()
}

func (s *SolverTestSuite) SetupSuite(ctx context.Context) {
	s.TestSuite.SetupSuite(ctx)

	eth, simd := s.ChainA, s.ChainB
	s.Require().NotNil(eth, "Ethereum chain (ChainA) is nil")
	s.Require().NotNil(simd, "Cosmos chain (ChainB) is nil")

	s.Require().True(s.Run("Set up environment", func() {
		err := os.Chdir("../..")
		s.Require().NoError(err)

		s.key, err = crypto.GenerateKey()
		s.Require().NoError(err)
		testKeyAddress := crypto.PubkeyToAddress(s.key.PublicKey).Hex()

		s.deployer, err = eth.BuildWallet(ctx, "deployer", "")
		s.Require().NoError(err)
		s.Require().NotNil(s.deployer, "Deployer wallet is nil")

		// get faucet private key from string
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

	s.Require().True(s.Run("Deploy contracts", func() {
		var (
			stdout []byte
			stderr []byte
			err    error
		)

		s.T().Logf("Deploying contracts with sender: %s", s.deployer.FormattedAddress())

		stdout, stderr, err = eth.ForgeScript(ctx, s.deployer.KeyName(), ethereum.ForgeScriptOpts{
			ContractRootDir: ".", // Point to project root as os.ChDir(../..) is called as first step
			// in "Set up environment" suite
			SolidityContract: "./tests/e2e/scripts/MockE2ETestDeploy.s.sol",
			RawOptions: []string{
				"--json",
				"--force",                                 // sometimes forge cache returns a nothing to compile error
				"-vvvv",                                   // Add verbose logging
				"--sender", s.deployer.FormattedAddress(), // This, combined with the keyname, makes msg.sender the deployer
			},
		})

		s.Require().NoError(err, fmt.Sprintf("error deploying contracts: \nstderr: %s\nstdout: %s\nerr: %s", stderr, stdout, err))

		ethClient, err := ethclient.Dial(eth.GetHostRPCAddress())
		s.Require().NoError(err)
		s.Require().NotNil(ethClient)

		s.contractAddresses = s.GetEthContractsFromDeployOutput(string(stdout))
		s.erc20Contract, err = erc20.NewContract(ethcommon.HexToAddress(s.contractAddresses.Erc20), ethClient)
		s.Require().NoError(err)

		balance, err := ethClient.BalanceAt(ctx, ethcommon.HexToAddress(s.deployer.FormattedAddress()), nil)
		if err != nil {
			s.T().Logf("Failed to get balance: %v", err)
			return
		}
		s.T().Logf("Deployer balance: %s", balance.String())

		s.ftgContract, err = fast_transfer_gateway.NewFastTransferGateway(ethcommon.HexToAddress(s.contractAddresses.FastTransferGateway), ethClient)
		s.Require().NoError(err)
	}))

	s.Require().True(s.Run("Fund address with ERC20", func() {
		tx, err := s.erc20Contract.Transfer(s.GetTransactOpts(s.faucet), crypto.PubkeyToAddress(s.key.PublicKey), big.NewInt(testvalues.InitialBalance))
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
		// Verify that the contracts have been deployed
		s.Require().True(s.Run("Verify ERC20 Genesis", func() {
			userBalance, err := s.erc20Contract.BalanceOf(nil, crypto.PubkeyToAddress(s.key.PublicKey))
			s.Require().NoError(err)
			s.Require().Equal(testvalues.InitialBalance, userBalance.Int64())
		}))
	}))
}
