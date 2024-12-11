package e2e

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/skip-mev/fast-transfer-solver/e2e/testvalues"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/chain/ethereum"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
)

type ForgeScriptReturnValues struct {
	InternalType string `json:"internal_type"`
	Value        string `json:"value"`
}

type ForgeDeployOutput struct {
	Returns map[string]ForgeScriptReturnValues `json:"returns"`
}

type DeployedContracts struct {
	Erc20               string
	FastTransferGateway string
	Mailbox             string
	Ism                 string
	MerkleHook          string
	ValidatorAnnounce   string
}

type HyperlaneAddresses struct {
	mailbox           ethcommon.Address
	ism               ethcommon.Address
	merkleHook        ethcommon.Address
	validatorAnnounce ethcommon.Address
}

func parseHyperlaneAddresses(output string) HyperlaneAddresses {
	// The output format is: "mailbox:0x...,ism:0x...,merkleHook:0x...,validatorAnnounce:0x..."
	parts := strings.Split(output, ",")
	addresses := HyperlaneAddresses{}

	for _, part := range parts {
		keyValue := strings.Split(part, ":")
		if len(keyValue) != 2 {
			continue
		}

		address := ethcommon.HexToAddress(keyValue[1])
		switch keyValue[0] {
		case "mailbox":
			addresses.mailbox = address
		case "ism":
			addresses.ism = address
		case "merkleHook":
			addresses.merkleHook = address
		case "validatorAnnounce":
			addresses.validatorAnnounce = address
		}
	}

	return addresses
}

// FundAddressChainB sends funds to the given address on Chain B.
// The amount sent is 1,000,000,000 of the chain's denom.
func (s *TestSuite) FundAddressChainB(ctx context.Context, address string) {
	s.fundAddress(ctx, s.ChainB, s.UserB.KeyName(), address)
}

// BroadcastMessages broadcasts the provided messages to the given chain and signs them on behalf of the provided user.
// Once the broadcast response is returned, we wait for two blocks to be created on chain.
func (s *TestSuite) BroadcastMessages(ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, gas uint64, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	sdk.GetConfig().SetBech32PrefixForAccount(chain.Config().Bech32Prefix, chain.Config().Bech32Prefix+sdk.PrefixPublic)
	sdk.GetConfig().SetBech32PrefixForValidator(
		chain.Config().Bech32Prefix+sdk.PrefixValidator+sdk.PrefixOperator,
		chain.Config().Bech32Prefix+sdk.PrefixValidator+sdk.PrefixOperator+sdk.PrefixPublic,
	)

	broadcaster := cosmos.NewBroadcaster(s.T(), chain)

	broadcaster.ConfigureClientContextOptions(func(clientContext client.Context) client.Context {
		return clientContext.
			WithCodec(chain.Config().EncodingConfig.Codec).
			WithChainID(chain.Config().ChainID).
			WithTxConfig(chain.Config().EncodingConfig.TxConfig)
	})

	broadcaster.ConfigureFactoryOptions(func(factory tx.Factory) tx.Factory {
		return factory.WithGas(gas)
	})

	resp, err := cosmos.BroadcastTx(ctx, broadcaster, user, msgs...)
	if err != nil {
		return nil, err
	}

	// wait for 2 blocks for the transaction to be included
	s.Require().NoError(testutil.WaitForBlocks(ctx, 2, chain))

	return &resp, nil
}

// fundAddress sends funds to the given address on the given chain
func (s *TestSuite) fundAddress(ctx context.Context, chain *cosmos.CosmosChain, keyName, address string) {
	err := chain.SendFunds(ctx, keyName, ibc.WalletAmount{
		Address: address,
		Denom:   chain.Config().Denom,
		Amount:  sdkmath.NewInt(1_000_000_000),
	})
	s.Require().NoError(err)

	// wait for 2 blocks for the funds to be received
	err = testutil.WaitForBlocks(ctx, 2, chain)
	s.Require().NoError(err)
}

func (s *TestSuite) GetEthContractsFromDeployOutput(stdout string) DeployedContracts {
	// Extract the JSON part using regex that matches forge's JSON output format
	re := regexp.MustCompile(`"value":"({.*?})"`)
	matches := re.FindStringSubmatch(stdout)
	if len(matches) != 2 {
		s.T().Fatalf("Failed to find JSON in forge output")
	}

	jsonStr := matches[1]
	// Unescape the JSON string
	jsonStr = strings.ReplaceAll(jsonStr, `\`, ``)

	var contracts DeployedContracts
	err := json.Unmarshal([]byte(jsonStr), &contracts)
	s.Require().NoError(err)

	// Verify all required fields are present
	s.Require().NotEmpty(contracts.Erc20)
	s.Require().NotEmpty(contracts.FastTransferGateway)
	s.Require().NotEmpty(contracts.Mailbox)
	s.Require().NotEmpty(contracts.Ism)
	s.Require().NotEmpty(contracts.MerkleHook)
	s.Require().NotEmpty(contracts.ValidatorAnnounce)

	return contracts
}

// GetRelayerUsers returns two ibc.Wallet instances which can be used for the relayer users
// on the two chains.
func (s *TestSuite) GetRelayerUsers(ctx context.Context) (ibc.Wallet, ibc.Wallet) {
	eth, simd := s.ChainA, s.ChainB

	ethUsers := interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), testvalues.StartingEthBalance, eth)

	cosmosUserFunds := sdkmath.NewInt(testvalues.InitialBalance)
	cosmosUsers := interchaintest.GetAndFundTestUsers(s.T(), ctx, s.T().Name(), cosmosUserFunds, simd)

	return ethUsers[0], cosmosUsers[0]
}

// GetEvmEvent parses the logs in the given receipt and returns the first event that can be parsed
func GetEvmEvent[T any](receipt *ethtypes.Receipt, parseFn func(log ethtypes.Log) (*T, error)) (event *T, err error) {
	for _, l := range receipt.Logs {
		event, err = parseFn(*l)
		if err == nil && event != nil {
			break
		}
	}

	if event == nil {
		err = fmt.Errorf("event not found")
	}

	return
}

func (s *TestSuite) GetTxReciept(ctx context.Context, chain *ethereum.EthereumChain, hash ethcommon.Hash) *ethtypes.Receipt {
	ethClient, err := ethclient.Dial(chain.GetHostRPCAddress())
	s.Require().NoError(err)

	var receipt *ethtypes.Receipt
	err = testutil.WaitForCondition(time.Second*10, time.Second, func() (bool, error) {
		receipt, err = ethClient.TransactionReceipt(ctx, hash)
		if err != nil {
			return false, nil
		}

		return receipt != nil, nil
	})
	s.Require().NoError(err)
	return receipt
}

func (s *TestSuite) GetTransactOpts(key *ecdsa.PrivateKey) *bind.TransactOpts {
	chainIDStr, err := strconv.ParseInt(s.ChainA.Config().ChainID, 10, 64)
	s.Require().NoError(err)
	chainID := big.NewInt(chainIDStr)

	txOpts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	s.Require().NoError(err)

	return txOpts
}

func IsLowercase(s string) bool {
	for _, r := range s {
		if !unicode.IsLower(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}
