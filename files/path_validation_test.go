package files

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ValidatePath", func() {
	Context("Non-ASCII character support", func() {
		It("should allow various non-ASCII characters", func() {
			validPaths := []string{
				"café.txt",
				"测试.txt",
				"файл.txt",
				"日本語.txt",
				"émojis😀.txt",
				"subdir/文件.txt",
				"한글/파일.txt",
				"العربية/ملف.txt",
				"ελληνικά/αρχείο.txt",
			}

			for _, path := range validPaths {
				err := ValidatePath(path)
				Expect(err).NotTo(HaveOccurred(), "Path %s should be valid", path)
			}
		})

		It("should still reject dangerous control characters", func() {
			invalidPaths := []string{
				"file\x00.txt", // null byte
				"file\x01.txt", // control character
				"file\x1f.txt", // control character
				"file\n.txt",   // newline
				"file\r.txt",   // carriage return
			}

			for _, path := range invalidPaths {
				err := ValidatePath(path)
				Expect(err).To(HaveOccurred(), "Path %s should be invalid", path)
				Expect(err.Error()).To(ContainSubstring("control characters"))
			}
		})

		It("should allow tabs in filenames", func() {
			err := ValidatePath("file\twith\ttabs.txt")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("Existing security validations", func() {
		It("should reject path traversal attempts", func() {
			invalidPaths := []string{
				"../etc/passwd",
				"subdir/../../../etc/passwd",
				"..\\windows\\system32",
			}

			for _, path := range invalidPaths {
				err := ValidatePath(path)
				Expect(err).To(HaveOccurred(), "Path %s should be invalid", path)
				Expect(err.Error()).To(ContainSubstring("access parent directories"))
			}
		})

		It("should reject blocked characters", func() {
			blacklistedChars := []string{"{", "}", "[", "]", "?", "*", ":", "<", ">", "|"}

			for _, char := range blacklistedChars {
				path := "file" + char + ".txt"
				err := ValidatePath(path)
				Expect(err).To(HaveOccurred(), "Path %s should be invalid", path)
				Expect(err.Error()).To(ContainSubstring("illegal characters"))
			}
		})

		It("should allow $ character for Java inner classes", func() {
			validPaths := []string{
				"WebConsoleStarter$OsgiUtil.class",
				"OuterClass$InnerClass.class",
				"Service$1$Helper.class",
				"apache-activemq-5.19.0/webapps/admin/WEB-INF/classes/org/apache/activemq/web/WebConsoleStarter$OsgiUtil.class",
			}

			for _, path := range validPaths {
				err := ValidatePath(path)
				Expect(err).NotTo(HaveOccurred(), "Path %s should be valid", path)
			}
		})

		It("should reject blocked prefixes", func() {
			blockedPaths := []string{
				"/etc/passwd",
				"/var/log/system.log",
				"/tmp/exploit",
				"/proc/version",
				"/run/secrets",
				"/dev/null",
			}

			for _, path := range blockedPaths {
				err := ValidatePath(path)
				Expect(err).To(HaveOccurred(), "Path %s should be invalid", path)
				Expect(err.Error()).To(ContainSubstring("blocked prefix"))
			}
		})

		It("should allow valid normal paths", func() {
			validPaths := []string{
				"file.txt",
				"subdir/file.txt",
				"another_dir/subdir/file-name.txt",
				"file123.log",
				"data.json",
			}

			for _, path := range validPaths {
				err := ValidatePath(path)
				Expect(err).NotTo(HaveOccurred(), "Path %s should be valid", path)
			}
		})
	})
})
