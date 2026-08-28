package httpretty

import (
	"bytes"
	"io"
	"math"
)

func (p *printer) printResponseBodyPrefix(contentType string, maxLength int64, body io.ReadCloser, knownLength int64) io.ReadCloser {
	if maxLength <= 0 {
		maxLength = maxDefaultUnknownReadable
	}
	readLimit := maxLength
	if maxLength < math.MaxInt64 {
		readLimit++
	}
	prefix, err := io.ReadAll(io.LimitReader(body, readLimit))
	restored := newBodyReaderBuf(bytes.NewReader(prefix), body)
	if err != nil {
		p.printf("* cannot read body: %v (%d bytes read)\n", err, len(prefix))
		return restored
	}

	truncated := int64(len(prefix)) > maxLength
	if truncated {
		prefix = prefix[:maxLength]
	}
	if len(prefix) > 0 {
		p.printBodyReader(contentType, bytes.NewReader(prefix))
	}
	if truncated {
		if knownLength > 0 {
			p.printf("* body truncated after %d bytes (total %d bytes)\n", maxLength, knownLength)
		} else {
			p.printf("* body truncated after %d bytes\n", maxLength)
		}
	}
	return restored
}
