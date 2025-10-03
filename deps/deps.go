package deps

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
	"time"

	gotemplate "text/template"

	osExec "os/exec"

	"github.com/Masterminds/semver/v3"
	"github.com/flanksource/commons/exec"
	"github.com/flanksource/commons/files"
	"github.com/flanksource/commons/is"
	"github.com/flanksource/commons/net"
	"github.com/flanksource/commons/utils"
	log "github.com/sirupsen/logrus"
)

// VersionCheckMode defines how version checking should be performed
type VersionCheckMode int

const (
	// VersionCheckNone skips version checking entirely
	VersionCheckNone VersionCheckMode = iota
	// VersionCheckExact requires an exact version match
	VersionCheckExact
	// VersionCheckMinimum requires the installed version to be at least the specified version
	VersionCheckMinimum
	// VersionCheckCompatible allows compatible versions (same major version)
	VersionCheckCompatible
)

// InstallOptions configures the installation behavior
type InstallOptions struct {
	BinDir       string
	Force        bool
	VersionCheck VersionCheckMode
	GOOS         string
	GOARCH       string
	Timeout      time.Duration
	SkipChecksum bool
	PreferLocal  bool
}

// InstallOption is a functional option for configuring installation
type InstallOption func(*InstallOptions)

// Dependency is a struct referring to a version and the templated path
// to download the dependency on the different OS platforms
type Dependency struct {
	Version        string
	Linux          string
	LinuxARM       string
	Macosx         string
	MacosxARM      string
	Windows        string
	Go             string
	Docker         string
	Template       string
	BinaryName     string
	PreInstalled   []string
	VersionCommand string            // Command to get version (e.g., "--version")
	VersionPattern string            // Regex pattern to extract version from output
	Checksums      map[string]string // Platform -> checksum mapping
}

type Process struct {
	Process *osExec.Cmd
}

func (p Process) Exec(args ...any) error {
	return exec.Execf(p.Process.Path+" "+p.Process.Args[1], args...)
}

// BinaryFunc is an interface to executing a binary, downloading it necessary
type BinaryFunc func(msg string, args ...any) error

// BinaryFuncWithEnv is an interface to executing a binary, downloading it necessary
type BinaryFuncWithEnv func(msg string, env map[string]string, args ...any) error

// Option functions for configuring installation

// WithBinDir sets the binary installation directory
func WithBinDir(dir string) InstallOption {
	return func(opts *InstallOptions) {
		opts.BinDir = dir
	}
}

// WithForce enables or disables forced reinstallation
func WithForce(force bool) InstallOption {
	return func(opts *InstallOptions) {
		opts.Force = force
	}
}

// WithVersionCheck sets the version checking mode
func WithVersionCheck(mode VersionCheckMode) InstallOption {
	return func(opts *InstallOptions) {
		opts.VersionCheck = mode
	}
}

// WithOS overrides the target OS and architecture
func WithOS(goos, goarch string) InstallOption {
	return func(opts *InstallOptions) {
		opts.GOOS = goos
		opts.GOARCH = goarch
	}
}

// WithTimeout sets the download timeout
func WithTimeout(timeout time.Duration) InstallOption {
	return func(opts *InstallOptions) {
		opts.Timeout = timeout
	}
}

// WithSkipChecksum enables or disables checksum verification
func WithSkipChecksum(skip bool) InstallOption {
	return func(opts *InstallOptions) {
		opts.SkipChecksum = skip
	}
}

// WithPreferLocal prefers locally installed binaries over downloading
func WithPreferLocal(prefer bool) InstallOption {
	return func(opts *InstallOptions) {
		opts.PreferLocal = prefer
	}
}

func (dependency *Dependency) GetPath(name string, binDir string) (string, error) {
	data := map[string]string{"os": runtime.GOOS, "platform": runtime.GOARCH, "version": dependency.Version}
	if dependency.BinaryName != "" {
		templated, err := template(dependency.BinaryName, data)
		if err != nil {
			return "", err
		}
		return path.Join(binDir, templated), nil
	} else {
		return path.Join(binDir, name), nil
	}
}

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
	return func(msg string, args ...any) error {
		bin := fmt.Sprintf("%s/%s", binDir, name)
		if !files.Exists(binDir) {
			if err := os.MkdirAll(binDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", binDir, err)
			}
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
	"gojsontoyaml": {
		Version: "0.15.0",
		Linux:   "https://github.com/hongkailiu/gojsontoyaml/releases/download/e8bd32d/gojsontoyaml",
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
		Version:        "v1.15.3",
		Template:       "https://storage.googleapis.com/kubernetes-release/release/{{.version}}/bin/{{.os}}/{{.platform}}/kubectl",
		VersionCommand: "version",
		VersionPattern: `Client Version:\s*v?(\d+\.\d+\.\d+)`,
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
		Version:        "v13.0.5",
		Linux:          "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-linux-static-x86-64.tar.xz",
		LinuxARM:       "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-ubuntu-aarch64.tar.xz ",
		Windows:        "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-windows-x86-64.zip",
		Macosx:         "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-macos-x86-64.tar.xz",
		MacosxARM:      "https://github.com/PostgREST/postgrest/releases/download/{{.version}}/postgrest-{{.version}}-macos-aarch64.tar.xz",
		BinaryName:     "postgrest",
		VersionCommand: "--help",
		VersionPattern: `PostgREST\s+v?(\d+\.\d+\.\d+)`,
	},
	"wal-g": {
		Version:        "v3.0.5",
		Linux:          "https://github.com/wal-g/wal-g/releases/download/{{.version}}/wal-g-pg-ubuntu-20.04-amd64.tar.gz",
		LinuxARM:       "https://github.com/wal-g/wal-g/releases/download/{{.version}}/wal-g-pg-ubuntu-20.04-aarch64.tar.gz",
		BinaryName:     "wal-g",
		VersionCommand: "--version",
		VersionPattern: `wal-g\s+version\s+v?(\d+\.\d+\.\d+)`,
		// Note: WAL-G does not provide pre-built macOS or Windows binaries
	},
	"task": {
		Version:        "v3.44.1",
		Template:       "https://github.com/go-task/task/releases/download/{{.version}}/task_{{.os}}_{{.platform}}.tar.gz",
		BinaryName:     "task",
		VersionCommand: "--version",
		VersionPattern: `Task\s+version:\s+v?(\d+\.\d+\.\d+)`,
	},
	"postgres": {
		Version:        "16.1.0",
		PreInstalled:   []string{"postgres"},
		VersionCommand: "--version",
		VersionPattern: `postgres\s+\(PostgreSQL\)\s+(\d+\.\d+(?:\.\d+)?)`,
		// Note: Uses custom InstallPostgres function due to complex extraction process
	},
	"yq": {
		Version:        "v4.16.2",
		Template:       "https://github.com/mikefarah/yq/releases/download/{{.version}}/yq_{{.os}}_{{.platform}}",
		VersionCommand: "--version",
		VersionPattern: `yq\s+.*version\s+v?(\d+\.\d+\.\d+)`,
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
	"trivy": {
		Version: "0.40.0", // without the "v" prefix
		Linux:   "https://github.com/aquasecurity/trivy/releases/download/v{{.version}}/trivy_{{.version}}_Linux-64bit.tar.gz",
		Windows: "https://github.com/aquasecurity/trivy/releases/download/v{{.version}}/trivy_{{.version}}_windows-64bit.zip",
		Macosx:  "https://github.com/aquasecurity/trivy/releases/download/v{{.version}}/trivy_{{.version}}_macOS-64bit.tar.gz",
		// BinaryName: "trivy-{{.version}}", // Custom name not supported right now. https://github.com/flanksource/commons/issues/68
	},
}

// verifyChecksum verifies the SHA256 checksum of a file
func verifyChecksum(filepath, expectedChecksum string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	expectedChecksum = strings.TrimSpace(expectedChecksum)

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

// Install installs a dependency with configurable options
func Install(name, version string, opts ...InstallOption) error {
	name = strings.ToLower(name)

	// Set default options
	options := &InstallOptions{
		BinDir:       "/usr/local/bin",
		Force:        false,
		VersionCheck: VersionCheckNone,
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		Timeout:      5 * time.Minute,
		SkipChecksum: false,
		PreferLocal:  false,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	// Handle PostgreSQL specially due to complex extraction process
	if name == "postgres" {
		if version == "" {
			if dependency, ok := dependencies[name]; ok {
				version = dependency.Version
			} else {
				version = "16.1.0"
			}
		}
		return InstallPostgres(version, options.BinDir)
	}

	dependency, ok := dependencies[name]
	if !ok {
		return fmt.Errorf("dependency %s not found", name)
	}

	if len(strings.TrimSpace(version)) == 0 {
		version = dependency.Version
	}

	// Check if binary already exists and meets version requirements
	if !options.Force {
		bin, err := dependency.GetPathWithOptions(name, options.BinDir, options.GOOS, options.GOARCH)
		if err != nil {
			return err
		}

		if is.File(bin) {
			// If version checking is disabled, skip reinstall
			if options.VersionCheck == VersionCheckNone {
				log.Debugf("%s already exists", bin)
				return nil
			}

			// Check if installed version meets requirements
			if shouldSkipInstall, err := checkVersionRequirement(bin, version, options.VersionCheck, &dependency); err == nil && shouldSkipInstall {
				log.Debugf("%s already meets version requirement", bin)
				return nil
			}
		} else if options.PreferLocal && len(dependency.PreInstalled) > 0 {
			// Check for pre-installed binaries first
			for _, binName := range dependency.PreInstalled {
				if Which(binName) {
					if options.VersionCheck == VersionCheckNone {
						log.Debugf("Using pre-installed %s", binName)
						return nil
					}
					if shouldSkipInstall, err := checkVersionRequirementByCommand(binName, version, options.VersionCheck, &dependency); err == nil && shouldSkipInstall {
						log.Debugf("Pre-installed %s meets version requirement", binName)
						return nil
					}
				}
			}
		}
	}

	bin, err := dependency.GetPathWithOptions(name, options.BinDir, options.GOOS, options.GOARCH)
	if err != nil {
		return err
	}

	var urlPath string
	switch options.GOOS {
	case "linux":
		urlPath = dependency.Linux
		if strings.HasPrefix(options.GOARCH, "arm") && dependency.LinuxARM != "" {
			urlPath = dependency.LinuxARM
		}
	case "darwin":
		urlPath = dependency.Macosx
		if strings.HasPrefix(options.GOARCH, "arm") && dependency.MacosxARM != "" {
			urlPath = dependency.MacosxARM
		}
	case "windows":
		urlPath = dependency.Windows
	}

	data := map[string]string{"os": options.GOOS, "platform": options.GOARCH, "version": version}
	if urlPath == "" && dependency.Template != "" {
		urlPath, err = template(dependency.Template, data)
		if err != nil {
			return err
		}
	}

	if urlPath != "" {
		url, err := template(urlPath, data)
		if err != nil {
			return err
		}
		log.Infof("Installing %s@%s (from=%s, to:%s)", name, version, url, bin)

		// Create bin directory if it doesn't exist
		if !files.Exists(options.BinDir) {
			if err := os.MkdirAll(options.BinDir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", options.BinDir, err)
			}
		}

		err = downloadWithOptions(url, bin, options, &dependency, data)
		if err != nil {
			return fmt.Errorf("failed to download %s: %+v", name, err)
		}
		if err := os.Chmod(bin, 0755); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", name, err)
		}
	} else if dependency.Go != "" {
		url, _ := template(dependency.Go, data)
		log.Infof("Installing via go get %s (%s) -> %s", name, version, url)
		if err := exec.Execf("GOPATH=$PWD/.go go get %s", url); err != nil {
			return err
		}
		if err := os.Rename(".go/bin/"+name, bin); err != nil {
			return err
		}
	}
	return nil
}

// InstallDependency installs a binary to binDir, if ver is nil then the default version is used
func InstallDependency(name, ver string, binDir string) error {
	return Install(name, ver, WithBinDir(binDir))
}

// Binary returns a function that can be called to execute the binary
func Binary(name, ver string, binDir string) BinaryFunc {
	binDir = absolutePath(binDir)

	dependency, ok := dependencies[name]
	if !ok {
		return func(msg string, args ...any) error {
			if Which(name) {
				return exec.Execf(name+" "+msg, args...)
			}
			return fmt.Errorf("cannot find preinstalled dependency: %s", name)
		}
	}

	if dependency.Docker != "" {
		return func(msg string, args ...any) error {
			cwd, _ := os.Getwd()
			docker := fmt.Sprintf("docker run --rm -v %s:%s -w %s %s:%s ", cwd, cwd, cwd, dependency.Docker, ver)
			return exec.Execf(docker+msg, args...)
		}
	}

	if len(dependency.PreInstalled) > 0 {
		return func(msg string, args ...any) error {
			for _, bin := range dependency.PreInstalled {
				if Which(bin) {
					return exec.Execf(bin+" "+msg, args...)
				}
			}
			return fmt.Errorf("cannot find preinstalled dependency: %s", strings.Join(dependency.PreInstalled, ","))
		}
	}

	return func(msg string, args ...any) error {
		if err := InstallDependency(name, ver, binDir); err != nil {
			return err
		}
		bin, err := dependency.GetPath(name, binDir)
		if err != nil {
			return err
		}
		return exec.Execf(bin+" "+msg, args...)
	}

}

// BinaryWithOptions returns a function that can be called to execute the binary with configurable options
func BinaryWithOptions(name, ver string, opts ...InstallOption) BinaryFunc {
	// Set default options
	options := &InstallOptions{
		BinDir:       "/usr/local/bin",
		Force:        false,
		VersionCheck: VersionCheckNone,
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		Timeout:      5 * time.Minute,
		SkipChecksum: false,
		PreferLocal:  false,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(options)
	}

	options.BinDir = absolutePath(options.BinDir)
	dependency, ok := dependencies[name]
	if !ok {
		return func(msg string, args ...any) error {
			if Which(name) {
				return exec.Execf(name+" "+msg, args...)
			}
			return fmt.Errorf("cannot find preinstalled dependency: %s", name)
		}
	}

	if dependency.Docker != "" {
		return func(msg string, args ...any) error {
			cwd, _ := os.Getwd()
			docker := fmt.Sprintf("docker run --rm -v %s:%s -w %s %s:%s ", cwd, cwd, cwd, dependency.Docker, ver)
			return exec.Execf(docker+msg, args...)
		}
	}

	if len(dependency.PreInstalled) > 0 {
		return func(msg string, args ...any) error {
			for _, bin := range dependency.PreInstalled {
				if Which(bin) {
					return exec.Execf(bin+" "+msg, args...)
				}
			}
			return fmt.Errorf("cannot find preinstalled dependency: %s", strings.Join(dependency.PreInstalled, ","))
		}
	}

	return func(msg string, args ...any) error {
		if err := Install(name, ver, opts...); err != nil {
			return err
		}
		bin, err := dependency.GetPathWithOptions(name, options.BinDir, options.GOOS, options.GOARCH)
		if err != nil {
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
		tmp, _ := os.MkdirTemp("", "")
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

// GetPathWithOptions returns the binary path using specified OS and architecture
func (dependency *Dependency) GetPathWithOptions(name string, binDir string, goos, goarch string) (string, error) {
	data := map[string]string{"os": goos, "platform": goarch, "version": dependency.Version}
	if dependency.BinaryName != "" {
		templated, err := template(dependency.BinaryName, data)
		if err != nil {
			return "", err
		}
		return path.Join(binDir, templated), nil
	} else {
		return path.Join(binDir, name), nil
	}
}

// GetInstalledVersion extracts version from binary output
func GetInstalledVersion(binaryPath string, dependency *Dependency) (string, error) {
	if dependency.VersionCommand == "" {
		dependency.VersionCommand = "--version"
	}

	cmd := osExec.Command(binaryPath, dependency.VersionCommand)
	output, err := cmd.Output()
	if err != nil {
		// Try alternative version commands
		for _, altCmd := range []string{"-v", "version", "--version"} {
			if altCmd != dependency.VersionCommand {
				cmd := osExec.Command(binaryPath, altCmd)
				if output, err = cmd.Output(); err == nil {
					break
				}
			}
		}
		if err != nil {
			return "", fmt.Errorf("failed to get version from %s: %w", binaryPath, err)
		}
	}

	return ExtractVersionFromOutput(string(output), dependency.VersionPattern)
}

// GetInstalledVersionByCommand extracts version by running command directly
func GetInstalledVersionByCommand(command string, dependency *Dependency) (string, error) {
	if dependency.VersionCommand == "" {
		dependency.VersionCommand = "--version"
	}

	cmd := osExec.Command(command, dependency.VersionCommand)
	output, err := cmd.Output()
	if err != nil {
		// Try alternative version commands
		for _, altCmd := range []string{"-v", "version", "--version"} {
			if altCmd != dependency.VersionCommand {
				cmd := osExec.Command(command, altCmd)
				if output, err = cmd.Output(); err == nil {
					break
				}
			}
		}
		if err != nil {
			return "", fmt.Errorf("failed to get version from %s: %w", command, err)
		}
	}

	return ExtractVersionFromOutput(string(output), dependency.VersionPattern)
}

// ExtractVersionFromOutput extracts version using regex pattern
func ExtractVersionFromOutput(output, pattern string) (string, error) {
	if pattern == "" {
		// Default pattern for common version formats
		pattern = `v?(\d+(?:\.\d+)*(?:\.\d+)*(?:-[a-zA-Z0-9-_.]+)?)`
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid version pattern: %w", err)
	}

	matches := re.FindStringSubmatch(output)
	if len(matches) < 2 {
		return "", fmt.Errorf("version not found in output")
	}

	return strings.TrimPrefix(matches[1], "v"), nil
}

// CompareVersions compares two version strings using semver
func CompareVersions(installed, required string, mode VersionCheckMode) (bool, error) {
	if mode == VersionCheckNone {
		return true, nil
	}

	// Clean up version strings and ensure they have v prefix for semver parsing
	installed = strings.TrimSpace(installed)
	required = strings.TrimSpace(required)

	if !strings.HasPrefix(installed, "v") {
		installed = "v" + installed
	}
	if !strings.HasPrefix(required, "v") {
		required = "v" + required
	}

	if mode == VersionCheckExact {
		return strings.TrimPrefix(installed, "v") == strings.TrimPrefix(required, "v"), nil
	}

	// Parse versions using semver
	installedVer, err := semver.NewVersion(installed)
	if err != nil {
		return false, fmt.Errorf("failed to parse installed version %s: %w", installed, err)
	}

	requiredVer, err := semver.NewVersion(required)
	if err != nil {
		return false, fmt.Errorf("failed to parse required version %s: %w", required, err)
	}

	switch mode {
	case VersionCheckMinimum:
		return installedVer.GreaterThan(requiredVer) || installedVer.Equal(requiredVer), nil
	case VersionCheckCompatible:
		// Same major version and installed >= required
		return installedVer.Major() == requiredVer.Major() &&
			(installedVer.GreaterThan(requiredVer) || installedVer.Equal(requiredVer)), nil
	}

	return false, fmt.Errorf("unknown version check mode: %d", mode)
}

// checkVersionRequirement checks if installed version meets requirement
func checkVersionRequirement(binaryPath, requiredVersion string, mode VersionCheckMode, dependency *Dependency) (bool, error) {
	installedVersion, err := GetInstalledVersion(binaryPath, dependency)
	if err != nil {
		return false, err
	}

	return CompareVersions(installedVersion, requiredVersion, mode)
}

// checkVersionRequirementByCommand checks version requirement by command name
func checkVersionRequirementByCommand(command, requiredVersion string, mode VersionCheckMode, dependency *Dependency) (bool, error) {
	installedVersion, err := GetInstalledVersionByCommand(command, dependency)
	if err != nil {
		return false, err
	}

	return CompareVersions(installedVersion, requiredVersion, mode)
}

// downloadWithOptions downloads with checksum verification if enabled
func downloadWithOptions(url, dest string, options *InstallOptions, dependency *Dependency, data map[string]string) error {
	if !options.SkipChecksum && dependency.Checksums != nil {
		platform := fmt.Sprintf("%s-%s", options.GOOS, options.GOARCH)
		if expectedChecksum, ok := dependency.Checksums[platform]; ok {
			return downloadWithChecksum(url, dest, expectedChecksum)
		}
	}

	return download(url, dest)
}

// downloadWithChecksum downloads and verifies checksum
func downloadWithChecksum(url, dest, expectedChecksum string) error {
	if err := net.Download(url, dest); err != nil {
		return err
	}

	return verifyFileChecksum(dest, expectedChecksum)
}

// verifyFileChecksum verifies SHA256 checksum of a file
func verifyFileChecksum(filepath, expectedChecksum string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	expectedChecksum = strings.TrimSpace(expectedChecksum)

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func template(template string, vars map[string]string) (string, error) {
	tpl := gotemplate.New("")

	tpl, err := tpl.Parse(template)

	if err != nil {
		return "", fmt.Errorf("invalid template %s: %v", strings.Split(template, "\n")[0], err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("error executing template %s: %v", strings.Split(template, "\n")[0], err)
	}
	return buf.String(), nil
}
