package hyperlane

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	mailbox "github.com/skip-mev/fast-transfer-solver/shared/contracts/hyperlane/Mailbox"
	merkle "github.com/skip-mev/fast-transfer-solver/shared/contracts/hyperlane/MerkleTreeHook"
	ism "github.com/skip-mev/fast-transfer-solver/shared/contracts/hyperlane/MultisigIsm"
	validator "github.com/skip-mev/fast-transfer-solver/shared/contracts/hyperlane/ValidatorAnnounce"
)

type TestMailbox struct {
	*mailbox.Mailbox
}

type TestIsm struct {
	*ism.MultisigIsm
}

type MerkleTreeHook struct {
	*merkle.MerkleTreeHook
}

type ValidatorAnnounce struct {
	*validator.ValidatorAnnounce
}

func NewTestMailbox(address common.Address, client *ethclient.Client) (*TestMailbox, error) {
	contract, err := mailbox.NewMailbox(address, client)
	return &TestMailbox{contract}, err
}

func NewTestIsm(address common.Address, client *ethclient.Client) (*TestIsm, error) {
	contract, err := ism.NewMultisigIsm(address, client)
	return &TestIsm{contract}, err
}

func NewMerkleTreeHook(address common.Address, client *ethclient.Client) (*MerkleTreeHook, error) {
	contract, err := merkle.NewMerkleTreeHook(address, client)
	return &MerkleTreeHook{contract}, err
}

func NewValidatorAnnounce(address common.Address, client *ethclient.Client) (*ValidatorAnnounce, error) {
	contract, err := validator.NewValidatorAnnounce(address, client)
	return &ValidatorAnnounce{contract}, err
}
