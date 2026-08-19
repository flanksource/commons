package logger

import (
	"io"
	"log/slog"
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
				"username": "j****",
				"password": "s****",
			},
		},
		{
			name: "empty remover",
			args: map[string]any{
				"username": nil,
				"password": "secret",
			},
			want: map[string]any{
				"password": "s****",
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
					"username": "j****",
					"password": "s****",
				},
				"token": "s****",
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
						"username": "j****",
						"password": "s****",
					},
				},
				"token": "s****",
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

func TestIsJSONLoggerFollowsConcreteLogger(t *testing.T) {
	newLogger := func(json bool) SlogLogger {
		return SlogLogger{
			Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
			Level:  &slog.LevelVar{},
			json:   json,
		}
	}
	plain := newLogger(false)
	jsonLogger := newLogger(true)

	if IsJSONLogger(plain) {
		t.Fatal("plain logger reported JSON output")
	}
	if !IsJSONLogger(jsonLogger) {
		t.Fatal("JSON logger reported pretty output")
	}
	if !IsJSONLogger(jsonLogger.WithValues("provider", "postgres")) {
		t.Fatal("WithValues lost JSON output mode")
	}
	if !IsJSONLogger(jsonLogger.WithV(Trace)) {
		t.Fatal("WithV lost JSON output mode")
	}
}
