package deps

// import (
// 	"fmt"
// 	"io/ioutil"
// 	"os"
// 	"path"
// 	"runtime"
// 	"testing"

// 	"github.com/flanksource/commons/files"
// 	"github.com/flanksource/commons/utils"
// )

// func TestInstallDependency(t *testing.T) {
// 	dir, err := ioutil.TempDir("", "commons-test-deps")
// 	fmt.Printf("Created dir %s\n", dir)
// 	if err != nil {
// 		t.Errorf("failed to create temporary directory %v", err)
// 	}
// 	defer os.RemoveAll(dir)

// 	for name, dependency := range dependencies {
// 		err := InstallDependency(name, dependency.Version, dir)
// 		if err != nil {
// 			t.Errorf("Failed to install %s: %v", name, err)
// 			continue
// 		}
// 		customName := dependencies[name].BinaryName
// 		if customName != "" {
// 			name = utils.Interpolate(customName, map[string]string{"os": runtime.GOOS, "platform": runtime.GOARCH})
// 		}
// 		if !files.Exists(path.Join(dir, name)) {
// 			t.Errorf("Failed to install %s. %s/%s does not exist", name, dir, name)
// 		}
// 	}
// }
