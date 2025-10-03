// Package exec provides simplified command execution utilities for running
// shell commands with enhanced error handling and output management.
//
// The package wraps Go's os/exec with convenience functions that handle
// common patterns like forwarding output to console, capturing combined
// output, and ignoring errors for optional operations.
//
// Key Features:
//   - Execute commands with automatic stdout/stderr forwarding
//   - Safe execution that never panics on errors
//   - Environment variable support
//   - Printf-style argument formatting
//   - Combined output capture
//
// Basic Execution:
//
//	// Execute a command and forward output to console
//	err := exec.Exec("ls -la")
//
//	// Execute with formatted arguments
//	err := exec.Execf("echo 'Hello %s'", "World")
//
//	// Execute with custom environment variables
//	env := map[string]string{
//		"DEBUG": "true",
//		"CONFIG_PATH": "/etc/app",
//	}
//	err := exec.ExecfWithEnv("./script.sh", env)
//
// Safe Execution:
//
// SafeExec executes commands and returns success status instead of errors,
// useful for optional operations or when you want to continue on failure:
//
//	// Try to get git branch, continue if git isn't available
//	branch, ok := exec.SafeExec("git rev-parse --abbrev-ref HEAD")
//	if ok {
//		fmt.Printf("Current branch: %s\n", branch)
//	}
//
//	// Check if a command exists
//	_, exists := exec.SafeExec("which docker")
//	if exists {
//		// Docker is installed
//	}
//
// Output Handling:
//
//	// Exec forwards both stdout and stderr to console
//	err := exec.Exec("make build")
//	// User sees all output in real-time
//
//	// SafeExec captures output and returns it
//	output, ok := exec.SafeExec("cat /etc/hostname")
//	if ok {
//		fmt.Println("Hostname:", output)
//	}
//
// Environment Variables:
//
//	// Set environment for command execution
//	env := map[string]string{
//		"PATH":         "/usr/local/bin:/usr/bin",
//		"DATABASE_URL": "postgres://localhost/mydb",
//	}
//	err := exec.ExecfWithEnv("migrate up", env)
//
// Error Handling:
//
//	// Exec returns error with command output on failure
//	if err := exec.Exec("./deploy.sh"); err != nil {
//		// err contains both error and stderr output
//		log.Fatal(err)
//	}
//
//	// SafeExec returns false on any error
//	output, ok := exec.SafeExec("risky-command")
//	if !ok {
//		// Command failed, output is empty string
//		log.Warn("Command failed, using default")
//	}
//
// Formatted Commands:
//
//	// Use printf-style formatting for dynamic commands
//	filename := "data.json"
//	err := exec.Execf("cat %s | jq .", filename)
//
//	// Multiple arguments
//	src := "/source"
//	dst := "/dest"
//	err := exec.Execf("rsync -av %s %s", src, dst)
//
// Common Use Cases:
//
//	// Run build commands
//	exec.Exec("go build -o app ./cmd/server")
//
//	// Check tool availability
//	if _, ok := exec.SafeExec("which kubectl"); !ok {
//		log.Fatal("kubectl not found in PATH")
//	}
//
//	// Get version information
//	version, ok := exec.SafeExec("app --version")
//
//	// Run tests with environment
//	env := map[string]string{"TEST_ENV": "integration"}
//	exec.ExecfWithEnv("go test ./...", env)
//
// Note: All commands are executed through bash -c, providing full shell
// functionality including pipes, redirects, and variable expansion.
package exec
