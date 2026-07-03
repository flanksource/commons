package logger

import "testing"

func FuzzStripSecrets(f *testing.F) {
	seeds := []string{
		"password=secret",
		"token: abc123",
		"https://user:password@example.com/path?token=abc",
		"Authorization: Bearer token",
		"plain text",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		_ = StripSecrets(input)
		_ = PrintableSecret(input)
	})
}
