package sealfile

// PathType defines how paths should be returned
type PathType int

const (
	DirectoryPath PathType = iota
	HTTPPath
)

const _privateKey = "A7!xM3pL#9zQwR2@tF6vH8jK$1nB5cD0"

// Config holds configuration for the file library
type Config struct {
	EncryptionKey string
	BaseURL       string
	PublicDir     string
	TempDir       string
	PathType      PathType
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		EncryptionKey: _privateKey,
		BaseURL:       "http://localhost:8080",
		PublicDir:     "./public",
		TempDir:       "./temp",
		PathType:      DirectoryPath,
	}
}
