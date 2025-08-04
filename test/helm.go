package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

// HelmChart represents a Helm chart with fluent interface
type HelmChart struct {
	releaseName    string
	namespace      string
	chartPath      string
	values         map[string]interface{}
	wait           bool
	timeout        time.Duration
	passwordSecret string
	colorOutput    bool
	dryRun         bool

	// Command execution state
	runner     *CommandRunner
	lastResult CommandResult
	lastError  error
}

// NewHelmChart creates a new HelmChart builder
func NewHelmChart(chartPath string) *HelmChart {
	return &HelmChart{
		chartPath:   chartPath,
		colorOutput: true,
		timeout:     5 * time.Minute,
		values:      make(map[string]interface{}),
		runner:      NewCommandRunner(true),
	}
}

// Release sets the release name
func (h *HelmChart) Release(name string) *HelmChart {
	h.releaseName = name
	return h
}

// Namespace sets the namespace
func (h *HelmChart) Namespace(ns string) *HelmChart {
	h.namespace = ns
	return h
}

// Values sets or merges Helm values
func (h *HelmChart) Values(values map[string]interface{}) *HelmChart {
	for k, v := range values {
		h.values[k] = v
	}
	return h
}

// SetValue sets a single value using dot notation
func (h *HelmChart) SetValue(key string, value interface{}) *HelmChart {
	parts := strings.Split(key, ".")
	m := h.values
	for i, part := range parts {
		if i == len(parts)-1 {
			m[part] = value
		} else {
			if _, ok := m[part]; !ok {
				m[part] = make(map[string]interface{})
			}
			m = m[part].(map[string]interface{})
		}
	}
	return h
}

// Wait enables waiting for resources to be ready
func (h *HelmChart) Wait() *HelmChart {
	h.wait = true
	return h
}

// WaitFor sets the wait timeout
func (h *HelmChart) WaitFor(timeout time.Duration) *HelmChart {
	h.wait = true
	h.timeout = timeout
	return h
}

// WithPassword creates a password secret and uses it
func (h *HelmChart) WithPassword(secretName string) *HelmChart {
	h.passwordSecret = secretName
	return h
}

// DryRun enables dry-run mode
func (h *HelmChart) DryRun() *HelmChart {
	h.dryRun = true
	return h
}

// NoColor disables colored output
func (h *HelmChart) NoColor() *HelmChart {
	h.colorOutput = false
	h.runner = NewCommandRunner(false)
	return h
}

// Install installs the Helm chart
func (h *HelmChart) Install() *HelmChart {
	if h.releaseName == "" {
		h.lastError = fmt.Errorf("release name is required")
		return h
	}

	h.runner.Printf(colorYellow, colorBold, "=== Helm Install: %s ===", h.releaseName)

	// Handle password secret if specified
	if h.passwordSecret != "" {
		if err := h.createPasswordSecret(); err != nil {
			h.lastError = err
			return h
		}
	}

	args := []string{"install", h.releaseName, h.chartPath}
	args = h.appendCommonArgs(args)

	if h.dryRun {
		args = append(args, "--dry-run")
	}

	h.lastResult = h.runner.RunCommand("helm", args...)
	if h.lastResult.Err != nil {
		h.lastError = fmt.Errorf("helm install failed: %s", h.lastResult.String())
		h.collectDiagnostics()
	}
	return h
}

// Upgrade upgrades the Helm release
func (h *HelmChart) Upgrade() *HelmChart {
	if h.releaseName == "" {
		h.lastError = fmt.Errorf("release name is required")
		return h
	}

	h.runner.Printf(colorYellow, colorBold, "=== Helm Upgrade: %s ===", h.releaseName)

	// Handle password secret if specified
	if h.passwordSecret != "" {
		if err := h.createPasswordSecret(); err != nil {
			h.lastError = err
			return h
		}
	}

	args := []string{"upgrade", h.releaseName, h.chartPath}
	args = h.appendCommonArgs(args)

	if h.dryRun {
		args = append(args, "--dry-run")
	}

	h.lastResult = h.runner.RunCommand("helm", args...)
	if h.lastResult.Err != nil {
		h.lastError = fmt.Errorf("helm upgrade failed: %s", h.lastResult.String())
		h.collectDiagnostics()
	}
	return h
}

// Delete deletes the Helm release
func (h *HelmChart) Delete() *HelmChart {
	if h.releaseName == "" {
		h.lastError = fmt.Errorf("release name is required")
		return h
	}

	args := []string{"delete", h.releaseName}
	if h.namespace != "" {
		args = append(args, "--namespace", h.namespace)
	}
	if h.wait {
		args = append(args, "--wait")
	}

	h.lastResult = h.runner.RunCommand("helm", args...)
	if h.lastResult.Err != nil && !strings.Contains(h.lastResult.Stderr, "not found") {
		h.lastError = fmt.Errorf("helm delete failed: %s", h.lastResult.String())
	}
	return h
}

// GetPod returns a Pod accessor for the current release
func (h *HelmChart) GetPod(selector string) *Pod {
	return &Pod{
		namespace:   h.namespace,
		selector:    selector,
		helm:        h,
		colorOutput: h.colorOutput,
	}
}

// GetStatefulSet returns a StatefulSet accessor
func (h *HelmChart) GetStatefulSet(name string) *StatefulSet {
	return &StatefulSet{
		name:        name,
		namespace:   h.namespace,
		helm:        h,
		colorOutput: h.colorOutput,
	}
}

// GetSecret returns a Secret accessor
func (h *HelmChart) GetSecret(name string) *Secret {
	return &Secret{
		name:        name,
		namespace:   h.namespace,
		helm:        h,
		colorOutput: h.colorOutput,
	}
}

// GetConfigMap returns a ConfigMap accessor
func (h *HelmChart) GetConfigMap(name string) *ConfigMap {
	return &ConfigMap{
		name:        name,
		namespace:   h.namespace,
		helm:        h,
		colorOutput: h.colorOutput,
	}
}

// GetPVC returns a PersistentVolumeClaim accessor
func (h *HelmChart) GetPVC(name string) *PVC {
	return &PVC{
		name:        name,
		namespace:   h.namespace,
		helm:        h,
		colorOutput: h.colorOutput,
	}
}

// Status returns the Helm release status
func (h *HelmChart) Status() (string, error) {
	args := []string{"status", h.releaseName}
	if h.namespace != "" {
		args = append(args, "-n", h.namespace)
	}
	result := h.runner.RunCommand("helm", args...)
	return result.Stdout, result.Err
}

// Error returns the last error
func (h *HelmChart) Error() error {
	return h.lastError
}

// Result returns the last command result
func (h *HelmChart) Result() CommandResult {
	return h.lastResult
}

// MustSucceed panics if there was an error
func (h *HelmChart) MustSucceed() *HelmChart {
	if h.lastError != nil {
		panic(h.lastError)
	}
	return h
}

// Pod represents a Kubernetes pod with fluent interface
type Pod struct {
	namespace   string
	selector    string
	name        string
	container   string
	helm        *HelmChart
	colorOutput bool
	lastResult  CommandResult
	lastError   error
}

// Container sets the container name
func (p *Pod) Container(name string) *Pod {
	p.container = name
	return p
}

// WaitReady waits for the pod to be ready
func (p *Pod) WaitReady() *Pod {
	return p.WaitFor("condition=Ready", 2*time.Minute)
}

// WaitFor waits for a specific condition
func (p *Pod) WaitFor(condition string, timeout time.Duration) *Pod {
	args := []string{"wait", "pod"}
	if p.namespace != "" {
		args = append(args, "-n", p.namespace)
	}
	if p.selector != "" {
		args = append(args, "-l", p.selector)
	}
	args = append(args, "--for="+condition, "--timeout="+timeout.String())

	p.lastResult = p.runCommand("kubectl", args...)
	if p.lastResult.Err != nil {
		p.lastError = fmt.Errorf("wait failed: %s", p.lastResult.String())
	}
	return p
}

// Exec executes a command in the pod
func (p *Pod) Exec(command string) *Pod {
	// Get pod name if not set
	if p.name == "" && p.selector != "" {
		if err := p.resolvePodName(); err != nil {
			p.lastError = err
			return p
		}
	}

	args := []string{"exec", "-n", p.namespace, p.name}
	if p.container != "" {
		args = append(args, "-c", p.container)
	}
	args = append(args, "--", "bash", "-c", command)

	p.lastResult = p.runCommand("kubectl", args...)
	if p.lastResult.Err != nil {
		p.lastError = fmt.Errorf("exec failed: %s", p.lastResult.String())
	}
	return p
}

// GetLogs retrieves pod logs
func (p *Pod) GetLogs(lines ...int) string {
	// Get pod name if not set
	if p.name == "" && p.selector != "" {
		if err := p.resolvePodName(); err != nil {
			p.lastError = err
			return ""
		}
	}

	args := []string{"logs", "-n", p.namespace, p.name}
	if p.container != "" {
		args = append(args, "-c", p.container)
	}
	if len(lines) > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", lines[0]))
	}

	p.lastResult = p.runCommand("kubectl", args...)
	return p.lastResult.Stdout
}

// Status returns the pod status
func (p *Pod) Status() (string, error) {
	// Get pod name if not set
	if p.name == "" && p.selector != "" {
		if err := p.resolvePodName(); err != nil {
			return "", err
		}
	}

	args := []string{"get", "pod", p.name, "-n", p.namespace,
		"-o", "jsonpath={.status.phase}"}
	p.lastResult = p.runCommand("kubectl", args...)
	return strings.TrimSpace(p.lastResult.Stdout), p.lastResult.Err
}

// Result returns the last command result
func (p *Pod) Result() string {
	return p.lastResult.Stdout
}

// Error returns the last error
func (p *Pod) Error() error {
	return p.lastError
}

// MustSucceed panics if there was an error
func (p *Pod) MustSucceed() *Pod {
	if p.lastError != nil {
		panic(p.lastError)
	}
	return p
}

// StatefulSet represents a Kubernetes StatefulSet
type StatefulSet struct {
	name        string
	namespace   string
	helm        *HelmChart
	colorOutput bool
	lastResult  CommandResult
	lastError   error
}

// WaitReady waits for the StatefulSet to be ready
func (s *StatefulSet) WaitReady() *StatefulSet {
	return s.WaitFor(2 * time.Minute)
}

// WaitFor waits for the StatefulSet rollout to complete
func (s *StatefulSet) WaitFor(timeout time.Duration) *StatefulSet {
	args := []string{"rollout", "status", "statefulset", s.name,
		"-n", s.namespace, "--timeout=" + timeout.String()}

	s.lastResult = s.runCommand("kubectl", args...)
	if s.lastResult.Err != nil {
		s.lastError = fmt.Errorf("rollout wait failed: %s", s.lastResult.String())
	}
	return s
}

// GetReplicas returns the number of ready replicas
func (s *StatefulSet) GetReplicas() (int, error) {
	args := []string{"get", "statefulset", s.name, "-n", s.namespace,
		"-o", "jsonpath={.status.readyReplicas}"}
	s.lastResult = s.runCommand("kubectl", args...)
	if s.lastResult.Err != nil {
		return 0, s.lastResult.Err
	}

	if s.lastResult.Stdout == "" {
		return 0, nil
	}

	var replicas int
	if _, err := fmt.Sscanf(s.lastResult.Stdout, "%d", &replicas); err != nil {
		return 0, fmt.Errorf("failed to parse replicas: %w", err)
	}
	return replicas, nil
}

// GetGeneration returns the current generation
func (s *StatefulSet) GetGeneration() (int64, error) {
	args := []string{"get", "statefulset", s.name, "-n", s.namespace,
		"-o", "jsonpath={.metadata.generation}"}
	s.lastResult = s.runCommand("kubectl", args...)
	if s.lastResult.Err != nil {
		return 0, s.lastResult.Err
	}

	var gen int64
	if _, err := fmt.Sscanf(strings.TrimSpace(s.lastResult.Stdout), "%d", &gen); err != nil {
		return 0, fmt.Errorf("failed to parse generation: %w", err)
	}
	return gen, nil
}

// Secret represents a Kubernetes Secret
type Secret struct {
	name        string
	namespace   string
	helm        *HelmChart
	colorOutput bool
	lastResult  CommandResult
}

// Get retrieves a secret value by key
func (s *Secret) Get(key string) (string, error) {
	args := []string{"get", "secret", s.name, "-n", s.namespace,
		"-o", fmt.Sprintf("jsonpath={.data.%s}", key)}
	s.lastResult = s.runCommand("kubectl", args...)
	if s.lastResult.Err != nil {
		return "", s.lastResult.Err
	}

	// Decode base64
	cmd := exec.Command("base64", "-d")
	cmd.Stdin = strings.NewReader(s.lastResult.Stdout)
	decoded, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	return string(decoded), nil
}

// ConfigMap represents a Kubernetes ConfigMap
type ConfigMap struct {
	name        string
	namespace   string
	helm        *HelmChart
	colorOutput bool
	lastResult  CommandResult
}

// Get retrieves a ConfigMap value by key
func (c *ConfigMap) Get(key string) (string, error) {
	escapedKey := strings.ReplaceAll(key, ".", "\\.")
	args := []string{"get", "configmap", c.name, "-n", c.namespace,
		"-o", fmt.Sprintf("jsonpath={.data['%s']}", escapedKey)}
	c.lastResult = c.runCommand("kubectl", args...)
	return c.lastResult.Stdout, c.lastResult.Err
}

// PVC represents a PersistentVolumeClaim
type PVC struct {
	name        string
	namespace   string
	helm        *HelmChart
	colorOutput bool
	lastResult  CommandResult
}

// Status returns the PVC status
func (p *PVC) Status() (map[string]interface{}, error) {
	args := []string{"get", "pvc", p.name, "-n", p.namespace, "-o", "json"}
	p.lastResult = p.runCommand("kubectl", args...)
	if p.lastResult.Err != nil {
		return nil, p.lastResult.Err
	}

	var pvc map[string]interface{}
	if err := json.Unmarshal([]byte(p.lastResult.Stdout), &pvc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal PVC: %w", err)
	}

	return pvc, nil
}

// Helper methods

func (h *HelmChart) appendCommonArgs(args []string) []string {
	if h.namespace != "" {
		args = append(args, "--namespace", h.namespace)
	}
	if h.wait {
		args = append(args, "--wait")
	}
	if h.timeout > 0 {
		args = append(args, "--timeout", h.timeout.String())
	}

	// Add values if any
	if len(h.values) > 0 {
		valuesYaml, err := yaml.Marshal(h.values)
		if err != nil {
			h.lastError = fmt.Errorf("failed to marshal values: %w", err)
			return args
		}

		// Write values to temp file
		tempFile := fmt.Sprintf("/tmp/helm-values-%d.yaml", time.Now().UnixNano())
		cmd := exec.Command("sh", "-c", fmt.Sprintf("cat > %s", tempFile))
		cmd.Stdin = bytes.NewReader(valuesYaml)
		if err := cmd.Run(); err != nil {
			h.lastError = fmt.Errorf("failed to write values file: %w", err)
			return args
		}

		args = append(args, "--values", tempFile)
		// Note: In production, should defer cleanup of temp file
	}

	return args
}

func (h *HelmChart) createPasswordSecret() error {
	password := fmt.Sprintf("pass-%d", time.Now().Unix())

	h.runner.Printf(colorYellow, colorBold, "Creating password secret: %s", h.passwordSecret)

	// Create the secret
	result := h.runner.RunCommand("kubectl", "create", "secret", "generic", h.passwordSecret,
		"--from-literal=password="+password,
		"-n", h.namespace,
		"--dry-run=client", "-o", "yaml")

	if result.Err != nil {
		return fmt.Errorf("failed to generate secret yaml: %s", result.String())
	}

	// Apply the secret
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(result.Stdout)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	// Add the secret reference to values
	if h.values == nil {
		h.values = make(map[string]interface{})
	}
	h.values["database"] = map[string]interface{}{
		"existingSecret": h.passwordSecret,
		"secretKey":      "password",
	}

	h.runner.Printf(colorGray, "", "Secret created with password: %s", password)
	return nil
}

func (h *HelmChart) collectDiagnostics() {
	if !h.colorOutput {
		return
	}

	h.runner.Printf(colorRed, colorBold, "=== Collecting Diagnostics ===")

	// Get Helm release status
	h.runner.Printf(colorBlue, "", "● Helm Release Status:")
	h.runner.RunCommand("helm", "status", h.releaseName, "-n", h.namespace)

	// Get pods
	h.runner.Printf(colorBlue, "", "● Pods in namespace %s:", h.namespace)
	h.runner.RunCommand("kubectl", "get", "pods", "-n", h.namespace, "-o", "wide")

	// Get events
	h.runner.Printf(colorBlue, "", "● Recent Events:")
	h.runner.RunCommand("kubectl", "get", "events", "-n", h.namespace,
		"--sort-by=.lastTimestamp")

	h.runner.Printf(colorYellow, colorBold, "=== End of Diagnostics ===")
}

// Similar runCommand methods for Pod, StatefulSet, etc.
func (p *Pod) runCommand(name string, args ...string) CommandResult {
	return p.helm.runner.RunCommand(name, args...)
}

func (s *StatefulSet) runCommand(name string, args ...string) CommandResult {
	return s.helm.runner.RunCommand(name, args...)
}

func (sec *Secret) runCommand(name string, args ...string) CommandResult {
	return sec.helm.runner.RunCommand(name, args...)
}

func (c *ConfigMap) runCommand(name string, args ...string) CommandResult {
	return c.helm.runner.RunCommand(name, args...)
}

func (p *PVC) runCommand(name string, args ...string) CommandResult {
	return p.helm.runner.RunCommand(name, args...)
}

func (p *Pod) resolvePodName() error {
	args := []string{"get", "pods", "-n", p.namespace, "-l", p.selector,
		"-o", "jsonpath={.items[0].metadata.name}"}
	p.lastResult = p.runCommand("kubectl", args...)
	if p.lastResult.Err != nil {
		return fmt.Errorf("failed to get pod name: %w", p.lastResult.Err)
	}
	p.name = strings.TrimSpace(p.lastResult.Stdout)
	if p.name == "" {
		return fmt.Errorf("no pod found with selector: %s", p.selector)
	}
	return nil
}

// Namespace represents a Kubernetes namespace with fluent interface
type Namespace struct {
	name        string
	colorOutput bool
	runner      *CommandRunner
	lastResult  CommandResult
	lastError   error
}

// NewNamespace creates a new Namespace accessor
func NewNamespace(name string) *Namespace {
	return &Namespace{
		name:        name,
		colorOutput: true,
		runner:      NewCommandRunner(true),
	}
}

// Create creates the namespace
func (n *Namespace) Create() *Namespace {
	result := n.runCommand("kubectl", "create", "namespace", n.name)
	if result.Err != nil && strings.Contains(result.Stderr, "already exists") {
		// Namespace already exists, that's ok
		n.lastError = nil
	} else if result.Err != nil {
		n.lastError = fmt.Errorf("failed to create namespace: %s", result.String())
	}
	n.lastResult = result
	return n
}

// Delete deletes the namespace
func (n *Namespace) Delete() *Namespace {
	result := n.runCommand("kubectl", "delete", "namespace", n.name, "--wait=false")
	if result.Err != nil {
		n.lastError = fmt.Errorf("failed to delete namespace: %s", result.String())
	}
	n.lastResult = result
	return n
}

// MustSucceed panics if there was an error
func (n *Namespace) MustSucceed() *Namespace {
	if n.lastError != nil {
		panic(n.lastError)
	}
	return n
}

func (n *Namespace) runCommand(name string, args ...string) CommandResult {
	return n.runner.RunCommand(name, args...)
}
