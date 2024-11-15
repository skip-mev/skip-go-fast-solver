package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/keys"
	"os"
)

func main() {
	solverKeys := map[string]map[string]string{
		"chain1": {"private_key": "key1"},
		"chain2": {"private_key": "key2"},
	}
	jsonData, _ := json.Marshal(solverKeys)

	// Create a test AES key
	aesKey := []byte("0123456789abcdef0123456789abcdef")
	aesKeyHex := hex.EncodeToString(aesKey)

	// Encrypt the test data
	encryptedData := encryptSolverKeys(jsonData, aesKey)
	encryptedDataHex := hex.EncodeToString(encryptedData)

	// Create a temporary file with encrypted data
	testFile := createTempFileWithContent([]byte(encryptedDataHex))
	defer os.Remove(testFile.Name())

	// Load keys from encrypted file
	result, _ := keys.LoadKeyStoreFromEncryptedFile(testFile.Name(), aesKeyHex)
	resultJson, _ := json.Marshal(result)

	// This is just an example, of course don't log your real private keys
	fmt.Printf("Decrypted keys %s", string(resultJson))
}

func createTempFileWithContent(content []byte) *os.File {
	file, _ := os.CreateTemp("", "keystore_test")

	_, _ = file.Write(content)

	_ = file.Close()

	return file
}

func encryptSolverKeys(data []byte, key []byte) []byte {
	block, _ := aes.NewCipher(key)

	gcm, _ := cipher.NewGCM(block)

	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nil, nonce, data, nil)
	return append(nonce, ciphertext...)
}
