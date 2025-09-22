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

var blacklistedPathSymbols = "${}[]?*:<>|"
var blockedPrefixes = []string{"/run/", "/proc/", "/etc/", "/var/", "/tmp/", "/dev/"}

// safeJoinEvalSymlinks joins base and path, resolves symlinks, and ensures result is inside base
func safeJoinEvalSymlinks(base, p string) (string, error) {
	joined := filepath.Join(base, p)
	resolved, err := filepath.EvalSymlinks(joined)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	// If file does not exist yet (may be new), fallback to cleaned join
	if err != nil && os.IsNotExist(err) {
		resolved = filepath.Clean(joined)
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absBase, absResolved)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("illegal file path: %s escapes base directory %s", absResolved, absBase)
	}
	return absResolved, nil
}

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

// Unarchive extracts the contents of an archive to the dest directory
func Unarchive(src, dest string) error {
	logger.Debugf("Unarchiving %s to %s", src, dest)
	if strings.HasSuffix(src, ".zip") {
		return Unzip(src, dest)
	} else if strings.HasSuffix(src, ".tar") || strings.HasSuffix(src, ".tgz") || strings.HasSuffix(src, ".tar.gz") {
		return Untar(src, dest)
	}

	return fmt.Errorf("unknown format type %s", src)
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

// Unzip the source file to the target directory
func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		if err := ValidatePath(f.Name); err != nil {
			return err
		}

		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", fpath, err)
			}
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
				fdir = fpath[:lastIndex]
			}

			err = os.MkdirAll(fdir, f.Mode())
			if err != nil {
				return err
			}
			f, err := os.OpenFile(
				fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
	}
	return nil

}

// Untar extracts all files in tarball to the target directory
func Untar(tarball, target string) error {
	return UntarWithFilter(tarball, target, nil)
}

// UntarWithFilter extracts all files in tarball to the target directory, passing each file to filter
// if the filter returns "" then the file is ignored, otherwise the return string is used as the relative
// destination path
func UntarWithFilter(tarball, target string, filter FileFilter) error {
	var reader io.Reader
	file, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer file.Close()
	reader = file
	if strings.HasSuffix(tarball, ".tar.gz") || strings.HasSuffix(tarball, ".tgz") {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return err
		}
	} else if strings.HasSuffix(tarball, ".tar.xz") || strings.HasSuffix(tarball, ".txz") {
		reader, err = xz.NewReader(reader)
		if err != nil {
			return err
		}
	}

	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		info := header.FileInfo()
		if err := ValidatePath(header.Name); err != nil {
			return err
		}
		extractPath, err := safeJoinEvalSymlinks(target, header.Name)
		if err != nil {
			return fmt.Errorf("invalid extracted path: %w", err)
		}
		path := extractPath
		if filter != nil {
			fp := filter(info)
			if fp == "" {
				continue
			}
			newPath, err := safeJoinEvalSymlinks(target, fp)
			if err != nil {
				return fmt.Errorf("invalid filtered path: %w", err)
			}
			path = newPath
		}
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		parent := filepath.Dir(path)

		if _, err := os.Stat(parent); os.IsNotExist(err) {
			if err := os.MkdirAll(parent, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", parent, err)
			}
		}

		switch header.Typeflag {
		case tar.TypeReg:
			file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s (mode=%v) %s", path, info.Mode(), err)
			}
			defer file.Close()
			_, err = io.Copy(file, tarReader)
			if err != nil {
				return err
			}

		case tar.TypeSymlink:
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("failed to remove symlink %s: %w", path, err)
			}

			// Validate the symlink target stays within extraction dir
			linkTarget := header.Linkname
			// Only check relative paths; absolute always forbidden
			if filepath.IsAbs(linkTarget) {
				return fmt.Errorf("symlink target %s is absolute and not allowed", linkTarget)
			evalTarget, err := safeJoinEvalSymlinks(filepath.Dir(path), linkTarget)
			if err != nil {
				return fmt.Errorf("invalid symlink target from %s to %s: %w", path, linkTarget, err)
			}
			if !strings.HasPrefix(evalTarget, filepath.Clean(target)+string(os.PathSeparator)) && filepath.Clean(evalTarget) != filepath.Clean(target) {
				return fmt.Errorf("symlink %s target %s would escape extraction root", path, linkTarget)
			}
			if err := os.Symlink(linkTarget, path); err != nil {
				return fmt.Errorf("failed to create symlink %s -> %s: %w", path, linkTarget, err)
			}
			}

		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
			continue
		}

	}
	return nil
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
