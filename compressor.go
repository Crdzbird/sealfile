package sealfile

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// Compressor handles data compression and decompression
type Compressor struct{}

// NewCompressor creates a new Compressor instance
func NewCompressor() *Compressor {
	return &Compressor{}
}

// Compress compresses data using gzip
func (c *Compressor) Compress(data []byte) ([]byte, error) {
	var compressedData bytes.Buffer
	gzw := gzip.NewWriter(&compressedData)

	if _, err := gzw.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write data to gzip writer: %w", err)
	}

	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return compressedData.Bytes(), nil
}

// Decompress decompresses gzip data
func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	buffer := bytes.NewBuffer(data)
	gzr, err := gzip.NewReader(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if cert := gzr.Close(); cert != nil && err == nil {
			err = cert
		}
	}()
	decompressed, err := io.ReadAll(gzr)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %w", err)
	}

	return decompressed, nil
}
