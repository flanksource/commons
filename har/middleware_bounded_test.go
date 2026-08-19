package har

import (
	"bytes"
	"io"
	"testing"
)

type countingReadCloser struct {
	*bytes.Reader
	read int
}

func (r *countingReadCloser) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.read += n
	return n, err
}

func (r *countingReadCloser) Close() error { return nil }

func TestReadBodyBoundsCaptureAndRestoresUnreadStream(t *testing.T) {
	content := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	source := &countingReadCloser{Reader: bytes.NewReader(content)}

	result, restored := readBody(source, 8, int64(len(content)))
	t.Cleanup(func() { _ = restored.Close() })

	if source.read != 9 {
		t.Fatalf("capture read %d bytes, want limit plus sentinel 9", source.read)
	}
	if result.text != "01234567" {
		t.Fatalf("captured %q, want first 8 bytes", result.text)
	}
	if !result.truncated {
		t.Fatal("expected capture to be marked truncated")
	}
	if result.totalSize != int64(len(content)) {
		t.Fatalf("total size = %d, want %d", result.totalSize, len(content))
	}

	full, err := io.ReadAll(restored)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if !bytes.Equal(full, content) {
		t.Fatalf("restored body = %q, want %q", full, content)
	}
}

func TestReadBodyReportsUnknownTruncatedSize(t *testing.T) {
	for _, contentLength := range []int64{-1, 0} {
		source := &countingReadCloser{Reader: bytes.NewReader([]byte("0123456789"))}
		result, restored := readBody(source, 4, contentLength)
		t.Cleanup(func() { _ = restored.Close() })

		if result.totalSize != -1 {
			t.Fatalf("content length %d: total size = %d, want unknown size -1", contentLength, result.totalSize)
		}
	}
}
