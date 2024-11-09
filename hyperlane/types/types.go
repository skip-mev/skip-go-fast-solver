package types

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

type ValidatorStorageLocation struct {
	Validator       string
	StorageLocation string
}

type SignedCheckpoint struct {
	Value               CheckpointWithMessageID `json:"value"`
	Signature           Signature               `json:"signature"`
	SerializedSignature string                  `json:"serialized_signature"`
}

type CheckpointWithMessageID struct {
	Checkpoint Checkpoint `json:"checkpoint"`
	MessageID  string     `json:"message_id"`
}

func (c SignedCheckpoint) Digest() ([]byte, error) {
	domainHash, err := c.Value.Checkpoint.DomainHash()
	if err != nil {
		return nil, fmt.Errorf("computing domain hash of checkpoint: %w", err)
	}
	root, err := hex.DecodeString(strings.TrimPrefix(c.Value.Checkpoint.Root, "0x"))
	if err != nil {
		return nil, fmt.Errorf("hex decoding checkpoint root: %w", err)
	}
	messageID, err := hex.DecodeString(strings.TrimPrefix(c.Value.MessageID, "0x"))
	if err != nil {
		return nil, fmt.Errorf("hex decoding checkpoint messaggeID: %w", err)
	}
	var buf bytes.Buffer
	if err = binary.Write(&buf, binary.BigEndian, c.Value.Checkpoint.Index); err != nil {
		return nil, fmt.Errorf("writing checkpoint index to byte buffer: %w", err)
	}
	return crypto.Keccak256Hash(domainHash, root, buf.Bytes(), messageID).Bytes(), nil
}

type Checkpoint struct {
	MerkleTreeHookAddress string `json:"merkle_tree_hook_address"`
	MailboxDomain         uint32 `json:"mailbox_domain"`
	Root                  string `json:"root"`
	Index                 uint32 `json:"index"`
}

func (c Checkpoint) DomainHash() ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.BigEndian, c.MailboxDomain); err != nil {
		return nil, fmt.Errorf("writing mailbox domain to byte buffer: %w", err)
	}
	merkle, err := hex.DecodeString(strings.TrimPrefix(c.MerkleTreeHookAddress, "0x"))
	if err != nil {
		return nil, fmt.Errorf("hex decoding merkle tree hook address: %w", err)
	}
	return crypto.Keccak256Hash(buf.Bytes(), merkle, []byte("HYPERLANE")).Bytes(), nil
}

type Signature struct {
	R string `json:"R"`
	S string `json:"S"`
	V byte   `json:"V"`
}

func (signature Signature) Bytes() ([]byte, error) {
	var buf []byte
	sig := bytes.NewBuffer(buf)

	rsBytes, err := signature.RSBytes()
	if err != nil {
		return nil, fmt.Errorf("getting R S bytes of signature: %w", err)
	}
	if _, err = sig.Write(rsBytes); err != nil {
		return nil, fmt.Errorf("writing signature.R to sig buf: %w", err)
	}

	if err := binary.Write(sig, binary.BigEndian, signature.V); err != nil {
		return nil, fmt.Errorf("writing signature.V to sig buf: %w", err)
	}
	return sig.Bytes(), nil
}

func (signature Signature) RSBytes() ([]byte, error) {
	var buf []byte
	sig := bytes.NewBuffer(buf)

	r, err := hex.DecodeString(strings.TrimPrefix(signature.R, "0x"))
	if err != nil {
		return nil, fmt.Errorf("hex decoding signature R value: %w", err)
	}
	if _, err = sig.Write(r); err != nil {
		return nil, fmt.Errorf("writing signature.R to sig buf: %w", err)
	}

	s, err := hex.DecodeString(strings.TrimPrefix(signature.S, "0x"))
	if err != nil {
		return nil, fmt.Errorf("hex decoding signature S value: %w", err)
	}
	if _, err = sig.Write(s); err != nil {
		return nil, fmt.Errorf("writing signature.S to sig buf: %w", err)
	}
	return sig.Bytes(), nil
}

func (signature Signature) RecoverPubKey(digest []byte) ([]byte, error) {
	// Ensure the signature length is 65 (r, s, and v)
	signatureBytes, err := signature.Bytes()
	if err != nil {
		return nil, err
	}
	if len(signatureBytes) != 65 {
		return nil, fmt.Errorf("invalid signature length: %d", len(signatureBytes))
	}

	// v should be either 27 or 28, but it needs to be in the form of 0 or 1 for `secp256k1.RecoverPubkey`
	v := signatureBytes[64]
	if v < 27 {
		return nil, fmt.Errorf("invalid recovery id: %d", v)
	}
	signatureBytes[64] = v - 27

	// Recover the public key using secp256k1's RecoverPubkey
	pubKeyBytes, err := secp256k1.RecoverPubkey(digest, signatureBytes)
	if err != nil {
		return nil, err
	}

	return pubKeyBytes, nil
}

type MultiSigSignedCheckpoint struct {
	Checkpoint CheckpointWithMessageID
	Signatures []Signature
}

const (
	MERKLE_TREE_ADDRESS_LEN    = 32
	SIGNED_CHECKPOINT_ROOT_LEN = 32
	VALIDATOR_SIGNATURE_LENGTH = 65
)

func (c MultiSigSignedCheckpoint) ToMetadata() ([]byte, error) {
	/**
	 * Format of metadata we need to construct:
	 * [   0:  32] Origin merkle tree address
	 * [  32:  64] Signed checkpoint root
	 * [  64:  68] Signed checkpoint index
	 * [  68:????] Validator signatures (length := threshold * 65)
	 */
	var buf []byte
	metadata := bytes.NewBuffer(buf)

	hook, err := hex.DecodeString(strings.TrimPrefix(c.Checkpoint.Checkpoint.MerkleTreeHookAddress, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decoding hex merkle tree checkpoint address: %w", err)
	}
	n, err := metadata.Write(hook)
	if err != nil {
		return nil, fmt.Errorf("writing merkle tree contract addr %s to message metadata: %w", c.Checkpoint.Checkpoint.MerkleTreeHookAddress, err)
	}
	if n != MERKLE_TREE_ADDRESS_LEN {
		return nil, fmt.Errorf("invalid length for merkle tree contract addr, expected %d, got %d", MERKLE_TREE_ADDRESS_LEN, n)
	}

	root, err := hex.DecodeString(strings.TrimPrefix(c.Checkpoint.Checkpoint.Root, "0x"))
	if err != nil {
		return nil, fmt.Errorf("decoding hex checkpoint root: %w", err)
	}
	n, err = metadata.Write(root)
	if err != nil {
		return nil, fmt.Errorf("writing signed checkpoint root %s to message metadata: %w", root, err)
	}
	if n != SIGNED_CHECKPOINT_ROOT_LEN {
		return nil, fmt.Errorf("invalid length for signed checkpoint root, expected %d, got %d", SIGNED_CHECKPOINT_ROOT_LEN, n)
	}

	index := c.Checkpoint.Checkpoint.Index
	err = binary.Write(metadata, binary.BigEndian, index)
	if err != nil {
		return nil, fmt.Errorf("writing signed checkpoint index %d to message metadata: %w", index, err)
	}

	for _, signature := range c.Signatures {
		sigBytes, err := signature.Bytes()
		if err != nil {
			return nil, fmt.Errorf("converting signature to bytes: %w", err)
		}

		n, err = metadata.Write(sigBytes)
		if err != nil {
			return nil, fmt.Errorf("writing signature bytes %s to message metadata: %w", string(sigBytes), err)
		}
		if n != VALIDATOR_SIGNATURE_LENGTH {
			return nil, fmt.Errorf("invalid length for signature, expected %d, got %d", VALIDATOR_SIGNATURE_LENGTH, n)
		}
	}

	return metadata.Bytes(), nil
}

type MailboxDispatchEvent struct {
	Recipient         string
	Message           string
	DestinationDomain string
	SenderMailbox     string
	Sender            string
	MessageID         string
}

type MailboxMerkleHookPostDispatchEvent struct {
	MessageID string `json:"message_id"`
	Index     uint64 `json:"index"`
}
