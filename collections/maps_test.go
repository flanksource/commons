package collections

import "testing"

func TestSortedMap(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "simple",
			args: args{
				labels: map[string]string{
					"b": "b",
					"a": "a",
					"c": "c",
				},
			},
			want: "a=a,b=b,c=c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SortedMap(tt.args.labels); got != tt.want {
				t.Errorf("SortedMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
