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
//
// The recorder is a pass-through: p is forwarded verbatim to the ResponseWriter
// the caller handed to Middleware, so this adds no response content of its own.
func (rr *responseRecorder) Write(p []byte) (int, error) {
	rr.record(p)
	rr.size += int64(len(p))
	return rr.ResponseWriter.Write(p)
}

// WriteHeader sends an HTTP response header with the provided
// status code, and records it.
func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.ResponseWriter.WriteHeader(statusCode)
	rr.statusCode = statusCode
}

// record buffers the response prefix used for logging, keeping at most
// maxReadableBody bytes so a large response is logged truncated rather than
// held in memory in full.
func (rr *responseRecorder) record(p []byte) {
	if rr.buf == nil {
		return
	}
	readable := len(p)
	if rr.maxReadableBody > 0 {
		remaining := rr.maxReadableBody - int64(rr.buf.Len())
		if remaining <= 0 {
			return
		}
		if int64(readable) > remaining {
			readable = int(remaining)
		}
	}
	_, _ = rr.buf.Write(p[:readable])
}
