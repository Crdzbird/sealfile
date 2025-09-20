package sealfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileManager manages secure file operations
type FileManager struct {
	config     *Config
	encryptor  *Encryptor
	compressor *Compressor
}

// FileOperation represents a file operation for batch processing
type FileOperation struct {
	Data     []byte
	Path     string
	Filename string
	Error    error
}

// CopyOptions defines options for file copying
type CopyOptions struct {
	DecryptBeforeCopy bool
	OverwriteExisting bool
	CreateDirectories bool
}

// NewFileManager creates a new FileManager instance
func NewFileManager(config *Config) (*FileManager, error) {
	if config == nil {
		config = DefaultConfig()
	}
	if config.Pepper == "" {
		return nil, fmt.Errorf("pepper is required for enhanced security")
	}
	encryptor, err := NewEncryptor(config.EncryptionKey, config.Pepper)
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

// CreateMultipleEncryptedFiles creates multiple encrypted files from a list of file operations
func (fm *FileManager) CreateMultipleEncryptedFiles(operations []FileOperation, maxConcurrency int) []FileOperation {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}
	results := make([]FileOperation, len(operations))
	copy(results, operations)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			op := &results[index]
			sf := fm.NewSecureFile(op.Data, op.Path, op.Filename)
			if err := sf.SaveEncrypted(); err != nil {
				op.Error = fmt.Errorf("failed to encrypt and save file %s: %w", op.Filename, err)
				return
			}
			op.Error = nil
		}(i)
	}

	wg.Wait()
	return results
}

// DecryptMultipleFiles decrypts multiple files from a list of file operations
func (fm *FileManager) DecryptMultipleFiles(operations []FileOperation, maxConcurrency int) []FileOperation {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}
	results := make([]FileOperation, len(operations))
	copy(results, operations)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			op := &results[index]
			sf, err := fm.LoadSecureFileFromDisk(op.Path, op.Filename)
			if err != nil {
				op.Error = fmt.Errorf("failed to decrypt file %s: %w", op.Filename, err)
				return
			}
			op.Data = sf.Data
			op.Error = nil
		}(i)
	}
	wg.Wait()
	return results
}

// CopyFileToNewLocation copies a file to a new location with optional decryption
func (fm *FileManager) CopyFileToNewLocation(sourcePath, sourceFilename, destPath, destFilename string, options CopyOptions) error {
	if options.CreateDirectories {
		if err := EnsureDirectory(destPath); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}
	destFullPath := filepath.Join(destPath, destFilename)
	if !options.OverwriteExisting {
		if _, err := os.Stat(destFullPath); err == nil {
			return fmt.Errorf("destination file already exists: %s", destFullPath)
		}
	}
	if options.DecryptBeforeCopy {
		return fm.copyWithDecryption(sourcePath, sourceFilename, destPath, destFilename)
	}
	return fm.copyEncryptedFile(sourcePath, sourceFilename, destPath, destFilename)
}

// copyWithDecryption decrypts the file and saves the unencrypted version
func (fm *FileManager) copyWithDecryption(sourcePath, sourceFilename, destPath, destFilename string) error {
	sourceFile, err := fm.LoadSecureFileFromDisk(sourcePath, sourceFilename)
	if err != nil {
		return fmt.Errorf("failed to load source file: %w", err)
	}
	destFullPath := filepath.Join(destPath, destFilename)
	if err := os.WriteFile(destFullPath, sourceFile.Data, 0644); err != nil {
		return fmt.Errorf("failed to write unencrypted file: %w", err)
	}
	return nil
}

// copyEncryptedFile copies the encrypted file as-is
func (fm *FileManager) copyEncryptedFile(sourcePath, sourceFilename, destPath, destFilename string) error {
	sourceFullPath := filepath.Join(sourcePath, sourceFilename)
	destFullPath := filepath.Join(destPath, destFilename)
	data, err := os.ReadFile(sourceFullPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}
	if err := os.WriteFile(destFullPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	return nil
}

// BatchCopyFiles copies multiple files to new locations with optional decryption
func (fm *FileManager) BatchCopyFiles(copyOperations []CopyOperation, maxConcurrency int) []CopyResult {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}
	results := make([]CopyResult, len(copyOperations))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)
	for i, op := range copyOperations {
		wg.Add(1)
		go func(index int, operation CopyOperation) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			err := fm.CopyFileToNewLocation(
				operation.SourcePath,
				operation.SourceFilename,
				operation.DestPath,
				operation.DestFilename,
				operation.Options,
			)
			results[index] = CopyResult{
				SourcePath:     operation.SourcePath,
				SourceFilename: operation.SourceFilename,
				DestPath:       operation.DestPath,
				DestFilename:   operation.DestFilename,
				Success:        err == nil,
				Error:          err,
			}
		}(i, op)
	}
	wg.Wait()
	return results
}

// GetConfig returns the current configuration
func (fm *FileManager) GetConfig() *Config {
	return fm.config
}

// UpdateConfig updates the configuration (creates new encryptor if key/pepper changed)
func (fm *FileManager) UpdateConfig(config *Config) error {
	if config.Pepper == "" {
		return fmt.Errorf("pepper is required for enhanced security")
	}
	keyChanged := config.EncryptionKey != fm.config.EncryptionKey
	pepperChanged := config.Pepper != fm.config.Pepper
	if keyChanged || pepperChanged {
		encryptor, err := NewEncryptor(config.EncryptionKey, config.Pepper)
		if err != nil {
			return fmt.Errorf("failed to create new encryptor: %w", err)
		}
		fm.encryptor = encryptor
	}
	fm.config = config
	return nil
}

// VerifyPepper verifies that the current pepper matches the provided one
func (fm *FileManager) VerifyPepper(pepper string) bool {
	return fm.encryptor.VerifyPepper(pepper)
}

// RotatePepper updates the pepper (warning: existing encrypted files will need re-encryption)
func (fm *FileManager) RotatePepper(newPepper string) error {
	if err := fm.encryptor.UpdatePepper(newPepper); err != nil {
		return fmt.Errorf("failed to update pepper: %w", err)
	}
	fm.config.Pepper = newPepper
	return nil
}

// ReEncryptFile re-encrypts a file with the current salt+pepper configuration
func (fm *FileManager) ReEncryptFile(path, filename string) error {
	sf, err := fm.LoadSecureFileFromDisk(path, filename)
	if err != nil {
		return fmt.Errorf("failed to load file for re-encryption: %w", err)
	}
	if err := sf.SaveEncrypted(); err != nil {
		return fmt.Errorf("failed to re-encrypt file: %w", err)
	}
	return nil
}

// ReEncryptMultipleFiles re-encrypts multiple files with current salt+pepper configuration
func (fm *FileManager) ReEncryptMultipleFiles(operations []FileOperation, maxConcurrency int) []FileOperation {
	if maxConcurrency <= 0 {
		maxConcurrency = 5
	}
	results := make([]FileOperation, len(operations))
	copy(results, operations)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxConcurrency)
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			op := &results[index]
			err := fm.ReEncryptFile(op.Path, op.Filename)
			if err != nil {
				op.Error = fmt.Errorf("failed to re-encrypt file %s: %w", op.Filename, err)
				return
			}
			op.Error = nil
		}(i)
	}
	wg.Wait()
	return results
}

// CopyOperation represents a file copy operation
type CopyOperation struct {
	SourcePath     string
	SourceFilename string
	DestPath       string
	DestFilename   string
	Options        CopyOptions
}

// CopyResult represents the result of a file copy operation
type CopyResult struct {
	SourcePath     string
	SourceFilename string
	DestPath       string
	DestFilename   string
	Success        bool
	Error          error
}
