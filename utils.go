package sealfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File type detection functions

// IsImageFile checks if a file is an image based on extension
func IsImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tiff", ".svg"}

	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// IsVideoFile checks if a file is a video based on extension
func IsVideoFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	videoExts := []string{".mp4", ".mov", ".avi", ".mkv", ".webm", ".flv", ".wmv", ".m4v"}

	for _, vidExt := range videoExts {
		if ext == vidExt {
			return true
		}
	}
	return false
}

// IsAudioFile checks if a file is an audio file based on extension
func IsAudioFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	audioExts := []string{".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma", ".m4a"}

	for _, audExt := range audioExts {
		if ext == audExt {
			return true
		}
	}
	return false
}

// IsDocumentFile checks if a file is a document based on extension
func IsDocumentFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	docExts := []string{".pdf", ".doc", ".docx", ".txt", ".rtf", ".odt", ".pages"}

	for _, docExt := range docExts {
		if ext == docExt {
			return true
		}
	}
	return false
}

// File manipulation functions

// CreateTempFile creates a temporary file with the given data
func CreateTempFile(dir, filename string, data []byte) (*os.File, error) {
	tempName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filename)
	tempPath := filepath.Join(dir, tempName)

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	file, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := file.Write(data); err != nil {
		err := file.Close()
		if err != nil {
			return nil, err
		}
		err = os.Remove(tempPath)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to write data to temp file: %w", err)
	}

	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		err := file.Close()
		if err != nil {
			return nil, err
		}
		err = os.Remove(tempPath)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	return file, nil
}

// GetFileNameWithoutExtension returns filename without extension
func GetFileNameWithoutExtension(filename string) string {
	return strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
}

// GetFileExtension returns the file extension (including the dot)
func GetFileExtension(filename string) string {
	return filepath.Ext(filename)
}

// SanitizeFilename removes or replaces invalid characters in filename
func SanitizeFilename(filename string) string {
	// Replace invalid characters with underscore
	invalidChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename

	for _, char := range invalidChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Remove leading/trailing spaces and dots
	sanitized = strings.Trim(sanitized, " .")

	// Ensure filename is not empty
	if sanitized == "" {
		return "untitled"
	}

	return sanitized
}

// EnsureDirectory creates directory if it doesn't exist
func EnsureDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	return nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filepath string) (int64, error) {
	info, err := os.Stat(filepath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	return info.Size(), nil
}
