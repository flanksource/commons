package deps

import (
	"fmt"
	"github.com/moshloop/commons/exec"
	"github.com/moshloop/commons/files"
	"github.com/moshloop/commons/is"
	"github.com/moshloop/commons/net"
	"github.com/moshloop/commons/utils"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path"
	"runtime"

	log "github.com/sirupsen/logrus"
)

// Dependency is a struct referring to a version and the templated path
// to download the dependency on the different OS platforms
type Dependency struct {
	Version           string
	Linux, Macosx, Go string
}

var dependencies = map[string]Dependency{
	"gomplate": Dependency{
		Version: "v3.5.0",
		Linux:   "https://github.com/hairyhenderson/gomplate/releases/download/{{.version}}/gomplate_linux-amd64",
		Macosx:  "https://github.com/hairyhenderson/gomplate/releases/download/{{.version}}/gomplate_darwin-amd64",
	},
	"konfigadm": Dependency{
		Version: "v0.3.6",
		Linux:   "https://github.com/moshloop/konfigadm/releases/download/{{.version}}/konfigadm",
		Macosx:  "https://github.com/moshloop/konfigadm/releases/download/{{.version}}/konfigadm_osx",
	},
	"jb": Dependency{
		Version: "v0.1.0",
		Linux:   "https://github.com/jsonnet-bundler/jsonnet-bundler/releases/download/{{.version}}/jb-linux-amd64",
		Macosx:  "https://github.com/jsonnet-bundler/jsonnet-bundler/releases/download/{{.version}}/jb-darwin-amd64",
	},
	"jsonnet": Dependency{
		Version: "v0.13.0",
		Go:      "github.com/google/go-jsonnet/cmd/jsonnet@{{.version}}",
	},
	"sonobuoy": Dependency{
		Version: "0.15.0",
		Linux:   "https://github.com/heptio/sonobuoy/releases/download/v{.version}}/sonobuoy_{{.version}}_linux_amd64.tar.gz",
		Macosx:  "https://github.com/heptio/sonobuoy/releases/download/v{{.version}}/sonobuoy_{{.version}}_darwin_amd64.tar.gz",
	},
	"govc": Dependency{
		Version: "v0.20.0",
		Linux:   "https://github.com/vmware/govmomi/releases/download/{{.version}}/govc_linux_amd64.gz",
		Macosx:  "https://github.com/vmware/govmomi/releases/download/{{.version}}/govc_darwin_amd64.gz",
	},
	"gojsontoyaml": Dependency{
		Version: "0.15.0",
		Go:      "github.com/brancz/gojsontoyaml",
	},
	"kustomize": Dependency{
		Version: "v3.0.2",
		Go:      "sigs.k8s.io/kustomize/v3/cmd/kustomize",
	},
	"pgo": Dependency{
		Version: "4.0.1",
		Linux:   "https://github.com/CrunchyData/postgres-operator/releases/download/s{{.version}}/pgo",
		Macosx:  "https://github.com/CrunchyData/postgres-operator/releases/download/{{.version}}/pgo-mac",
	},
	"helm": Dependency{
		Version: "v2.13.0",
		Linux:   "https://storage.googleapis.com/kubernetes-helm/helm-{{.version}}-linux-amd64.tar.gz",
		Macosx:  "https://storage.googleapis.com/kubernetes-helm/helm-{{.version}}-darwin-amd64.tar.gz",
	},
	"helmfile": Dependency{
		Version: "v0.45.3",
		Macosx:  "https://github.com/roboll/helmfile/releases/download/{{.version}}/helmfile_darwin_amd64",
		Linux:   "https://github.com/roboll/helmfile/releases/download/{{.version}}/helmfile_linux_amd64",
	},
	"aws-iam-authenticator": Dependency{
		Version: "1.13.7/2019-06-11",
		Linux:   "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/bin/linux/amd64/aws-iam-authenticator",
		Macosx:  "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/bin/darwin/amd64/aws-iam-authenticator",
	},
	"kubectl": Dependency{
		Version: "v1.15.3",
		Linux:   "https://storage.googleapis.com/kubernetes-release/release/{{.version}}/bin/linux/amd64/kubectl",
	},
	"terraform": Dependency{
		Version: "0.12.",
		Linux:   "https://releases.hashicorp.com/terraform/{{.version}}/terraform_{{.version}}_linux_amd64.zip",
		Macosx:  "https://releases.hashicorp.com/terraform/{{.version}}/terraform_{{.version}}_darwin_amd64.zip",
	},
	"eksctl": Dependency{
		Version: "0.4.3",
		Linux:   "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_Linux_amd64.tar.gz",
		Macosx:  "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_Darwin_amd64.tar.gz",
	},
}

// InstallDependencies takes a map of supported dependencies and their version and
// installs them to the specified binDir
func InstallDependencies(deps map[string]string, binDir string) error {
	os.Mkdir(binDir, 0755)
	for name, ver := range deps {
		bin := fmt.Sprintf("%s/%s", binDir, name)
		if is.File(bin) {
			log.Debugf("%s already exists", bin)
			continue
		}

		dependency, ok := dependencies[name]
		if !ok {
			return errors.New("Unknown dependency " + name)
		}

		path := dependency.Linux
		if runtime.GOOS == "darwin" {
			path = dependency.Macosx
		}
		if path != "" {
			url := utils.Interpolate(path, map[string]string{"version": ver})
			log.Infof("Installing %s (%s) -> %s", name, ver, url)
			err := download(url, bin)
			if err != nil {
				return fmt.Errorf("failed to download %s: %+v", name, err)
			}
			if err := os.Chmod(bin, 0755); err != nil {
				return fmt.Errorf("failed to make %s executable", name)
			}
		} else if dependency.Go != "" {
			url := utils.Interpolate(dependency.Go, map[string]string{"version": ver})
			log.Infof("Installing via go get %s (%s) -> %s", name, ver, url)
			if err := exec.Execf("GOPATH=$PWD/.go go get %s", url); err != nil {
				return err
			}
			if err := os.Rename(".go/bin/"+name, bin); err != nil {
				return err
			}
		}
	}
	return nil
}

func download(url, bin string) error {
	if is.Archive(url) {
		tmp, _ := ioutil.TempDir("", "")
		file := path.Join(tmp, path.Base(url))
		net.Download(url, file)
		defer os.Remove(file)
		return files.UnarchiveExecutables(file, path.Dir(bin))
	}
	return net.Download(url, bin)
}
