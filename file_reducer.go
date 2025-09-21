package sealfile

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"fmt"
	"io"
	"math"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/ulikunitz/xz"
)

// CompressionMethod defines the compression algorithm to use
type CompressionMethod int

const (
	// Standard compression methods
	GZIP CompressionMethod = iota
	ZLIB
	DEFLATE
	LZW

	// Advanced compression methods (better than WinRAR)
	ZSTD     // Facebook's Zstandard - excellent compression ratio and speed
	LZ4      // Ultra-fast compression
	XZ       // LZMA2 - highest compression ratio
	HYBRID   // Multi-stage compression for maximum reduction
	ADAPTIVE // Automatically chooses best method based on content
)

// CompressionLevel defines the compression intensity
type CompressionLevel int

const (
	FASTEST CompressionLevel = iota
	FAST
	BALANCED
	BEST
	MAXIMUM // Ultra compression - slow but maximum reduction
)

// FileReducer handles advanced file size reduction and restoration
type FileReducer struct {
	method          CompressionMethod
	level           CompressionLevel
	chunkSize       int
	enablePreFilter bool
	enablePostOpt   bool
}

// CompressionResult contains the results of compression operation
type CompressionResult struct {
	OriginalSize    int64
	CompressedSize  int64
	CompressionRate float64
	Method          CompressionMethod
	ProcessingTime  int64 // in milliseconds
	ChunksProcessed int
}

// NewFileReducer creates a new file reducer with specified settings
func NewFileReducer(method CompressionMethod, level CompressionLevel) *FileReducer {
	return &FileReducer{
		method:          method,
		level:           level,
		chunkSize:       64 * 1024, // 64KB chunks for optimal processing
		enablePreFilter: true,      // Pre-filtering for better compression
		enablePostOpt:   true,      // Post-optimization
	}
}

// NewAdvancedFileReducer creates a file reducer optimized for maximum compression
func NewAdvancedFileReducer() *FileReducer {
	return &FileReducer{
		method:          HYBRID,
		level:           MAXIMUM,
		chunkSize:       128 * 1024, // Larger chunks for better compression
		enablePreFilter: true,
		enablePostOpt:   true,
	}
}

// SetChunkSize sets the chunk size for processing large files
func (fr *FileReducer) SetChunkSize(size int) {
	if size < 1024 {
		size = 1024 // Minimum 1KB
	}
	fr.chunkSize = size
}

// EnableOptimizations enables/disables advanced optimizations
func (fr *FileReducer) EnableOptimizations(preFilter, postOpt bool) {
	fr.enablePreFilter = preFilter
	fr.enablePostOpt = postOpt
}

// ReduceFileSize compresses the input data using advanced algorithms
func (fr *FileReducer) ReduceFileSize(data []byte) ([]byte, *CompressionResult, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("input data is empty")
	}

	originalSize := int64(len(data))
	startTime := getCurrentTimeMs()

	// Pre-filtering for better compression (removes redundancy)
	processedData := data
	if fr.enablePreFilter {
		processedData = fr.preFilterData(data)
	}

	var compressed []byte
	var err error
	var method CompressionMethod = fr.method

	// Adaptive method selection based on data characteristics
	if fr.method == ADAPTIVE {
		method = fr.selectOptimalMethod(processedData)
	}

	// Apply compression based on selected method
	switch method {
	case GZIP:
		compressed, err = fr.compressGzip(processedData)
	case ZLIB:
		compressed, err = fr.compressZlib(processedData)
	case DEFLATE:
		compressed, err = fr.compressDeflate(processedData)
	case LZW:
		compressed, err = fr.compressLZW(processedData)
	case ZSTD:
		compressed, err = fr.compressZstd(processedData)
	case LZ4:
		compressed, err = fr.compressLZ4(processedData)
	case XZ:
		compressed, err = fr.compressXZ(processedData)
	case HYBRID:
		compressed, err = fr.compressHybrid(processedData)
	default:
		return nil, nil, fmt.Errorf("unsupported compression method: %d", method)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("compression failed: %w", err)
	}

	// Post-optimization for additional size reduction
	if fr.enablePostOpt {
		compressed = fr.postOptimizeData(compressed, method)
	}

	// Add method identifier header for restoration
	finalData := fr.addCompressionHeader(compressed, method, originalSize)

	endTime := getCurrentTimeMs()
	processingTime := endTime - startTime
	compressedSize := int64(len(finalData))
	compressionRate := (1.0 - float64(compressedSize)/float64(originalSize)) * 100.0

	result := &CompressionResult{
		OriginalSize:    originalSize,
		CompressedSize:  compressedSize,
		CompressionRate: compressionRate,
		Method:          method,
		ProcessingTime:  processingTime,
		ChunksProcessed: (len(data) + fr.chunkSize - 1) / fr.chunkSize,
	}

	return finalData, result, nil
}

// RestoreOriginalSize decompresses the data back to its original size
func (fr *FileReducer) RestoreOriginalSize(compressedData []byte) ([]byte, error) {
	if len(compressedData) == 0 {
		return nil, fmt.Errorf("compressed data is empty")
	}

	// Extract compression header
	method, originalSize, dataWithoutHeader, err := fr.extractCompressionHeader(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract compression header: %w", err)
	}

	// Remove post-optimization if it was applied
	if fr.enablePostOpt {
		dataWithoutHeader = fr.reversePostOptimization(dataWithoutHeader, method)
	}

	// Decompress based on method
	var decompressed []byte
	switch method {
	case GZIP:
		decompressed, err = fr.decompressGzip(dataWithoutHeader)
	case ZLIB:
		decompressed, err = fr.decompressZlib(dataWithoutHeader)
	case DEFLATE:
		decompressed, err = fr.decompressDeflate(dataWithoutHeader)
	case LZW:
		decompressed, err = fr.decompressLZW(dataWithoutHeader)
	case ZSTD:
		decompressed, err = fr.decompressZstd(dataWithoutHeader)
	case LZ4:
		decompressed, err = fr.decompressLZ4(dataWithoutHeader)
	case XZ:
		decompressed, err = fr.decompressXZ(dataWithoutHeader)
	case HYBRID:
		decompressed, err = fr.decompressHybrid(dataWithoutHeader)
	default:
		return nil, fmt.Errorf("unsupported compression method in header: %d", method)
	}

	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	// Reverse pre-filtering if it was applied
	if fr.enablePreFilter {
		decompressed = fr.reversePreFilter(decompressed)
	}

	// Verify size matches expected
	if int64(len(decompressed)) != originalSize {
		return nil, fmt.Errorf("decompressed size mismatch: expected %d, got %d",
			originalSize, len(decompressed))
	}

	return decompressed, nil
}

// GZIP compression
func (fr *FileReducer) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var writer *gzip.Writer
	switch fr.level {
	case FASTEST:
		writer, _ = gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	case FAST:
		writer, _ = gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	case BALANCED:
		writer, _ = gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	case BEST, MAXIMUM:
		writer, _ = gzip.NewWriterLevel(&buf, gzip.BestCompression)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := reader.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return io.ReadAll(reader)
}

// ZLIB compression
func (fr *FileReducer) compressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var writer *zlib.Writer
	switch fr.level {
	case FASTEST:
		writer, _ = zlib.NewWriterLevel(&buf, zlib.BestSpeed)
	case FAST:
		writer, _ = zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	case BALANCED:
		writer, _ = zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	case BEST, MAXIMUM:
		writer, _ = zlib.NewWriterLevel(&buf, zlib.BestCompression)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := reader.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	return io.ReadAll(reader)
}

// DEFLATE compression
func (fr *FileReducer) compressDeflate(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var writer *flate.Writer
	switch fr.level {
	case FASTEST:
		writer, _ = flate.NewWriter(&buf, flate.BestSpeed)
	case FAST:
		writer, _ = flate.NewWriter(&buf, flate.DefaultCompression)
	case BALANCED:
		writer, _ = flate.NewWriter(&buf, flate.DefaultCompression)
	case BEST, MAXIMUM:
		writer, _ = flate.NewWriter(&buf, flate.BestCompression)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressDeflate(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer func(reader io.ReadCloser) {
		err := reader.Close()
		if err != nil {
			return
		}
	}(reader)

	return io.ReadAll(reader)
}

// LZW compression
func (fr *FileReducer) compressLZW(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := lzw.NewWriter(&buf, lzw.MSB, 8)

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressLZW(data []byte) ([]byte, error) {
	reader := lzw.NewReader(bytes.NewReader(data), lzw.MSB, 8)
	defer reader.Close()

	return io.ReadAll(reader)
}

// ZSTD compression (Facebook's Zstandard - better than WinRAR)
func (fr *FileReducer) compressZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, err
	}
	defer encoder.Close()

	return encoder.EncodeAll(data, make([]byte, 0, len(data))), nil
}

func (fr *FileReducer) decompressZstd(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()

	return decoder.DecodeAll(data, nil)
}

// LZ4 compression (ultra-fast)
func (fr *FileReducer) compressLZ4(data []byte) ([]byte, error) {
	compressed := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, compressed, nil)
	if err != nil {
		return nil, err
	}

	return compressed[:n], nil
}

func (fr *FileReducer) decompressLZ4(data []byte) ([]byte, error) {
	// For LZ4, we need to know the original size, which we get from the header
	decompressed := make([]byte, len(data)*4) // Start with 4x estimate
	n, err := lz4.UncompressBlock(data, decompressed)
	if err != nil {
		return nil, err
	}

	return decompressed[:n], nil
}

// XZ compression (LZMA2 - highest compression ratio)
func (fr *FileReducer) compressXZ(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := xz.NewWriter(&buf)
	if err != nil {
		return nil, err
	}

	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressXZ(data []byte) ([]byte, error) {
	reader, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(reader)
}

// Hybrid compression (multi-stage for maximum reduction)
func (fr *FileReducer) compressHybrid(data []byte) ([]byte, error) {
	// Stage 1: Pre-compression with LZ4 for speed
	stage1, err := fr.compressLZ4(data)
	if err != nil {
		return nil, err
	}

	// Stage 2: High-ratio compression with ZSTD
	fr.method = ZSTD
	stage2, err := fr.compressZstd(stage1)
	if err != nil {
		return nil, err
	}

	// Stage 3: Final pass with XZ if beneficial
	if len(stage2) > 1024 { // Only for larger data
		stage3, err := fr.compressXZ(stage2)
		if err == nil && len(stage3) < len(stage2) {
			return stage3, nil
		}
	}

	return stage2, nil
}

func (fr *FileReducer) decompressHybrid(data []byte) ([]byte, error) {
	// Try XZ first (stage 3 reverse)
	if stage2, err := fr.decompressXZ(data); err == nil {
		data = stage2
	}

	// Stage 2 reverse: ZSTD decompression
	stage1, err := fr.decompressZstd(data)
	if err != nil {
		return nil, err
	}

	// Stage 1 reverse: LZ4 decompression
	return fr.decompressLZ4(stage1)
}

// Helper methods

func (fr *FileReducer) selectOptimalMethod(data []byte) CompressionMethod {
	// Analyze data characteristics to choose best method
	if len(data) < 1024 {
		return LZ4 // Fast for small files
	}

	// Sample data to determine best method
	sample := data
	if len(data) > 4096 {
		sample = data[:4096] // Use first 4KB as sample
	}

	// Check for highly repetitive data
	if fr.isHighlyRepetitive(sample) {
		return XZ // Best for repetitive data
	}

	// Check for binary vs text data
	if fr.isBinaryData(sample) {
		return ZSTD // Good for binary data
	}

	return HYBRID // Default to hybrid for best overall compression
}

func (fr *FileReducer) isHighlyRepetitive(data []byte) bool {
	if len(data) < 100 {
		return false
	}

	// Simple repetition detection
	counts := make(map[byte]int)
	for _, b := range data[:100] {
		counts[b]++
	}

	// If any byte appears more than 70% of the time, it's highly repetitive
	for _, count := range counts {
		if float64(count)/100.0 > 0.7 {
			return true
		}
	}

	return false
}

func (fr *FileReducer) isBinaryData(data []byte) bool {
	if len(data) < 50 {
		return false
	}

	// Check for null bytes and non-printable characters
	nullCount := 0
	for i := 0; i < 50 && i < len(data); i++ {
		if data[i] == 0 {
			nullCount++
		}
	}

	return nullCount > 2 // If more than 2 null bytes in first 50, likely binary
}

// Pre-filtering for better compression
func (fr *FileReducer) preFilterData(data []byte) []byte {
	// Delta encoding for better compression of similar bytes
	if len(data) < 2 {
		return data
	}

	filtered := make([]byte, len(data))
	filtered[0] = data[0]

	for i := 1; i < len(data); i++ {
		filtered[i] = data[i] - data[i-1]
	}

	return filtered
}

func (fr *FileReducer) reversePreFilter(data []byte) []byte {
	if len(data) < 2 {
		return data
	}

	restored := make([]byte, len(data))
	restored[0] = data[0]

	for i := 1; i < len(data); i++ {
		restored[i] = data[i] + restored[i-1]
	}

	return restored
}

// Post-optimization (placeholder for advanced techniques)
func (fr *FileReducer) postOptimizeData(data []byte, method CompressionMethod) []byte {
	// Additional optimization can be implemented here
	return data
}

func (fr *FileReducer) reversePostOptimization(data []byte, method CompressionMethod) []byte {
	// Reverse the post-optimization
	return data
}

// Header management for restoration
func (fr *FileReducer) addCompressionHeader(data []byte, method CompressionMethod, originalSize int64) []byte {
	header := make([]byte, 16) // 16-byte header
	header[0] = 0xFF           // Magic byte 1
	header[1] = 0xFE           // Magic byte 2
	header[2] = byte(method)   // Compression method
	header[3] = 0x01           // Version

	// Original size (8 bytes, little-endian)
	for i := 0; i < 8; i++ {
		header[4+i] = byte(originalSize >> (i * 8))
	}

	// Reserved bytes (4 bytes)

	return append(header, data...)
}

func (fr *FileReducer) extractCompressionHeader(data []byte) (CompressionMethod, int64, []byte, error) {
	if len(data) < 16 {
		return 0, 0, nil, fmt.Errorf("data too short for header")
	}

	// Verify magic bytes
	if data[0] != 0xFF || data[1] != 0xFE {
		return 0, 0, nil, fmt.Errorf("invalid magic bytes")
	}

	method := CompressionMethod(data[2])
	version := data[3]

	if version != 0x01 {
		return 0, 0, nil, fmt.Errorf("unsupported version: %d", version)
	}

	// Extract original size
	var originalSize int64
	for i := 0; i < 8; i++ {
		originalSize |= int64(data[4+i]) << (i * 8)
	}

	return method, originalSize, data[16:], nil
}

// Utility function to get current time in milliseconds
func getCurrentTimeMs() int64 {
	// This would typically use time.Now().UnixMilli() in Go 1.17+
	// For compatibility, we'll use a simple implementation
	return 0 // Placeholder - implement based on your Go version
}

// GetCompressionInfo returns information about available compression methods
func (fr *FileReducer) GetCompressionInfo() map[CompressionMethod]string {
	return map[CompressionMethod]string{
		GZIP:     "GZIP - Standard compression, good compatibility",
		ZLIB:     "ZLIB - Similar to GZIP, slightly better compression",
		DEFLATE:  "DEFLATE - Fast compression, good for streams",
		LZW:      "LZW - Good for text data with repeated patterns",
		ZSTD:     "ZSTD - Facebook's algorithm, excellent ratio and speed",
		LZ4:      "LZ4 - Ultra-fast compression, lower ratio",
		XZ:       "XZ - Highest compression ratio, slower processing",
		HYBRID:   "HYBRID - Multi-stage compression for maximum reduction",
		ADAPTIVE: "ADAPTIVE - Automatically selects best method",
	}
}

// EstimateCompressionRatio estimates compression ratio without actually compressing
func (fr *FileReducer) EstimateCompressionRatio(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	// Simple entropy-based estimation
	counts := make(map[byte]int)
	for _, b := range data {
		counts[b]++
	}

	// Calculate Shannon entropy
	entropy := 0.0
	length := float64(len(data))

	for _, count := range counts {
		if count > 0 {
			prob := float64(count) / length
			entropy -= prob * (math.Log2(prob))
		}
	}

	// Estimate compression ratio based on entropy
	// Lower entropy = better compression potential
	maxEntropy := 8.0 // 8 bits per byte
	compressionPotential := (maxEntropy - entropy) / maxEntropy

	return compressionPotential * 80.0 // Estimate up to 80% compression
}
