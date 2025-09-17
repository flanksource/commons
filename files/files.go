package files

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"unicode"

	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/flanksource/commons/logger"
	"github.com/ulikunitz/xz"
)

// Archive represents the result of an archive extraction operation
type Archive struct {
	Source      string   // Path to source archive
	Destination string   // Extraction destination
	Files       []string // Successfully extracted files
	Directories []string // Created directories
	Symlinks    []string // Created symlinks
	Skipped     []string // Skipped entries
	Errors      []error  // Non-fatal errors encountered
	Overwritten []string // Files that were overwritten during extraction

	// Size metrics
	CompressedSize   int64   // Size of the archive file
	ExtractedSize    int64   // Total size of extracted content
	CompressionRatio float64 // Ratio of extracted to compressed size
}

// UnarchiveOptions configures archive extraction behavior
type UnarchiveOptions struct {
	Overwrite bool // Allow overwriting existing files
}

// UnarchiveOption is a functional option for configuring archive extraction
type UnarchiveOption func(*UnarchiveOptions)

// WithOverwrite sets whether existing files should be overwritten during extraction
func WithOverwrite(overwrite bool) UnarchiveOption {
	return func(opts *UnarchiveOptions) {
		opts.Overwrite = overwrite
	}
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// String returns a human-readable summary of the archive extraction
func (a *Archive) String() string {
	var parts []string

	// Main extraction info
	if len(a.Files) > 0 || a.ExtractedSize > 0 {
		if a.ExtractedSize > 0 {
			parts = append(parts, fmt.Sprintf("Extracted %d files (%s)", len(a.Files), formatBytes(a.ExtractedSize)))
		} else {
			parts = append(parts, fmt.Sprintf("Extracted %d files", len(a.Files)))
		}
	}

	if len(a.Directories) > 0 {
		parts = append(parts, fmt.Sprintf("%d directories", len(a.Directories)))
	}

	if len(a.Symlinks) > 0 {
		parts = append(parts, fmt.Sprintf("%d symlinks", len(a.Symlinks)))
	}

	// Source info
	sourcePart := fmt.Sprintf("from %s", filepath.Base(a.Source))
	if a.CompressedSize > 0 {
		sourcePart += fmt.Sprintf(" (%s)", formatBytes(a.CompressedSize))
	}
	parts = append(parts, sourcePart)

	// Destination
	parts = append(parts, fmt.Sprintf("to %s", a.Destination))

	result := strings.Join(parts, ", ")

	// Add compression ratio if available
	if a.CompressionRatio > 1 {
		result += fmt.Sprintf("\nCompression ratio: %.2f:1", a.CompressionRatio)
	}

	// Add issues if any
	var issues []string
	if len(a.Skipped) > 0 {
		issues = append(issues, fmt.Sprintf("%d skipped", len(a.Skipped)))
	}
	if len(a.Errors) > 0 {
		issues = append(issues, fmt.Sprintf("%d errors", len(a.Errors)))
	}
	if len(issues) > 0 {
		result += fmt.Sprintf(" (%s)", strings.Join(issues, ", "))
	}

	return result
}

var blacklistedPathSymbols = "${}[]?*:<>|"
var blockedPrefixes = []string{"/run/", "/proc/", "/etc/", "/var/", "/tmp/", "/dev/"}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {

		if s[i] > unicode.MaxASCII || unicode.IsControl(rune(s[i])) {
			return false
		}
	}
	return true

}

// ValidatePath validates a single path for security issues
func ValidatePath(path string) error {

	// Check for non-ASCII characters
	if !isASCII(path) {
		return fmt.Errorf("path %s contains non-ASCII characters", path)
	}

	// Check for illegal characters
	if strings.ContainsAny(path, blacklistedPathSymbols) {
		return fmt.Errorf("path %s contains illegal characters", path)
	}

	// Check for path traversal attempts

	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return fmt.Errorf("path %s attempts to access parent directories which is not allowed", path)
	}

	cleanPath := filepath.Clean(path)
	// Check for blocked prefixes
	for _, prefix := range blockedPrefixes {
		if strings.HasPrefix(cleanPath, prefix) {
			return fmt.Errorf("path %s contains a blocked prefix: %s", path, prefix)
		}
	}

	return nil
}

func DoubleStarGlob(root string, paths []string) ([]string, error) {
	unfoldedPaths := []string{}

	for _, path := range paths {
		if !strings.HasPrefix(path, root) {
			path = filepath.Join(root, path)
		}
		if err := ValidatePath(path); err != nil {
			return nil, err
		}
		matched, err := doublestar.FilepathGlob(path)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern. path=%s; %w", path, err)
		}

		if len(matched) == 0 {
			// Absolute path, not a glob
			matched = append(matched, path)
		}

		unfoldedPaths = append(unfoldedPaths, matched...)
	}

	return unfoldedPaths, nil
}

// Deprecated: use DoubleStarGlob instead
func UnfoldGlobs(paths ...string) ([]string, error) {
	unfoldedPaths := make([]string, 0, len(paths))

	for _, path := range paths {
		matched, err := filepath.Glob(path)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern. path=%s; %w", path, err)
		}

		if len(matched) == 0 {
			// Absolute path, not a glob
			matched = append(matched, path)
		}

		unfoldedPaths = append(unfoldedPaths, matched...)
	}

	return unfoldedPaths, nil
}

// GzipFile takes the path to a file and returns a Gzip comppressed byte slice
func GzipFile(path string) ([]byte, error) {
	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(contents)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	result := buf.Bytes()
	return result, nil
}

// UnarchiveSimple extracts the contents of an archive to the dest directory (returns only error for backwards compatibility)
func UnarchiveSimple(src, dest string) error {
	_, err := Unarchive(src, dest)
	return err
}

// Unarchive extracts an archive and returns detailed results with optional configuration
func Unarchive(src, dest string, options ...UnarchiveOption) (*Archive, error) {
	// Apply default options
	opts := &UnarchiveOptions{
		Overwrite: false, // Default: don't overwrite existing files
	}
	for _, option := range options {
		option(opts)
	}

	logger.Debugf("Unarchiving %s to %s (overwrite=%v)", src, dest, opts.Overwrite)
	if strings.HasSuffix(src, ".zip") || strings.HasSuffix(src, ".jar") {
		return unzipWithResult(src, dest, opts)
	} else if strings.HasSuffix(src, ".tar") || strings.HasSuffix(src, ".tgz") || strings.HasSuffix(src, ".tar.gz") || strings.HasSuffix(src, ".tar.xz") || strings.HasSuffix(src, ".txz") {
		return UntarWithFilterAndResult(src, dest, nil, opts)
	}

	return nil, fmt.Errorf("unknown format type %s", src)
}

// UnarchiveWithResult is deprecated, use Unarchive instead
func UnarchiveWithResult(src, dest string) (*Archive, error) {
	return Unarchive(src, dest)
}

// UnarchiveExecutables extracts all executable's to the dest directory, ignoring any path's specified by the archive
func UnarchiveExecutables(src, dest string) error {
	logger.Debugf("Unarchiving %s to %s", src, dest)
	if strings.HasSuffix(src, ".zip") {
		return Unzip(src, dest)
	} else if strings.HasSuffix(src, ".tar") || strings.HasSuffix(src, ".tgz") || strings.HasSuffix(src, ".tar.gz") || strings.HasSuffix(src, ".tar.xz") || strings.HasSuffix(src, ".txz") {
		return UntarWithFilter(src, dest, func(header os.FileInfo) string {
			if fmt.Sprintf("%v", header.Mode()&0100) != "---x------" {
				return ""
			}
			return path.Base(header.Name())
		})
	} else if strings.HasSuffix(src, ".xz") {
		return Unxz(src, dest)
	}

	return fmt.Errorf("unknown format type %s", src)
}

func Unxz(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// decompress buffer and write output to stdout
	r, err := xz.NewReader(reader)
	if err != nil {
		return err
	}
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()
	if _, err = io.Copy(writer, r); err != nil {
		return err
	}
	return nil
}

// Ungzip the source file to the target directory
func Ungzip(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	target = filepath.Join(target, archive.Name)
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	return err
}

// FileFilter is a function used for filtering files
type FileFilter func(header os.FileInfo) string

// unzipWithResult extracts zip archive and returns detailed results
func unzipWithResult(src, dest string, opts *UnarchiveOptions) (*Archive, error) {
	absSrc, err := filepath.Abs(src)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination path: %w", err)
	}

	// Initialize archive result
	archive := &Archive{
		Source:      absSrc,
		Destination: absDest,
		Files:       make([]string, 0),
		Directories: make([]string, 0),
		Symlinks:    make([]string, 0),
		Skipped:     make([]string, 0),
		Errors:      make([]error, 0),
		Overwritten: make([]string, 0),
	}

	// Get compressed size from file stat
	if stat, err := os.Stat(src); err == nil {
		archive.CompressedSize = stat.Size()
	}

	logger.V(3).Infof("Unzip: starting extraction of %s to %s", absSrc, absDest)

	r, err := zip.OpenReader(src)
	if err != nil {
		return archive, err
	}
	defer r.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return archive, fmt.Errorf("failed to create target directory %s: %w", absDest, err)
	}

	// Open the target directory as a root for secure file operations
	root, err := os.OpenRoot(dest)
	if err != nil {
		return archive, fmt.Errorf("failed to open target directory as root %s: %w", absDest, err)
	}
	defer root.Close()

	for _, f := range r.File {
		if err := ValidatePath(f.Name); err != nil {
			return archive, err
		}

		rc, err := f.Open()
		if err != nil {
			return archive, err
		}

		path := f.Name
		info := f.FileInfo()

		if info.IsDir() {
			_ = rc.Close()
			dirMode := info.Mode() & os.ModePerm
			if dirMode == 0 {
				dirMode = 0755
			}
			if err := root.MkdirAll(path, dirMode); err != nil {
				return archive, fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			archive.Directories = append(archive.Directories, path)
			continue
		}

		// Handle symlinks if present in zip
		if info.Mode()&os.ModeSymlink != 0 {
			linkData, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return archive, fmt.Errorf("failed to read symlink target for %s: %w", path, err)
			}
			linkTarget := string(linkData)

			// Validate symlink target - only allow relative paths within extraction dir
			if filepath.IsAbs(linkTarget) || !filepath.IsLocal(linkTarget) {
				return archive, fmt.Errorf("symlink target %s is absolute and not allowed", linkTarget)
			}

			// Create parent directory
			parent := filepath.Dir(path)
			if parent != "." && parent != "" {
				if err := root.MkdirAll(parent, 0755); err != nil {
					return archive, fmt.Errorf("failed to create directory %s: %w", parent, err)
				}
			}

			if err := root.Symlink(linkTarget, path); err != nil {
				return archive, fmt.Errorf("failed to create symlink %s -> %s: %w", path, linkTarget, err)
			}
			archive.Symlinks = append(archive.Symlinks, path)
			continue
		}

		// Regular file
		// Create parent directory
		parent := filepath.Dir(path)
		if parent != "." && parent != "" {
			if err := root.MkdirAll(parent, 0755); err != nil {
				_ = rc.Close()
				return archive, fmt.Errorf("failed to create directory %s: %w", parent, err)
			}
		}

		// Check for existing file
		fileExists := false
		if _, err := root.Stat(path); err == nil {
			fileExists = true
			if !opts.Overwrite {
				_ = rc.Close()
				return archive, fmt.Errorf("file %s already exists", path)
			}
			archive.Overwritten = append(archive.Overwritten, path)
		}

		// Open file with appropriate flags
		flags := os.O_CREATE | os.O_RDWR
		if fileExists && opts.Overwrite {
			flags |= os.O_TRUNC
		}
		file, err := root.OpenFile(path, flags, info.Mode())
		if err != nil {
			_ = rc.Close()
			return archive, fmt.Errorf("failed to create file %s: %w", path, err)
		}

		bytesWritten, err := io.Copy(file, rc)
		_ = file.Close()
		_ = rc.Close()
		if err != nil {
			return archive, fmt.Errorf("failed to write file %s: %w", path, err)
		}

		archive.Files = append(archive.Files, path)
		archive.ExtractedSize += bytesWritten
	}

	// Calculate compression ratio
	if archive.CompressedSize > 0 && archive.ExtractedSize > 0 {
		archive.CompressionRatio = float64(archive.ExtractedSize) / float64(archive.CompressedSize)
	}

	logger.V(3).Infof("Unzip: extraction complete for %s", archive)
	return archive, nil
}

// Unzip the source file to the target directory (backwards compatibility wrapper)
func Unzip(src, dest string) error {
	_, err := Unarchive(src, dest)
	return err
}


// Untar extracts all files in tarball to the target directory (backwards compatibility wrapper)
func Untar(tarball, target string) error {
	_, err := Unarchive(tarball, target)
	return err
}

// UntarWithFilter extracts all files in tarball to the target directory, passing each file to filter
// if the filter returns "" then the file is ignored, otherwise the return string is used as the relative
// destination path
func UntarWithFilter(tarball, target string, filter FileFilter) error {
	opts := &UnarchiveOptions{Overwrite: true} // Maintain backward compatibility
	_, err := UntarWithFilterAndResult(tarball, target, filter, opts)
	return err
}

// UntarWithResult extracts all files in tarball to the target directory and returns detailed results
func UntarWithResult(tarball, target string) (*Archive, error) {
	opts := &UnarchiveOptions{Overwrite: true} // Maintain backward compatibility
	return UntarWithFilterAndResult(tarball, target, nil, opts)
}

// UntarWithFilterAndResult extracts all files in tarball with filter and returns detailed results
func UntarWithFilterAndResult(tarball, target string, filter FileFilter, opts *UnarchiveOptions) (*Archive, error) {
	absTarball, err := filepath.Abs(tarball)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tarball path: %w", err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve target path: %w", err)
	}

	// Initialize archive result
	archive := &Archive{
		Source:      absTarball,
		Destination: absTarget,
		Files:       make([]string, 0),
		Directories: make([]string, 0),
		Symlinks:    make([]string, 0),
		Skipped:     make([]string, 0),
		Errors:      make([]error, 0),
		Overwritten: make([]string, 0),
	}

	// Get compressed size from file stat
	if stat, err := os.Stat(tarball); err == nil {
		archive.CompressedSize = stat.Size()
	}

	logger.V(3).Infof("Untar: starting extraction of %s to %s", absTarball, absTarget)

	var reader io.Reader
	file, err := os.Open(tarball)
	if err != nil {
		return archive, err
	}
	defer file.Close()
	reader = file

	// Detect and handle compression
	if strings.HasSuffix(tarball, ".tar.gz") || strings.HasSuffix(tarball, ".tgz") {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return archive, fmt.Errorf("failed to create gzip reader: %w", err)
		}
	} else if strings.HasSuffix(tarball, ".tar.xz") || strings.HasSuffix(tarball, ".txz") {
		reader, err = xz.NewReader(reader)
		if err != nil {
			return archive, fmt.Errorf("failed to create xz reader: %w", err)
		}
	}

	tarReader := tar.NewReader(reader)

	if err := os.MkdirAll(target, 0755); err != nil {
		return archive, fmt.Errorf("failed to create target directory %s: %w", absTarget, err)
	}

	// Open the target directory as a root for secure file operations
	root, err := os.OpenRoot(target)
	if err != nil {
		return archive, fmt.Errorf("failed to open target directory as root %s: %w", absTarget, err)
	}
	defer root.Close()

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return archive, fmt.Errorf("error reading tar entry: %w", err)
		}

		info := header.FileInfo()
		path := header.Name
		if err := ValidatePath(header.Name); err != nil {
			return archive, err
		}

		if filter != nil {
			fp := filter(info)
			if fp == "" {
				archive.Skipped = append(archive.Skipped, path)
				continue
			}
		}

		if info.IsDir() {
			dirMode := info.Mode() & os.ModePerm // Extract permission bits only
			if dirMode == 0 {
				dirMode = 0755 // Default directory permissions
			}
			if err = root.MkdirAll(path, dirMode); err != nil {
				return archive, fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			archive.Directories = append(archive.Directories, path)
			continue
		}

		parent := filepath.Dir(path)
		if parent != "." && parent != "" {
			if err := root.MkdirAll(parent, 0755); err != nil {
				return archive, fmt.Errorf("failed to create directory %s: %w", parent, err)
			}
		}

		switch header.Typeflag {
		case tar.TypeReg:
			// Check for existing file
			fileExists := false
			if _, err := root.Stat(path); err == nil {
				fileExists = true
				if !opts.Overwrite {
					return archive, fmt.Errorf("file %s already exists", path)
				}
				archive.Overwritten = append(archive.Overwritten, path)
			}

			// Open file with appropriate flags
			flags := os.O_CREATE | os.O_RDWR
			if fileExists && opts.Overwrite {
				flags |= os.O_TRUNC
			}
			file, err := root.OpenFile(path, flags, os.FileMode(header.Mode))
			if err != nil {
				return archive, fmt.Errorf("failed to create file %s (mode=%v) %s", path, info.Mode(), err)
			}
			bytesWritten, err := io.Copy(file, tarReader)
			_ = file.Close()
			if err != nil {
				return archive, fmt.Errorf("failed to write file %s: %w", path, err)
			}
			archive.Files = append(archive.Files, path)
			archive.ExtractedSize += bytesWritten

		case tar.TypeSymlink:
			// Validate the symlink target stays within extraction dir
			linkTarget := header.Linkname
			// Only check relative paths; absolute always forbidden
			if filepath.IsAbs(linkTarget) || !filepath.IsLocal(linkTarget) {
				return archive, fmt.Errorf("symlink target %s is absolute and not allowed", linkTarget)
			}

			if err := root.Symlink(linkTarget, path); err != nil {
				return archive, fmt.Errorf("failed to create symlink %s -> %s: %w", path, linkTarget, err)
			}
			archive.Symlinks = append(archive.Symlinks, path)

		case tar.TypeDir:
			dirMode := os.FileMode(header.Mode) & os.ModePerm // Extract permission bits only
			if dirMode == 0 {
				dirMode = 0755 // Default directory permissions
			}
			if err := root.MkdirAll(path, dirMode); err != nil {
				return archive, fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			archive.Directories = append(archive.Directories, path)

		case tar.TypeLink:
			// Hard link - copy the target file content
			linkTarget := header.Linkname
			targetFile, err := root.Open(linkTarget)
			if err != nil {
				return nil, fmt.Errorf("cannot read hard link target %s for %s: %v", linkTarget, path, err)
			}

			// Check for existing file
			fileExists := false
			if _, err := root.Stat(path); err == nil {
				fileExists = true
				if !opts.Overwrite {
					_ = targetFile.Close()
					return archive, fmt.Errorf("file %s already exists", path)
				}
				archive.Overwritten = append(archive.Overwritten, path)
			}

			// Open file with appropriate flags
			flags := os.O_CREATE | os.O_RDWR
			if fileExists && opts.Overwrite {
				flags |= os.O_TRUNC
			}
			file, err := root.OpenFile(path, flags, os.FileMode(header.Mode))
			if err != nil {
				_ = targetFile.Close()
				return nil, fmt.Errorf("failed to create hard link file %s (mode=%v): %w", path, info.Mode(), err)
			}

			bytesWritten, err := io.Copy(file, targetFile)
			_ = targetFile.Close()
			_ = file.Close()

			if err != nil {
				return archive, fmt.Errorf("failed to copy content for hard link %s: %w", path, err)
			}
			archive.Files = append(archive.Files, path)
			archive.ExtractedSize += bytesWritten

		default:
			return nil, fmt.Errorf("unsupported file type %c (%d) for %s", header.Typeflag, header.Typeflag, path)
		}
	}

	// Calculate compression ratio
	if archive.CompressedSize > 0 && archive.ExtractedSize > 0 {
		archive.CompressionRatio = float64(archive.ExtractedSize) / float64(archive.CompressedSize)
	}

	logger.V(3).Infof("Untar: extraction complete for %s", archive)

	return archive, nil
}

// SafeRead reads a path and returns the text contents or nil
func SafeRead(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// Copy a file from src to dst
func Copy(src string, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	return err
}

func CopyFromReader(src io.Reader, dst string, mode os.FileMode) (int64, error) {
	dir := path.Dir(dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, err
	}
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	nBytes, err := io.Copy(f, src)
	return nBytes, err
}

// Exists returns true if the file exists
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsValidPathType(input string, extensions ...string) bool {
	if strings.Contains(input, "\n") {
		return false
	}
	for _, ext := range extensions {
		if strings.Trim(filepath.Ext(input), ".") == ext {
			return true
		}
	}
	return false
}

// GetBaseName returns the base part of the filename without the extension
func GetBaseName(filename string) string {
	filename = path.Base(filename)
	parts := strings.Split(filename, ".")
	if len(parts) == 1 {
		return filename
	}
	return strings.Join(parts[0:len(parts)-1], ".")
}

// TempFileName generates a temporary filename for
func TempFileName(prefix, suffix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes) //nolint:errCheck
	return filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes)+suffix)
}

// Zip the source file/directory into the target destination
func Zip(src, dst string) error {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := w.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return err
		}

		return nil
	}
	return filepath.Walk(src, walker)
}
