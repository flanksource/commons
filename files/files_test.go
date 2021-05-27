package files

import "testing"

func TestIsValidPathType(t *testing.T) {
	type args struct {
		input      string
		extensions []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"workingyaml", args{"patch1.yaml", []string{"yaml", "yml", "json"}}, true},
		{"workingyml", args{"patch1.yml", []string{"yaml", "yml", "json"}}, true},
		{"workingjson", args{"patch1.json", []string{"yaml", "yml", "json"}}, true},
		{"wrongext", args{"patch1.txt", []string{"yaml", "yml", "json"}}, false},
		{"israw", args{"Kind: pod\nMetadata:\n  name: test", []string{"yaml", "yml", "json"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPathType(tt.args.input, tt.args.extensions...); got != tt.want {
				t.Errorf("IsValidPathType() = %v, want %v", got, tt.want)
			}
		})
	}
}
