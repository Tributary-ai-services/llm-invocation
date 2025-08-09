package crypto

import (
	"bytes"
	"fmt"
	"testing"
)

func TestEncryptDecryptWithPassword(t *testing.T) {
	// Test cases
	testCases := []struct {
		name      string
		plaintext string
		password  string
	}{
		{
			name:      "simple text",
			plaintext: "hello world",
			password:  "password123",
		},
		{
			name:      "api key",
			plaintext: "sk-1234567890abcdef",
			password:  "user-secret-password",
		},
		{
			name:      "long text",
			plaintext: "This is a longer piece of text that should still encrypt and decrypt correctly with our PBKDF2 + AES-256-GCM implementation",
			password:  "complex-password-with-special-chars!@#$%",
		},
		{
			name:      "unicode text",
			plaintext: "Hello 世界! 🌍 Encryption test with émojis and spéciål chars",
			password:  "unicode-密码-🔐",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encryptedData, err := EncryptWithPassword([]byte(tc.plaintext), []byte(tc.password))
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Verify encrypted data structure
			if len(encryptedData.Ciphertext) == 0 {
				t.Error("Ciphertext is empty")
			}
			if len(encryptedData.Salt) == 0 {
				t.Error("Salt is empty")
			}
			if len(encryptedData.Nonce) == 0 {
				t.Error("Nonce is empty")
			}

			// Decrypt
			decryptedBytes, err := DecryptWithPassword(encryptedData, []byte(tc.password))
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Verify decrypted content
			decrypted := string(decryptedBytes)
			if decrypted != tc.plaintext {
				t.Errorf("Decrypted text doesn't match original.\nExpected: %q\nGot: %q", tc.plaintext, decrypted)
			}
		})
	}
}

func TestDecryptWithWrongPassword(t *testing.T) {
	plaintext := "secret api key"
	correctPassword := "correct-password"
	wrongPassword := "wrong-password"

	// Encrypt with correct password
	encryptedData, err := EncryptWithPassword([]byte(plaintext), []byte(correctPassword))
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Try to decrypt with wrong password
	_, err = DecryptWithPassword(encryptedData, []byte(wrongPassword))
	if err == nil {
		t.Error("Decryption should have failed with wrong password")
	}
}

func TestKeyDerivation(t *testing.T) {
	password := []byte("test-password")
	salt := []byte("test-salt-12345")

	// Derive key twice with same inputs
	key1 := KeyDerivation(password, salt)
	key2 := KeyDerivation(password, salt)

	// Should produce identical results
	if !bytes.Equal(key1, key2) {
		t.Error("Key derivation should be deterministic")
	}

	// Key should be 32 bytes (256 bits)
	if len(key1) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key1))
	}

	// Different salt should produce different key
	differentSalt := []byte("different-salt")
	key3 := KeyDerivation(password, differentSalt)
	if bytes.Equal(key1, key3) {
		t.Error("Different salts should produce different keys")
	}

	// Different password should produce different key
	differentPassword := []byte("different-password")
	key4 := KeyDerivation(differentPassword, salt)
	if bytes.Equal(key1, key4) {
		t.Error("Different passwords should produce different keys")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	lengths := []int{16, 32, 64, 128}

	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			bytes1, err := GenerateRandomBytes(length)
			if err != nil {
				t.Fatalf("Failed to generate random bytes: %v", err)
			}

			bytes2, err := GenerateRandomBytes(length)
			if err != nil {
				t.Fatalf("Failed to generate random bytes: %v", err)
			}

			// Check length
			if len(bytes1) != length {
				t.Errorf("Expected length %d, got %d", length, len(bytes1))
			}

			// Should be different (extremely unlikely to be same)
			if bytes.Equal(bytes1, bytes2) {
				t.Error("Two random byte sequences should not be identical")
			}

			// Should not be all zeros
			allZeros := make([]byte, length)
			if bytes.Equal(bytes1, allZeros) {
				t.Error("Random bytes should not be all zeros")
			}
		})
	}
}

func TestCreateKeyFingerprint(t *testing.T) {
	testKeys := []string{
		"sk-1234567890abcdef",
		"sk-ant-api-key-example",
		"google-api-key-example",
		"",
		"short",
		"very-long-api-key-that-exceeds-normal-lengths-for-testing-purposes",
	}

	for i, key := range testKeys {
		t.Run(fmt.Sprintf("key_%d", i), func(t *testing.T) {
			fingerprint1 := CreateKeyFingerprint([]byte(key))
			fingerprint2 := CreateKeyFingerprint([]byte(key))

			// Should be consistent
			if fingerprint1 != fingerprint2 {
				t.Error("Fingerprint should be consistent for same input")
			}

			// Should be non-empty (except for empty input)
			if key != "" && fingerprint1 == "" {
				t.Error("Fingerprint should not be empty for non-empty input")
			}

			// Should be hex string
			if key != "" && len(fingerprint1) == 0 {
				t.Error("Fingerprint should be non-empty hex string")
			}
		})
	}

	// Different keys should produce different fingerprints
	key1 := "sk-key-one"
	key2 := "sk-key-two"
	
	fp1 := CreateKeyFingerprint([]byte(key1))
	fp2 := CreateKeyFingerprint([]byte(key2))
	
	if fp1 == fp2 {
		t.Error("Different keys should produce different fingerprints")
	}
}

func TestSecureZero(t *testing.T) {
	// Create test data
	testData := []byte("sensitive-api-key-data")
	originalLength := len(testData)

	// Make a copy to verify it gets zeroed
	testCopy := make([]byte, len(testData))
	copy(testCopy, testData)

	// Zero the data
	SecureZero(testCopy)

	// Verify length unchanged
	if len(testCopy) != originalLength {
		t.Errorf("SecureZero changed slice length: expected %d, got %d", originalLength, len(testCopy))
	}

	// Verify all bytes are zero
	for i, b := range testCopy {
		if b != 0 {
			t.Errorf("Byte at index %d not zeroed: got %d", i, b)
		}
	}

	// Verify original data is unchanged (to ensure we're testing correctly)
	if bytes.Equal(testData, testCopy) {
		t.Error("Original data should not equal zeroed data")
	}
}

func BenchmarkEncryptWithPassword(b *testing.B) {
	plaintext := []byte("sk-1234567890abcdefghijklmnopqrstuvwxyz")
	password := []byte("benchmark-password-123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptWithPassword(plaintext, password)
		if err != nil {
			b.Fatalf("Encryption failed: %v", err)
		}
	}
}

func BenchmarkDecryptWithPassword(b *testing.B) {
	plaintext := []byte("sk-1234567890abcdefghijklmnopqrstuvwxyz")
	password := []byte("benchmark-password-123")

	// Encrypt once
	encData, err := EncryptWithPassword(plaintext, password)
	if err != nil {
		b.Fatalf("Setup encryption failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecryptWithPassword(encData, password)
		if err != nil {
			b.Fatalf("Decryption failed: %v", err)
		}
	}
}

func BenchmarkKeyDerivation(b *testing.B) {
	password := []byte("benchmark-password")
	salt := []byte("benchmark-salt-16")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KeyDerivation(password, salt)
	}
}