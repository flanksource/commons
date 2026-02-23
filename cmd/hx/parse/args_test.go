package parse

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPositionalArgs(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantURL   string
		wantItems []string
		wantErr   bool
	}{
		{
			name:    "URL only",
			args:    []string{"https://example.com"},
			wantURL: "https://example.com",
		},
		{
			name:      "method and URL",
			args:      []string{"GET", "https://example.com"},
			wantURL:   "https://example.com",
			wantItems: nil,
		},
		{
			name:      "URL with data items",
			args:      []string{"https://example.com", "name=test", "count:=5"},
			wantURL:   "https://example.com",
			wantItems: []string{"name=test", "count:=5"},
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "method only",
			args:    []string{"GET"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PositionalArgs(tt.args)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, got.URL)
			if tt.wantItems != nil {
				assert.Equal(t, tt.wantItems, got.Items)
			}
		})
	}
}

func TestEffectiveMethod(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		hasBody        bool
		methodOverride string
		want           string
	}{
		{name: "default GET", want: "GET"},
		{name: "POST when body present", hasBody: true, want: "POST"},
		{name: "explicit method", method: "PUT", want: "PUT"},
		{name: "override wins", method: "GET", methodOverride: "PATCH", want: "PATCH"},
		{name: "override case insensitive", methodOverride: "delete", want: "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Args{Method: tt.method}
			assert.Equal(t, tt.want, a.EffectiveMethod(tt.hasBody, tt.methodOverride))
		})
	}
}
