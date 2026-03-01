package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /get", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"url": r.URL.String()})
	})

	mux.HandleFunc("GET /bearer", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeJSON(w, map[string]any{"token": auth})
	})

	mux.HandleFunc("GET /status/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := 200
		if c := r.PathValue("code"); c != "" {
			_ = json.Unmarshal([]byte(c), &code)
		}
		w.WriteHeader(code)
	})

	return httptest.NewServer(mux)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func harFixturePath(t *testing.T) string {
	t.Helper()
	dir := "testdata"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, t.Name()+".har")
}

func TestHARExportToFile(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	path := harFixturePath(t)
	flagHAROutput = path
	flagQuiet = true
	defer func() { flagHAROutput = ""; flagQuiet = false }()

	require.NoError(t, run(rootCmd, []string{srv.URL + "/get"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var file map[string]any
	require.NoError(t, json.Unmarshal(data, &file))

	harLog := file["log"].(map[string]any)
	assert.Equal(t, "1.2", harLog["version"])
	assert.NotNil(t, harLog["pages"], "pages key must be present for HAR spec compliance")
	entries := harLog["entries"].([]any)
	require.Len(t, entries, 1)

	entry := entries[0].(map[string]any)
	reqMap := entry["request"].(map[string]any)
	assert.Equal(t, "GET", reqMap["method"])
	assert.Contains(t, reqMap["url"], "/get")

	respMap := entry["response"].(map[string]any)
	assert.Equal(t, float64(200), respMap["status"])
}

func TestHARExportAuthRedacted(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	path := harFixturePath(t)
	flagHAROutput = path
	flagQuiet = true
	flagToken = "supersecrettoken"
	defer func() { flagHAROutput = ""; flagQuiet = false; flagToken = "" }()

	require.NoError(t, run(rootCmd, []string{srv.URL + "/bearer"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "supersecrettoken", "token must be redacted in HAR output")
}

func TestNonOKStatusReturnsError(t *testing.T) {
	srv := testServer()
	defer srv.Close()

	tests := []struct {
		name       string
		code       int
		wantErr    bool
		wantStatus string
	}{
		{name: "200 returns nil", code: 200, wantErr: false},
		{name: "201 returns nil", code: 201, wantErr: false},
		{name: "301 returns nil", code: 301, wantErr: false},
		{name: "404 returns error", code: 404, wantErr: true, wantStatus: "404 Not Found"},
		{name: "500 returns error", code: 500, wantErr: true, wantStatus: "500 Internal Server Error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flagQuiet = true
			flagNoFollow = true
			defer func() { flagQuiet = false; flagNoFollow = false }()

			err := run(rootCmd, []string{fmt.Sprintf("%s/status/%d", srv.URL, tc.code)})

			if !tc.wantErr {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			var statusErr *httpStatusError
			require.True(t, errors.As(err, &statusErr), "expected httpStatusError, got %T: %v", err, err)
			assert.Equal(t, tc.code, statusErr.code)
			assert.Equal(t, tc.wantStatus, statusErr.status)
		})
	}
}
