package deps

import (
	"fmt"
	"os"

	"testing"

	"github.com/flanksource/commons/files"
)

func TestInstallDependency(t *testing.T) {
	dir, err := os.MkdirTemp("", "commons-test-deps")
	fmt.Printf("Created dir %s\n", dir)
	if err != nil {
		t.Errorf("failed to create temporary directory %v", err)
	}
	defer os.RemoveAll(dir)

	for name, dependency := range dependencies {
		t.Run(name, func(t *testing.T) {
			t.Logf("Installing %s", name)
			err := InstallDependency(name, dependency.Version, dir)
			if err != nil {
				t.Errorf("Failed to download %s: %v", name, err)
				return
			}
			if len(dependency.PreInstalled) > 0 || len(dependency.Docker) > 0 {
				return
			}

			path, err := dependency.GetPath(name, dir)
			if err != nil {
				t.Errorf("Failed to install %s. ", err)
			}
			if !files.Exists(path) {
				t.Errorf("Failed to install %s. %s does not exist", name, path)
			}
		})

	}
}
