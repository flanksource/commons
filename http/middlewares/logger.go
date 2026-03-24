package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/api"
	"github.com/flanksource/commons/console"
	commonsCtx "github.com/flanksource/commons/context"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/logger/httpretty"
)

type jsonFormatter struct{}

func (j *jsonFormatter) Match(mediatype string) bool {
	return strings.Contains(mediatype, "json")
}

func (j *jsonFormatter) Format(w io.Writer, src []byte) error {
	if !json.Valid(src) {
		if err := json.Unmarshal(src, &json.RawMessage{}); err != nil {
			return err
		}
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, src, "", "    "); err != nil {
		return err
	}
	fmt.Fprint(w, api.CodeBlock("json", indented.String()).ANSI())
	return nil
}

type formURLEncodedFormatter struct{}

func (f *formURLEncodedFormatter) Match(mediatype string) bool {
	return mediatype == "application/x-www-form-urlencoded"
}

func (f *formURLEncodedFormatter) Format(w io.Writer, src []byte) error {
	values, err := url.ParseQuery(string(src))
	if err != nil {
		return err
	}
	m := make(map[string]string)
	for k, v := range values {
		m[k] = strings.Join(v, ",")
	}
	fmt.Fprint(w, clicky.Map(m).ANSI())
	return nil
}

func getLogger(req *http.Request) logger.Logger {
	if req == nil || req.Context() == nil {
		return logger.GetLogger()
	}
	return commonsCtx.LoggerFromContext(req.Context())
}

func newContextLogger(config TraceConfig) Middleware {
	return func(rt http.RoundTripper) http.RoundTripper {
		l := &httpretty.Logger{
			TLS:            config.TLS,
			RequestHeader:  config.Headers,
			RequestBody:    config.Body,
			ResponseHeader: config.ResponseHeaders,
			ResponseBody:   config.Response,
			Auth:           config.Auth,
			Colors:         true,
			Formatters:     []httpretty.Formatter{&jsonFormatter{}, &formURLEncodedFormatter{}},
		}
		l.SkipHeader(logger.SensitiveHeaders)

		var buf bytes.Buffer
		l.SetOutput(&buf)
		inner := l.RoundTripper(rt)

		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			buf.Reset()
			start := time.Now()
			resp, err := inner.RoundTrip(req)
			elapsed := time.Since(start)
			if buf.Len() > 0 {
				msg := buf.String()
				if config.Timing {
					suffix := ""
					if resp != nil {
						suffix = fmt.Sprintf(" %d", resp.StatusCode)
					} else if err != nil {
						suffix = " error"
					}
					suffix += fmt.Sprintf(" %s", elapsed.Truncate(time.Millisecond))

					lines := strings.Split(msg, "\n")
					for i, line := range lines {
						if strings.Contains(line, req.Method) {
							lines[i] = strings.TrimRight(line, "\r\n") + suffix
							break
						}
					}
					msg = strings.Join(lines, "\n")
				}
				getLogger(req).Infof(strings.TrimSpace(msg))
			}
			return resp, err
		})
	}
}

func statusColor(code int) func(string, ...interface{}) string {
	if code >= 200 && code < 300 {
		return console.Greenf
	} else if code >= 400 {
		return console.Redf
	}
	return console.Yellowf
}

func newContextAccessLog() Middleware {
	return func(rt http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
			start := time.Now()
			resp, err := rt.RoundTrip(req)
			elapsed := time.Since(start)
			log := getLogger(req)
			if err != nil {
				log.Infof("%s %s %s %s", console.Bluef(req.Method), console.Yellowf("%s", req.URL), console.Redf("error"), elapsed.Truncate(time.Millisecond))
				return nil, err
			}
			log.Infof("%s %s %s %s", console.Bluef(req.Method), console.Yellowf("%s", req.URL), statusColor(resp.StatusCode)("%d", resp.StatusCode), elapsed.Truncate(time.Millisecond))
			return resp, nil
		})
	}
}

func NewLogger(config TraceConfig) Middleware {
	if config.AccessLog {
		return newContextAccessLog()
	}
	return newContextLogger(config)
}
