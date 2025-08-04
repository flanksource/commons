package test

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Kind struct {
	Version     string `yaml:"version"`
	Name        string `yaml:"name"`
	UseExisting bool   `yaml:"use_existing"`
	ColorOutput bool   `yaml:"color_output"`
	
	runner     *CommandRunner
	lastResult CommandResult
	lastError  error
}

// NewKind creates a new Kind cluster manager
func NewKind(name string) *Kind {
	if name == "" {
		name = "kind"
	}
	return &Kind{
		Name:        name,
		Version:     "latest",
		ColorOutput: true,
		runner:      NewCommandRunner(true),
	}
}

// WithVersion sets the kind version to use
func (k *Kind) WithVersion(version string) *Kind {
	k.Version = version
	return k
}

// NoColor disables colored output
func (k *Kind) NoColor() *Kind {
	k.ColorOutput = false
	k.runner = NewCommandRunner(false)
	return k
}

// GetOrCreate gets an existing kind cluster or creates a new one
func (k *Kind) GetOrCreate() *Kind {
	k.runner.Printf(colorYellow, colorBold, "=== Kind Cluster: %s ===", k.Name)
	
	// Check if cluster already exists
	result := k.runner.RunCommandQuiet("kind", "get", "clusters")
	if result.Err == nil {
		clusters := strings.Split(strings.TrimSpace(result.Stdout), "\n")
		for _, cluster := range clusters {
			if cluster == k.Name {
				k.runner.Printf(colorGray, "", "Using existing cluster: %s", k.Name)
				k.Use()
				return k
			}
		}
	}
	
	// Create new cluster
	k.runner.Printf(colorBlue, "", "Creating new cluster: %s", k.Name)
	
	args := []string{"create", "cluster", "--name", k.Name}
	if k.Version != "" && k.Version != "latest" {
		args = append(args, "--image", fmt.Sprintf("kindest/node:%s", k.Version))
	}
	
	k.lastResult = k.runner.RunCommand("kind", args...)
	if k.lastResult.Err != nil {
		k.lastError = fmt.Errorf("failed to create kind cluster: %s", k.lastResult.String())
		return k
	}
	
	// Wait for cluster to be ready
	k.runner.Printf(colorGray, "", "Waiting for cluster to be ready...")
	k.waitForCluster()
	
	k.Use()
	return k
}

// Use updates KUBECONFIG to use the kind cluster
func (k *Kind) Use() *Kind {
	k.runner.Printf(colorBlue, "", "Switching to cluster context: kind-%s", k.Name)
	
	// Export kubeconfig for the kind cluster
	result := k.runner.RunCommandQuiet("kind", "export", "kubeconfig", "--name", k.Name)
	if result.Err != nil {
		k.lastError = fmt.Errorf("failed to export kubeconfig: %s", result.String())
		return k
	}
	
	// Set the current context
	contextName := fmt.Sprintf("kind-%s", k.Name)
	k.lastResult = k.runner.RunCommand("kubectl", "config", "use-context", contextName)
	if k.lastResult.Err != nil {
		k.lastError = fmt.Errorf("failed to switch context: %s", k.lastResult.String())
		return k
	}
	
	// Verify connection
	k.runner.Printf(colorGray, "", "Verifying cluster connection...")
	result = k.runner.RunCommandQuiet("kubectl", "cluster-info", "--context", contextName)
	if result.Err != nil {
		k.lastError = fmt.Errorf("failed to verify cluster connection: %s", result.String())
		return k
	}
	
	k.runner.Printf(colorGray, "", "Successfully connected to cluster: %s", k.Name)
	return k
}

// Delete deletes the kind cluster
func (k *Kind) Delete() *Kind {
	k.runner.Printf(colorYellow, colorBold, "=== Deleting Kind Cluster: %s ===", k.Name)
	
	k.lastResult = k.runner.RunCommand("kind", "delete", "cluster", "--name", k.Name)
	if k.lastResult.Err != nil {
		k.lastError = fmt.Errorf("failed to delete kind cluster: %s", k.lastResult.String())
	}
	return k
}

// LoadImage loads a docker image into the kind cluster
func (k *Kind) LoadImage(image string) *Kind {
	k.runner.Printf(colorBlue, "", "Loading image into cluster: %s", image)
	
	k.lastResult = k.runner.RunCommand("kind", "load", "docker-image", image, "--name", k.Name)
	if k.lastResult.Err != nil {
		k.lastError = fmt.Errorf("failed to load image: %s", k.lastResult.String())
	}
	return k
}

// GetKubeconfig returns the kubeconfig for the kind cluster
func (k *Kind) GetKubeconfig() (string, error) {
	result := k.runner.RunCommandQuiet("kind", "get", "kubeconfig", "--name", k.Name)
	if result.Err != nil {
		return "", fmt.Errorf("failed to get kubeconfig: %s", result.String())
	}
	return result.Stdout, nil
}

// Exists checks if the kind cluster exists
func (k *Kind) Exists() bool {
	result := k.runner.RunCommandQuiet("kind", "get", "clusters")
	if result.Err != nil {
		return false
	}
	
	clusters := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	for _, cluster := range clusters {
		if cluster == k.Name {
			return true
		}
	}
	return false
}

// Error returns the last error
func (k *Kind) Error() error {
	return k.lastError
}

// Result returns the last command result
func (k *Kind) Result() CommandResult {
	return k.lastResult
}

// MustSucceed panics if there was an error
func (k *Kind) MustSucceed() *Kind {
	if k.lastError != nil {
		panic(k.lastError)
	}
	return k
}

// waitForCluster waits for the cluster to be ready
func (k *Kind) waitForCluster() {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		result := k.runner.RunCommandQuiet("kubectl", "get", "nodes")
		if result.Err == nil && strings.Contains(result.Stdout, "Ready") {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

// SetKubeconfig sets the KUBECONFIG environment variable to use the kind cluster
func (k *Kind) SetKubeconfig() *Kind {
	kubeconfig, err := k.GetKubeconfig()
	if err != nil {
		k.lastError = err
		return k
	}
	
	// Write kubeconfig to temp file
	tempFile := fmt.Sprintf("/tmp/kind-%s-kubeconfig-%d", k.Name, time.Now().UnixNano())
	cmd := k.runner.RunCommandQuiet("sh", "-c", fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", tempFile, kubeconfig))
	if cmd.Err != nil {
		k.lastError = fmt.Errorf("failed to write kubeconfig: %w", cmd.Err)
		return k
	}
	
	// Set KUBECONFIG environment variable
	os.Setenv("KUBECONFIG", tempFile)
	k.runner.Printf(colorGray, "", "KUBECONFIG set to: %s", tempFile)
	
	return k
}
