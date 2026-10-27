package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/flanksource/commons/har"
	"github.com/flanksource/commons/har/harpretty"
)

func main() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, `{"status":"ok"}`); err != nil {
			panic(err)
		}
	}))
	defer server.Close()
	collector := har.NewCollector(har.DefaultConfig())
	client := &http.Client{Transport: collector.Middleware()(http.DefaultTransport)}
	response, err := client.Get(server.URL + "/status?token=example-secret")
	if err != nil {
		panic(err)
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		panic(err)
	}
	if err := response.Body.Close(); err != nil {
		panic(err)
	}
	entries := collector.Entries()
	fmt.Printf("Captured %d request\n", len(entries))
	for _, entry := range entries {
		fmt.Println(harpretty.Entry(entry).Pretty().String())
	}
}
