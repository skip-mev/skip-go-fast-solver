package ethereum

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	interchain_security_module "github.com/skip-mev/go-fast-solver/shared/contracts/hyperlane/InterchainSecurityModule"
	mailbox "github.com/skip-mev/go-fast-solver/shared/contracts/hyperlane/Mailbox"
	multisig_ism "github.com/skip-mev/go-fast-solver/shared/contracts/hyperlane/MultisigIsm"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/skip-mev/go-fast-solver/hyperlane/types"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/evmrpc"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"github.com/skip-mev/go-fast-solver/shared/signing"
	"github.com/skip-mev/go-fast-solver/shared/signing/evm"
)

type TxPriceOracle interface {
	TxFeeUUSDC(ctx context.Context, tx *ethtypes.Transaction) (*big.Int, error)
}

type HyperlaneClient struct {
	client          evmrpc.EVMChainRPC
	chainID         string
	hyperlaneDomain string
	mailboxAddress  common.Address
	keystore        keys.KeyStore

	ismAddress     *common.Address
	ismAddressLock sync.RWMutex

	txPriceOracle TxPriceOracle
}

func NewHyperlaneClient(
	ctx context.Context,
	hyperlaneDomain string,
	manager evmrpc.EVMRPCClientManager,
	keystore keys.KeyStore,
	priceOracle TxPriceOracle,
) (*HyperlaneClient, error) {
	chainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(hyperlaneDomain)
	if err != nil {
		return nil, fmt.Errorf("getting chainID from hyperlane domain %s: %w", hyperlaneDomain, err)
	}

	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return nil, fmt.Errorf("getting config for chain %s: %w", chainID, err)
	}

	client, err := manager.GetClient(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("getting rpc client for chainID %s: %w", chainID, err)
	}

	return &HyperlaneClient{
		client:          client,
		chainID:         chainID,
		hyperlaneDomain: hyperlaneDomain,
		mailboxAddress:  common.HexToAddress(chainConfig.Relayer.MailboxAddress),
		keystore:        keystore,
		txPriceOracle:   priceOracle,
	}, nil
}

func (c *HyperlaneClient) GetHyperlaneDispatch(ctx context.Context, domain, originChainID, initiateTxHash string) (*types.MailboxDispatchEvent, *types.MailboxMerkleHookPostDispatchEvent, error) {
	panic("not implemented")
}

func (c *HyperlaneClient) HasBeenDelivered(ctx context.Context, domain string, messageID string) (bool, error) {
	if domain != c.hyperlaneDomain {
		return false, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}

	destinationMailbox, err := mailbox.NewMailbox(c.mailboxAddress, c.client.Client())
	if err != nil {
		return false, fmt.Errorf("creating mailbox contract caller for address %s: %w", c.mailboxAddress.String(), err)
	}
	destinationMailboxSession := mailbox.MailboxSession{
		Contract: destinationMailbox,
		CallOpts: bind.CallOpts{Context: ctx},
	}

	messageIDBytes, err := hex.DecodeString(messageID)
	if err != nil {
		return false, fmt.Errorf("decoding messageID %s: %w", messageID, err)

	}

	var messageIDBytes32 [32]byte
	copy(messageIDBytes32[:], messageIDBytes)
	delivered, err := destinationMailboxSession.Delivered(messageIDBytes32)
	if err != nil {
		return false, fmt.Errorf("querying destination mailbox at %s to see if message %s was delivered: %w", c.mailboxAddress.String(), messageID, err)
	}

	return delivered, nil
}

func (c *HyperlaneClient) ISMType(ctx context.Context, domain string, recipient string) (uint8, error) {
	if domain != c.hyperlaneDomain {
		return 0, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}

	ismAddress, err := c.getISMAddress(ctx, recipient)
	if err != nil {
		return 0, fmt.Errorf("getting ism address for recipeint %s on domain %s: %w", recipient, domain, err)
	}

	ism, err := interchain_security_module.NewInterchainSecurityModuleCaller(ismAddress, c.client.Client())
	if err != nil {
		return 0, fmt.Errorf("creating ism contract caller for address %s: %w", ismAddress.String(), err)
	}
	ismSession := interchain_security_module.InterchainSecurityModuleCallerSession{
		Contract: ism,
		CallOpts: bind.CallOpts{Context: ctx},
	}

	ismType, err := ismSession.ModuleType()
	if err != nil {
		return 0, fmt.Errorf("getting ism type for ism address %s: %w", ismAddress.String(), err)
	}

	return ismType, nil
}

const (
	ismTypeMessageIDMultisig = 5
)

func (c *HyperlaneClient) ValidatorsAndThreshold(
	ctx context.Context,
	domain string,
	recipient string,
	message string,
) ([]common.Address, uint8, error) {
	if domain != c.hyperlaneDomain {
		return nil, 0, fmt.Errorf("expected domain %s but got %s", c.hyperlaneDomain, domain)
	}

	ismAddress, err := c.getISMAddress(ctx, recipient)
	if err != nil {
		return nil, 0, fmt.Errorf("getting ism address for recipeint %s on domain %s: %w", recipient, domain, err)
	}

	ismType, err := c.ISMType(ctx, domain, recipient)
	if err != nil {
		return nil, 0, fmt.Errorf("getting ism type for recipient %s on domain %s: %w", recipient, domain, err)
	}

	switch ismType {
	case ismTypeMessageIDMultisig:
		multisigISM, err := multisig_ism.NewMultisigIsmCaller(ismAddress, c.client.Client())
		if err != nil {
			return nil, 0, fmt.Errorf("creating multisign ism contract caller for address %s: %w", ismAddress.String(), err)
		}
		multisigISMSession := multisig_ism.MultisigIsmCallerSession{Contract: multisigISM, CallOpts: bind.CallOpts{Context: ctx}}

		validatorsAndThreshold, err := multisigISMSession.ValidatorsAndThreshold([]byte(message))
		if err != nil {
			return nil, 0, fmt.Errorf("fetching validators and threshold from multisig ism at address %s: %w", ismAddress.String(), err)
		}

		return validatorsAndThreshold.Validators, validatorsAndThreshold.Threshold, nil
	default:
		return nil, 0, fmt.Errorf("ism type %d not supported", ismType)
	}
}

func (c *HyperlaneClient) getISMAddress(ctx context.Context, recipient string) (common.Address, error) {
	c.ismAddressLock.RLock()
	if c.ismAddress != nil {
		defer c.ismAddressLock.RUnlock()
		return *c.ismAddress, nil
	}
	c.ismAddressLock.RUnlock()

	destinationMailbox, err := mailbox.NewMailbox(c.mailboxAddress, c.client.Client())
	if err != nil {
		return common.Address{}, fmt.Errorf("creating mailbox contract caller for address %s: %w", c.mailboxAddress.String(), err)
	}
	destinationMailboxSession := mailbox.MailboxSession{
		Contract: destinationMailbox,
		CallOpts: bind.CallOpts{Context: ctx},
	}
	ismAddress, err := destinationMailboxSession.RecipientIsm(common.HexToAddress(recipient))
	if err != nil {
		return common.Address{}, fmt.Errorf("getting ism address for recipient %s: %w", recipient, err)
	}

	c.ismAddressLock.Lock()
	defer c.ismAddressLock.Unlock()
	c.ismAddress = &ismAddress

	return ismAddress, nil
}

func (c *HyperlaneClient) Process(ctx context.Context, domain string, message []byte, metadata []byte) ([]byte, error) {
	destinationMailbox, err := mailbox.NewMailbox(c.mailboxAddress, c.client.Client())
	if err != nil {
		return nil, fmt.Errorf("creating mailbox contract caller for address %s: %w", c.mailboxAddress.String(), err)
	}

	signer, err := c.signer(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("getting signer: %w", err)
	}

	addr, err := c.address(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("getting address: %w", err)
	}

	processTx, err := destinationMailbox.Process(&bind.TransactOpts{
		From:    addr,
		Context: ctx,
		Signer:  signer,
	}, metadata, message)
	if err != nil {
		return nil, fmt.Errorf("processing message on destination mailbox: %w", err)
	}

	return processTx.Hash().Bytes(), nil
}

func (c *HyperlaneClient) QuoteProcessUUSDC(ctx context.Context, domain string, message []byte, metadata []byte) (*big.Int, error) {
	destinationMailbox, err := mailbox.NewMailbox(c.mailboxAddress, c.client.Client())
	if err != nil {
		return nil, fmt.Errorf("creating mailbox contract caller for address %s: %w", c.mailboxAddress.String(), err)
	}

	signer, err := c.signer(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("getting signer: %w", err)
	}

	addr, err := c.address(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("getting address: %w", err)
	}

	unsentProcessTx, err := destinationMailbox.Process(&bind.TransactOpts{
		From:    addr,
		Context: ctx,
		Signer:  signer,
		// NoSend dry runs the tx, this will populate the tx with all necessary
		// gas estimates from the node needed to get the tx fee in uusdc
		NoSend: true,
	}, metadata, message)
	if err != nil {
		return nil, fmt.Errorf("simulating process tx: %w", err)
	}

	txFeeUUSDC, err := c.txPriceOracle.TxFeeUUSDC(ctx, unsentProcessTx)
	if err != nil {
		return nil, fmt.Errorf("getting tx fee in uusdc from gas oracle: %w", err)
	}

	return txFeeUUSDC, nil
}

func (c *HyperlaneClient) MerkleTreeLeafCount(ctx context.Context, domain string) (uint64, error) {
	panic("not implemented")
}

func (c *HyperlaneClient) ValidatorStorageLocations(
	ctx context.Context,
	domain string,
	validators []common.Address,
) ([]*types.ValidatorStorageLocation, error) {
	panic("not implemented")
}

func (c *HyperlaneClient) IsContract(ctx context.Context, domain, address string) (bool, error) {
	contractCode, err := c.client.CodeAt(ctx, address, nil)
	if err != nil {
		return false, err
	}

	return len(contractCode) > 0, nil
}

func (c *HyperlaneClient) address(ctx context.Context, domain string) (common.Address, error) {
	destinationChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(domain)
	if err != nil {
		return common.Address{}, fmt.Errorf("getting chainID for hyperlane domain %s: %w", domain, err)
	}

	destinationChainConfig, err := config.GetConfigReader(ctx).GetChainConfig(destinationChainID)
	if err != nil {
		return common.Address{}, fmt.Errorf("getting chain config for chainID %s: %w", destinationChainID, err)
	}

	return common.HexToAddress(destinationChainConfig.SolverAddress), nil
}

func (c *HyperlaneClient) signer(ctx context.Context, domain string) (bind.SignerFn, error) {
	destinationChainID, err := config.GetConfigReader(ctx).GetChainIDByHyperlaneDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("getting chainID for hyperlane domain %s: %w", domain, err)
	}

	privateKeyStr, ok := c.keystore.GetPrivateKey(destinationChainID)
	if !ok {
		return nil, fmt.Errorf("relayer private key not found for chainID %s", destinationChainID)
	}
	if privateKeyStr[:2] == "0x" {
		privateKeyStr = privateKeyStr[2:]
	}

	privateKey, err := crypto.HexToECDSA(string(privateKeyStr))
	if err != nil {
		return nil, fmt.Errorf("creating private key from string: %w", err)
	}

	return evm.EthereumSignerToBindSignerFn(
		signing.NewLocalEthereumSigner(privateKey),
		destinationChainID,
	), nil
}
