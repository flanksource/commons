package deps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/flanksource/commons/files"
)

func TestInstallDependency(t *testing.T) {
	dir, err := ioutil.TempDir("", "commons-test-deps")
	fmt.Printf("Created dir %s\n", dir)
	if err != nil {
		t.Errorf("failed to create temporary directory %v", err)
	}
	defer os.RemoveAll(dir)

	for name, dependency := range dependencies {
		t.Run(name, func(t *testing.T) {
			err := InstallDependency(name, dependency.Version, dir)
			if err != nil {
				t.Errorf("Failed to download %s: %v", name, err)
				return
			}
			if len(dependency.PreInstalled) > 0 || len(dependency.Docker) > 0 {
				return
			}

			if !files.Exists(path.Join(dir, name)) {
				t.Errorf("Failed to install %s. %s/%s does not exist", name, dir, name)
			}
		})
	}
}
