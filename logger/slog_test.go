package logger

import (
	"bytes"
	"encoding/json"
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
	originalFlags := *flags
	originalOutput := GetOutput()
	t.Cleanup(func() {
		*flags = originalFlags
		SetOutput(originalOutput)
	})
	flags.jsonLogs = true
	flags.color = false

	assertRecord := func(output *bytes.Buffer, message, name string) {
		t.Helper()
		var record map[string]any
		if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
			t.Fatalf("decode JSON log record: %v\noutput: %s", err, output.String())
		}
		if record["msg"] != message {
			t.Fatalf("message = %v, want %q", record["msg"], message)
		}
		if name != "" && record["logger"] != name {
			t.Fatalf("logger = %v, want %q", record["logger"], name)
		}
	}

	var sharedOutput bytes.Buffer
	SetOutput(&sharedOutput)
	namedLogger := New("constructor")
	namedLogger.Infof("from New")
	if !IsJSONLogger(namedLogger) {
		t.Fatal("New did not report JSON output")
	}
	assertRecord(&sharedOutput, "from New", "constructor")

	var writerOutput bytes.Buffer
	jsonLogger := NewWithWriter(&writerOutput)
	jsonLogger.Infof("from NewWithWriter")
	if !IsJSONLogger(jsonLogger) {
		t.Fatal("NewWithWriter did not report JSON output")
	}
	assertRecord(&writerOutput, "from NewWithWriter", "")

	if !IsJSONLogger(jsonLogger.WithValues("provider", "postgres")) {
		t.Fatal("WithValues lost JSON output mode")
	}
	if !IsJSONLogger(jsonLogger.WithV(Trace)) {
		t.Fatal("WithV lost JSON output mode")
	}
}
