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
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/ulikunitz/xz"
)

// CompressionMethod defines the compression algorithm to use
type CompressionMethod int

const (
	// GZIP Standard compression methods
	GZIP CompressionMethod = iota
	// ZLIB Standard compression methods
	ZLIB
	// DEFLATE Standard compression methods
	DEFLATE
	// LZW Standard compression methods
	LZW

	// ZSTD Advanced compression methods (better than WinRAR)
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
	startTime := time.Now().UnixMilli()

	// Pre-filtering for better compression (removes redundancy)
	processedData := data
	if fr.enablePreFilter {
		processedData = fr.preFilterData(data)
	}

	var compressed []byte
	var err error
	var method = fr.method

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

	endTime := time.Now().UnixMilli()
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
		decompressed, err = fr.decompressLZ4(dataWithoutHeader, originalSize)
	case XZ:
		decompressed, err = fr.decompressXZ(dataWithoutHeader)
	case HYBRID:
		decompressed, err = fr.decompressHybrid(dataWithoutHeader, originalSize)
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
	var err error
	switch fr.level {
	case FASTEST:
		writer, err = gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	case FAST:
		writer, err = gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	case BALANCED:
		writer, err = gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	case BEST, MAXIMUM:
		writer, err = gzip.NewWriterLevel(&buf, gzip.BestCompression)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write gzip data: %w", err)
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			// Log close error but don't override main error
		}
	}()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read gzip data: %w", err)
	}
	return result, nil
}

// ZLIB compression
func (fr *FileReducer) compressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var writer *zlib.Writer
	var err error
	switch fr.level {
	case FASTEST:
		writer, err = zlib.NewWriterLevel(&buf, zlib.BestSpeed)
	case FAST:
		writer, err = zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	case BALANCED:
		writer, err = zlib.NewWriterLevel(&buf, zlib.DefaultCompression)
	case BEST, MAXIMUM:
		writer, err = zlib.NewWriterLevel(&buf, zlib.BestCompression)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create zlib writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write zlib data: %w", err)
	}
	if writer != nil {
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close zlib writer: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			// Log close error but don't override main error
		}
	}()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read zlib data: %w", err)
	}
	return result, nil
}

// DEFLATE compression
func (fr *FileReducer) compressDeflate(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	var writer *flate.Writer
	var err error
	switch fr.level {
	case FASTEST:
		writer, err = flate.NewWriter(&buf, flate.BestSpeed)
	case FAST:
		writer, err = flate.NewWriter(&buf, flate.DefaultCompression)
	case BALANCED:
		writer, err = flate.NewWriter(&buf, flate.DefaultCompression)
	case BEST, MAXIMUM:
		writer, err = flate.NewWriter(&buf, flate.BestCompression)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create deflate writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write deflate data: %w", err)
	}

	if writer != nil {
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close deflate writer: %w", err)
		}
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressDeflate(data []byte) ([]byte, error) {
	reader := flate.NewReader(bytes.NewReader(data))
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			return
		}
	}()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read deflate data: %w", err)
	}
	return result, nil
}

// LZW compression
func (fr *FileReducer) compressLZW(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := lzw.NewWriter(&buf, lzw.MSB, 8)

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write LZW data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close LZW writer: %w", err)
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressLZW(data []byte) ([]byte, error) {
	reader := lzw.NewReader(bytes.NewReader(data), lzw.MSB, 8)
	defer func() {
		if closeErr := reader.Close(); closeErr != nil {
			return
		}
	}()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read LZW data: %w", err)
	}
	return result, nil
}

// ZSTD compression (Facebook's Zstandard - better than WinRAR)
func (fr *FileReducer) compressZstd(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, fmt.Errorf("failed to create ZSTD encoder: %w", err)
	}
	defer func(encoder *zstd.Encoder) {
		err := encoder.Close()
		if err != nil {
			return
		}
	}(encoder)

	return encoder.EncodeAll(data, make([]byte, 0, len(data))), nil
}

func (fr *FileReducer) decompressZstd(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ZSTD decoder: %w", err)
	}
	defer decoder.Close()

	result, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decode ZSTD data: %w", err)
	}
	return result, nil
}

// LZ4 compression (ultra-fast)
func (fr *FileReducer) compressLZ4(data []byte) ([]byte, error) {
	compressed := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, compressed, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to compress LZ4 data: %w", err)
	}

	return compressed[:n], nil
}

func (fr *FileReducer) decompressLZ4(data []byte, originalSize int64) ([]byte, error) {
	decompressed := make([]byte, originalSize)
	n, err := lz4.UncompressBlock(data, decompressed)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress LZ4 data: %w", err)
	}

	return decompressed[:n], nil
}

// XZ compression (LZMA2 - highest compression ratio)
func (fr *FileReducer) compressXZ(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := xz.NewWriter(&buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create XZ writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write XZ data: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close XZ writer: %w", err)
	}

	return buf.Bytes(), nil
}

func (fr *FileReducer) decompressXZ(data []byte) ([]byte, error) {
	reader, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create XZ reader: %w", err)
	}

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read XZ data: %w", err)
	}
	return result, nil
}

// Hybrid compression (multi-stage for maximum reduction)
func (fr *FileReducer) compressHybrid(data []byte) ([]byte, error) {
	// Check if data is already compressed
	if fr.isAlreadyCompressed(data) {
		// For already compressed data, use a different strategy
		return fr.compressAlreadyCompressedData(data)
	}

	// Original hybrid approach for uncompressed data
	// Stage 1: Pre-compression with LZ4 for speed
	stage1, err := fr.compressLZ4(data)
	if err != nil {
		return nil, fmt.Errorf("hybrid stage 1 failed: %w", err)
	}

	// Stage 2: High-ratio compression with ZSTD
	stage2, err := fr.compressZstd(stage1)
	if err != nil {
		return nil, fmt.Errorf("hybrid stage 2 failed: %w", err)
	}

	// Stage 3: Final pass with XZ if beneficial
	if len(stage2) > 1024 {
		stage3, err := fr.compressXZ(stage2)
		if err == nil && len(stage3) < len(stage2) {
			return stage3, nil
		}
	}

	return stage2, nil
}

func (fr *FileReducer) decompressHybrid(data []byte, originalSize int64) ([]byte, error) {
	// Try XZ first (stage 3 reverse)
	if stage2, err := fr.decompressXZ(data); err == nil {
		data = stage2
	}

	// Stage 2 reverse: ZSTD decompression
	stage1, err := fr.decompressZstd(data)
	if err != nil {
		return nil, fmt.Errorf("hybrid decompression stage 2 failed: %w", err)
	}

	// Stage 1 reverse: LZ4 decompression
	result, err := fr.decompressLZ4(stage1, originalSize)
	if err != nil {
		return nil, fmt.Errorf("hybrid decompression stage 1 failed: %w", err)
	}

	return result, nil
}

// Helper methods

func (fr *FileReducer) selectOptimalMethod(data []byte) CompressionMethod {
	// Analyze data characteristics to choose best method
	if len(data) < 1024 {
		return LZ4 // Fast for small files
	}

	// Check if data is already compressed (like ZIP, PPTX, DOCX, etc.)
	if fr.isAlreadyCompressed(data) {
		return LZ4 // Use fast compression for already compressed data
	}

	// Sample data to determine best method
	sample := data
	if len(data) > 8192 {
		sample = data[:8192] // Use first 8KB as sample for better analysis
	}

	// Check for highly repetitive data
	if fr.isHighlyRepetitive(sample) {
		return XZ // Best for repetitive data
	}

	// Check for binary vs text data
	if fr.isBinaryData(sample) {
		// For binary data, try ZSTD first
		return ZSTD
	}

	return HYBRID // Default to hybrid for best overall compression
}

// isAlreadyCompressed detects if data is already compressed (ZIP, GZIP, etc.)
func (fr *FileReducer) isAlreadyCompressed(data []byte) bool {
	if len(data) < 4 {
		return false
	}

	// Check common compressed file signatures
	signatures := [][]byte{
		// Archive formats
		{0x50, 0x4B, 0x03, 0x04},                   // ZIP/PPTX/DOCX/XLSX (PK..)
		{0x50, 0x4B, 0x05, 0x06},                   // ZIP empty archive
		{0x50, 0x4B, 0x07, 0x08},                   // ZIP spanned archive
		{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x00}, // RAR archive
		{0x52, 0x61, 0x72, 0x21, 0x1A, 0x07, 0x01}, // RAR archive v5
		{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C},       // 7-Zip
		{0x1F, 0x9D},                               // compress (.Z)
		{0x1F, 0xA0},                               // compress (.Z)
		{0x42, 0x5A, 0x68},                         // BZIP2
		{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},       // XZ
		{0x28, 0xB5, 0x2F, 0xFD},                   // ZSTD
		{0x04, 0x22, 0x4D, 0x18},                   // LZ4
		{0x1F, 0x8B, 0x08},                         // GZIP
		{0x78, 0x01},                               // ZLIB (best compression)
		{0x78, 0x9C},                               // ZLIB (default compression)
		{0x78, 0xDA},                               // ZLIB (best compression)
		{0x78, 0x5E},                               // ZLIB (fast compression)
		{0x60, 0xEA},                               // ARJ compressed archive
		{0x4C, 0x5A, 0x49, 0x50},                   // LZIP
		{0x4D, 0x5A, 0x90, 0x00, 0x03, 0x00},       // Windows PE (compressed)

		// Image formats (already compressed)
		{0xFF, 0xD8, 0xFF, 0xE0},                         // JPEG/JFIF
		{0xFF, 0xD8, 0xFF, 0xE1},                         // JPEG/EXIF
		{0xFF, 0xD8, 0xFF, 0xE2},                         // JPEG
		{0xFF, 0xD8, 0xFF, 0xE3},                         // JPEG
		{0xFF, 0xD8, 0xFF, 0xE8},                         // JPEG/SPIFF
		{0xFF, 0xD8, 0xFF, 0xDB},                         // JPEG
		{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, // PNG
		{0x47, 0x49, 0x46, 0x38, 0x37, 0x61},             // GIF87a
		{0x47, 0x49, 0x46, 0x38, 0x39, 0x61},             // GIF89a
		{0x42, 0x4D},                                     // BMP (can be compressed)
		{0x00, 0x00, 0x01, 0x00},                         // ICO (can be compressed)
		{0x00, 0x00, 0x02, 0x00},                         // CUR (can be compressed)
		{0x49, 0x49, 0x2A, 0x00},                         // TIFF (little endian, can be compressed)
		{0x4D, 0x4D, 0x00, 0x2A},                         // TIFF (big endian, can be compressed)
		{0x52, 0x49, 0x46, 0x46},                         // WEBP (RIFF container, check for WEBP)
		{0x38, 0x42, 0x50, 0x53},                         // PSD (Photoshop, can be compressed)

		// Video formats (heavily compressed)
		{0x00, 0x00, 0x00, 0x14, 0x66, 0x74, 0x79, 0x70}, // MP4 (14-byte header + ftyp)
		{0x00, 0x00, 0x00, 0x18, 0x66, 0x74, 0x79, 0x70}, // MP4 (24-byte header + ftyp)
		{0x00, 0x00, 0x00, 0x1C, 0x66, 0x74, 0x79, 0x70}, // MP4 (28-byte header + ftyp)
		{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70}, // MP4 (32-byte header + ftyp)
		{0x66, 0x74, 0x79, 0x70},                         // MP4 (ftyp at start)
		{0x1A, 0x45, 0xDF, 0xA3},                         // MKV/WEBM
		{0x52, 0x49, 0x46, 0x46},                         // AVI (RIFF format)
		{0x30, 0x26, 0xB2, 0x75, 0x8E, 0x66, 0xCF},       // WMV/ASF
		{0x46, 0x4C, 0x56, 0x01},                         // FLV
		{0x00, 0x00, 0x01, 0xBA},                         // MPEG
		{0x00, 0x00, 0x01, 0xB3},                         // MPEG video
		{0x47},                                           // MPEG-TS (Transport Stream)

		// Audio formats (compressed)
		{0xFF, 0xFB},       // MP3 (MPEG-1 Layer 3)
		{0xFF, 0xF3},       // MP3 (MPEG-1 Layer 3)
		{0xFF, 0xF2},       // MP3 (MPEG-1 Layer 3)
		{0x49, 0x44, 0x33}, // MP3 with ID3v2
		{0x66, 0x4C, 0x61, 0x43, 0x00, 0x00, 0x00, 0x22},                   // FLAC
		{0x4F, 0x67, 0x67, 0x53},                                           // OGG (Ogg Vorbis)
		{0x52, 0x49, 0x46, 0x46},                                           // WAV (RIFF format, can be compressed)
		{0x4D, 0x54, 0x68, 0x64},                                           // MIDI
		{0x2E, 0x52, 0x4D, 0x46},                                           // RealMedia
		{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x4D, 0x34, 0x41}, // M4A

		// Document formats (often compressed internally)
		{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, // Microsoft Office (old format)
		{0x25, 0x50, 0x44, 0x46, 0x2D},                   // PDF
		{0x7B, 0x5C, 0x72, 0x74, 0x66, 0x31},             // RTF
		{0xEF, 0xBB, 0xBF},                               // UTF-8 BOM (text files can be compressed)
		{0xFF, 0xFE},                                     // UTF-16 LE BOM
		{0xFE, 0xFF},                                     // UTF-16 BE BOM
		{0x00, 0x00, 0xFE, 0xFF},                         // UTF-32 BE BOM
		{0xFF, 0xFE, 0x00, 0x00},                         // UTF-32 LE BOM

		// Executable formats (can be compressed)
		{0x4D, 0x5A},             // Windows PE/EXE
		{0x7F, 0x45, 0x4C, 0x46}, // ELF (Linux executable)
		{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O (macOS 32-bit)
		{0xFE, 0xED, 0xFA, 0xCF}, // Mach-O (macOS 64-bit)
		{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O (reverse byte order)
		{0xCF, 0xFA, 0xED, 0xFE}, // Mach-O (reverse byte order)
		{0xCA, 0xFE, 0xBA, 0xBE}, // Java class file

		// Database formats (often compressed)
		{0x53, 0x51, 0x4C, 0x69, 0x74, 0x65, 0x20, 0x66}, // SQLite
		{0x00, 0x01, 0x00, 0x00, 0x53, 0x74, 0x61, 0x6E}, // Access DB

		// Font formats (can be compressed)
		{0x00, 0x01, 0x00, 0x00, 0x00}, // TTF
		{0x4F, 0x54, 0x54, 0x4F, 0x00}, // OTF
		{0x77, 0x4F, 0x46, 0x46},       // WOFF
		{0x77, 0x4F, 0x46, 0x32},       // WOFF2

		// CAD formats (often compressed)
		{0x41, 0x43, 0x31, 0x30},             // AutoCAD DWG
		{0x41, 0x43, 0x31, 0x30, 0x31, 0x32}, // AutoCAD DWG

		// Virtual disk formats (compressed)
		{0x56, 0x4D, 0x44, 0x4B},                         // VMDK
		{0x51, 0x46, 0x49, 0xFB, 0x00, 0x00, 0x00},       // QCOW2
		{0x63, 0x6F, 0x6E, 0x65, 0x63, 0x74, 0x69, 0x78}, // VHD

		// Apple formats
		{0x62, 0x6F, 0x6F, 0x6B, 0x00, 0x00, 0x00, 0x00}, // Apple Alias
		{0x62, 0x70, 0x6C, 0x69, 0x73, 0x74, 0x30, 0x30}, // Apple Binary Plist

		// Game/3D formats (often compressed)
		{0x89, 0x48, 0x44, 0x46, 0x0D, 0x0A, 0x1A, 0x0A}, // HDF5
		{0x42, 0x4C, 0x45, 0x4E, 0x44, 0x45, 0x52},       // Blender

		// Disk image formats
		{0x45, 0x52, 0x02, 0x00, 0x00, 0x00},       // Toast disk image
		{0x8B, 0x45, 0x52, 0x02, 0x00, 0x00, 0x00}, // Toast disk image
		{0x78, 0x01, 0x73, 0x0D, 0x62, 0x62, 0x60}, // Apple Disk Image (compressed)

		// Backup formats (compressed)
		{0x42, 0x41, 0x43, 0x4B, 0x4D, 0x49, 0x44, 0x42}, // BackupBuddy
		{0x53, 0x50, 0x46, 0x49},                         // Sphinx backup

		// Scientific data formats (often compressed)
		{0x43, 0x44, 0x46, 0x01}, // NetCDF
		{0x43, 0x44, 0x46, 0x02}, // NetCDF v2
		{0x89, 0x48, 0x44, 0x46}, // HDF4/HDF5

		// Web formats (compressed)
		{0x1F, 0x8B}, // GZIP (web compression)
		{0x42, 0x52}, // Brotli

		// Cryptocurrency/blockchain
		{0xF9, 0xBE, 0xB4, 0xD9}, // Bitcoin block

		// Nintendo formats
		{0x4E, 0x45, 0x53, 0x1A}, // NES ROM

		// Android formats
		{0x64, 0x65, 0x78, 0x0A, 0x30, 0x33, 0x35, 0x00}, // Android DEX

		// Windows formats
		{0x52, 0x65, 0x67, 0x45, 0x64, 0x69, 0x74}, // Windows Registry
		{0x43, 0x57, 0x53},                         // Shockwave Flash (compressed)
		{0x46, 0x57, 0x53},                         // Shockwave Flash (uncompressed)
		{0x5A, 0x57, 0x53},                         // Shockwave Flash (LZMA compressed)
	}

	for _, sig := range signatures {
		if len(data) >= len(sig) {
			match := true
			for i, b := range sig {
				if data[i] != b {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}

	// Check entropy - high entropy suggests already compressed data
	entropy := fr.calculateEntropy(data[:mins(4096, len(data))])
	// If entropy is very high (> 7.5 out of 8), likely already compressed
	return entropy > 7.5
}

// calculateEntropy calculates Shannon entropy of data
func (fr *FileReducer) calculateEntropy(data []byte) float64 {
	if len(data) == 0 {
		return 0.0
	}

	// Count frequency of each byte
	freq := make([]int, 256)
	for _, b := range data {
		freq[b]++
	}

	// Calculate entropy
	entropy := 0.0
	length := float64(len(data))

	for _, count := range freq {
		if count > 0 {
			prob := float64(count) / length
			entropy -= prob * math.Log2(prob)
		}
	}

	return entropy
}

// compressAlreadyCompressedData uses specialized techniques for compressed data
func (fr *FileReducer) compressAlreadyCompressedData(data []byte) ([]byte, error) {
	// Strategy 1: Try to find patterns in the compressed data
	processed := fr.applyAdvancedPreprocessing(data)

	// Strategy 2: Use LZ4 for minimal overhead
	lz4Result, err := fr.compressLZ4(processed)
	if err == nil {
		// If LZ4 provides any benefit, use it
		if len(lz4Result) < len(data) {
			return lz4Result, nil
		}
	}

	// Strategy 3: Try ZSTD with lowest compression level for speed
	zstdResult, err := fr.compressZstdFast(processed)
	if err == nil {
		if len(zstdResult) < len(data) {
			return zstdResult, nil
		}
	}

	// If no compression helps, return original data with minimal header
	return data, nil
}

// compressZstdFast uses ZSTD with fastest settings
func (fr *FileReducer) compressZstdFast(data []byte) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedFastest))
	if err != nil {
		return nil, err
	}
	defer func(encoder *zstd.Encoder) {
		err := encoder.Close()
		if err != nil {
			return
		}
	}(encoder)

	return encoder.EncodeAll(data, make([]byte, 0, len(data))), nil
}

// applyAdvancedPreprocessing applies advanced preprocessing for compressed data
func (fr *FileReducer) applyAdvancedPreprocessing(data []byte) []byte {
	if len(data) < 16 {
		return data
	}

	// Strategy 1: Look for repeating sequences in compressed data
	processed := fr.findAndReplaceSequences(data)

	// Strategy 2: Byte frequency reordering
	if len(processed) > 256 {
		processed = fr.applyBurrowsWheelerTransform(processed)
	}

	return processed
}

// findAndReplaceSequences finds repeating byte sequences
func (fr *FileReducer) findAndReplaceSequences(data []byte) []byte {
	if len(data) < 32 {
		return data
	}

	// Simple implementation - look for 4-byte sequences that repeat
	sequences := make(map[string][]int)

	// Find all 4-byte sequences and their positions
	for i := 0; i <= len(data)-4; i++ {
		seq := string(data[i : i+4])
		sequences[seq] = append(sequences[seq], i)
	}

	// If no sequences repeat enough, return original
	maxRepeats := 0
	for _, positions := range sequences {
		if len(positions) > maxRepeats {
			maxRepeats = len(positions)
		}
	}

	if maxRepeats < 3 {
		return data
	}

	// For now, return original (full implementation would be more complex)
	return data
}

// applyBurrowsWheelerTransform applies BWT for better compressibility
func (fr *FileReducer) applyBurrowsWheelerTransform(data []byte) []byte {
	// Simplified BWT - in practice, this would be a full implementation
	// For now, just return the data (placeholder)
	return data
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
	entropy := fr.calculateEntropy(data)

	// Estimate compression ratio based on entropy
	// Lower entropy = better compression potential
	maxEntropy := 8.0 // 8 bits per byte
	compressionPotential := (maxEntropy - entropy) / maxEntropy

	return compressionPotential * 80.0 // Estimate up to 80% compression
}

func mins(a, b int) int {
	if a < b {
		return a
	}
	return b
}
