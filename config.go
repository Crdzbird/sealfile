package sealfile

import (
	"crypto/rand"
	"fmt"
)

// PathType defines how paths should be returned
type PathType int

// Private default key (should be overridden in production)
const _privateKey = "A7!xM3pL#9zQwR2@tF6vH8jK$1nB5cD0"

// Private default pepper (should be overridden in production)
const _privatePepper = "j^3ñZ!8r$L0@wF5+N2*Xv7#y4&Tk9=Q6h1ñ"

const (
	DirectoryPath PathType = iota
	HTTPPath
)

// Config holds configuration for the file library
type Config struct {
	EncryptionKey string
	Pepper        string // Additional secret for enhanced security
	BaseURL       string
	PublicDir     string
	TempDir       string
	PathType      PathType
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		EncryptionKey: _privateKey,
		Pepper:        _privatePepper,
		BaseURL:       "http://localhost:8080",
		PublicDir:     "./public",
		TempDir:       "./temp",
		PathType:      DirectoryPath,
	}
}

// GenerateRandomPepper generates a cryptographically secure random pepper
func GenerateRandomPepper(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random pepper: %w", err)
	}

	// Convert to hex string
	pepper := fmt.Sprintf("%x", bytes)
	return pepper, nil
}
