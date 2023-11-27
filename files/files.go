package files

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/hashicorp/go-getter"
	"github.com/ulikunitz/xz"
)

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
	} else if strings.HasSuffix(src, ".tar") || strings.HasSuffix(src, ".tgz") || strings.HasSuffix(src, ".tar.gz") || strings.HasSuffix(src, ".tar.xz") {
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
	} else if strings.HasSuffix(tarball, ".tar.xz") {
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
		path := filepath.Join(target, header.Name)
		if filter != nil {
			path = filter(info)
			if path == "" {
				continue
			}
			path = filepath.Join(target, path)
		}
		if info.IsDir() {
			if err = os.MkdirAll(path, info.Mode()); err != nil {
				return err
			}
			continue
		}

		file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return err
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

// Getter gets a directory or file using the Hashicorp go-getter library
// See https://github.com/hashicorp/go-getter
func Getter(url, dst string) error {
	pwd, _ := os.Getwd()

	stashed := false
	if Exists(dst + "/.git") {
		cmd := exec.Command("git", "stash")
		cmd.Dir = pwd

		cmd.Dir = dst
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		if err := cmd.Run(); err == nil {
			stashed = true
		}

	}
	client := &getter.Client{
		Ctx:     context.TODO(),
		Src:     url,
		Dst:     dst,
		Pwd:     pwd,
		Mode:    getter.ClientModeDir,
		Options: []getter.ClientOption{},
	}
	logger.Infof("Downloading %s -> %s", url, dst)
	err := client.Get()
	if stashed {
		cmd := exec.Command("git", "stash", "pop")
		cmd.Dir = dst
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	}
	return err
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
