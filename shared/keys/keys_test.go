package keys

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"crypto/aes"
	"crypto/cipher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadKeyStoreFromPlaintextFile(t *testing.T) {
	// Create a temporary file with test data
	testData := map[string]map[string]string{
		"chain1": {"private_key": "key1"},
		"chain2": {"private_key": "key2"},
	}
	testFile := createTempFile(t, testData)
	defer os.Remove(testFile.Name())

	// Test the function
	result, err := LoadKeyStoreFromPlaintextFile(testFile.Name())
	require.NoError(t, err, "LoadKeyStoreFromPlaintextFile should not return an error")

	// Check the results
	expected := map[string]string{
		"chain1": "key1",
		"chain2": "key2",
	}
	assert.Equal(t, expected, result, "Loaded key store should match expected data")
}

func TestLoadKeyStoreFromEnv(t *testing.T) {
	// Set up test environment variable
	testData := map[string]map[string]string{
		"chain1": {"private_key": "key1"},
		"chain2": {"private_key": "key2"},
	}
	jsonData, err := json.Marshal(testData)
	require.NoError(t, err, "JSON marshaling should not fail")

	os.Setenv("SOLVER_KEYS", string(jsonData))
	defer os.Unsetenv("SOLVER_KEYS")

	// Test the function
	result, err := LoadKeyStoreFromEnv()
	require.NoError(t, err, "LoadKeyStoreFromEnv should not return an error")

	// Check the results
	expected := map[string]string{
		"chain1": "key1",
		"chain2": "key2",
	}
	assert.Equal(t, expected, result, "Loaded key store should match expected data")
}

func TestLoadKeyStoreFromEncryptedFile(t *testing.T) {
	// Create test data
	testData := map[string]map[string]string{
		"chain1": {"private_key": "key1"},
		"chain2": {"private_key": "key2"},
	}
	jsonData, err := json.Marshal(testData)
	require.NoError(t, err, "JSON marshaling should not fail")

	// Create a test AES key
	aesKey := []byte("0123456789abcdef0123456789abcdef")
	aesKeyHex := hex.EncodeToString(aesKey)

	// set key environment variable
	os.Setenv("AES_KEY_HEX", aesKeyHex)

	// Encrypt the test data
	encryptedData := encryptTestData(t, jsonData, aesKey)
	encryptedDataHex := hex.EncodeToString(encryptedData)

	// Create a temporary file with encrypted data
	testFile := createTempFileWithContent(t, []byte(encryptedDataHex))
	defer os.Remove(testFile.Name())

	// Test the function
	result, err := LoadKeyStoreFromEncryptedFile(testFile.Name())
	require.NoError(t, err, "LoadKeyStoreFromEncryptedFile should not return an error")

	// Check the results
	expected := map[string]string{
		"chain1": "key1",
		"chain2": "key2",
	}
	assert.Equal(t, expected, result, "Loaded key store should match expected data")
}

// Helper functions

func createTempFile(t *testing.T, data map[string]map[string]string) *os.File {
	file, err := os.CreateTemp("", "keystore_test")
	require.NoError(t, err, "Failed to create temp file")

	jsonData, err := json.Marshal(data)
	require.NoError(t, err, "JSON marshaling should not fail")

	_, err = file.Write(jsonData)
	require.NoError(t, err, "Failed to write to temp file")

	err = file.Close()
	require.NoError(t, err, "Failed to close temp file")

	return file
}

func createTempFileWithContent(t *testing.T, content []byte) *os.File {
	file, err := os.CreateTemp("", "keystore_test")
	require.NoError(t, err, "Failed to create temp file")

	_, err = file.Write(content)
	require.NoError(t, err, "Failed to write to temp file")

	err = file.Close()
	require.NoError(t, err, "Failed to close temp file")

	return file
}

func encryptTestData(t *testing.T, data []byte, key []byte) []byte {
	block, err := aes.NewCipher(key)
	require.NoError(t, err, "Failed to create AES cipher")

	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err, "Failed to create GCM")

	nonce := make([]byte, gcm.NonceSize())
	ciphertext := gcm.Seal(nil, nonce, data, nil)
	return append(nonce, ciphertext...)
}
