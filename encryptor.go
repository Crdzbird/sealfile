package sealfile

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

// Encryptor handles AES encryption and decryption
type Encryptor struct {
	key       []byte
	cipherKey cipher.Block
	cipherGCM cipher.AEAD
}

// NewEncryptor creates a new Encryptor with the provided key
func NewEncryptor(key string) (*Encryptor, error) {
	e := &Encryptor{}
	e.setKey(key)

	var err error
	e.cipherKey, err = aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	e.cipherGCM, err = cipher.NewGCM(e.cipherKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return e, nil
}

// setKey pads or truncates the key to valid AES key sizes (16, 24, or 32 bytes)
func (e *Encryptor) setKey(key string) {
	keyBytes := []byte(key)
	switch len(keyBytes) {
	case 16, 24, 32:
		e.key = keyBytes
	default:
		// Default to 32 bytes (AES-256)
		e.key = make([]byte, 32)
		copy(e.key, keyBytes)
	}
}

// Encrypt encrypts data using AES-GCM
func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, e.cipherGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	encrypted := e.cipherGCM.Seal(nonce, nonce, data, nil)
	return encrypted, nil
}

// Decrypt decrypts AES-GCM encrypted data
func (e *Encryptor) Decrypt(encryptedData []byte) ([]byte, error) {
	nonceSize := e.cipherGCM.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("encrypted data too short")
	}

	nonce := encryptedData[:nonceSize]
	ciphertext := encryptedData[nonceSize:]

	decrypted, err := e.cipherGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return decrypted, nil
}
