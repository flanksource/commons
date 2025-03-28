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

func TestPathTraversalVuln(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"traversal", args{"../../etc/passwd"}, true},
		{"absolute", args{"/etc/passwd"}, true},
		{"relative", args{"./config.yaml"}, false},
		{"normal", args{"config.yaml"}, false},
		{"double_dot", args{"../config.yaml"}, true},
		{"hidden", args{".hidden/config.yaml"}, false},
		{"multiple_traversal", args{"../../../etc/passwd"}, true},
		{"windows_absolute", args{"C:\\Windows\\System32"}, true},
		{"windows_traversal", args{"..\\..\\Windows\\System32"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidatePath(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
