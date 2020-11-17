package deps

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/flanksource/commons/files"
	"github.com/flanksource/commons/utils"
)

func TestInstallDependency(t *testing.T) {
	err := os.MkdirAll("/tmp/.bin", os.ModeDir)
	if err != nil {
		t.Error("Failed to create test directory.")
	}
	for name, dependency := range dependencies {
		InstallDependency(name, dependency.Version, "/tmp/.bin")
		customName := dependencies[name].BinaryName
		if customName != "" {
			name = utils.Interpolate(customName, map[string]string{"os": runtime.GOOS, "platform": runtime.GOARCH})
		}
		if !files.Exists(fmt.Sprintf("/tmp/.bin/%s", name)) {
			t.Errorf("Failed to install %s. /tmp/.bin/%s does not exist", name, name)
		}
	}
}
