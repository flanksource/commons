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
	"github.com/flanksource/commons/text"
	"github.com/flanksource/commons/utils"
	log "github.com/sirupsen/logrus"
)

// Dependency is a struct referring to a version and the templated path
// to download the dependency on the different OS platforms
type Dependency struct {
	Version                            string
	Linux, Macosx, Windows, Go, Docker string
	Template                           string
	BinaryName                         string
	PreInstalled                       []string
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
	name = strings.ToLower(name)
	binDir = absolutePath(binDir)
	return func(msg string, args ...interface{}) error {
		bin := fmt.Sprintf("%s/%s", binDir, name)
		if !files.Exists(binDir) {
			os.MkdirAll(binDir, 0755)
		}
		if err := InstallDependency(name, ver, binDir); err != nil {
			return err
		}
		customName := dependencies[name].BinaryName
		if customName != "" {
			templated := utils.Interpolate(customName, map[string]string{"os": runtime.GOOS, "platform": runtime.GOARCH})
			bin = fmt.Sprintf("%s/%s", binDir, templated)
		}
		return exec.ExecfWithEnv(bin+" "+msg, env, args...)
	}
}

var dependencies = map[string]Dependency{
	"jq": {
		Version: "1.6",
		Linux:   "https://github.com/stedolan/jq/releases/download/jq-{{.version}}/jq-linux64",
		Windows: "https://github.com/stedolan/jq/releases/download/jq-{{.version}}/jq-win64.exe",
		Macosx:  "https://github.com/stedolan/jq/releases/download/jq-{{.version}}/jq-osx-amd64",
	},
	"gomplate": {
		Version:  "v3.5.0",
		Template: "https://github.com/hairyhenderson/gomplate/releases/download/{{.version}}/gomplate_{{.os}}-{{.platform}}",
	},
	"konfigadm": {
		Version: "v0.12.0",
		Linux:   "https://github.com/flanksource/konfigadm/releases/download/{{.version}}/konfigadm",
		Macosx:  "https://github.com/flanksource/konfigadm/releases/download/{{.version}}/konfigadm_osx",
	},
	"jb": {
		Version:  "v0.1.0",
		Template: "https://github.com/jsonnet-bundler/jsonnet-bundler/releases/download/{{.version}}/jb-{{.os}}-{{.platform}}",
	},
	"jsonnet": {
		Version: "0.14",
		Docker:  "docker.io/bitnami/jsonnet",
	},
	"sonobuoy": {
		Version:  "0.55.1",
		Template: "https://github.com/vmware-tanzu/sonobuoy/releases/download/v{{.version}}/sonobuoy_{{.version}}_{{.os}}_{{.platform}}.tar.gz",
	},
	"govc": {
		Version:  "v0.27.4",
		Template: "https://github.com/vmware/govmomi/releases/download/{{.version}}/govc_{{.os | title}}_{{ ternary \"x86_64\" \"armv6\" (eq .platform \"amd64\")}}.tar.gz",
	},
	"gojsontoyaml": {
		Version: "0.15.0",
		Linux:   "github.com/hongkailiu/gojsontoyaml/releases/download/e8bd32d/gojsontoyaml",
	},
	"yaml-cli": {
		Version:  "v1.0.2",
		Template: "https://github.com/flanksource/yaml-cli/releases/download/{{.version}}/yaml_{{.os}}-{{.platform}}",
	},
	"kind": {
		Version:  "0.6.1",
		Template: "https://github.com/kubernetes-sigs/kind/releases/download/v{{.version}}/kind-{{.os}}-{{.platform}}",
	},
	"pgo": {
		Version: "4.0.1",
		Linux:   "https://github.com/CrunchyData/postgres-operator/releases/download/{{.version}}/pgo",
		Macosx:  "https://github.com/CrunchyData/postgres-operator/releases/download/{{.version}}/pgo-mac",
	},
	"helm": {
		Version:  "v3.7.2",
		Template: "https://get.helm.sh/helm-{{.version}}-{{.os}}-{{.platform}}.tar.gz",
	},
	"helmfile": {
		Version:  "v0.45.3",
		Template: "https://github.com/roboll/helmfile/releases/download/{{.version}}/helmfile_{{.os}}_{{.platform}}",
	},
	"aws-iam-authenticator": {
		Version:  "1.13.7/2019-06-11",
		Template: "https://amazon-eks.s3-us-west-2.amazonaws.com/{{.version}}/bin/{{.os}}/{{.platform}}/aws-iam-authenticator",
	},
	"kubectl": {
		Version:  "v1.15.3",
		Template: "https://storage.googleapis.com/kubernetes-release/release/{{.version}}/bin/{{.os}}/{{.platform}}/kubectl",
	},
	"terraform": {
		Version:  "1.1.7",
		Template: "https://releases.hashicorp.com/terraform/{{.version}}/terraform_{{.version}}_{{.os}}_{{.platform}}.zip",
	},
	"go-getter": {
		Version:  "1.5.10",
		Template: "https://github.com/hashicorp/go-getter/releases/download/v{{.version}}/go-getter_{{.version}}_{{.os}}_{{.platform}}.zip",
	},
	"expenv": {
		Version:  "v1.2.0",
		Template: "https://github.com/TheWolfNL/expenv/releases/download/{{.version}}/expenv_{{.os}}_{{.platform}}",
	},
	"velero": {
		Version:  "v1.2.0",
		Template: "https://github.com/heptio/velero/releases/download/{{.version}}/velero-{{.version}}-{{.os}}-{{.platform}}.tar.gz",
	},
	"ketall": {
		Version:    "v1.3.8",
		Template:   "https://github.com/corneliusweig/ketall/releases/download/{{.version}}/get-all-{{.platform}}-{{.os}}.tar.gz",
		BinaryName: "get-all-{{.platform}}-{{.os}}",
	},
	"sops": {
		Version:  "v3.5.0",
		Template: "https://github.com/mozilla/sops/releases/download/{{.version}}/sops-{{.version}}.{{.os}}",
	},
	"kubeseal": {
		Version:  "v0.10.0",
		Template: "https://github.com/bitnami-labs/sealed-secrets/releases/download/{{.version}}/kubeseal-{{.os}}-{{.platform}}",
	},
	"packer": {
		Version:  "1.5.5",
		Template: "https://releases.hashicorp.com/packer/{{.version}}/packer_{{.version}}_{{.os}}_{{.platform}}.zip",
	},
	"reg": {
		Version:  "v0.16.1",
		Template: "https://github.com/genuinetools/reg/releases/download/{{.version}}/reg-{{.os}}-{{.platform}}",
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
	//the kubebuilder testenv binaries are all in the same tarball
	//installing any one will result in all three being installed (kubectl not listed here due to map collision)
	"etcd": {
		Version:  "1.19.2",
		Template: "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-{{.version}}-{{.os}}-{{.platform}}.tar.gz",
	},
	"kube-apiserver": {
		Version:  "1.19.2",
		Template: "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-{{.version}}-{{.os}}-{{.platform}}.tar.gz",
	},
	"postgrest": {
		Version:    "v9.0.0.20211220",
		Linux:      "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-linux-static-x64.tar.xz",
		Windows:    "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-windows-x64.zip",
		Macosx:     "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-macos-x64.tar.xz",
		BinaryName: "postgrest",
	},
	"yq": {
		Version:  "v4.16.2",
		Template: "https://github.com/mikefarah/yq/releases/download/{{.version}}/yq_{{.os}}_{{.platform}}",
	},
	"karina": {
		Version:  "v0.61.0",
		Template: "https://github.com/flanksource/karina/releases/download/{{.version}}/karina_{{.os}}-{{.platform}}",
	},
	"canary-checker": {
		Version:  "v0.38.74",
		Template: "https://github.com/flanksource/canary-checker/releases/download/{{.version}}/canary-checker_{{.os}}_{{.platform}}",
	},
	"eksctl": {
		Version: "v0.86.0",
		Linux:   "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_Linux_amd64.tar.gz",
		Windows: "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_Windows_amd64.tar.gz",
		Macosx:  "https://github.com/weaveworks/eksctl/releases/download/{{.version}}/eksctl_Darwin_amd64.tar.gz",
	},
}

// InstallDependency installs a binary to binDir, if ver is nil then the default version is used
func InstallDependency(name, ver string, binDir string) error {
	name = strings.ToLower(name)
	dependency, ok := dependencies[name]
	if !ok {
		return fmt.Errorf("dependency %s not found", name)
	}
	var bin string
	if len(strings.TrimSpace(ver)) == 0 {
		ver = dependency.Version
	}
	data := map[string]string{"os": runtime.GOOS, "platform": runtime.GOARCH, "version": ver}
	if dependency.BinaryName != "" {
		templated, err := text.Template(dependency.BinaryName, data)
		if err != nil {
			return err
		}
		bin = fmt.Sprintf("%s/%s", binDir, templated)
	} else {
		bin = fmt.Sprintf("%s/%s", binDir, name)
	}

	finalBin := path.Join(binDir, name)

	if is.File(finalBin) {
		log.Debugf("%s already exists", finalBin)
		return nil
	}

	var urlPath string
	var err error
	if runtime.GOOS == "linux" {
		urlPath = dependency.Linux
	} else if runtime.GOOS == "darwin" {
		urlPath = dependency.Macosx
	} else if runtime.GOOS == "windows" {
		urlPath = dependency.Windows
	}

	if urlPath == "" && dependency.Template != "" {
		urlPath, err = text.Template(dependency.Template, data)
		if err != nil {
			return err
		}
	}
	if urlPath != "" {
		url, err := text.Template(urlPath, data)
		if err != nil {
			return err
		}
		log.Infof("Installing %s (%s) -> %s", name, ver, url)
		err = download(url, bin)
		if err != nil {
			return fmt.Errorf("failed to download %s: %+v", name, err)
		}
		if err := os.Chmod(bin, 0755); err != nil {
			return fmt.Errorf("failed to make %s executable", name)
		}
		if dependency.BinaryName != "" {
			if err := os.Rename(bin, finalBin); err != nil {
				return fmt.Errorf("failed to rename %s to %s: %v", bin, finalBin, err)
			}
		}
	} else if dependency.Go != "" {
		//FIXME this only works if the PWD is in the GOPATH
		url, _ := text.Template(dependency.Go, data)
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
		if err := InstallDependency(name, ver, binDir); err != nil {
			return err
		}
		return exec.Execf(bin+" "+msg, args...)
	}

}

// InstallDependencies takes a map of supported dependencies and their version and
// installs them to the specified binDir
func InstallDependencies(deps map[string]string, binDir string) error {
	_ = os.Mkdir(binDir, 0755)
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
		if err := net.Download(url, file); err != nil {
			return fmt.Errorf("failed to download %s: %+v", url, err)
		}
		defer os.Remove(file)
		return files.UnarchiveExecutables(file, path.Dir(bin))
	}
	return net.Download(url, bin)
}

func Which(cmd string) bool {
	_, err := osExec.LookPath(cmd)
	return err == nil
}
