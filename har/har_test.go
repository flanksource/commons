package har

import (
	"testing"

	"github.com/flanksource/commons/properties"
)

func TestDefaultConfig_MaxBodySizeProperty(t *testing.T) {
	cases := []struct {
		name  string
		value string
		set   bool
		want  int64
	}{
		{name: "unset keeps default", set: false, want: defaultMaxBodySize},
		{name: "override lowers cap", value: "1048576", set: true, want: 1048576},
		{name: "IEC suffix lowers cap", value: "1MiB", set: true, want: 1024 * 1024},
		{name: "SI suffix lowers cap", value: "2MB", set: true, want: 2 * 1000 * 1000},
		{name: "zero disables truncation", value: "0", set: true, want: 0},
		{name: "unparseable keeps default", value: "huge", set: true, want: defaultMaxBodySize},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := ""
			if tc.set {
				value = tc.value
			}
			properties.Set(MaxBodySizeProperty, value)
			defer properties.Set(MaxBodySizeProperty, "")

			if got := DefaultConfig().MaxBodySize; got != tc.want {
				t.Errorf("MaxBodySize = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestDefaultMaxBodySizeIsFourMiB(t *testing.T) {
	if defaultMaxBodySize != 4*1024*1024 {
		t.Fatalf("defaultMaxBodySize = %d, want 4194304", defaultMaxBodySize)
	}
	if MaxBodySizeProperty != "http.har.response.body.length" {
		t.Fatalf("MaxBodySizeProperty = %q, want http.har.response.body.length", MaxBodySizeProperty)
	}
}
