package output

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/flanksource/clicky/api"
	"golang.org/x/term"
)

type Mode int

const (
	ModeAuto    Mode = iota
	ModeQuiet        // body only, no decoration
	ModeHeaders      // headers only
	ModeRaw          // no colors
)

type Options struct {
	Mode      Mode
	Verbosity int // 0=default, 1=-v, 2=-vv, 3=-vvv
}

func (o Options) IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func (o Options) UseColor() bool {
	if o.Mode == ModeRaw || o.Mode == ModeQuiet {
		return false
	}
	return o.IsTTY()
}

func PrintResponse(resp *http.Response, body []byte, opts Options) error {
	w := os.Stdout

	if opts.Mode == ModeQuiet || (!opts.IsTTY() && opts.Mode == ModeAuto && opts.Verbosity == 0) {
		_, err := w.Write(body)
		if len(body) > 0 && body[len(body)-1] != '\n' {
			fmt.Fprintln(w)
		}
		return err
	}

	if opts.Verbosity >= 2 && resp.TLS != nil {
		printTLS(w, resp.TLS, opts.UseColor())
	}

	if opts.Verbosity >= 1 || opts.Mode == ModeHeaders {
		printStatusLine(w, resp, opts.UseColor())
		printHeaders(w, resp.Header, opts.UseColor())
		if opts.Mode == ModeHeaders {
			return nil
		}
		fmt.Fprintln(w)
	}

	return printBody(w, body, resp.Header.Get("Content-Type"), opts.UseColor())
}

func PrintRequest(req *http.Request, body []byte, opts Options) {
	if opts.Verbosity < 1 {
		return
	}
	w := os.Stdout
	useColor := opts.UseColor()

	line := fmt.Sprintf("%s %s %s", req.Method, req.URL.RequestURI(), req.Proto)
	if useColor {
		fmt.Fprintln(w, api.Text{Content: line, Style: "text-blue-400 font-bold"}.ANSI())
	} else {
		fmt.Fprintln(w, line)
	}

	printHeaders(w, req.Header, useColor)

	if len(body) > 0 {
		fmt.Fprintln(w)
		_ = printBody(w, body, req.Header.Get("Content-Type"), useColor)
	}
	fmt.Fprintln(w)
}

func printStatusLine(w io.Writer, resp *http.Response, useColor bool) {
	line := fmt.Sprintf("%s %s", resp.Proto, resp.Status)
	if !useColor {
		fmt.Fprintln(w, line)
		return
	}

	style := statusStyle(resp.StatusCode)
	fmt.Fprintln(w, api.Text{Content: line, Style: style}.ANSI())
}

func printHeaders(w io.Writer, headers http.Header, useColor bool) {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := strings.Join(headers[k], ", ")
		if useColor {
			fmt.Fprintln(w, api.Text{}.
				Append(k+": ", "text-cyan-400").
				Append(v).ANSI())
		} else {
			fmt.Fprintf(w, "%s: %s\n", k, v)
		}
	}
}

func printTLS(w io.Writer, state *tls.ConnectionState, useColor bool) {
	if len(state.PeerCertificates) == 0 {
		line := fmt.Sprintf("TLS: %s", tlsVersionName(state.Version))
		if useColor {
			fmt.Fprintln(w, api.Text{Content: line, Style: "text-green-400"}.ANSI())
		} else {
			fmt.Fprintln(w, line)
		}
		return
	}

	cert := state.PeerCertificates[0]
	line := fmt.Sprintf("TLS: CN=%s, notafter=%s, issuedBy=%s",
		cert.Subject.CommonName,
		cert.NotAfter.Format("2006-01-02"),
		cert.Issuer.CommonName)

	style := "text-green-400"
	if time.Now().After(cert.NotAfter) || time.Now().Before(cert.NotBefore) {
		style = "text-red-400"
	}

	if useColor {
		fmt.Fprintln(w, api.Text{Content: line, Style: style}.ANSI())
	} else {
		fmt.Fprintln(w, line)
	}
}

func tlsVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("unknown (0x%04x)", version)
	}
}

func printBody(w io.Writer, body []byte, contentType string, useColor bool) error {
	if len(body) == 0 {
		return nil
	}

	if isJSON(contentType) || json.Valid(body) {
		var indented json.RawMessage
		if err := json.Unmarshal(body, &indented); err == nil {
			pretty, err := json.MarshalIndent(indented, "", "  ")
			if err == nil {
				body = pretty
			}
		}

		if useColor {
			fmt.Fprintln(w, api.CodeBlock("json", string(body)).ANSI())
			return nil
		}
	}

	_, err := w.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		fmt.Fprintln(w)
	}
	return err
}

func isJSON(contentType string) bool {
	return strings.Contains(contentType, "json")
}

func statusStyle(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "text-green-400 font-bold"
	case code >= 300 && code < 400:
		return "text-yellow-400 font-bold"
	default:
		return "text-red-400 font-bold"
	}
}
