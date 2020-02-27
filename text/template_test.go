package text

import (
	"testing"

	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
)

func TestTemplate(t *testing.T) {
	vals := map[string]interface{}{
		"var1": "val1",
	}
	g := gomega.NewWithT(t)
	fixtures := map[string]string{
		"{{.var1}}":                "val1",
		`{{ "" | default "foo" }}`: "foo",
	}

	for tpl, val := range fixtures {
		t.Run(tpl, func(t *testing.T) {
			result, err := Template(tpl, vals)
			g.Expect(err).To(BeNil())
			g.Expect(result).To(Equal(val))
		})
	}
}
