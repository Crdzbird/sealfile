# SealFile

A lightweight and developer-friendly Go library for encrypting and decrypting files.  
It supports single or multiple files, automatically detects file type by extension (image, video, audio, doc, etc.), and uses a dynamic private key mechanism with shrinking for enhanced security.

---

## Features

- 🔒 **Encrypt/Decrypt** any file or batch of files.
- 📂 **File type detection** by extension (e.g., `.jpg`, `.mp4`, `.mp3`, `.pdf`, etc.).
- 🔑 **Dynamic private keys** with shrinking, ensuring secure key management.
- 🛠️ **Developer-friendly API** with simple function calls.
- ⚡ **Lightweight**: minimal dependencies, pure Go implementation.

---

## Installation

```bash
go get github.com/crdzbird/sealfile
```

---

## Example Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/crdzbird/sealfile"
)

func main() {
	basicEncryptionExample()
	pathConfigurationExample()
	batchProcessingExample()
	utilityFunctionsExample()
}

func basicEncryptionExample() {
	fmt.Println("=== Basic Encryption Example ===")

	// Create configuration
	config := &sealfile.Config{
		EncryptionKey: "my-super-secret-32-byte-key-string",
		BaseURL:       "https://api.example.com/files",
		PublicDir:     "./public",
		TempDir:       "./temp",
		PathType:      sealfile.DirectoryPath,
	}

	// Create file manager
	fm, err := sealfile.NewFileManager(config)
	if err != nil {
		log.Fatal("Failed to create file manager:", err)
	}

	// Sample sensitive data
	sensitiveData := []byte("This is confidential information that needs encryption!")

	// Save encrypted file
	secureFile, err := fm.SaveDataAsSecureFile(sensitiveData, "./secure", "confidential.dat")
	if err != nil {
		log.Fatal("Failed to save secure file:", err)
	}

	fmt.Printf("✓ File encrypted and saved at: %s\n", secureFile.GetFullPath())

	// Load and decrypt file
	loadedFile, err := fm.LoadSecureFileFromDisk("./secure", "confidential.dat")
	if err != nil {
		log.Fatal("Failed to load secure file:", err)
	}

	fmt.Printf("✓ Decrypted content: %s\n", string(loadedFile.Data))

	// Clean up
	if err := loadedFile.Delete(); err != nil {
		fmt.Printf("Warning: Failed to delete file: %v\n", err)
	}
}

func pathConfigurationExample() {
	fmt.Println("\n=== Path Configuration Example ===")

	config := &sealfile.Config{
		EncryptionKey: "path-example-32-byte-key-for-demo",
		BaseURL:       "https://cdn.myapp.com/api/files",
		PublicDir:     "./public",
		TempDir:       "./temp",
		PathType:      sealfile.HTTPPath,
	}

	fm, err := sealfile.NewFileManager(config)
	if err != nil {
		log.Fatal("Failed to create file manager:", err)
	}

	// Create file in subdirectory
	imageData := []byte("fake-image-data-here")
	secureFile := fm.NewSecureFile(imageData, "./public/images/avatars", "user-avatar.jpg")

	if err := secureFile.SaveEncrypted(); err != nil {
		log.Fatal("Failed to save file:", err)
	}

	fmt.Println("Path Examples:")
	fmt.Printf("  HTTP URL:        %s\n", secureFile.GetURL())
	fmt.Printf("  Configured Path: %s\n", secureFile.GetPath())
	fmt.Printf("  Directory Path:  %s\n", secureFile.GetDirectoryPath())
	fmt.Printf("  Full File Path:  %s\n", secureFile.GetFullPath())

	// Change path type dynamically
	config.PathType = sealfile.DirectoryPath
	fmt.Printf("  After changing to DirectoryPath: %s\n", secureFile.GetPath())

	// Clean up
	//secureFile.Delete()
}

func batchProcessingExample() {
	fmt.Println("\n=== Batch Processing Example ===")

	config := sealfile.DefaultConfig()
	config.EncryptionKey = "batch-processing-demo-key-32-bytes"

	fm, err := sealfile.NewFileManager(config)
	if err != nil {
		log.Fatal("Failed to create file manager:", err)
	}

	// Create multiple files
	files := []*sealfile.SecureFile{
		fm.NewSecureFile([]byte("Content of file 1"), "./batch", "file1.txt"),
		fm.NewSecureFile([]byte("Content of file 2"), "./batch", "file2.txt"),
		fm.NewSecureFile([]byte("Content of file 3"), "./batch", "file3.txt"),
		fm.NewSecureFile([]byte("Content of file 4"), "./batch", "file4.txt"),
	}

	// Create batch processor with concurrency of 2
	bp := sealfile.NewBatchProcessor(fm, 2)

	// Save all files concurrently
	fmt.Println("Saving files concurrently...")
	errors := bp.SaveAllFiles(files)

	for i, err := range errors {
		if err != nil {
			fmt.Printf("  ✗ File %d failed: %v\n", i+1, err)
		} else {
			fmt.Printf("  ✓ File %d saved successfully\n", i+1)
		}
	}

	// Load all files concurrently
	fmt.Println("Loading files concurrently...")
	errors = bp.LoadAllFiles(files)

	for i, err := range errors {
		if err != nil {
			fmt.Printf("  ✗ File %d failed to load: %v\n", i+1, err)
		} else {
			fmt.Printf("  ✓ File %d loaded: %s\n", i+1, string(files[i].Data))
		}
	}

	// Clean up all files
	//bp.DeleteAllFiles(files)
}

func utilityFunctionsExample() {
	fmt.Println("\n=== Utility Functions Example ===")

	testFiles := []string{
		"photo.jpg", "video.mp4", "song.mp3", "document.pdf",
		"image.PNG", "movie.AVI", "track.MP3", "text.TXT",
	}

	fmt.Println("File Type Detection:")
	for _, filename := range testFiles {
		fmt.Printf("  %-12s | Image: %-5t | Video: %-5t | Audio: %-5t | Document: %-5t\n",
			filename,
			sealfile.IsImageFile(filename),
			sealfile.IsVideoFile(filename),
			sealfile.IsAudioFile(filename),
			sealfile.IsDocumentFile(filename))
	}

	// Filename utilities
	fmt.Println("\nFilename Utilities:")
	testName := "my-document.pdf"
	fmt.Printf("  Original: %s\n", testName)
	fmt.Printf("  Without extension: %s\n", sealfile.GetFileNameWithoutExtension(testName))
	fmt.Printf("  Extension only: %s\n", sealfile.GetFileExtension(testName))

	// Sanitize filename
	unsafeName := "file/with:invalid*chars?.txt"
	fmt.Printf("  Unsafe name: %s\n", unsafeName)
	fmt.Printf("  Sanitized: %s\n", sealfile.SanitizeFilename(unsafeName))
}
```