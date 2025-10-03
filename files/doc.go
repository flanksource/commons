// Package files provides utilities for file operations, archive handling,
// and path manipulation.
//
// The package offers comprehensive support for working with various archive
// formats (tar, zip, gzip, xz), file path validation, glob pattern matching,
// and common file operations.
//
// Key Features:
//   - Archive extraction with detailed results and statistics
//   - Support for tar, tar.gz, tar.xz, and zip formats
//   - Path traversal vulnerability protection
//   - Glob pattern matching with doublestar support
//   - File compression and decompression
//   - Safe file operations with validation
//
// Archive Extraction:
//
//	// Simple extraction
//	err := files.UnarchiveSimple("package.tar.gz", "/dest")
//
//	// Extraction with detailed results
//	result, err := files.Unarchive("package.tar.gz", "/dest")
//	fmt.Printf("Extracted %d files (%s)\n", len(result.Files), result.String())
//
//	// Extraction with options
//	result, err := files.Unarchive("archive.zip", "/dest",
//		files.WithOverwrite(true))
//
//	// Extract only executables
//	err := files.UnarchiveExecutables("binaries.tar.gz", "/usr/local/bin")
//
// Compression:
//
//	// Gzip a file
//	data, err := files.GzipFile("document.txt")
//
//	// Create a zip archive
//	err := files.Zip("/source/dir", "archive.zip")
//
//	// Decompress xz
//	err := files.Unxz("file.xz", "file.txt")
//
//	// Decompress gzip
//	err := files.Ungzip("file.gz", "file.txt")
//
// Path Operations:
//
//	// Validate path (check for traversal attacks)
//	err := files.ValidatePath("../../etc/passwd") // Returns error
//
//	// Check file type by extension
//	if files.IsValidPathType("config.yaml", ".yaml", ".yml") {
//		// Process YAML file
//	}
//
//	// Get base name without extension
//	name := files.GetBaseName("config.yaml") // "config"
//
// Glob Matching:
//
//	// Expand glob patterns
//	matches, err := files.UnfoldGlobs("**/*.go", "**/*.yaml")
//
//	// Match with doublestar patterns
//	files, err := files.DoubleStarGlob("/project", []string{"**/*.go"})
//
// File Operations:
//
//	// Check if file exists
//	if files.Exists("/path/to/file") {
//		// File exists
//	}
//
//	// Safe read (returns empty string on error)
//	content := files.SafeRead("/path/to/file")
//
//	// Copy file
//	err := files.Copy("/source/file", "/dest/file")
//
//	// Copy from reader with permissions
//	n, err := files.CopyFromReader(reader, "/dest/file", 0644)
//
//	// Create temp file with custom name
//	tmpFile := files.TempFileName("prefix-", ".txt")
//
// Archive Results:
//
// The Archive type provides detailed information about extraction operations:
//
//	result, _ := files.Unarchive("package.tar.gz", "/dest")
//	fmt.Printf("Files: %d\n", len(result.Files))
//	fmt.Printf("Directories: %d\n", len(result.Directories))
//	fmt.Printf("Symlinks: %d\n", len(result.Symlinks))
//	fmt.Printf("Size: %s -> %s (%.1f%% compression)\n",
//		formatSize(result.CompressedSize),
//		formatSize(result.ExtractedSize),
//		result.CompressionRatio*100)
//
// Security:
//
// The package includes protection against path traversal vulnerabilities
// when extracting archives. All paths are validated to ensure they don't
// escape the destination directory.
package files
