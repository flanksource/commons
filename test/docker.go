package test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DockerClient provides Docker operations with colored output
type DockerClient struct {
	runner *CommandRunner
}

// NewDockerClient creates a new DockerClient
func NewDockerClient(colorOutput bool) *DockerClient {
	return &DockerClient{
		runner: NewCommandRunner(colorOutput),
	}
}

// InstallClient installs Docker client if not already present
func InstallClient() error {
	runner := NewCommandRunner(true)

	// Check if docker is already installed
	result := runner.RunCommandQuiet("which", "docker")
	if result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "" {
		runner.Printf(colorGray, "", "Docker client already installed at: %s", strings.TrimSpace(result.Stdout))
		return nil
	}

	runner.Printf(colorBlue, colorBold, "Installing Docker client...")

	// Detect OS and architecture
	unameResult := runner.RunCommandQuiet("uname", "-s")
	if unameResult.ExitCode != 0 {
		return fmt.Errorf("failed to detect OS: %v", unameResult.Err)
	}
	osName := strings.ToLower(strings.TrimSpace(unameResult.Stdout))

	archResult := runner.RunCommandQuiet("uname", "-m")
	if archResult.ExitCode != 0 {
		return fmt.Errorf("failed to detect architecture: %v", archResult.Err)
	}
	arch := strings.TrimSpace(archResult.Stdout)

	// Map architecture names
	switch arch {
	case "x86_64":
		arch = "amd64"
	case "aarch64", "arm64":
		arch = "arm64"
	}

	// Download URL
	downloadURL := fmt.Sprintf("https://download.docker.com/%s/static/stable/%s/docker-24.0.7.tgz", osName, arch)

	// Create temp directory
	tempDir := "/tmp/docker-install"
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download Docker
	runner.Printf(colorGray, "", "Downloading from: %s", downloadURL)
	downloadResult := runner.RunCommand("curl", "-fsSL", "-o", filepath.Join(tempDir, "docker.tgz"), downloadURL)
	if downloadResult.ExitCode != 0 {
		return fmt.Errorf("failed to download Docker: %v", downloadResult.Err)
	}

	// Extract
	extractResult := runner.RunCommand("tar", "-xzf", filepath.Join(tempDir, "docker.tgz"), "-C", tempDir)
	if extractResult.ExitCode != 0 {
		return fmt.Errorf("failed to extract Docker: %v", extractResult.Err)
	}

	// Install to /usr/local/bin
	installPath := "/usr/local/bin/docker"
	runner.Printf(colorGray, "", "Installing to: %s", installPath)

	// Try to copy with sudo if regular copy fails
	copyResult := runner.RunCommandQuiet("cp", filepath.Join(tempDir, "docker/docker"), installPath)
	if copyResult.ExitCode != 0 {
		runner.Printf(colorYellow, "", "Regular copy failed, trying with sudo...")
		sudoResult := runner.RunCommand("sudo", "cp", filepath.Join(tempDir, "docker/docker"), installPath)
		if sudoResult.ExitCode != 0 {
			return fmt.Errorf("failed to install Docker binary: %v", sudoResult.Err)
		}

		// Make executable
		chmodResult := runner.RunCommand("sudo", "chmod", "+x", installPath)
		if chmodResult.ExitCode != 0 {
			return fmt.Errorf("failed to make Docker executable: %v", chmodResult.Err)
		}
	} else {
		// Make executable without sudo
		chmodResult := runner.RunCommand("chmod", "+x", installPath)
		if chmodResult.ExitCode != 0 {
			return fmt.Errorf("failed to make Docker executable: %v", chmodResult.Err)
		}
	}

	// Verify installation
	verifyResult := runner.RunCommand("docker", "--version")
	if verifyResult.ExitCode != 0 {
		return fmt.Errorf("Docker installation verification failed: %v", verifyResult.Err)
	}

	runner.Printf(colorGray, colorBold, "Docker client installed successfully")
	return nil
}

// InstallBuildx installs Docker buildx plugin
func InstallBuildx() error {
	runner := NewCommandRunner(true)

	// Check if buildx is already available
	result := runner.RunCommandQuiet("docker", "buildx", "version")
	if result.ExitCode == 0 {
		runner.Printf(colorGray, "", "Docker buildx already installed: %s", strings.TrimSpace(result.Stdout))
		return nil
	}

	runner.Printf(colorBlue, colorBold, "Installing Docker buildx plugin...")

	// Detect OS and architecture
	unameResult := runner.RunCommandQuiet("uname", "-s")
	if unameResult.ExitCode != 0 {
		return fmt.Errorf("failed to detect OS: %v", unameResult.Err)
	}
	osName := strings.ToLower(strings.TrimSpace(unameResult.Stdout))

	archResult := runner.RunCommandQuiet("uname", "-m")
	if archResult.ExitCode != 0 {
		return fmt.Errorf("failed to detect architecture: %v", archResult.Err)
	}
	arch := strings.TrimSpace(archResult.Stdout)

	// Map architecture names
	switch arch {
	case "x86_64":
		arch = "amd64"
	case "aarch64", "arm64":
		arch = "arm64"
	}

	// Create plugin directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	pluginDir := filepath.Join(homeDir, ".docker", "cli-plugins")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Download buildx
	buildxVersion := "v0.12.0"
	downloadURL := fmt.Sprintf("https://github.com/docker/buildx/releases/download/%s/buildx-%s.%s-%s",
		buildxVersion, buildxVersion, osName, arch)

	buildxPath := filepath.Join(pluginDir, "docker-buildx")

	runner.Printf(colorGray, "", "Downloading from: %s", downloadURL)
	downloadResult := runner.RunCommand("curl", "-fsSL", "-o", buildxPath, downloadURL)
	if downloadResult.ExitCode != 0 {
		return fmt.Errorf("failed to download buildx: %v", downloadResult.Err)
	}

	// Make executable
	chmodResult := runner.RunCommand("chmod", "+x", buildxPath)
	if chmodResult.ExitCode != 0 {
		return fmt.Errorf("failed to make buildx executable: %v", chmodResult.Err)
	}

	// Verify installation
	verifyResult := runner.RunCommand("docker", "buildx", "version")
	if verifyResult.ExitCode != 0 {
		return fmt.Errorf("buildx installation verification failed: %v", verifyResult.Err)
	}

	// Create and use a new builder instance
	runner.Printf(colorGray, "", "Creating buildx builder instance...")
	createResult := runner.RunCommand("docker", "buildx", "create", "--use", "--name", "mybuilder", "--driver", "docker-container")
	if createResult.ExitCode != 0 {
		// Builder might already exist, try to use it
		useResult := runner.RunCommand("docker", "buildx", "use", "mybuilder")
		if useResult.ExitCode != 0 {
			runner.Printf(colorYellow, "", "Warning: Could not create or use buildx builder")
		}
	}

	runner.Printf(colorGray, colorBold, "Docker buildx installed successfully")
	return nil
}

// LoadImage loads a Docker image from a tar file
func LoadImage(imagePath string) error {
	client := NewDockerClient(true)
	client.runner.Printf(colorBlue, colorBold, "Loading Docker image from: %s", imagePath)

	result := client.runner.RunCommand("docker", "load", "-i", imagePath)
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to load image: %v", result.Err)
	}

	// Extract loaded image name from output
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Loaded image:") {
			imageName := strings.TrimSpace(strings.TrimPrefix(line, "Loaded image:"))
			client.runner.Printf(colorGray, colorBold, "Successfully loaded image: %s", imageName)
			break
		}
	}

	return nil
}

// Container represents a running Docker container
type Container struct {
	ID     string
	Name   string
	Image  string
	client *DockerClient
}

// ContainerOptions provides options for running a container
type ContainerOptions struct {
	Name          string
	Image         string
	Command       []string
	Env           map[string]string
	Ports         map[string]string // host:container
	Volumes       map[string]string // host:container
	Network       string
	Detach        bool
	Remove        bool
	Privileged    bool
	HealthCheck   string
	HealthRetries int
	WorkingDir    string
	User          string
	Labels        map[string]string
}

// Run creates and starts a new Docker container
func Run(opts ContainerOptions) (*Container, error) {
	client := NewDockerClient(true)

	if opts.Image == "" {
		return nil, fmt.Errorf("image is required")
	}

	args := []string{"run"}

	if opts.Detach {
		args = append(args, "-d")
	}

	if opts.Remove {
		args = append(args, "--rm")
	}

	if opts.Privileged {
		args = append(args, "--privileged")
	}

	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}

	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}

	if opts.WorkingDir != "" {
		args = append(args, "-w", opts.WorkingDir)
	}

	if opts.User != "" {
		args = append(args, "-u", opts.User)
	}

	// Add environment variables
	for key, value := range opts.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add port mappings
	for hostPort, containerPort := range opts.Ports {
		args = append(args, "-p", fmt.Sprintf("%s:%s", hostPort, containerPort))
	}

	// Add volume mappings
	for hostPath, containerPath := range opts.Volumes {
		args = append(args, "-v", fmt.Sprintf("%s:%s", hostPath, containerPath))
	}

	// Add labels
	for key, value := range opts.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add health check if specified
	if opts.HealthCheck != "" {
		args = append(args, "--health-cmd", opts.HealthCheck)
		if opts.HealthRetries > 0 {
			args = append(args, "--health-retries", strconv.Itoa(opts.HealthRetries))
		}
	}

	// Add image
	args = append(args, opts.Image)

	// Add command if specified
	if len(opts.Command) > 0 {
		args = append(args, opts.Command...)
	}

	client.runner.Printf(colorBlue, colorBold, "Starting container: %s", opts.Name)

	result := client.runner.RunCommand("docker", args...)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to run container: %v", result.Err)
	}

	containerID := strings.TrimSpace(result.Stdout)
	if containerID == "" {
		return nil, fmt.Errorf("no container ID returned")
	}

	container := &Container{
		ID:     containerID,
		Name:   opts.Name,
		Image:  opts.Image,
		client: client,
	}

	// If health check is specified, wait for it
	if opts.HealthCheck != "" && opts.Detach {
		client.runner.Printf(colorGray, "", "Waiting for container to be healthy...")
		maxWait := 30 * time.Second
		if err := container.waitForHealth(maxWait); err != nil {
			return container, fmt.Errorf("container failed health check: %w", err)
		}
	}

	return container, nil
}

// Stop stops the container
func (c *Container) Stop() error {
	c.client.runner.Printf(colorYellow, "", "Stopping container: %s", c.getIdentifier())
	result := c.client.runner.RunCommand("docker", "stop", c.ID)
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to stop container: %v", result.Err)
	}
	return nil
}

// Start starts a stopped container
func (c *Container) Start() error {
	c.client.runner.Printf(colorBlue, "", "Starting container: %s", c.getIdentifier())
	result := c.client.runner.RunCommand("docker", "start", c.ID)
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to start container: %v", result.Err)
	}
	return nil
}

// Delete removes the container
func (c *Container) Delete() error {
	c.client.runner.Printf(colorRed, "", "Deleting container: %s", c.getIdentifier())
	result := c.client.runner.RunCommand("docker", "rm", "-f", c.ID)
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to delete container: %v", result.Err)
	}
	return nil
}

// Logs retrieves container logs
func (c *Container) Logs() (string, error) {
	result := c.client.runner.RunCommandQuiet("docker", "logs", c.ID)
	if result.ExitCode != 0 {
		return "", fmt.Errorf("failed to get logs: %v", result.Err)
	}
	return result.Stdout + result.Stderr, nil
}

// Exec executes a command inside the container
func (c *Container) Exec(command ...string) (string, error) {
	args := []string{"exec", c.ID}
	args = append(args, command...)

	c.client.runner.Printf(colorGray, "", "Executing in container: %s", strings.Join(command, " "))
	result := c.client.runner.RunCommand("docker", args...)
	if result.ExitCode != 0 {
		return result.Stdout + result.Stderr, fmt.Errorf("command failed with exit code %d: %v", result.ExitCode, result.Err)
	}
	return result.Stdout, nil
}

// GetFile copies a file from the container to the host
func (c *Container) GetFile(containerPath, hostPath string) error {
	c.client.runner.Printf(colorGray, "", "Copying from container: %s:%s -> %s", c.getIdentifier(), containerPath, hostPath)

	// Ensure host directory exists
	hostDir := filepath.Dir(hostPath)
	if err := os.MkdirAll(hostDir, 0755); err != nil {
		return fmt.Errorf("failed to create host directory: %w", err)
	}

	result := c.client.runner.RunCommand("docker", "cp", fmt.Sprintf("%s:%s", c.ID, containerPath), hostPath)
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to copy file: %v", result.Err)
	}
	return nil
}

// WaitFor waits for a specific port to be available in the container
func (c *Container) WaitFor(port int, timeout time.Duration) error {
	c.client.runner.Printf(colorGray, "", "Waiting for port %d to be available...", port)

	// Get container IP
	inspectResult := c.client.runner.RunCommandQuiet("docker", "inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", c.ID)
	if inspectResult.ExitCode != 0 {
		return fmt.Errorf("failed to get container IP: %v", inspectResult.Err)
	}

	containerIP := strings.TrimSpace(inspectResult.Stdout)
	if containerIP == "" {
		// Try localhost if no IP (might be using host network)
		containerIP = "127.0.0.1"
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", containerIP, port), 1*time.Second)
		if err == nil {
			conn.Close()
			c.client.runner.Printf(colorGray, colorBold, "Port %d is now available", port)
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for port %d after %v", port, timeout)
}

// getIdentifier returns the container name if available, otherwise the ID
func (c *Container) getIdentifier() string {
	if c.Name != "" {
		return c.Name
	}
	return c.ID[:12] // First 12 chars of ID
}

// waitForHealth waits for the container to become healthy
func (c *Container) waitForHealth(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		// Check container health
		result := c.client.runner.RunCommandQuiet("docker", "inspect", "--format", "{{.State.Health.Status}}", c.ID)
		if result.ExitCode == 0 {
			health := strings.TrimSpace(result.Stdout)
			if health == "healthy" {
				c.client.runner.Printf(colorGray, colorBold, "Container is healthy")
				return nil
			}
			if health == "unhealthy" {
				// Get last health check log
				logsResult := c.client.runner.RunCommandQuiet("docker", "inspect", "--format", "{{json .State.Health.Log}}", c.ID)
				if logsResult.ExitCode == 0 {
					var logs []interface{}
					if err := json.Unmarshal([]byte(logsResult.Stdout), &logs); err == nil && len(logs) > 0 {
						c.client.runner.Printf(colorRed, "", "Health check failed: %v", logs[len(logs)-1])
					}
				}
				return fmt.Errorf("container is unhealthy")
			}
		}

		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timeout waiting for container to be healthy after %v", timeout)
}

// StreamLogs streams container logs in real-time
func (c *Container) StreamLogs(ctx context.Context, writer io.Writer) error {
	cmd := exec.Command("docker", "logs", "-f", c.ID)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start log streaming: %w", err)
	}

	// Stream both stdout and stderr
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fmt.Fprintln(writer, scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintln(writer, scanner.Text())
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	return nil
}

// CopyTo copies a file or directory from the host to the container
func (c *Container) CopyTo(hostPath, containerPath string) error {
	c.client.runner.Printf(colorGray, "", "Copying to container: %s -> %s:%s", hostPath, c.getIdentifier(), containerPath)

	result := c.client.runner.RunCommand("docker", "cp", hostPath, fmt.Sprintf("%s:%s", c.ID, containerPath))
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to copy to container: %v", result.Err)
	}
	return nil
}

// Inspect returns detailed information about the container
func (c *Container) Inspect() (map[string]interface{}, error) {
	result := c.client.runner.RunCommandQuiet("docker", "inspect", c.ID)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to inspect container: %v", result.Err)
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &inspectData); err != nil {
		return nil, fmt.Errorf("failed to parse inspect data: %w", err)
	}

	if len(inspectData) == 0 {
		return nil, fmt.Errorf("no inspect data returned")
	}

	return inspectData[0], nil
}

// ExecInteractive executes a command interactively (with TTY)
func (c *Container) ExecInteractive(command ...string) error {
	args := []string{"exec", "-it", c.ID}
	args = append(args, command...)

	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// WaitForLog waits for a specific log message to appear
func (c *Container) WaitForLog(pattern string, timeout time.Duration) error {
	c.client.runner.Printf(colorGray, "", "Waiting for log pattern: %s", pattern)

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		logs, err := c.Logs()
		if err != nil {
			return fmt.Errorf("failed to get logs: %w", err)
		}

		if strings.Contains(logs, pattern) {
			c.client.runner.Printf(colorGray, colorBold, "Found log pattern: %s", pattern)
			return nil
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for log pattern '%s' after %v", pattern, timeout)
}

// GetPort returns the host port mapped to a container port
func (c *Container) GetPort(containerPort string) (string, error) {
	result := c.client.runner.RunCommandQuiet("docker", "port", c.ID, containerPort)
	if result.ExitCode != 0 {
		return "", fmt.Errorf("failed to get port mapping: %v", result.Err)
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "" {
		return "", fmt.Errorf("no port mapping found for %s", containerPort)
	}

	// Extract host port from output like "0.0.0.0:32768"
	parts := strings.Split(output, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("unexpected port output format: %s", output)
	}

	return parts[len(parts)-1], nil
}

// Commit creates a new image from the container's changes
func (c *Container) Commit(imageName string) error {
	c.client.runner.Printf(colorBlue, "", "Committing container to image: %s", imageName)

	result := c.client.runner.RunCommand("docker", "commit", c.ID, imageName)
	if result.ExitCode != 0 {
		return fmt.Errorf("failed to commit container: %v", result.Err)
	}

	c.client.runner.Printf(colorGray, colorBold, "Successfully created image: %s", imageName)
	return nil
}

// Stats returns resource usage statistics for the container
func (c *Container) Stats() (map[string]interface{}, error) {
	result := c.client.runner.RunCommandQuiet("docker", "stats", "--no-stream", "--format", "{{json .}}", c.ID)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to get stats: %v", result.Err)
	}

	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &stats); err != nil {
		return nil, fmt.Errorf("failed to parse stats: %w", err)
	}

	return stats, nil
}

// Volume represents a Docker volume
type Volume struct {
	Name   string
	Driver string
	client *DockerClient
}

// VolumeOptions provides options for creating a volume
type VolumeOptions struct {
	Name   string
	Driver string // e.g., "local"
	Labels map[string]string
	Opts   map[string]string // Driver specific options
}

// CreateVolume creates a new Docker volume
func CreateVolume(opts VolumeOptions) (*Volume, error) {
	client := NewDockerClient(true)

	args := []string{"volume", "create"}

	if opts.Name != "" {
		args = append(args, "--name", opts.Name)
	}

	if opts.Driver != "" {
		args = append(args, "--driver", opts.Driver)
	}

	// Add labels
	for key, value := range opts.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add driver options
	for key, value := range opts.Opts {
		args = append(args, "--opt", fmt.Sprintf("%s=%s", key, value))
	}

	client.runner.Printf(colorBlue, colorBold, "Creating volume: %s", opts.Name)

	result := client.runner.RunCommand("docker", args...)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to create volume: %v", result.Err)
	}

	volumeName := strings.TrimSpace(result.Stdout)
	if volumeName == "" && opts.Name != "" {
		volumeName = opts.Name
	}

	volume := &Volume{
		Name:   volumeName,
		Driver: opts.Driver,
		client: client,
	}

	client.runner.Printf(colorGray, colorBold, "Successfully created volume: %s", volumeName)
	return volume, nil
}

// GetVolume retrieves an existing Docker volume
func GetVolume(name string) (*Volume, error) {
	client := NewDockerClient(true)

	// Check if volume exists
	result := client.runner.RunCommandQuiet("docker", "volume", "inspect", name)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("volume not found: %s", name)
	}

	var volumeData []map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &volumeData); err != nil {
		return nil, fmt.Errorf("failed to parse volume data: %w", err)
	}

	if len(volumeData) == 0 {
		return nil, fmt.Errorf("no volume data returned")
	}

	driver := ""
	if d, ok := volumeData[0]["Driver"].(string); ok {
		driver = d
	}

	return &Volume{
		Name:   name,
		Driver: driver,
		client: client,
	}, nil
}

// Delete removes the volume
func (v *Volume) Delete() error {
	v.client.runner.Printf(colorRed, "", "Deleting volume: %s", v.Name)

	result := v.client.runner.RunCommand("docker", "volume", "rm", v.Name)
	if result.ExitCode != 0 {
		// Try force delete
		v.client.runner.Printf(colorYellow, "", "Regular delete failed, trying force delete...")
		forceResult := v.client.runner.RunCommand("docker", "volume", "rm", "-f", v.Name)
		if forceResult.ExitCode != 0 {
			return fmt.Errorf("failed to delete volume: %v", forceResult.Err)
		}
	}

	v.client.runner.Printf(colorGray, colorBold, "Successfully deleted volume: %s", v.Name)
	return nil
}

// Inspect returns detailed information about the volume
func (v *Volume) Inspect() (map[string]interface{}, error) {
	result := v.client.runner.RunCommandQuiet("docker", "volume", "inspect", v.Name)
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to inspect volume: %v", result.Err)
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &inspectData); err != nil {
		return nil, fmt.Errorf("failed to parse inspect data: %w", err)
	}

	if len(inspectData) == 0 {
		return nil, fmt.Errorf("no inspect data returned")
	}

	return inspectData[0], nil
}

// ListVolumes lists all Docker volumes
func ListVolumes() ([]*Volume, error) {
	client := NewDockerClient(true)

	result := client.runner.RunCommandQuiet("docker", "volume", "ls", "--format", "{{json .}}")
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list volumes: %v", result.Err)
	}

	var volumes []*Volume
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var volumeData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &volumeData); err != nil {
			continue
		}

		name, _ := volumeData["Name"].(string)
		driver, _ := volumeData["Driver"].(string)

		if name != "" {
			volumes = append(volumes, &Volume{
				Name:   name,
				Driver: driver,
				client: client,
			})
		}
	}

	return volumes, nil
}

// PruneVolumes removes all unused volumes
func PruneVolumes() ([]string, error) {
	client := NewDockerClient(true)

	client.runner.Printf(colorYellow, colorBold, "Pruning unused volumes...")

	result := client.runner.RunCommand("docker", "volume", "prune", "-f")
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to prune volumes: %v", result.Err)
	}

	// Parse output to find deleted volumes
	var deletedVolumes []string
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Deleted Volumes:") && !strings.HasPrefix(line, "Total reclaimed space:") {
			deletedVolumes = append(deletedVolumes, line)
		}
	}

	client.runner.Printf(colorGray, colorBold, "Pruned %d volumes", len(deletedVolumes))
	return deletedVolumes, nil
}

// GetMountpoint returns the mountpoint path of the volume on the host
func (v *Volume) GetMountpoint() (string, error) {
	inspectData, err := v.Inspect()
	if err != nil {
		return "", err
	}

	mountpoint, ok := inspectData["Mountpoint"].(string)
	if !ok {
		return "", fmt.Errorf("mountpoint not found in volume data")
	}

	return mountpoint, nil
}

// GetLabels returns the labels of the volume
func (v *Volume) GetLabels() (map[string]string, error) {
	inspectData, err := v.Inspect()
	if err != nil {
		return nil, err
	}

	labelsRaw, ok := inspectData["Labels"]
	if !ok || labelsRaw == nil {
		return make(map[string]string), nil
	}

	labels := make(map[string]string)
	if labelsMap, ok := labelsRaw.(map[string]interface{}); ok {
		for key, value := range labelsMap {
			if strValue, ok := value.(string); ok {
				labels[key] = strValue
			}
		}
	}

	return labels, nil
}

// CopyFromVolume copies data from a Docker volume to the host
func (v *Volume) CopyFromVolume(volumePath, hostPath string) error {
	v.client.runner.Printf(colorGray, "", "Copying from volume: %s:%s -> %s", v.Name, volumePath, hostPath)

	// Create a temporary container to access the volume
	container, err := Run(ContainerOptions{
		Name:    fmt.Sprintf("volume-copy-%s-%d", v.Name, time.Now().Unix()),
		Image:   "alpine:latest",
		Command: []string{"sleep", "10"},
		Volumes: map[string]string{
			v.Name: "/volume",
		},
		Remove: true,
		Detach: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create temporary container: %w", err)
	}
	defer func() {
		if err := container.Delete(); err != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete container: %v", err)
		}
	}()

	// Copy the file from the container
	containerPath := filepath.Join("/volume", volumePath)
	return container.GetFile(containerPath, hostPath)
}

// CopyToVolume copies data from the host to a Docker volume
func (v *Volume) CopyToVolume(hostPath, volumePath string) error {
	v.client.runner.Printf(colorGray, "", "Copying to volume: %s -> %s:%s", hostPath, v.Name, volumePath)

	// Create a temporary container to access the volume
	container, err := Run(ContainerOptions{
		Name:    fmt.Sprintf("volume-copy-%s-%d", v.Name, time.Now().Unix()),
		Image:   "alpine:latest",
		Command: []string{"sleep", "10"},
		Volumes: map[string]string{
			v.Name: "/volume",
		},
		Remove: true,
		Detach: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create temporary container: %w", err)
	}
	defer func() {
		if err := container.Delete(); err != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete container: %v", err)
		}
	}()

	// Copy the file to the container
	containerPath := filepath.Join("/volume", volumePath)
	return container.CopyTo(hostPath, containerPath)
}

// BackupVolume creates a tar backup of the volume
func (v *Volume) BackupVolume(backupPath string) error {
	v.client.runner.Printf(colorBlue, colorBold, "Backing up volume: %s to %s", v.Name, backupPath)

	// Ensure backup directory exists
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Create a container to tar the volume
	container, err := Run(ContainerOptions{
		Name:    fmt.Sprintf("volume-backup-%s-%d", v.Name, time.Now().Unix()),
		Image:   "alpine:latest",
		Command: []string{"tar", "-czf", "/backup.tar.gz", "-C", "/volume", "."},
		Volumes: map[string]string{
			v.Name: "/volume",
		},
		Remove: false, // Don't remove automatically so we can copy the backup
	})
	if err != nil {
		return fmt.Errorf("failed to create backup container: %w", err)
	}
	defer func() {
		if err := container.Delete(); err != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete container: %v", err)
		}
	}()

	// Wait for tar to complete
	time.Sleep(2 * time.Second)

	// Copy the backup file
	if err := container.GetFile("/backup.tar.gz", backupPath); err != nil {
		return fmt.Errorf("failed to copy backup: %w", err)
	}

	v.client.runner.Printf(colorGray, colorBold, "Successfully backed up volume to: %s", backupPath)
	return nil
}

// RestoreVolume restores a volume from a tar backup
func (v *Volume) RestoreVolume(backupPath string) error {
	v.client.runner.Printf(colorBlue, colorBold, "Restoring volume: %s from %s", v.Name, backupPath)

	// Check if backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Create a container to extract the backup
	container, err := Run(ContainerOptions{
		Name:    fmt.Sprintf("volume-restore-%s-%d", v.Name, time.Now().Unix()),
		Image:   "alpine:latest",
		Command: []string{"sleep", "30"},
		Volumes: map[string]string{
			v.Name: "/volume",
		},
		Remove: true,
		Detach: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create restore container: %w", err)
	}
	defer func() {
		if err := container.Delete(); err != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete container: %v", err)
		}
	}()

	// Copy backup to container
	if err := container.CopyTo(backupPath, "/backup.tar.gz"); err != nil {
		return fmt.Errorf("failed to copy backup to container: %w", err)
	}

	// Clear existing data and extract backup
	_, err = container.Exec("sh", "-c", "rm -rf /volume/* && tar -xzf /backup.tar.gz -C /volume")
	if err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	v.client.runner.Printf(colorGray, colorBold, "Successfully restored volume from: %s", backupPath)
	return nil
}

// CloneVolume creates a copy of the volume with a new name
func (v *Volume) CloneVolume(newName string) (*Volume, error) {
	v.client.runner.Printf(colorBlue, colorBold, "Cloning volume: %s to %s", v.Name, newName)

	// Create new volume
	newVolume, err := CreateVolume(VolumeOptions{
		Name:   newName,
		Driver: v.Driver,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new volume: %w", err)
	}

	// Create containers to copy data
	sourceContainer, err := Run(ContainerOptions{
		Name:    fmt.Sprintf("volume-clone-src-%d", time.Now().Unix()),
		Image:   "alpine:latest",
		Command: []string{"sleep", "30"},
		Volumes: map[string]string{
			v.Name: "/source:ro",
		},
		Remove: true,
		Detach: true,
	})
	if err != nil {
		if deleteErr := newVolume.Delete(); deleteErr != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete volume: %v", deleteErr)
		}
		return nil, fmt.Errorf("failed to create source container: %w", err)
	}
	defer func() {
		if err := sourceContainer.Delete(); err != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete container: %v", err)
		}
	}()

	destContainer, err := Run(ContainerOptions{
		Name:    fmt.Sprintf("volume-clone-dest-%d", time.Now().Unix()),
		Image:   "alpine:latest",
		Command: []string{"sleep", "30"},
		Volumes: map[string]string{
			newName: "/dest",
		},
		Remove: true,
		Detach: true,
	})
	if err != nil {
		if deleteErr := newVolume.Delete(); deleteErr != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete volume: %v", deleteErr)
		}
		return nil, fmt.Errorf("failed to create destination container: %w", err)
	}
	defer func() {
		if err := destContainer.Delete(); err != nil {
			v.client.runner.Printf(colorRed, "", "Failed to delete container: %v", err)
		}
	}()

	// Copy data between volumes using tar
	_, err = sourceContainer.Exec("sh", "-c", "cd /source && tar -cf - . | docker exec -i "+destContainer.ID+" tar -xf - -C /dest")
	if err != nil {
		// Try alternative method
		v.client.runner.Printf(colorYellow, "", "Direct copy failed, trying alternative method...")

		// Create tar in source
		_, err = sourceContainer.Exec("tar", "-czf", "/tmp/data.tar.gz", "-C", "/source", ".")
		if err != nil {
			if deleteErr := newVolume.Delete(); deleteErr != nil {
				v.client.runner.Printf(colorRed, "", "Failed to delete volume: %v", deleteErr)
			}
			return nil, fmt.Errorf("failed to create tar: %w", err)
		}

		// Copy tar to host
		tempFile := fmt.Sprintf("/tmp/volume-clone-%d.tar.gz", time.Now().Unix())
		if err := sourceContainer.GetFile("/tmp/data.tar.gz", tempFile); err != nil {
			if deleteErr := newVolume.Delete(); deleteErr != nil {
				v.client.runner.Printf(colorRed, "", "Failed to delete volume: %v", deleteErr)
			}
			return nil, fmt.Errorf("failed to copy tar to host: %w", err)
		}
		defer os.Remove(tempFile)

		// Copy tar to destination
		if err := destContainer.CopyTo(tempFile, "/tmp/data.tar.gz"); err != nil {
			if deleteErr := newVolume.Delete(); deleteErr != nil {
				v.client.runner.Printf(colorRed, "", "Failed to delete volume: %v", deleteErr)
			}
			return nil, fmt.Errorf("failed to copy tar to destination: %w", err)
		}

		// Extract in destination
		_, err = destContainer.Exec("tar", "-xzf", "/tmp/data.tar.gz", "-C", "/dest")
		if err != nil {
			if deleteErr := newVolume.Delete(); deleteErr != nil {
				v.client.runner.Printf(colorRed, "", "Failed to delete volume: %v", deleteErr)
			}
			return nil, fmt.Errorf("failed to extract tar: %w", err)
		}
	}

	v.client.runner.Printf(colorGray, colorBold, "Successfully cloned volume: %s -> %s", v.Name, newName)
	return newVolume, nil
}

// GetSize returns the size of the volume in bytes
func (v *Volume) GetSize() (int64, error) {
	// Create a container to check volume size
	container, err := Run(ContainerOptions{
		Name:    fmt.Sprintf("volume-size-%s-%d", v.Name, time.Now().Unix()),
		Image:   "alpine:latest",
		Command: []string{"du", "-sb", "/volume"},
		Volumes: map[string]string{
			v.Name: "/volume:ro",
		},
		Remove: true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create container for size check: %w", err)
	}

	// Parse output
	output := strings.TrimSpace(container.client.runner.RunCommandQuiet("docker", "logs", container.ID).Stdout)
	parts := strings.Fields(output)
	if len(parts) < 1 {
		return 0, fmt.Errorf("unexpected du output: %s", output)
	}

	size, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	}

	return size, nil
}
