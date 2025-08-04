package test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// CommandResult holds the result of a command execution
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// String returns a formatted string of the command result
func (r CommandResult) String() string {
	return fmt.Sprintf("ExitCode: %d\nStdout:\n%s\nStderr:\n%s\nError: %v",
		r.ExitCode, r.Stdout, r.Stderr, r.Err)
}

// CommandRunner provides command execution with optional colored output
type CommandRunner struct {
	ColorOutput bool
}

// NewCommandRunner creates a new CommandRunner
func NewCommandRunner(colorOutput bool) *CommandRunner {
	return &CommandRunner{ColorOutput: colorOutput}
}

// RunCommand executes a command and returns the result
func (c *CommandRunner) RunCommand(name string, args ...string) CommandResult {
	if c.ColorOutput {
		fmt.Printf("%s%s>>> Executing: %s %s%s\n", colorBlue, colorBold, name, strings.Join(args, " "), colorReset)
	}

	cmd := exec.Command(name, args...)

	// Create pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return CommandResult{
			Err:      fmt.Errorf("failed to create stdout pipe: %w", err),
			ExitCode: -1,
		}
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return CommandResult{
			Err:      fmt.Errorf("failed to create stderr pipe: %w", err),
			ExitCode: -1,
		}
	}

	// Buffers to capture output
	var stdout, stderr bytes.Buffer

	// Start the command
	if err := cmd.Start(); err != nil {
		return CommandResult{
			Err:      fmt.Errorf("failed to start command: %w", err),
			ExitCode: -1,
		}
	}

	// Stream output in real-time with colors
	var wg sync.WaitGroup
	wg.Add(2)

	go c.streamOutput(stdoutPipe, "stdout", colorGray, &stdout, &wg)
	go c.streamOutput(stderrPipe, "stderr", colorRed, &stderr, &wg)

	// Wait for output streaming to complete
	wg.Wait()

	// Wait for command to complete
	err = cmd.Wait()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	result := CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}

	// Print exit status
	if c.ColorOutput {
		if result.Err != nil {
			fmt.Printf("%s%s<<< Command failed with exit code %d%s\n", colorRed, colorBold, result.ExitCode, colorReset)
		} else {
			fmt.Printf("%s<<< Command completed successfully%s\n", colorGray, colorReset)
		}
		fmt.Println() // Add blank line for readability
	}

	return result
}

// RunCommandQuiet executes a command without output streaming
func (c *CommandRunner) RunCommandQuiet(name string, args ...string) CommandResult {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}

	return CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

func (c *CommandRunner) streamOutput(reader io.Reader, prefix string, color string, buffer *bytes.Buffer, wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		buffer.WriteString(line + "\n")
		if c.ColorOutput {
			fmt.Printf("%s%s%s: %s%s\n", color, prefix, colorReset, color, line+colorReset)
		}
	}
}

// Printf prints a formatted colored message
func (c *CommandRunner) Printf(color, style, format string, args ...interface{}) {
	if c.ColorOutput {
		fmt.Printf("%s%s%s%s\n", color, style, fmt.Sprintf(format, args...), colorReset)
	} else {
		fmt.Printf(format+"\n", args...)
	}
}