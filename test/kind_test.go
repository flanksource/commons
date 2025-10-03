package test

import (
	"os"
	"testing"
)

func TestKindCluster(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test")
	}

	t.Run("GetOrCreate", func(t *testing.T) {
		kind := NewKind("test-cluster").NoColor()

		// Create or get cluster
		kind.GetOrCreate()
		if kind.Error() != nil {
			t.Fatalf("Failed to get or create cluster: %v", kind.Error())
		}

		// Verify cluster exists
		if !kind.Exists() {
			t.Error("Cluster should exist after GetOrCreate")
		}
	})

	t.Run("Use", func(t *testing.T) {
		kind := NewKind("test-cluster").NoColor()

		// Use the cluster
		kind.Use()
		if kind.Error() != nil {
			t.Fatalf("Failed to use cluster: %v", kind.Error())
		}

		// Verify we can get kubeconfig
		kubeconfig, err := kind.GetKubeconfig()
		if err != nil {
			t.Fatalf("Failed to get kubeconfig: %v", err)
		}
		if kubeconfig == "" {
			t.Error("Kubeconfig should not be empty")
		}
	})

	t.Run("LoadImage", func(t *testing.T) {
		kind := NewKind("test-cluster").NoColor()

		// Skip if docker image doesn't exist
		runner := NewCommandRunner(false)
		result := runner.RunCommandQuiet("docker", "images", "-q", "nginx:latest")
		if result.Err != nil || result.Stdout == "" {
			t.Skip("nginx:latest image not available")
		}

		// Load image into cluster
		kind.LoadImage("nginx:latest")
		if kind.Error() != nil {
			t.Fatalf("Failed to load image: %v", kind.Error())
		}
	})

	t.Run("Delete", func(t *testing.T) {
		kind := NewKind("test-cluster").NoColor()

		// Delete the cluster
		kind.Delete()
		if kind.Error() != nil {
			t.Fatalf("Failed to delete cluster: %v", kind.Error())
		}

		// Verify cluster no longer exists
		if kind.Exists() {
			t.Error("Cluster should not exist after Delete")
		}
	})
}

func TestKindBuilderPattern(t *testing.T) {
	kind := NewKind("builder-test").
		WithVersion("v1.27.0").
		NoColor()

	if kind.Name != "builder-test" {
		t.Errorf("Expected name to be 'builder-test', got %s", kind.Name)
	}

	if kind.Version != "v1.27.0" {
		t.Errorf("Expected version to be 'v1.27.0', got %s", kind.Version)
	}

	if kind.ColorOutput {
		t.Error("Expected ColorOutput to be false")
	}
}

func TestCommandRunner(t *testing.T) {
	runner := NewCommandRunner(false)

	t.Run("RunCommand", func(t *testing.T) {
		result := runner.RunCommand("echo", "hello")
		if result.Err != nil {
			t.Fatalf("Command failed: %v", result.Err)
		}
		if result.ExitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", result.ExitCode)
		}
		if result.Stdout != "hello\n" {
			t.Errorf("Expected 'hello\\n', got %q", result.Stdout)
		}
	})

	t.Run("RunCommandQuiet", func(t *testing.T) {
		result := runner.RunCommandQuiet("echo", "quiet")
		if result.Err != nil {
			t.Fatalf("Command failed: %v", result.Err)
		}
		if result.Stdout != "quiet\n" {
			t.Errorf("Expected 'quiet\\n', got %q", result.Stdout)
		}
	})

	t.Run("CommandFailure", func(t *testing.T) {
		result := runner.RunCommand("false")
		if result.Err == nil {
			t.Error("Expected command to fail")
		}
		if result.ExitCode == 0 {
			t.Error("Expected non-zero exit code")
		}
	})
}
