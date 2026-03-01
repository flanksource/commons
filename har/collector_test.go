package har_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/flanksource/commons/har"
	commonshttp "github.com/flanksource/commons/http"
)

func TestCollector_AccumulatesMultipleEntries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	collector := har.NewCollector(har.DefaultConfig())
	client := commonshttp.NewClient().HARCollector(collector)

	for i := range 3 {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/req/%d", srv.URL, i), nil)
		resp, err := client.RoundTrip(req)
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		resp.Body.Close()
	}

	entries := collector.Entries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	for i, e := range entries {
		if e.Response.Status != 200 {
			t.Errorf("entry %d: expected status 200, got %d", i, e.Response.Status)
		}
	}
}

func TestCollector_RetryAccumulatesEntries(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount < 3 {
			// Close connection abruptly to cause a transport error
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server doesn't support hijacking")
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	collector := har.NewCollector(har.DefaultConfig())
	client := commonshttp.NewClient().
		Retry(3, 0, 1.0).
		HARCollector(collector)

	resp, err := client.R(context.Background()).Get(srv.URL + "/retry")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	entries := collector.Entries()
	// The middleware captures each roundTrip call. Failed attempts still produce
	// entries (with no response), and the final success also produces one.
	if len(entries) < 1 {
		t.Fatal("expected at least 1 entry from successful attempt")
	}
	// The last entry should be the successful response
	last := entries[len(entries)-1]
	if last.Response.Status != 200 {
		t.Errorf("last entry should be 200, got %d", last.Response.Status)
	}
}

func TestCollector_RedirectCapturesAllHops(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/start":
			http.Redirect(w, r, "/middle", http.StatusFound)
		case "/middle":
			http.Redirect(w, r, "/end", http.StatusFound)
		case "/end":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			fmt.Fprint(w, `{"done":true}`)
		}
	}))
	defer srv.Close()

	collector := har.NewCollector(har.DefaultConfig())
	client := commonshttp.NewClient().
		HARCollector(collector).
		RedirectPolicy(5)

	resp, err := client.R(context.Background()).Get(srv.URL + "/start")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	entries := collector.Entries()
	// 2 redirect hops captured by CheckRedirect + 1 final request captured by middleware
	if len(entries) < 3 {
		t.Fatalf("expected at least 3 entries (2 redirects + final), got %d", len(entries))
	}
}

func TestCollector_OAuthTokenRequestCaptured(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"access_token":"test-token","token_type":"bearer","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"data":"ok"}`)
	}))
	defer apiSrv.Close()

	collector := har.NewCollector(har.DefaultConfig())
	client := commonshttp.NewClient().
		HARCollector(collector).
		OAuth(commonshttp.OauthConfig{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			TokenURL:     tokenSrv.URL + "/token",
		})

	resp, err := client.R(context.Background()).Get(apiSrv.URL + "/api")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	entries := collector.Entries()
	// Should have at least 2 entries: the OAuth token fetch + the API request
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 entries (token + api), got %d", len(entries))
	}

	// Find the token request entry
	foundToken := false
	for _, e := range entries {
		if e.Request.Method == "POST" && e.Response.Status == 200 {
			if e.Response.Content.Text != "" {
				foundToken = true
			}
		}
	}
	if !foundToken {
		t.Error("expected to find OAuth token request entry in HAR")
	}
}

func TestCollector_OAuthHeaderCapturedInAPIRequest(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"access_token":"test-token-value","token_type":"bearer","expires_in":3600}`)
	}))
	defer tokenSrv.Close()

	var gotAuthHeader string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		fmt.Fprint(w, `{"data":"ok"}`)
	}))
	defer apiSrv.Close()

	collector := har.NewCollector(har.DefaultConfig())
	client := commonshttp.NewClient().
		HARCollector(collector).
		OAuth(commonshttp.OauthConfig{
			ClientID:     "test-client",
			ClientSecret: "test-secret",
			TokenURL:     tokenSrv.URL + "/token",
		})

	resp, err := client.R(context.Background()).Get(apiSrv.URL + "/api")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if gotAuthHeader != "Bearer test-token-value" {
		t.Fatalf("API server did not receive expected Authorization header, got %q", gotAuthHeader)
	}

	// Find the API request entry (GET to /api)
	var apiEntry *har.Entry
	for _, e := range collector.Entries() {
		if e.Request.Method == "GET" {
			apiEntry = &e
			break
		}
	}
	if apiEntry == nil {
		t.Fatal("expected to find API request entry in HAR")
	}

	// Verify Authorization header is present (redacted) in the HAR entry
	foundAuth := false
	for _, h := range apiEntry.Request.Headers {
		if h.Name == "Authorization" {
			foundAuth = true
			if h.Value == "Bearer test-token-value" {
				t.Error("Authorization header should be redacted, got raw value")
			}
			// Should be PrintableSecret format like "Bearer t****e"
			if h.Value == "" {
				t.Error("Authorization header should not be empty")
			}
		}
	}
	if !foundAuth {
		t.Error("expected Authorization header in API request HAR entry")
	}
}

func TestRedirectPolicy_NoFollow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/target", http.StatusFound)
	}))
	defer srv.Close()

	client := commonshttp.NewClient().RedirectPolicy(0)

	resp, err := client.R(context.Background()).Get(srv.URL + "/start")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302, got %d", resp.StatusCode)
	}
}
