package logger

import (
	"reflect"
	"testing"
)

func Test_walkMap(t *testing.T) {
	tests := []struct {
		name string
		args map[string]any
		want map[string]any
	}{
		{
			name: "simple",
			args: map[string]any{
				"username": "james",
				"password": "secret",
			},
			want: map[string]any{
				"username": "****",
				"password": "****",
			},
		},
		{
			name: "empty remover",
			args: map[string]any{
				"username": nil,
				"password": "secret",
			},
			want: map[string]any{
				"password": "****",
			},
		},
		{
			name: "nested",
			args: map[string]any{
				"auth": map[string]any{
					"role":     "editor",
					"username": "james",
					"password": "secret",
				},
				"token": "secret",
			},
			want: map[string]any{
				"auth": map[string]any{
					"role":     "editor",
					"username": "****",
					"password": "****",
				},
				"token": "****",
			},
		},
		{
			name: "nested level 3",
			args: map[string]any{
				"auth": map[string]any{
					"role": "editor",
					"cred": map[string]any{
						"username": "james",
						"password": "secret",
					},
				},
				"token": "secret",
			},
			want: map[string]any{
				"auth": map[string]any{
					"role": "editor",
					"cred": map[string]any{
						"username": "****",
						"password": "****",
					},
				},
				"token": "****",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := StripSecretsFromMap(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("walkMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
