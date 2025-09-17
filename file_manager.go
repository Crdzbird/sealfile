package sealfile

import (
	"fmt"
)

// FileManager manages secure file operations
type FileManager struct {
	config     *Config
	encryptor  *Encryptor
	compressor *Compressor
}

// NewFileManager creates a new FileManager instance
func NewFileManager(config *Config) (*FileManager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	encryptor, err := NewEncryptor(config.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}

	fm := &FileManager{
		config:     config,
		encryptor:  encryptor,
		compressor: NewCompressor(),
	}

	return fm, nil
}

// NewSecureFile creates a new SecureFile instance
func (fm *FileManager) NewSecureFile(data []byte, path, filename string) *SecureFile {
	return newSecureFile(data, path, filename, fm.config, fm.encryptor, fm.compressor)
}

// LoadSecureFileFromDisk loads a secure file from disk
func (fm *FileManager) LoadSecureFileFromDisk(path, filename string) (*SecureFile, error) {
	sf := fm.NewSecureFile(nil, path, filename)
	if err := sf.LoadDecrypted(); err != nil {
		return nil, err
	}
	return sf, nil
}

// SaveDataAsSecureFile saves raw data as a secure file
func (fm *FileManager) SaveDataAsSecureFile(data []byte, path, filename string) (*SecureFile, error) {
	sf := fm.NewSecureFile(data, path, filename)
	if err := sf.SaveEncrypted(); err != nil {
		return nil, err
	}
	return sf, nil
}

// GetConfig returns the current configuration
func (fm *FileManager) GetConfig() *Config {
	return fm.config
}

// UpdateConfig updates the configuration (creates new encryptor if key changed)
func (fm *FileManager) UpdateConfig(config *Config) error {
	if config.EncryptionKey != fm.config.EncryptionKey {
		encryptor, err := NewEncryptor(config.EncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to create new encryptor: %w", err)
		}
		fm.encryptor = encryptor
	}
	fm.config = config
	return nil
}
