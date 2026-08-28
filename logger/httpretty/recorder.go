package httpretty

import (
	"bytes"
	"io"
	"net/http"
)

type bodyCloser struct {
	r     io.Reader
	close func() error
}

func (bc *bodyCloser) Read(p []byte) (n int, err error) {
	return bc.r.Read(p)
}

func (bc *bodyCloser) Close() error {
	return bc.close()
}

func newBodyReaderBuf(buf io.Reader, body io.ReadCloser) *bodyCloser {
	return &bodyCloser{
		r:     io.MultiReader(buf, body),
		close: body.Close,
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode      int
	maxReadableBody int64
	size            int64
	buf             *bytes.Buffer
}

// Write the data to the connection as part of an HTTP reply, and records it.
func (rr *responseRecorder) Write(p []byte) (int, error) {
	n, err := rr.ResponseWriter.Write(p)
	rr.size += int64(n)
	readable := n
	if rr.maxReadableBody > 0 {
		remaining := rr.maxReadableBody - int64(rr.buf.Len())
		if remaining <= 0 {
			return n, err
		}
		if int64(readable) > remaining {
			readable = int(remaining)
		}
	}
	_, _ = rr.buf.Write(p[:readable])
	return n, err
}

// WriteHeader sends an HTTP response header with the provided
// status code, and records it.
func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.ResponseWriter.WriteHeader(statusCode)
	rr.statusCode = statusCode
}
