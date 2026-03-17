package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigestTransport(t *testing.T) {
	const (
		username = "testuser"
		password = "testpass"
		realm    = "testrealm"
		nonce    = "dcd98b7102dd2f0e8b11d0f600bfb0c093"
		opaque   = "5ccc069c403ebaf9f0171e9517f40e41"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("WWW-Authenticate",
				`Digest realm="`+realm+`", nonce="`+nonce+`", opaque="`+opaque+`", qop="auth", algorithm=MD5`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	transport := newDigestTransport(username, password, http.DefaultTransport)
	client := &http.Client{Transport: transport}

	resp, err := client.Get(server.URL + "/test")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestDigestTransportSecondRequestReusesAuth(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("WWW-Authenticate",
				`Digest realm="test", nonce="abc123", qop="auth", algorithm=MD5`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	transport := newDigestTransport("user", "pass", http.DefaultTransport)
	client := &http.Client{Transport: transport}

	// First request: 401 + retry = 2 server calls
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 2, callCount)

	// Second request: reuses cached auth = 1 additional server call
	resp, err = client.Get(server.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 3, callCount)
}

func TestParseWWWAuthenticate(t *testing.T) {
	header := `Digest realm="example.com", nonce="abc123", opaque="xyz", qop="auth", algorithm=SHA-256, userhash=true`
	wa, err := parseWWWAuthenticate(header)
	require.NoError(t, err)

	assert.Equal(t, "Digest", wa.authType)
	assert.Equal(t, "example.com", wa.realm)
	assert.Equal(t, "abc123", wa.nonce)
	assert.Equal(t, "xyz", wa.opaque)
	assert.Equal(t, "auth", wa.qop)
	assert.Equal(t, "SHA-256", wa.algorithm)
	assert.True(t, wa.userhash)
}

func TestParsePairs(t *testing.T) {
	input := `realm="test realm", nonce="abc", algorithm=MD5, qop="auth,auth-int"`
	pairs := parsePairs(input)

	assert.Equal(t, "test realm", pairs["realm"])
	assert.Equal(t, "abc", pairs["nonce"])
	assert.Equal(t, "MD5", pairs["algorithm"])
	assert.Equal(t, "auth,auth-int", pairs["qop"])
}

func TestDigestAuthString(t *testing.T) {
	a := &digestAuth{
		username:  "user",
		realm:     "realm",
		nonce:     "nonce123",
		uri:       "/path",
		response:  "resp",
		algorithm: "MD5",
		cnonce:    "cn",
		qop:       "auth",
		nc:        1,
	}

	s := a.String()
	assert.Contains(t, s, `username="user"`)
	assert.Contains(t, s, `realm="realm"`)
	assert.Contains(t, s, `nonce="nonce123"`)
	assert.Contains(t, s, `uri="/path"`)
	assert.Contains(t, s, `response="resp"`)
	assert.Contains(t, s, "algorithm=MD5")
	assert.Contains(t, s, `cnonce="cn"`)
	assert.Contains(t, s, "qop=auth")
	assert.Contains(t, s, "nc=00000001")
	assert.True(t, strings.HasPrefix(s, "Digest "))
}
