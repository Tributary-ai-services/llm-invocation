package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// AES-256 key size
	KeySize = 32
	// GCM nonce size
	NonceSize = 12
	// PBKDF2 iteration count (100,000 - secure but not too slow)
	PBKDF2Iterations = 100000
	// Salt size for key derivation
	SaltSize = 32
)

var (
	ErrInvalidKeySize   = errors.New("invalid key size")
	ErrInvalidNonceSize = errors.New("invalid nonce size")
	ErrDecryptionFailed = errors.New("decryption failed")
	ErrInvalidCiphertext = errors.New("invalid ciphertext format")
)

// EncryptedData represents encrypted data with all necessary components
type EncryptedData struct {
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	Salt       []byte `json:"salt"`
}

// KeyDerivation derives an AES-256 key from a password using PBKDF2
func KeyDerivation(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, PBKDF2Iterations, KeySize, sha256.New)
}

// GenerateSalt creates a cryptographically secure random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateNonce creates a cryptographically secure random nonce for GCM
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceSize)
	_, err := rand.Read(nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// EncryptWithPassword encrypts plaintext using AES-256-GCM with a password-derived key
func EncryptWithPassword(plaintext, password []byte) (*EncryptedData, error) {
	// Generate salt for key derivation
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	// Derive key from password
	key := KeyDerivation(password, salt)

	// Generate nonce
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Encrypt with derived key
	ciphertext, err := EncryptAESGCM(plaintext, key, nonce)
	if err != nil {
		return nil, err
	}

	return &EncryptedData{
		Ciphertext: ciphertext,
		Nonce:      nonce,
		Salt:       salt,
	}, nil
}

// DecryptWithPassword decrypts ciphertext using AES-256-GCM with a password-derived key
func DecryptWithPassword(encData *EncryptedData, password []byte) ([]byte, error) {
	if len(encData.Salt) != SaltSize {
		return nil, fmt.Errorf("invalid salt size: got %d, want %d", len(encData.Salt), SaltSize)
	}

	// Derive key from password and salt
	key := KeyDerivation(password, encData.Salt)

	// Decrypt
	return DecryptAESGCM(encData.Ciphertext, key, encData.Nonce)
}

// EncryptAESGCM encrypts plaintext using AES-256-GCM
func EncryptAESGCM(plaintext, key, nonce []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	if len(nonce) != NonceSize {
		return nil, ErrInvalidNonceSize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAESGCM decrypts ciphertext using AES-256-GCM
func DecryptAESGCM(ciphertext, key, nonce []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	if len(nonce) != NonceSize {
		return nil, ErrInvalidNonceSize
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// SecureZero overwrites memory with zeros (for clearing sensitive data)
func SecureZero(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return bytes, nil
}

// Hash creates a SHA-256 hash of the input (useful for key fingerprints)
func Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// CreateKeyFingerprint creates a human-readable fingerprint of a key
func CreateKeyFingerprint(key []byte) string {
	hash := Hash(key)
	// Return first 16 chars as hex string (8 bytes)
	return fmt.Sprintf("%x", hash[:8])
}