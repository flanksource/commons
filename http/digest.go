package http

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var errAuthRetryNeeded = errors.New("retry request with authentication")

const digestReadLimit int64 = 128 * 1024

type digestTransport struct {
	username  string
	password  string
	auth      *digestAuth
	mu        sync.Mutex
	transport http.RoundTripper
}

func newDigestTransport(username, password string, transport http.RoundTripper) *digestTransport {
	return &digestTransport{
		username:  username,
		password:  password,
		transport: transport,
	}
}

func (dt *digestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := req.Clone(req.Context())
	if req.Body != nil {
		defer req.Body.Close()
	}

	var bodyRead *bytes.Buffer
	var bodyLeft io.Reader
	if req.Body != nil && req.GetBody == nil {
		bodyRead = new(bytes.Buffer)
		bodyLeft = io.TeeReader(req.Body, bodyRead)
		reqCopy.Body = io.NopCloser(bodyLeft)
	}

	resp, err := dt.tryReq(reqCopy)
	if err == nil {
		return resp, nil
	}
	if !errors.Is(err, errAuthRetryNeeded) {
		return nil, err
	}

	drainBody(resp.Body)

	if req.Body != nil {
		if req.GetBody == nil {
			reqCopy.Body = io.NopCloser(io.MultiReader(bodyRead, bodyLeft))
		} else {
			newBody, err := req.GetBody()
			if err != nil {
				return nil, err
			}
			reqCopy.Body = newBody
		}
	}

	resp, err = dt.tryReq(reqCopy)
	if errors.Is(err, errAuthRetryNeeded) {
		return resp, nil
	}
	return resp, err
}

func (dt *digestTransport) tryReq(req *http.Request) (*http.Response, error) {
	dt.mu.Lock()
	auth := dt.auth

	if auth != nil {
		auth.refresh(req)
		dt.mu.Unlock()
		resp, err := dt.transport.RoundTrip(setDigestHeader(req, auth.String()))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 401 {
			return resp, nil
		}
		return dt.handleChallenge(resp, req)
	}

	dt.mu.Unlock()
	resp, err := dt.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 401 {
		return resp, nil
	}
	return dt.handleChallenge(resp, req)
}

func (dt *digestTransport) handleChallenge(resp *http.Response, req *http.Request) (*http.Response, error) {
	waString := resp.Header.Get("WWW-Authenticate")
	if waString == "" {
		return resp, nil
	}

	wa, err := parseWWWAuthenticate(waString)
	if err != nil {
		return nil, err
	}
	if wa.authType != "Digest" {
		return resp, nil
	}

	auth := newDigestAuth(wa, dt.username, dt.password, req)

	dt.mu.Lock()
	dt.auth = auth
	dt.mu.Unlock()

	return resp, errAuthRetryNeeded
}

func setDigestHeader(req *http.Request, authString string) *http.Request {
	req.Header.Set("Authorization", authString)
	return req
}

func drainBody(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, &io.LimitedReader{R: body, N: digestReadLimit})
	body.Close()
}

// --- digest auth computation ---

type digestAuth struct {
	algorithm string
	cnonce    string
	nc        int
	nonce     string
	opaque    string
	qop       string
	realm     string
	response  string
	uri       string
	userhash  bool
	username  string
	password  string
	userHash  string
}

const (
	algMD5        = "MD5"
	algMD5Sess    = "MD5-SESS"
	algSHA256     = "SHA-256"
	algSHA256Sess = "SHA-256-SESS"
)

func newDigestAuth(wa *wwwAuthenticate, username, password string, req *http.Request) *digestAuth {
	a := &digestAuth{
		algorithm: wa.algorithm,
		nonce:     wa.nonce,
		opaque:    wa.opaque,
		realm:     wa.realm,
		userhash:  wa.userhash,
		username:  username,
		password:  password,
	}
	a.refresh(req)
	return a
}

func (a *digestAuth) refresh(req *http.Request) {
	if a.userhash {
		a.userHash = a.hashStr(fmt.Sprintf("%s:%s", a.username, a.realm))
	}
	a.nc++
	a.cnonce = a.hashStr(fmt.Sprintf("%d:%s:my_value", time.Now().UnixNano(), a.username))
	a.uri = req.URL.RequestURI()
	a.response = a.computeResponse(req)
}

func (a *digestAuth) computeResponse(req *http.Request) string {
	kdSecret := a.hashStr(a.computeA1())
	kdData := fmt.Sprintf("%s:%08x:%s:%s:%s", a.nonce, a.nc, a.cnonce, a.qop, a.hashStr(a.computeA2(req)))
	return a.hashStr(fmt.Sprintf("%s:%s", kdSecret, kdData))
}

func (a *digestAuth) computeA1() string {
	alg := strings.ToUpper(a.algorithm)
	switch alg {
	case "", algMD5, algSHA256:
		return fmt.Sprintf("%s:%s:%s", a.username, a.realm, a.password)
	case algMD5Sess, algSHA256Sess:
		upHash := a.hashStr(fmt.Sprintf("%s:%s:%s", a.username, a.realm, a.password))
		return fmt.Sprintf("%s:%s:%s", upHash, a.nonce, a.cnonce)
	default:
		return ""
	}
}

func (a *digestAuth) computeA2(req *http.Request) string {
	if strings.Contains(a.qop, "auth-int") {
		h := a.hashStr("")
		if req.Body != nil {
			buf := new(bytes.Buffer)
			_, _ = buf.ReadFrom(req.Body)
			h = a.hashStr(buf.String())
			req.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
		}
		a.qop = "auth-int"
		return fmt.Sprintf("%s:%s:%s", req.Method, a.uri, h)
	}
	a.qop = "auth"
	return fmt.Sprintf("%s:%s", req.Method, a.uri)
}

func (a *digestAuth) hashStr(s string) string {
	var h hash.Hash
	alg := strings.ToUpper(a.algorithm)
	switch alg {
	case "", algMD5, algMD5Sess:
		h = md5.New()
	case algSHA256, algSHA256Sess:
		h = sha256.New()
	default:
		return ""
	}
	_, _ = io.WriteString(h, s)
	return hex.EncodeToString(h.Sum(nil))
}

func (a *digestAuth) String() string {
	var b strings.Builder
	b.WriteString("Digest ")

	if a.userhash {
		fmt.Fprintf(&b, "username=\"%s\", ", a.userHash)
	} else {
		fmt.Fprintf(&b, "username=\"%s\", ", a.username)
	}
	if a.realm != "" {
		fmt.Fprintf(&b, "realm=\"%s\", ", a.realm)
	}
	if a.nonce != "" {
		fmt.Fprintf(&b, "nonce=\"%s\", ", a.nonce)
	}
	if a.uri != "" {
		fmt.Fprintf(&b, "uri=\"%s\", ", a.uri)
	}
	if a.response != "" {
		fmt.Fprintf(&b, "response=\"%s\", ", a.response)
	}
	if a.algorithm != "" {
		fmt.Fprintf(&b, "algorithm=%s, ", a.algorithm)
	}
	if a.cnonce != "" {
		fmt.Fprintf(&b, "cnonce=\"%s\", ", a.cnonce)
	}
	if a.opaque != "" {
		fmt.Fprintf(&b, "opaque=\"%s\", ", a.opaque)
	}
	if a.qop != "" {
		fmt.Fprintf(&b, "qop=%s, ", a.qop)
	}
	if a.nc != 0 {
		fmt.Fprintf(&b, "nc=%08x, ", a.nc)
	}
	if a.userhash {
		b.WriteString("userhash=true, ")
	}

	return strings.TrimSuffix(b.String(), ", ")
}

// --- WWW-Authenticate header parsing ---

type wwwAuthenticate struct {
	algorithm string
	domain    string
	nonce     string
	opaque    string
	qop       string
	realm     string
	stale     bool
	charset   string
	userhash  bool
	authType  string
}

func parseWWWAuthenticate(header string) (*wwwAuthenticate, error) {
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return nil, errors.New("bad WWW-Authenticate header")
	}

	vals := parsePairs(parts[1])

	return &wwwAuthenticate{
		algorithm: vals["algorithm"],
		domain:    vals["domain"],
		nonce:     vals["nonce"],
		opaque:    vals["opaque"],
		qop:       vals["qop"],
		realm:     vals["realm"],
		stale:     strings.EqualFold(vals["stale"], "true"),
		charset:   vals["charset"],
		userhash:  strings.EqualFold(vals["userhash"], "true"),
		authType:  parts[0],
	}, nil
}

func parseList(value string) []string {
	var list []string
	var escape, quote bool
	b := new(bytes.Buffer)
	for _, r := range value {
		switch {
		case escape:
			b.WriteRune(r)
			escape = false
		case quote:
			if r == '\\' {
				escape = true
			} else {
				if r == '"' {
					quote = false
				}
				b.WriteRune(r)
			}
		case r == ',':
			list = append(list, strings.TrimSpace(b.String()))
			b.Reset()
		case r == '"':
			quote = true
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	if s := b.String(); s != "" {
		list = append(list, strings.TrimSpace(s))
	}
	return list
}

func parsePairs(value string) map[string]string {
	m := make(map[string]string)
	for _, pair := range parseList(strings.TrimSpace(value)) {
		i := strings.Index(pair, "=")
		switch {
		case i < 0:
			m[pair] = ""
		case i == len(pair)-1:
			m[pair[:i]] = ""
		default:
			v := pair[i+1:]
			if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
				v = v[1 : len(v)-1]
			}
			m[pair[:i]] = v
		}
	}
	return m
}
