package logger

import (
	"bytes"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestSetOutput_RedirectsNewLogger verifies that a logger constructed after
// SetOutput writes to the new destination.
func TestSetOutput_RedirectsNewLogger(t *testing.T) {
	original := GetOutput()
	t.Cleanup(func() { SetOutput(original) })

	var buf bytes.Buffer
	SetOutput(&buf)

	lg := New("setoutput-new")
	lg.Infof("hello %s", "world")

	if !strings.Contains(buf.String(), "hello world") {
		t.Fatalf("expected buffer to contain log line, got %q", buf.String())
	}
}

// TestSetOutput_RedirectsExistingLogger verifies that a logger constructed
// BEFORE SetOutput still honors the swap on its next write. This is the
// important guarantee — handlers capture the indirect writer, not the
// concrete fd, so a later swap retargets them without rebuilding.
func TestSetOutput_RedirectsExistingLogger(t *testing.T) {
	original := GetOutput()
	t.Cleanup(func() { SetOutput(original) })

	lg := New("setoutput-existing")

	var buf bytes.Buffer
	SetOutput(&buf)

	lg.Infof("retargeted %d", 42)

	if !strings.Contains(buf.String(), "retargeted 42") {
		t.Fatalf("existing logger did not retarget on SetOutput, buf=%q", buf.String())
	}
}

// TestSetOutput_RestoreSwap verifies that restoring the original writer
// via GetOutput + SetOutput stops sending to the intermediate buffer.
func TestSetOutput_RestoreSwap(t *testing.T) {
	original := GetOutput()
	t.Cleanup(func() { SetOutput(original) })

	lg := New("setoutput-restore")

	var buf bytes.Buffer
	saved := GetOutput()
	SetOutput(&buf)
	lg.Infof("recorded")
	SetOutput(saved)
	lg.Infof("not recorded")

	if !strings.Contains(buf.String(), "recorded") {
		t.Fatalf("expected first write in buf, got %q", buf.String())
	}
	if strings.Contains(buf.String(), "not recorded") {
		t.Fatalf("buf contains post-restore write, swap didn't restore, got %q", buf.String())
	}
}

// TestSetOutput_NilRestoresStderr verifies that SetOutput(nil) falls back
// to os.Stderr rather than panicking on every log write.
func TestSetOutput_NilRestoresStderr(t *testing.T) {
	original := GetOutput()
	t.Cleanup(func() { SetOutput(original) })

	SetOutput(nil)
	got := GetOutput()
	if got != io.Writer(os.Stderr) {
		t.Fatalf("SetOutput(nil) did not reset to os.Stderr, got %T", got)
	}
}

// TestSetOutput_ConcurrentSwaps verifies that a flood of concurrent SetOutput
// + log calls does not race or deadlock. Run with -race.
func TestSetOutput_ConcurrentSwaps(t *testing.T) {
	original := GetOutput()
	t.Cleanup(func() { SetOutput(original) })

	lg := New("setoutput-concurrent")

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				var b bytes.Buffer
				SetOutput(&b)
				lg.Infof("n=%d", j)
			}
		}()
	}
	wg.Wait()
}
