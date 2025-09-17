package sealfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SecureFile represents a file with encryption capabilities
type SecureFile struct {
	Path       string
	Filename   string
	Extension  string
	Data       []byte
	config     *Config
	encryptor  *Encryptor
	compressor *Compressor
}

// NewSecureFile creates a new SecureFile instance (internal use)
func newSecureFile(data []byte, path, filename string, config *Config, encryptor *Encryptor, compressor *Compressor) *SecureFile {
	return &SecureFile{
		Path:       path,
		Filename:   filename,
		Extension:  filepath.Ext(filename),
		Data:       data,
		config:     config,
		encryptor:  encryptor,
		compressor: compressor,
	}
}

// SaveEncrypted saves the file with encryption and compression
func (sf *SecureFile) SaveEncrypted() error {
	// Encrypt the data
	encrypted, err := sf.encryptor.Encrypt(sf.Data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Compress the encrypted data
	compressed, err := sf.compressor.Compress(encrypted)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// Ensure directory exists
	if err := sf.ensureDirectory(); err != nil {
		return err
	}

	// Write to file
	fullPath := filepath.Join(sf.Path, sf.Filename)
	if err := os.WriteFile(fullPath, compressed, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// LoadDecrypted loads and decrypts a file
func (sf *SecureFile) LoadDecrypted() error {
	fullPath := filepath.Join(sf.Path, sf.Filename)

	// Read compressed data
	compressed, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Decompress data
	encrypted, err := sf.compressor.Decompress(compressed)
	if err != nil {
		return fmt.Errorf("failed to decompress data: %w", err)
	}

	// Decrypt data
	sf.Data, err = sf.encryptor.Decrypt(encrypted)
	if err != nil {
		return fmt.Errorf("failed to decrypt data: %w", err)
	}

	return nil
}

// Delete removes the secure file from disk
func (sf *SecureFile) Delete() error {
	fullPath := filepath.Join(sf.Path, sf.Filename)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// GetPath returns the file path based on the configured path type
func (sf *SecureFile) GetPath() string {
	switch sf.config.PathType {
	case HTTPPath:
		return sf.GetURL()
	default:
		return sf.GetFullPath()
	}
}

// GetURL returns the HTTP URL for the file
func (sf *SecureFile) GetURL() string {
	relativePath := strings.TrimPrefix(sf.Path, sf.config.PublicDir)
	relativePath = strings.TrimPrefix(relativePath, "/")
	if relativePath != "" && !strings.HasPrefix(relativePath, "/") {
		relativePath = "/" + relativePath
	}
	return fmt.Sprintf("%s%s/%s", sf.config.BaseURL, relativePath, sf.Filename)
}

// GetDirectoryPath returns just the directory path
func (sf *SecureFile) GetDirectoryPath() string {
	return sf.Path
}

// GetFullPath returns the complete file path
func (sf *SecureFile) GetFullPath() string {
	return filepath.Join(sf.Path, sf.Filename)
}

// ensureDirectory creates the directory if it doesn't exist
func (sf *SecureFile) ensureDirectory() error {
	if _, err := os.Stat(sf.Path); os.IsNotExist(err) {
		if err := os.MkdirAll(sf.Path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}
