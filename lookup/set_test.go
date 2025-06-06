package lookup

import (
	"fmt"
	"reflect"
	"testing"

	. "github.com/onsi/gomega"
)

type Config struct {
	A ConfigA `json:"a"`
	B ConfigB `json:"b"`
}

type Disabled struct {
	Disabled bool   `json:"disabled"`
	Version  string `json:"version"`
}

type ConfigA struct {
	Disabled `json:",inline"`
	Foo      string `json:"foo"`
}

type ConfigB struct {
	Bar int `json:"bar"`
}

type Fixture struct {
	Key    string
	Value  string
	Expect func(*WithT, *Config, error)
}

func DeepFields(iface interface{}) []reflect.Value {
	fields := make([]reflect.Value, 0)
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)

		switch v.Kind() {
		case reflect.Struct:
			fields = append(fields, DeepFields(v.Interface())...)
		default:
			fields = append(fields, v)
		}
	}

	return fields
}

func TestSet(t *testing.T) {
	fixtures := []Fixture{
		{
			Key:   "a.foo",
			Value: "bar",
			Expect: func(g *WithT, cfg *Config, err error) {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(cfg.A.Foo).To(Equal("bar"))
			},
		},
		{
			Key:   "b.bar",
			Value: "3",
			Expect: func(g *WithT, cfg *Config, err error) {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(cfg.B.Bar).To(Equal(3))
			},
		},
		{
			Key:   "a.version",
			Value: "v1.2.3",
			Expect: func(g *WithT, cfg *Config, err error) {
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(cfg.A.Version).To(Equal("v1.2.3"))
			},
		},
	}

	for i, fixture := range fixtures {
		name := fmt.Sprintf("Test %d - %s", i, fixture.Key)
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)

			cfg := &Config{}
			err := Set(cfg, fixture.Key, fixture.Value)

			fixture.Expect(g, cfg, err)
		})
	}
}
