package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItems(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantHeaders map[string]string
		wantParams  map[string]string
		wantBody    map[string]any
		wantErr     bool
	}{
		{
			name:     "string value",
			args:     []string{"name=John"},
			wantBody: map[string]any{"name": "John"},
		},
		{
			name:     "raw JSON number",
			args:     []string{"count:=42"},
			wantBody: map[string]any{"count": float64(42)},
		},
		{
			name:     "raw JSON bool",
			args:     []string{"active:=true"},
			wantBody: map[string]any{"active": true},
		},
		{
			name:     "raw JSON array",
			args:     []string{`tags:=["a","b"]`},
			wantBody: map[string]any{"tags": []any{"a", "b"}},
		},
		{
			name:       "query param",
			args:       []string{"page==2"},
			wantParams: map[string]string{"page": "2"},
		},
		{
			name:        "header",
			args:        []string{"Content-Type:application/json"},
			wantHeaders: map[string]string{"Content-Type": "application/json"},
		},
		{
			name:        "header with space after colon",
			args:        []string{"Accept: text/html"},
			wantHeaders: map[string]string{"Accept": "text/html"},
		},
		{
			name: "mixed items",
			args: []string{"name=test", "count:=5", "page==1", "X-Token:abc"},
			wantBody: map[string]any{
				"name":  "test",
				"count": float64(5),
			},
			wantParams:  map[string]string{"page": "1"},
			wantHeaders: map[string]string{"X-Token": "abc"},
		},
		{
			name:    "invalid raw JSON",
			args:    []string{"bad:=notjson"},
			wantErr: true,
		},
		{
			name:    "unparseable item",
			args:    []string{"justtext"},
			wantErr: true,
		},
		{
			name:     "value with equals sign",
			args:     []string{"query=a=b"},
			wantBody: map[string]any{"query": "a=b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Items(tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			if tt.wantHeaders != nil {
				assert.Equal(t, tt.wantHeaders, got.Headers)
			}
			if tt.wantParams != nil {
				assert.Equal(t, tt.wantParams, got.QueryParams)
			}
			if tt.wantBody != nil {
				assert.Equal(t, tt.wantBody, got.Body)
			}
		})
	}
}

func TestParsedItemsBodyJSON(t *testing.T) {
	items := &ParsedItems{
		Body: map[string]any{"name": "test", "count": float64(5)},
	}
	data, err := items.BodyJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"name":"test"`)
	assert.Contains(t, string(data), `"count":5`)
}

func TestParsedItemsHasBody(t *testing.T) {
	empty := &ParsedItems{Body: map[string]any{}}
	assert.False(t, empty.HasBody())

	filled := &ParsedItems{Body: map[string]any{"k": "v"}}
	assert.True(t, filled.HasBody())
}
