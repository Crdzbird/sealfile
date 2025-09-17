package sealfile

import (
	"fmt"
)

// BatchProcessor processes multiple files concurrently
type BatchProcessor struct {
	fm          *FileManager
	concurrency int
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(fm *FileManager, concurrency int) *BatchProcessor {
	if concurrency <= 0 {
		concurrency = 5 // Default concurrency
	}
	return &BatchProcessor{
		fm:          fm,
		concurrency: concurrency,
	}
}

// ProcessFiles processes multiple files concurrently
func (bp *BatchProcessor) ProcessFiles(files []*SecureFile, processor func(*SecureFile) error) []error {
	semaphore := make(chan struct{}, bp.concurrency)
	errChan := make(chan error, len(files))

	for _, file := range files {
		go func(f *SecureFile) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore
			if err := processor(f); err != nil {
				errChan <- fmt.Errorf("failed to process file %s: %w", f.Filename, err)
				return
			}
			errChan <- nil
		}(file)
	}

	// Collect errors
	errors := make([]error, len(files))
	for i := 0; i < len(files); i++ {
		errors[i] = <-errChan
	}
	return errors
}

// SaveAllFiles saves multiple files concurrently
func (bp *BatchProcessor) SaveAllFiles(files []*SecureFile) []error {
	return bp.ProcessFiles(files, func(sf *SecureFile) error {
		return sf.SaveEncrypted()
	})
}

// LoadAllFiles loads multiple files concurrently
func (bp *BatchProcessor) LoadAllFiles(files []*SecureFile) []error {
	return bp.ProcessFiles(files, func(sf *SecureFile) error {
		return sf.LoadDecrypted()
	})
}

// DeleteAllFiles deletes multiple files concurrently
func (bp *BatchProcessor) DeleteAllFiles(files []*SecureFile) []error {
	return bp.ProcessFiles(files, func(sf *SecureFile) error {
		return sf.Delete()
	})
}
