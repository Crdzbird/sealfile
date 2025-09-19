package sealfile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// SaltSize Security constants
	SaltSize = 16 // 128 bits
	// KeyIterations defines the number of iterations for PBKDF2
	KeyIterations = 100000 // PBKDF2 iterations
	// KeyLength defines the length of the derived key
	KeyLength = 32 // AES-256
)

// Encryptor handles AES encryption and decryption with salt and pepper
type Encryptor struct {
	baseKey     []byte
	pepper      []byte
	cipherKey   cipher.Block
	cipherGCM   cipher.AEAD
	currentSalt []byte
}

// NewEncryptor creates a new Encryptor with the provided key and pepper
func NewEncryptor(key, pepper string) (*Encryptor, error) {
	e := &Encryptor{
		baseKey: []byte(key),
		pepper:  []byte(pepper),
	}

	// For initialization, use a temporary salt to set up the cipher
	tempSalt := make([]byte, SaltSize)
	if _, err := rand.Read(tempSalt); err != nil {
		return nil, fmt.Errorf("failed to generate temporary salt: %w", err)
	}

	if err := e.updateCipher(tempSalt); err != nil {
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	return e, nil
}

// updateCipher updates the cipher with a new salt
func (e *Encryptor) updateCipher(salt []byte) error {
	e.currentSalt = salt

	// Derive key using PBKDF2 with salt and pepper
	derivedKey := e.deriveKey(salt)

	var err error
	e.cipherKey, err = aes.NewCipher(derivedKey)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	e.cipherGCM, err = cipher.NewGCM(e.cipherKey)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	return nil
}

// deriveKey derives the encryption key using PBKDF2 with salt and pepper
func (e *Encryptor) deriveKey(salt []byte) []byte {
	// Combine base key with pepper for additional security
	keyMaterial := append(e.baseKey, e.pepper...)

	// Use PBKDF2 to derive the final encryption key
	derivedKey := pbkdf2.Key(keyMaterial, salt, KeyIterations, KeyLength, sha256.New)

	return derivedKey
}

// generateSalt creates a new random salt
func (e *Encryptor) generateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts data using AES-GCM with salt and pepper
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	// Generate a new salt for each encryption
	salt, err := e.generateSalt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Update cipher with new salt
	if err := e.updateCipher(salt); err != nil {
		return nil, fmt.Errorf("failed to update cipher with new salt: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, e.cipherGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := e.cipherGCM.Seal(nil, nonce, data, nil)

	// Format: [salt][nonce][ciphertext]
	result := make([]byte, 0, SaltSize+len(nonce)+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt decrypts AES-GCM encrypted data with salt and pepper
func (e *Encryptor) Decrypt(encryptedData []byte) ([]byte, error) {
	// Minimum size check: salt + nonce + some ciphertext
	minSize := SaltSize + e.cipherGCM.NonceSize() + 1
	if len(encryptedData) < minSize {
		return nil, fmt.Errorf("encrypted data too short: got %d bytes, need at least %d", len(encryptedData), minSize)
	}

	// Extract salt
	salt := encryptedData[:SaltSize]
	remaining := encryptedData[SaltSize:]

	// Update cipher with extracted salt
	if err := e.updateCipher(salt); err != nil {
		return nil, fmt.Errorf("failed to update cipher with extracted salt: %w", err)
	}

	// Extract nonce
	nonceSize := e.cipherGCM.NonceSize()
	if len(remaining) < nonceSize {
		return nil, fmt.Errorf("insufficient data for nonce: got %d bytes, need %d", len(remaining), nonceSize)
	}

	nonce := remaining[:nonceSize]
	ciphertext := remaining[nonceSize:]

	// Decrypt data
	decrypted, err := e.cipherGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return decrypted, nil
}

// GetCurrentSalt returns the current salt (for debugging/testing purposes)
func (e *Encryptor) GetCurrentSalt() []byte {
	salt := make([]byte, len(e.currentSalt))
	copy(salt, e.currentSalt)
	return salt
}

// VerifyPepper checks if the provided pepper matches the encryptor's pepper
func (e *Encryptor) VerifyPepper(pepper string) bool {
	return string(e.pepper) == pepper
}

// UpdatePepper updates the pepper (requires re-encryption of existing data)
func (e *Encryptor) UpdatePepper(newPepper string) error {
	e.pepper = []byte(newPepper)

	// Update cipher with current salt and new pepper
	if e.currentSalt != nil {
		if err := e.updateCipher(e.currentSalt); err != nil {
			return fmt.Errorf("failed to update cipher with new pepper: %w", err)
		}
	}

	return nil
}
