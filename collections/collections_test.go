package collections

import (
	"reflect"
	"testing"
)

func Test_MergeMap(t *testing.T) {
	type args struct {
		a map[string]string
		b map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "no overlaps",
			args: args{
				a: map[string]string{"name": "flanksource"},
				b: map[string]string{"foo": "bar"},
			},
			want: map[string]string{
				"name": "flanksource",
				"foo":  "bar",
			},
		},
		{
			name: "overlaps",
			args: args{
				a: map[string]string{"name": "flanksource", "foo": "baz"},
				b: map[string]string{"foo": "bar"},
			},
			want: map[string]string{
				"name": "flanksource",
				"foo":  "bar",
			},
		},
		{
			name: "overlaps II",
			args: args{
				a: map[string]string{"name": "github", "foo": "baz"},
				b: map[string]string{"name": "flanksource", "foo": "bar"},
			},
			want: map[string]string{
				"name": "flanksource",
				"foo":  "bar",
			},
		},
		{
			name: "ditto",
			args: args{
				a: map[string]string{"name": "flanksource", "foo": "bar"},
				b: map[string]string{"name": "flanksource", "foo": "bar"},
			},
			want: map[string]string{
				"name": "flanksource",
				"foo":  "bar",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeMap(tt.args.a, tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_KeyValueSliceToMap(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]string
	}{
		{name: "simple", args: []string{"name=flanksource"}, want: map[string]string{"name": "flanksource"}},
		{name: "white space", args: []string{"    name  =  flanksource   "}, want: map[string]string{"name": "flanksource"}},
		{name: "multiple-simple", args: []string{"name=flanksource", "foo=bar"}, want: map[string]string{"name": "flanksource", "foo": "bar"}},
		{name: "double-equal", args: []string{"name=foo=bar"}, want: map[string]string{"name": "foo=bar"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KeyValueSliceToMap(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
