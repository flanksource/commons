package httpretty

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/flanksource/commons/logger/httpretty/internal/color"
)

func newPrinter(l *Logger) printer {
	l.mu.Lock()
	defer l.mu.Unlock()
	return printer{
		logger:  l,
		flusher: l.flusher,
	}
}

type printer struct {
	flusher Flusher
	logger  *Logger
	buf     bytes.Buffer
}

func (p *printer) maybeOnReady() {
	if p.flusher == OnReady {
		p.flush()
	}
}

func (p *printer) flush() {
	if p.flusher == NoBuffer {
		return
	}
	p.logger.mu.Lock()
	defer p.logger.mu.Unlock()
	defer p.buf.Reset()
	w := p.logger.getWriter()
	fmt.Fprint(w, p.buf.String())
}

func (p *printer) print(a ...interface{}) {
	p.logger.mu.Lock()
	defer p.logger.mu.Unlock()
	w := p.logger.getWriter()
	if p.flusher == NoBuffer {
		fmt.Fprint(w, a...)
		return
	}
	fmt.Fprint(&p.buf, a...)
}

func (p *printer) println(a ...interface{}) {
	p.logger.mu.Lock()
	defer p.logger.mu.Unlock()
	w := p.logger.getWriter()
	if p.flusher == NoBuffer {
		fmt.Fprintln(w, a...)
		return
	}
	fmt.Fprintln(&p.buf, a...)
}

func (p *printer) printf(format string, a ...interface{}) {
	p.logger.mu.Lock()
	defer p.logger.mu.Unlock()
	w := p.logger.getWriter()
	if p.flusher == NoBuffer {
		fmt.Fprintf(w, format, a...)
		return
	}
	fmt.Fprintf(&p.buf, format, a...)
}

func (p *printer) printRequest(req *http.Request) {
	if p.logger.RequestHeader {
		p.printRequestHeader(req)
		p.maybeOnReady()
	}
	if p.logger.RequestBody && req.Body != nil {
		p.printRequestBody(req)
		p.maybeOnReady()
	}
}

func (p *printer) printRequestInfo(req *http.Request) {
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	p.printf("%s\n", p.format(color.FgBlue, host))
	p.printf("%s %s\n",
		p.format(color.FgBlue, color.Bold, req.Method),
		p.format(color.FgYellow, req.URL.Path))
	for key, values := range req.URL.Query() {
		p.printf("  %s: %s\n",
			p.format(color.FgBlue, key),
			p.format(color.FgYellow, strings.Join(values, ",")))
	}
}

// checkFilter checkes if the request is filtered and if the Request value is nil.
func (p *printer) checkFilter(req *http.Request) (skip bool) {
	filter := p.logger.getFilter()
	if req == nil {
		p.printf("> %s\n", p.format(color.FgRed, "error: null request"))
		return true
	}
	if filter == nil {
		return false
	}
	ok, err := safeFilter(filter, req)
	if err != nil {
		p.printf("* cannot filter request: %s: %s\n", p.format(color.FgBlue, fmt.Sprintf("%s %s", req.Method, req.URL)), p.format(color.FgRed, err.Error()))
		return false // never filter out the request if the filter errored
	}
	return ok
}

func safeFilter(filter Filter, req *http.Request) (skip bool, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("panic: %v", e)
		}
	}()
	return filter(req)
}

func (p *printer) printResponse(resp *http.Response) {
	if resp == nil {
		p.printf("< %s\n", p.format(color.FgRed, "error: null response"))
		p.maybeOnReady()
		return
	}
	if p.logger.ResponseHeader {
		p.printResponseHeader(resp.Proto, resp.Status, resp.Header)
		p.maybeOnReady()
	}
	if p.logger.ResponseBody && resp.Body != nil && (resp.Request == nil || resp.Request.Method != http.MethodHead) {
		p.printResponseBodyOut(resp)
		p.maybeOnReady()
	}
}

func (p *printer) checkBodyFiltered(h http.Header) (skip bool, err error) {
	if f := p.logger.getBodyFilter(); f != nil {
		defer func() {
			if e := recover(); e != nil {
				p.printf("* panic while filtering body: %v\n", e)
			}
		}()
		return f(h)
	}
	return false, nil
}

func (p *printer) printResponseBodyOut(resp *http.Response) {
	if resp.ContentLength == 0 {
		return
	}
	skip, err := p.checkBodyFiltered(resp.Header)
	if err != nil {
		p.printf("* %s\n", p.format(color.FgRed, "error on response body filter: ", err.Error()))
	}
	if skip {
		return
	}
	if contentType := resp.Header.Get("Content-Type"); contentType != "" && isBinaryMediatype(contentType) {
		p.println("* body contains binary data")
		return
	}
	contentType := resp.Header.Get("Content-Type")
	if p.logger.MaxResponseBody > 0 || resp.ContentLength == -1 {
		resp.Body = p.printResponseBodyPrefix(contentType, p.logger.MaxResponseBody, resp.Body, resp.ContentLength)
		return
	}
	var buf bytes.Buffer
	tee := io.TeeReader(resp.Body, &buf)
	defer resp.Body.Close()
	defer func() {
		resp.Body = io.NopCloser(&buf)
	}()
	p.printBodyReader(contentType, tee)
}

// isBinary uses heuristics to guess if file is binary (actually, "printable" in the terminal).
// See discussion at https://groups.google.com/forum/#!topic/golang-nuts/YeLL7L7SwWs
func isBinary(body []byte) bool {
	if len(body) > 512 {
		body = body[512:]
	}
	// If file contains UTF-8 OR UTF-16 BOM, consider it non-binary.
	// Reference: https://tools.ietf.org/html/draft-ietf-websec-mime-sniff-03#section-5
	if len(body) >= 3 && (bytes.Equal(body[:2], []byte{0xFE, 0xFF}) || // UTF-16BE BOM
		bytes.Equal(body[:2], []byte{0xFF, 0xFE}) || // UTF-16LE BOM
		bytes.Equal(body[:3], []byte{0xEF, 0xBB, 0xBF})) { // UTF-8 BOM
		return false
	}
	// If all of the first n octets are binary data octets, consider it binary.
	// Reference: https://github.com/golang/go/blob/349e7df2c3d0f9b5429e7c86121499c137faac7e/src/net/http/sniff.go#L297-L309
	// c.f. section 5, step 4.
	for _, b := range body {
		switch {
		case b <= 0x08,
			b == 0x0B,
			0x0E <= b && b <= 0x1A,
			0x1C <= b && b <= 0x1F:
			return true
		}
	}
	// Otherwise, check against a white list of binary mimetypes.
	mediatype, _, err := mime.ParseMediaType(http.DetectContentType(body))
	if err != nil {
		return false
	}
	return isBinaryMediatype(mediatype)
}

var binaryMediatypes = map[string]struct{}{
	"application/pdf":               {},
	"application/postscript":        {},
	"image":                         {}, // for practical reasons, any image (including SVG) is considered binary data
	"audio":                         {},
	"application/ogg":               {},
	"video":                         {},
	"application/vnd.ms-fontobject": {},
	"font":                          {},
	"application/x-gzip":            {},
	"application/zip":               {},
	"application/x-rar-compressed":  {},
	"application/wasm":              {},
}

func isBinaryMediatype(mediatype string) bool {
	if _, ok := binaryMediatypes[mediatype]; ok {
		return true
	}
	if parts := strings.SplitN(mediatype, "/", 2); len(parts) == 2 {
		if _, ok := binaryMediatypes[parts[0]]; ok {
			return true
		}
	}
	return false
}

const maxDefaultUnknownReadable = 4096 // bytes

func (p *printer) printBodyUnknownLength(contentType string, maxLength int64, r io.ReadCloser) (newBody io.ReadCloser) {
	if maxLength == 0 {
		maxLength = maxDefaultUnknownReadable
	}
	pb := make([]byte, maxLength+1) // read one extra bit to assure the length is longer than acceptable
	n, err := io.ReadFull(r, pb)
	pb = pb[0:n] // trim any nil symbols left after writing in the byte slice.
	buf := bytes.NewReader(pb)
	newBody = newBodyReaderBuf(buf, r)
	switch {
	// Server requests always return req.Body != nil, but the Reader returns io.EOF immediately.
	// Avoiding returning early to mitigate any risk of bad reader implementations that might
	// send something even after returning io.EOF if read again.
	case err == io.EOF && n == 0:
	case err == nil && int64(n) > maxLength:
		p.printf("* body is too long, skipping (contains more than %d bytes)\n", n-1)
	case err == io.ErrUnexpectedEOF || err == nil:
		// cannot pass same bytes reader below because we only read it once.
		p.printBodyReader(contentType, bytes.NewReader(pb))
	default:
		p.printf("* cannot read body: %v (%d bytes read)\n", err, n)
	}
	return
}

func (p *printer) printServerResponse(req *http.Request, rec *responseRecorder) {
	if p.logger.ResponseHeader {
		// TODO(henvic): see how httptest.ResponseRecorder adds extra headers due to Content-Type detection
		// and other stuff (Date). It would be interesting to show them here too (either as default or opt-in).
		p.printResponseHeader(req.Proto, fmt.Sprintf("%d %s", rec.statusCode, http.StatusText(rec.statusCode)), rec.Header())
	}
	if !p.logger.ResponseBody || rec.size == 0 {
		return
	}
	skip, err := p.checkBodyFiltered(rec.Header())
	if err != nil {
		p.printf("* %s\n", p.format(color.FgRed, "error on response body filter: ", err.Error()))
	}
	if skip {
		return
	}
	if mediatype := req.Header.Get("Content-Type"); mediatype != "" && isBinaryMediatype(mediatype) {
		p.println("* body contains binary data")
		return
	}
	if p.logger.MaxResponseBody > 0 && rec.size > p.logger.MaxResponseBody {
		p.printBodyReader(rec.Header().Get("Content-Type"), bytes.NewReader(rec.buf.Bytes()))
		p.printf("* body truncated after %d bytes (total %d bytes)\n", p.logger.MaxResponseBody, rec.size)
		return
	}
	p.printBodyReader(rec.Header().Get("Content-Type"), rec.buf)
}

func (p *printer) printResponseHeader(proto, status string, h http.Header) {
	p.printf("< %s %s\n",
		p.format(color.FgBlue, color.Bold, proto),
		p.format(color.FgRed, status))
	p.printHeaders('<', h)
	p.println()
}

func (p *printer) printBodyReader(contentType string, r io.Reader) {
	mediatype, _, _ := mime.ParseMediaType(contentType)
	body, err := io.ReadAll(r)
	if err != nil {
		p.printf("* cannot read body: %v\n", p.format(color.FgRed, err.Error()))
		return
	}
	if isBinary(body) {
		p.println("* body contains binary data")
		return
	}
	for _, f := range p.logger.Formatters {
		if ok := p.safeBodyMatch(f, mediatype); !ok {
			continue
		}
		var formatted bytes.Buffer
		switch err := p.safeBodyFormat(f, &formatted, body); {
		case err != nil:
			p.printf("* body cannot be formatted: %v\n%s\n", p.format(color.FgRed, err.Error()), string(body))
		default:
			p.println(formatted.String())
		}
		return
	}

	p.println(string(body))
}

func (p *printer) safeBodyMatch(f Formatter, mediatype string) bool {
	defer func() {
		if e := recover(); e != nil {
			p.printf("* panic while testing body format: %v\n", e)
		}
	}()
	return f.Match(mediatype)
}

func (p *printer) safeBodyFormat(f Formatter, w io.Writer, src []byte) (err error) {
	defer func() {
		// should not return panic as error because we want to try the next formatter
		if e := recover(); e != nil {
			err = fmt.Errorf("panic: %v", e)
		}
	}()
	return f.Format(w, src)
}

func (p *printer) format(s ...interface{}) string {
	if p.logger.Colors {
		return color.Format(s...)
	}
	return color.StripAttributes(s...)
}

func (p *printer) printRequestBody(req *http.Request) {
	// For client requests, a request with zero content-length and no body is also treated as unknown.
	if req.Body == nil {
		return
	}
	skip, err := p.checkBodyFiltered(req.Header)
	if err != nil {
		p.printf("* %s\n", p.format(color.FgRed, "error on request body filter: ", err.Error()))
	}
	if skip {
		return
	}
	if mediatype := req.Header.Get("Content-Type"); mediatype != "" && isBinaryMediatype(mediatype) {
		p.println("* body contains binary data")
		return
	}
	// TODO(henvic): add support for printing multipart/formdata information as body (to responses too).
	if p.logger.MaxRequestBody > 0 && req.ContentLength > p.logger.MaxRequestBody {
		p.printf("* body is too long (%d bytes) to print, skipping (longer than %d bytes)\n",
			req.ContentLength, p.logger.MaxRequestBody)
		return
	}
	contentType := req.Header.Get("Content-Type")
	if req.ContentLength > 0 {
		var buf bytes.Buffer
		tee := io.TeeReader(req.Body, &buf)
		defer req.Body.Close()
		defer func() {
			req.Body = io.NopCloser(&buf)
		}()
		p.printBodyReader(contentType, tee)
		return
	}
	if newBody := p.printBodyUnknownLength(contentType, p.logger.MaxRequestBody, req.Body); newBody != nil {
		req.Body = newBody
	}
}

func (p *printer) printTimeRequest() (end func()) {
	startRequest := time.Now()
	p.printf(" at %v\n", startRequest.Format(time.RFC3339))
	return func() {
		p.printf("* Request took %v\n", time.Since(startRequest))
	}
}
