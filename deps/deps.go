package deps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"

	osExec "os/exec"

	"github.com/flanksource/commons/exec"
	"github.com/flanksource/commons/files"
	"github.com/flanksource/commons/is"
	"github.com/flanksource/commons/net"
	"github.com/flanksource/commons/utils"
	log "github.com/sirupsen/logrus"
)

// Dependency is a struct referring to a version and the templated path
// to download the dependency on the different OS platforms
type Dependency struct {
	Version                   string
	Linux, Macosx, Go, Docker string
	BinaryName                string
	PreInstalled              []string
}

// BinaryFunc is an interface to executing a binary, downloading it necessary
type BinaryFunc func(msg string, args ...interface{}) error

// BinaryFuncWithEnv is an interface to executing a binary, downloading it necessary
type BinaryFuncWithEnv func(msg string, env map[string]string, args ...interface{}) error

func absolutePath(dir string) string {
	if !strings.HasPrefix("/", dir) {
		cwd, _ := os.Getwd()
		dir = cwd + "/" + dir
	}
	// dir, _ = os.Readlink(dir)
	return dir
}

// BinaryWithEnv returns a function that be called to execute the binary
func BinaryWithEnv(name, ver string, binDir string, env map[string]string) BinaryFunc {
	binDir = absolutePath(binDir)
	return func(msg string, args ...interface{}) error {
		bin := fmt.Sprintf("%s/%s", binDir, name)
		InstallDependency(name, ver, binDir)
		customName := dependencies[name].BinaryName
		if customName != "" {
			templated := utils.Interpolate(customName, map[string]string{"os": runtime.GOOS, "platform": runtime.GOARCH})
			bin = fmt.Sprintf("%s/%s", binDir, templated)
		}
		return exec.ExecfWithEnv(bin+" "+msg, env, args...)
	}
}

var dependencies = map[string]Dependency{
	"gomplate": {
		Version: "v3.5.0",
		Linux:   "https://github.com/hairyhenderson/gomplate/releases/download/{{.version}}/gomplate_linux-amd64",
		Macosx:  "https://github.com/hairyhenderson/gomplate/releases/download/{{.version}}/gomplate_darwin-amd64",
	},
	"konfigadm": {
		Version: "v0.3.6",
		Linux:   "https://github.com/moshloop/konfigadm/releases/download/{{.version}}/konfigadm",
		Macosx:  "https://github.com/moshloop/konfigadm/releases/download/{{.version}}/konfigadm_osx",
	},
	"jb": {
		Version: "v0.1.0",
		Linux:   "https://github.com/jsonnet-bundler/jsonnet-bundler/releases/download/{{.version}}/jb-linux-amd64",
		Macosx:  "https://github.com/jsonnet-bundler/jsonnet-bundler/releases/download/{{.version}}/jb-darwin-amd64",
	},
	"jsonnet": {
		Version: "0.14",
		Docker:  "docker.io/bitnami/jsonnet",
	},
	"sonobuoy": {
		Version: "0.16.4",
		Linux:   "https://github.com/heptio/sonobuoy/releases/download/v{{.version}}/sonobuoy_{{.version}}_linux_amd64.tar.gz",
		Macosx:  "https://github.com/heptio/sonobuoy/releases/download/v{{.version}}/sonobuoy_{{.version}}_darwin_amd64.tar.gz",
	},
	"govc": {
		Version: "v0.20.0",
		Linux:   "https://github.com/vmware/govmomi/releases/download/{{.version}}/govc_linux_amd64.gz",
		Macosx:  "https://github.com/vmware/govmomi/releases/download/{{.version}}/govc_darwin_amd64.gz",
	},
	"gojsontoyaml": {
		Version: "0.15.0",
		Linux:   "github.com/hongkailiu/gojsontoyaml/releases/download/e8bd32d/gojsontoyaml",
	},
	"kind": {
		Version: "0.6.1",
		Linux:   "https://github.com/kubernetes-sigs/kind/releases/download/v{{.version}}/kind-linux-amd64",
		Macosx:  "https://github.com/kubernetes-sigs/kind/releases/download/v{{.version}}/kind-darwin-amd64",
	},
	"pgo": {
		Version: "4.0.1",
		Linux:   "https://github.com/CrunchyData/postgres-operator/releases/download/{{.version}}/pgo",
		Macosx:  "https://github.com/CrunchyData/postgres-operator/releases/download/{{.version}}/pgo-mac",
	},
	"helm": {
		Version: "v2.13.0",
		Linux:   "https://storage.googleapis.com/kubernetes-helm/helm-{{.version}}-linux-amd64.tar.gz",
		Macosx:  "https://storage.googleapis.com/kubernetes-helm/helm-{{.version}}-darwin-amd64.tar.gz",
	},
	"helmfile": {
		Version: "v0.45.3",
		Macosx:  "https://github.com/roboll/helmfile/releases/download/{{.version}}/helmfile_darwin_amd64",
		Linux:   "https://github.com/roboll/helmfile/releases/download/{{.version}}/helmfile_linux_amd64",
	},
	"aws-iam-authenticator": {
		Version: "1.13.7/2019-06-11",
		Linux:   "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/bin/linux/amd64/aws-iam-authenticator",
		Macosx:  "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/bin/darwin/amd64/aws-iam-authenticator",
	},
	"kubectl": {
		Version: "v1.15.3",
		Linux:   "https://storage.googleapis.com/kubernetes-release/release/{{.version}}/bin/linux/amd64/kubectl",
		Macosx:  "https://storage.googleapis.com/kubernetes-release/release/{{.version}}/bin/darwin/amd64/kubectl",
	},
	"terraform": Dependency{
		Version: "0.12.",
		Linux:   "https://releases.hashicorp.com/terraform/{{.version}}/terraform_{{.version}}_linux_amd64.zip",
		Macosx:  "https://releases.hashicorp.com/terraform/{{.version}}/terraform_{{.version}}_darwin_amd64.zip",
	},
	"eksctl": {
		Version: "0.4.3",
		Linux:   "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_Linux_amd64.tar.gz",
		Macosx:  "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_Darwin_amd64.tar.gz",
	},
	// "go-getter": Dependency{
	// 	Version: "1.3",
	// 	Go:      "github.com/hashicorp/go-getter@{{.version}}",
	// },
	"expenv": {
		Version: "v1.2.0",
		Macosx:  "https://github.com/TheWolfNL/expenv/releases/download/{{.version}}/expenv_darwin_amd64",
		Linux:   "https://github.com/TheWolfNL/expenv/releases/download/{{.version}}/expenv_linux_amd64",
	},
	"velero": {
		Version: "v1.2.0",
		Macosx:  "https://github.com/heptio/velero/releases/download/{{.version}}/velero-{{.version}}-darwin-amd64.tar.gz",
		Linux:   "https://github.com/heptio/velero/releases/download/{{.version}}/velero-{{.version}}-linux-amd64.tar.gz",
	},
	"jx": {
		Version: "2.0.795",
		Macosx:  "https://github.com/jenkins-x/jx/releases/download/v2.0.795/jx-darwin-amd64.tar.gz",
		Linux:   "https://github.com/jenkins-x/jx/releases/download/v2.0.795/jx-linux-amd64.tar.gz",
	},
	"ketall": {
		Version:    "v1.3.0",
		Macosx:     "https://github.com/corneliusweig/ketall/releases/download/{{.version}}/get-all-amd64-darwin.tar.gz",
		Linux:      "https://github.com/corneliusweig/ketall/releases/download/{{.version}}/get-all-amd64-linux.tar.gz",
		BinaryName: "get-all-{{.platform}}-{{.os}}",
	},
	"sops": {
		Version: "v3.5.0",
		Linux:   "https://github.com/mozilla/sops/releases/download/{{.version}}/sops-{{.version}}.linux",
		Macosx:  "https://github.com/mozilla/sops/releases/download/{{.version}}/sops-{{.version}}.darwin",
	},
	"kubeseal": {
		Version: "v0.10.0",
		Linux:   "https://github.com/bitnami-labs/sealed-secrets/releases/download/{{.version}}/kubeseal-linux-amd64",
		Macosx:  "https://github.com/bitnami-labs/sealed-secrets/releases/download/{{.version}}/kubeseal-darwin-amd64",
	},
	"packer": {
		Version: "1.5.5",
		Macosx:  "https://releases.hashicorp.com/packer/{{.version}}/packer_{{.version}}_darwin_amd64.zip",
		Linux:   "https://releases.hashicorp.com/packer/{{.version}}/packer_{{.version}}_linux_amd64.zip",
	},
	"reg": {
		Version: "v0.16.1",
		Linux:   "https://github.com/genuinetools/reg/releases/download/{{.version}}/reg-linux-amd64",
		Macosx:  "https://github.com/genuinetools/reg/releases/download/{{.version}}/reg-darwin-amd64",
	},
	"mkisofs": {
		PreInstalled: []string{"mkisofs", "genisoimage"},
	},
	"qemu-img": {
		PreInstalled: []string{"qemu-img"},
	},
	"qemu-system": {
		PreInstalled: []string{"qemu-system-x86_64"},
	},
	"docker": {
		PreInstalled: []string{"docker", "crictl"},
	},
	"hdiutil": {},
	//the kubebuilder testenv binaries are all in the same tarball
	//installing any one will result in all three being installed (kubectl not listed here due to map collision)
	"etcd": {
		Version:      "1.19.2",
		Linux:        "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-{{.version}}-linux-amd64.tar.gz",
		Macosx:       "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-{{.version}}-darwin-amd64.tar.gz",
	},
	"kube-apiserver": {
		Version:      "1.19.2",
		Linux:        "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-{{.version}}-linux-amd64.tar.gz",
		Macosx:       "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-{{.version}}-darwin-amd64.tar.gz",
	},
}

// InstallDependency installs a binary to binDir, if ver is nil then the default version is used
func InstallDependency(name, ver string, binDir string) error {
	dependency, ok := dependencies[name]
	if !ok {
		return nil
	}
	var bin string
	if dependency.BinaryName != "" {
		templated := utils.Interpolate(dependency.BinaryName, map[string]string{"os": runtime.GOOS, "platform": runtime.GOARCH})
		bin = fmt.Sprintf("%s/%s", binDir, templated)
	} else {
		bin = fmt.Sprintf("%s/%s", binDir, name)
	}
	if is.File(bin) {
		log.Tracef("%s already exists", bin)
		return nil
	}

	if ver == "" {
		ver = dependency.Version
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
		//FIXME this only works if the PWD is in the GOPATH
		url := utils.Interpolate(dependency.Go, map[string]string{"version": ver})
		log.Infof("Installing via go get %s (%s) -> %s", name, ver, url)
		if err := exec.Execf("GOPATH=$PWD/.go go get %s", url); err != nil {
			return err
		}
		if err := os.Rename(".go/bin/"+name, bin); err != nil {
			return err
		}
	}
	return nil
}

// Binary returns a function that can be called to execute the binary
func Binary(name, ver string, binDir string) BinaryFunc {
	binDir = absolutePath(binDir)

	dependency, ok := dependencies[name]
	if !ok {
		return func(msg string, args ...interface{}) error {
			if Which(name) {
				return exec.Execf(name+" "+msg, args...)
			}
			return fmt.Errorf("cannot find preinstalled dependency: %s", name)
		}
	}

	if dependency.Docker != "" {
		return func(msg string, args ...interface{}) error {
			cwd, _ := os.Getwd()
			docker := fmt.Sprintf("docker run --rm -v %s:%s -w %s %s:%s ", cwd, cwd, cwd, dependency.Docker, ver)
			return exec.Execf(docker+msg, args...)
		}
	}

	if len(dependency.PreInstalled) > 0 {
		return func(msg string, args ...interface{}) error {
			for _, bin := range dependency.PreInstalled {
				if Which(bin) {
					return exec.Execf(bin+" "+msg, args...)
				}
			}
			return fmt.Errorf("cannot find preinstalled dependency: %s", strings.Join(dependency.PreInstalled, ","))
		}
	}

	return func(msg string, args ...interface{}) error {
		bin := fmt.Sprintf("%s/%s", binDir, name)
		InstallDependency(name, ver, binDir)
		return exec.Execf(bin+" "+msg, args...)
	}

}

// InstallDependencies takes a map of supported dependencies and their version and
// installs them to the specified binDir
func InstallDependencies(deps map[string]string, binDir string) error {
	os.Mkdir(binDir, 0755)
	for name, ver := range deps {
		if err := InstallDependency(name, ver, binDir); err != nil {
			return err
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

func Which(cmd string) bool {
	_, err := osExec.LookPath(cmd)
	return err == nil
}
