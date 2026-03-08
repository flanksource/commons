package har_test

import (
	"strings"
	"testing"

	"github.com/flanksource/commons/har"
)

func TestEntry_Pretty(t *testing.T) {
	entry := har.Entry{
		Time: 150.5,
		Request: har.Request{
			Method:      "POST",
			URL:         "https://api.example.com/v1/users",
			HTTPVersion: "HTTP/1.1",
		},
		Response: har.Response{
			Status:     201,
			StatusText: "Created",
			Content:    har.Content{Size: 42},
		},
	}

	output := entry.Pretty().String()
	for _, want := range []string{"POST", "https://api.example.com/v1/users", "201", "Created"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output: %s", want, output)
		}
	}
}

func TestEntry_Row(t *testing.T) {
	entry := har.Entry{
		Time: 100,
		Request: har.Request{
			Method: "GET",
			URL:    "https://example.com/api",
		},
		Response: har.Response{
			Status:     200,
			StatusText: "OK",
			Content:    har.Content{Size: 1024},
		},
	}

	row := entry.Row()
	if row["method"] != "GET" {
		t.Errorf("expected method=GET, got %v", row["method"])
	}
	if row["url"] != "https://example.com/api" {
		t.Errorf("expected url, got %v", row["url"])
	}
	if row["status"] == nil {
		t.Error("expected status to be set")
	}
	if row["duration"] == nil {
		t.Error("expected duration to be set")
	}
	if row["size"] == nil {
		t.Error("expected size to be set for non-zero content")
	}
}

func TestEntry_Columns(t *testing.T) {
	cols := har.Entry{}.Columns()
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}
	want := []string{"method", "url", "status", "duration", "size"}
	for i, w := range want {
		if names[i] != w {
			t.Errorf("column %d: expected %q, got %q", i, w, names[i])
		}
	}
}

func TestEntry_RowDetail_WithContent(t *testing.T) {
	entry := har.Entry{
		Request: har.Request{
			Method: "POST",
			URL:    "/api",
			Headers: []har.Header{
				{Name: "Content-Type", Value: "application/json"},
				{Name: "Accept", Value: "*/*"},
			},
			PostData: &har.PostData{
				MimeType: "application/json",
				Text:     `{"name":"test"}`,
			},
		},
		Response: har.Response{
			Status: 200,
			Headers: []har.Header{
				{Name: "Content-Type", Value: "application/json"},
			},
			Content: har.Content{
				MimeType: "application/json",
				Text:     `{"id":1}`,
			},
		},
	}

	detail := entry.RowDetail()
	if detail == nil {
		t.Fatal("expected non-nil RowDetail")
	}

	output := detail.String()
	for _, want := range []string{
		"Request Headers", "Content-Type", "Accept",
		"Request Body", `"name": "test"`,
		"Response Headers",
		"Response Body", `"id": 1`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in detail: %s", want, output)
		}
	}
}

func TestEntry_RowDetail_Truncated(t *testing.T) {
	entry := har.Entry{
		Request:  har.Request{Method: "GET", URL: "/big"},
		Response: har.Response{Status: 200, Content: har.Content{Text: "{}", Truncated: true}},
	}

	output := entry.RowDetail().String()
	if !strings.Contains(output, "truncated") {
		t.Errorf("expected truncated label, got: %s", output)
	}
}

func TestEntry_RowDetail_Empty(t *testing.T) {
	entry := har.Entry{
		Request:  har.Request{Method: "GET", URL: "/"},
		Response: har.Response{Status: 200},
	}

	if detail := entry.RowDetail(); detail != nil {
		t.Errorf("expected nil RowDetail for entry with no headers/bodies, got: %v", detail)
	}
}

func TestCollector_Table(t *testing.T) {
	collector := har.NewCollector(har.DefaultConfig())
	collector.Add(&har.Entry{
		Time:    100,
		Request: har.Request{Method: "GET", URL: "https://example.com/a"},
		Response: har.Response{
			Status: 200, StatusText: "OK",
			Headers: []har.Header{{Name: "X-Req-Id", Value: "abc"}},
			Content: har.Content{Size: 10, Text: "hello"},
		},
	})
	collector.Add(&har.Entry{
		Time:     50,
		Request:  har.Request{Method: "POST", URL: "https://example.com/b"},
		Response: har.Response{Status: 500, StatusText: "Internal Server Error"},
	})

	table := collector.Table()

	if len(table.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table.Rows))
	}
	if len(table.Headers) != 5 {
		t.Fatalf("expected 5 headers, got %d", len(table.Headers))
	}

	// First entry has response headers+body -> RowDetail should be populated
	if table.RowDetail == nil {
		t.Fatal("expected RowDetail to be populated")
	}
	if table.RowDetail[0] == nil {
		t.Error("first entry has headers/body, expected non-nil detail")
	}
	// Second entry has no headers/body
	if table.RowDetail[1] != nil {
		t.Error("second entry has no headers/body, expected nil detail")
	}
}

func TestCollector_Pretty(t *testing.T) {
	collector := har.NewCollector(har.DefaultConfig())
	collector.Add(&har.Entry{
		Time:     100,
		Request:  har.Request{Method: "GET", URL: "https://example.com/a"},
		Response: har.Response{Status: 200, StatusText: "OK"},
	})
	collector.Add(&har.Entry{
		Time:     200,
		Request:  har.Request{Method: "POST", URL: "https://example.com/b"},
		Response: har.Response{Status: 500, StatusText: "Internal Server Error"},
	})

	output := collector.Pretty().String()
	if !strings.Contains(output, "GET") || !strings.Contains(output, "500") {
		t.Errorf("expected both entries in output: %s", output)
	}
	if !strings.Contains(output, "───") {
		t.Errorf("expected separator: %s", output)
	}
}
