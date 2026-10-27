package dependencies_test

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDependencies(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SDK Dependency Guard")
}

var _ = DescribeTable("SDK dependency guard", func(pkg string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-f",
		"{{if not .Standard}}{{with .Module}}{{.Path}}{{else}}<missing module>{{end}} {{.ImportPath}}{{end}}", pkg)
	cmd.Dir = "../.."
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	Expect(err).NotTo(HaveOccurred(), "%s", output)

	var forbidden []string
	for line := range strings.SplitSeq(string(output), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] != "github.com/flanksource/commons" {
			forbidden = append(forbidden, line)
		}
	}
	slices.Sort(forbidden)
	Expect(forbidden).To(BeEmpty(), "%s must use only the standard library and commons; forbidden dependencies:\n%s", pkg, strings.Join(forbidden, "\n"))
},
	Entry("HAR capture", "./har"),
	Entry("transport middleware primitives", "./http/middlewares"),
)
