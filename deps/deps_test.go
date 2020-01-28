package deps

import (
	"fmt"
	"github.com/flanksource/commons/files"
	"github.com/flanksource/commons/utils"
	"runtime"
	"testing"
)

func TestInstallDependency(t *testing.T) {
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