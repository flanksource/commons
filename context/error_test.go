package context_test

import (
	"bytes"
	gocontext "context"
	"errors"
	"fmt"
	"testing"

	commonscontext "github.com/flanksource/commons/context"
	"github.com/flanksource/commons/logger"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type diagnosticError struct{}

func (diagnosticError) Error() string { return "diagnostic failure" }
func (diagnosticError) ToMap() map[string]any {
	return map[string]any{"detail": "retained", "error": "diagnostic failure"}
}

func TestContextErrors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Context Error Suite")
}

var _ = Describe("context errors", func() {
	It("retains structured metadata through standard error wrapping", func() {
		var output bytes.Buffer
		ctx := commonscontext.NewContext(gocontext.Background(), commonscontext.WithLogger(logger.NewWithWriter(&output)))
		ctx.Error(fmt.Errorf("outer: %w", diagnosticError{}), "operation %s", "failed")
		Expect(output.String()).To(ContainSubstring("retained"))
		Expect(output.String()).To(ContainSubstring("diagnostic failure"))
	})
	It("formats wrapped unstructured errors with their operation context", func() {
		var output bytes.Buffer
		ctx := commonscontext.NewContext(gocontext.Background(), commonscontext.WithLogger(logger.NewWithWriter(&output)))
		ctx.Error(fmt.Errorf("outer: %w", errors.New("cause")), "operation %s", "failed")
		Expect(output.String()).To(ContainSubstring("operation failed: outer: cause"))
	})
})
