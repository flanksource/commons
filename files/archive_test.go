package files

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArchive(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Archive Suite")
}

type testCase struct {
	name            string
	archivePath     string
	expectedArchive *Archive
	expectedError   string
}

var _ = Describe("Unarchive", func() {
	testCases := []testCase{
		{
			name:        "simple tar archive",
			archivePath: "testdata/archives/simple.tar",
			expectedArchive: &Archive{
				Files:       []string{"file1.txt", "file2.txt"},
				Directories: []string{},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "simple gzipped tar archive",
			archivePath: "testdata/archives/simple.tar.gz",
			expectedArchive: &Archive{
				Files:       []string{"file1.txt", "file2.txt"},
				Directories: []string{},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "simple xz compressed tar archive",
			archivePath: "testdata/archives/simple.tar.xz",
			expectedArchive: &Archive{
				Files:       []string{"file1.txt", "file2.txt"},
				Directories: []string{},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "nested tar with directories",
			archivePath: "testdata/archives/nested.tar",
			expectedArchive: &Archive{
				Files:       []string{"./file1.txt", "./file2.txt", "./subdir/file3.txt"},
				Directories: []string{"./", "./subdir/"},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "tar with safe symlinks",
			archivePath: "testdata/archives/safe_symlinks.tar",
			expectedArchive: &Archive{
				Files:       []string{"target.txt"},
				Directories: []string{},
				Symlinks:    []string{"safe_link"},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:          "tar with dangerous symlinks should fail",
			archivePath:   "testdata/archives/parent_symlinks.tar",
			expectedError: "is absolute and not allowed",
		},
		{
			name:        "simple zip archive",
			archivePath: "testdata/archives/simple.zip",
			expectedArchive: &Archive{
				Files:       []string{"file1.txt", "file2.txt"},
				Directories: []string{},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "jar archive",
			archivePath: "testdata/archives/test.jar",
			expectedArchive: &Archive{
				Files:       []string{"file1.txt", "file2.txt"},
				Directories: []string{},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "tgz archive",
			archivePath: "testdata/archives/nested.tgz",
			expectedArchive: &Archive{
				Files:       []string{"file1.txt", "file2.txt", "subdir/file3.txt"},
				Directories: []string{"subdir/"},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "txz archive",
			archivePath: "testdata/archives/nested.txz",
			expectedArchive: &Archive{
				Files:       []string{"file1.txt", "file2.txt", "subdir/file3.txt"},
				Directories: []string{"subdir/"},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "tar.xz with safe symlinks",
			archivePath: "testdata/archives/safe_postgres.tar.xz",
			expectedArchive: &Archive{
				Files:       []string{"./bin/postgres", "./bin/postgres.conf"},
				Directories: []string{"./", "./bin/"},
				Symlinks:    []string{"./bin/config_link"},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:          "tar.xz with parent symlinks should fail",
			archivePath:   "testdata/archives/symlinks.tar.xz",
			expectedError: "is absolute and not allowed",
		},
		{
			name:        "empty archive",
			archivePath: "testdata/archives/empty.tar",
			expectedArchive: &Archive{
				Files:       []string{},
				Directories: []string{},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "dirs only archive",
			archivePath: "testdata/archives/dirs_only.tar",
			expectedArchive: &Archive{
				Files:       []string{},
				Directories: []string{"./", "./subdir1/", "./subdir1/subdir2/", "./subdir3/"},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "hard links in tar",
			archivePath: "testdata/archives/hardlinks.tar",
			expectedArchive: &Archive{
				Files:       []string{"./hardlink.txt", "./original.txt"},
				Directories: []string{"./"},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:        "special characters in zip",
			archivePath: "testdata/archives/special_chars.zip",
			expectedArchive: &Archive{
				Files:       []string{"special_chars/file with spaces.txt", "special_chars/file_with_underscore.txt"},
				Directories: []string{},
				Symlinks:    []string{},
				Skipped:     []string{},
				Errors:      []error{},
			},
		},
		{
			name:          "corrupted archive",
			archivePath:   "testdata/archives/corrupted.tar.gz",
			expectedError: "unexpected EOF",
		},
		{
			name:          "unsupported format",
			archivePath:   "testdata/archives/nonexistent.rar",
			expectedError: "unknown format type",
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		It(tc.name, func() {
			// Create temporary directory for extraction
			tempDir, err := os.MkdirTemp("", "unarchive-test-")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			// Run the extraction
			archive, err := Unarchive(tc.archivePath, tempDir)

			if tc.expectedError != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(tc.expectedError))
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(archive).NotTo(BeNil())

				// Verify extracted files match expectations
				Expect(archive.Files).To(ConsistOf(tc.expectedArchive.Files))
				Expect(archive.Directories).To(ConsistOf(tc.expectedArchive.Directories))
				Expect(archive.Symlinks).To(ConsistOf(tc.expectedArchive.Symlinks))

				// Verify source and destination paths are set
				absArchivePath, _ := filepath.Abs(tc.archivePath)
				absTempDir, _ := filepath.Abs(tempDir)
				Expect(archive.Source).To(Equal(absArchivePath))
				Expect(archive.Destination).To(Equal(absTempDir))

				// Verify extracted size is greater than 0 for non-empty archives
				if len(tc.expectedArchive.Files) > 0 {
					Expect(archive.ExtractedSize).To(BeNumerically(">", 0))
				}

				// Verify compressed size is greater than 0
				Expect(archive.CompressedSize).To(BeNumerically(">", 0))

				// Verify actual files exist on disk
				for _, fileName := range tc.expectedArchive.Files {
					filePath := filepath.Join(tempDir, fileName)
					Expect(filePath).To(BeAnExistingFile())
				}

				// Verify directories exist on disk
				for _, dirName := range tc.expectedArchive.Directories {
					dirPath := filepath.Join(tempDir, dirName)
					Expect(dirPath).To(BeADirectory())
				}

				// Verify symlinks exist on disk
				for _, linkName := range tc.expectedArchive.Symlinks {
					linkPath := filepath.Join(tempDir, linkName)
					info, err := os.Lstat(linkPath)
					Expect(err).NotTo(HaveOccurred())
					Expect(info.Mode() & os.ModeSymlink).NotTo(BeZero())
				}
			}
		})
	}
})

var _ = Describe("UntarWithFilter", func() {
	It("should extract only executable files", func() {
		tempDir, err := os.MkdirTemp("", "filter-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		// Test executable filter
		err = UntarWithFilter("testdata/archives/executable.tar", tempDir, func(header os.FileInfo) string {
			if header.Mode()&0111 != 0 { // Has execute permission
				return header.Name()
			}
			return "" // Skip non-executable files
		})

		Expect(err).NotTo(HaveOccurred())

		// Verify only executable was extracted
		scriptPath := filepath.Join(tempDir, "script.sh")
		regularPath := filepath.Join(tempDir, "regular.txt")

		Expect(scriptPath).To(BeAnExistingFile())
		Expect(regularPath).NotTo(BeAnExistingFile())
	})

	It("should apply custom name filter", func() {
		tempDir, err := os.MkdirTemp("", "filter-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		// Test custom filter - only files containing "file1"
		err = UntarWithFilter("testdata/archives/simple.tar", tempDir, func(header os.FileInfo) string {
			if strings.Contains(header.Name(), "file1") {
				return header.Name()
			}
			return ""
		})

		Expect(err).NotTo(HaveOccurred())

		// Verify only file1 was extracted
		file1Path := filepath.Join(tempDir, "file1.txt")
		file2Path := filepath.Join(tempDir, "file2.txt")

		Expect(file1Path).To(BeAnExistingFile())
		Expect(file2Path).NotTo(BeAnExistingFile())
	})
})

var _ = Describe("UnarchiveExecutables", func() {
	It("should extract only executable files using the built-in filter", func() {
		tempDir, err := os.MkdirTemp("", "executable-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		err = UnarchiveExecutables("testdata/archives/executable.tar", tempDir)
		Expect(err).NotTo(HaveOccurred())

		// Verify only executable was extracted to base name
		scriptPath := filepath.Join(tempDir, "script.sh")
		regularPath := filepath.Join(tempDir, "regular.txt")

		Expect(scriptPath).To(BeAnExistingFile())
		Expect(regularPath).NotTo(BeAnExistingFile())
	})
})

var _ = Describe("Unarchive with Overwrite Options", func() {
	type overwriteTestCase struct {
		name            string
		archivePath     string
		overwriteOption bool
		expectedError   string
	}

	overwriteTestCases := []overwriteTestCase{
		{
			name:            "no overwrite with existing files should fail",
			archivePath:     "testdata/archives/simple.tar",
			overwriteOption: false,
			expectedError:   "file file1.txt already exists",
		},
		{
			name:            "with overwrite should succeed",
			archivePath:     "testdata/archives/simple.tar",
			overwriteOption: true,
			expectedError:   "",
		},
		{
			name:            "no overwrite with zip and existing files should fail",
			archivePath:     "testdata/archives/simple.zip",
			overwriteOption: false,
			expectedError:   "file file1.txt already exists",
		},
		{
			name:            "with overwrite for zip should succeed",
			archivePath:     "testdata/archives/simple.zip",
			overwriteOption: true,
			expectedError:   "",
		},
	}

	for _, tc := range overwriteTestCases {
		tc := tc // capture range variable
		It(tc.name, func() {
			// Create temporary directory for extraction
			tempDir, err := os.MkdirTemp("", "overwrite-test-")
			Expect(err).NotTo(HaveOccurred())
			defer os.RemoveAll(tempDir)

			// First extraction - should always succeed
			_, err = Unarchive(tc.archivePath, tempDir)
			Expect(err).NotTo(HaveOccurred())

			// Create different content in existing files to verify overwrite behavior
			existingFiles := []string{"file1.txt", "file2.txt"}
			originalContent := "ORIGINAL CONTENT - SHOULD BE REPLACED"
			for _, fileName := range existingFiles {
				filePath := filepath.Join(tempDir, fileName)
				err := os.WriteFile(filePath, []byte(originalContent), 0644)
				Expect(err).NotTo(HaveOccurred())
			}

			// Second extraction - behavior depends on overwrite option
			var archive *Archive
			if tc.overwriteOption {
				archive, err = Unarchive(tc.archivePath, tempDir, WithOverwrite(true))
			} else {
				archive, err = Unarchive(tc.archivePath, tempDir)
			}

			if tc.expectedError != "" {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(tc.expectedError))
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(archive).NotTo(BeNil())

				// Verify Overwritten list contains the expected files in exact order
				Expect(archive.Overwritten).To(Equal(existingFiles))

				// Verify files were actually overwritten (content changed)
				for _, fileName := range existingFiles {
					filePath := filepath.Join(tempDir, fileName)
					content, err := os.ReadFile(filePath)
					Expect(err).NotTo(HaveOccurred())
					Expect(string(content)).NotTo(ContainSubstring("ORIGINAL CONTENT"))
				}
			}
		})
	}

	It("should track mixed overwrite scenario correctly", func() {
		tempDir, err := os.MkdirTemp("", "mixed-overwrite-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		// Create only some files that will be overwritten
		existingFile := filepath.Join(tempDir, "file1.txt")
		err = os.WriteFile(existingFile, []byte("existing content"), 0644)
		Expect(err).NotTo(HaveOccurred())

		// Extract archive with overwrite enabled
		archive, err := Unarchive("testdata/archives/simple.tar", tempDir, WithOverwrite(true))
		Expect(err).NotTo(HaveOccurred())

		// Should only list the one file that was overwritten
		Expect(archive.Overwritten).To(Equal([]string{"file1.txt"}))

		// Should still list both extracted files
		Expect(archive.Files).To(ConsistOf([]string{"file1.txt", "file2.txt"}))
	})

	It("should handle hard links overwrite correctly", func() {
		tempDir, err := os.MkdirTemp("", "hardlink-overwrite-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		// First extraction
		_, err = Unarchive("testdata/archives/hardlinks.tar", tempDir)
		Expect(err).NotTo(HaveOccurred())

		// Modify existing files
		files := []string{"./original.txt", "./hardlink.txt"}
		for _, fileName := range files {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte("modified content"), 0644)
			Expect(err).NotTo(HaveOccurred())
		}

		// Second extraction with overwrite
		archive, err := Unarchive("testdata/archives/hardlinks.tar", tempDir, WithOverwrite(true))
		Expect(err).NotTo(HaveOccurred())

		// Should track both overwritten files with exact paths
		Expect(archive.Overwritten).To(Equal(files))
	})

	It("should verify error message contains the first conflicting file", func() {
		tempDir, err := os.MkdirTemp("", "error-message-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		// Create only the first file that would conflict
		existingFile := filepath.Join(tempDir, "file1.txt")
		err = os.WriteFile(existingFile, []byte("existing content"), 0644)
		Expect(err).NotTo(HaveOccurred())

		// Extract without overwrite should fail with specific filename
		_, err = Unarchive("testdata/archives/simple.tar", tempDir)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("file file1.txt already exists"))
	})

	It("should preserve path format from archive in Overwritten list", func() {
		tempDir, err := os.MkdirTemp("", "path-format-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		// First extraction
		_, err = Unarchive("testdata/archives/nested.tar", tempDir)
		Expect(err).NotTo(HaveOccurred())

		// Modify existing files (using the exact paths from the archive in correct order)
		existingFiles := []string{"./file2.txt", "./file1.txt", "./subdir/file3.txt"}
		for _, fileName := range existingFiles {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte("modified content"), 0644)
			Expect(err).NotTo(HaveOccurred())
		}

		// Second extraction with overwrite
		archive, err := Unarchive("testdata/archives/nested.tar", tempDir, WithOverwrite(true))
		Expect(err).NotTo(HaveOccurred())

		// Should preserve the exact path format from the archive
		Expect(archive.Overwritten).To(Equal(existingFiles))
	})

	It("should maintain overwrite order matching extraction order", func() {
		tempDir, err := os.MkdirTemp("", "order-test-")
		Expect(err).NotTo(HaveOccurred())
		defer os.RemoveAll(tempDir)

		// Create files in reverse order to test that overwrite order follows extraction order
		existingFiles := []string{"file2.txt", "file1.txt"}
		for _, fileName := range existingFiles {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte("existing content"), 0644)
			Expect(err).NotTo(HaveOccurred())
		}

		// Extract with overwrite
		archive, err := Unarchive("testdata/archives/simple.tar", tempDir, WithOverwrite(true))
		Expect(err).NotTo(HaveOccurred())

		// Overwritten should follow the order they appear in the archive, not filesystem order
		Expect(archive.Overwritten).To(Equal([]string{"file1.txt", "file2.txt"}))
	})
})
