package files

import (
	"fmt"
	"testing"
)

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

func TestResolveFile(t *testing.T) {
	type args struct {
		file string
		dest string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		//	{"incorrectDestination", args{"/Users/mrinalwahal/go/src/github.com/flanksource/regen", "/Users/mrinalwahal/go/src/github.com/flanksource/regen/file.temp"}, true},
		//	{"correctFile", args{"/Users/mrinalwahal/go/src/github.com/flanksource/regen/file.temp", ""}, false},
		//	{"incorrectFile", args{"https/Users/mrinalwahal/go/src/github.com/flanksource/regen/file1.temp", ""}, true},
		//	{"url", args{"github.com/mrinalwahal/portfolio/README.md", ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveFile(tt.args.file, tt.args.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("%v: error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			fmt.Println(got)
		})
	}
}
