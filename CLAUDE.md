# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

flanksource/commons is a Go utility library providing common functionality for the Flanksource ecosystem. It's a collection of reusable modules organized by concern: core utilities, infrastructure helpers, development tools, data processing, and system integration.

## Key Commands

```bash
# Testing
make test              # Run all tests with verbose output
go test ./...          # Run all tests (standard Go)
go test ./logger/...   # Run tests for a specific package

# Linting and Code Quality
make lint              # Run golangci-lint with project-specific rules

# Dependency Management
make tidy              # Clean up go.mod dependencies
go mod tidy            # Standard Go dependency cleanup

# Build
go build               # Build the dependency installer binary
go run main.go <dep>   # Download and install a dependency (e.g., go run main.go jq)
```

## Architecture and Code Organization

### Module Structure
The codebase follows a flat package structure where each directory represents a focused utility module:

- **HTTP Client (`http/`)**: Enhanced HTTP client with authentication middleware pattern. Supports Digest, NTLM, OAuth2, and custom auth. Uses functional options for configuration.
- **Logger (`logger/`)**: Dual logging system supporting both logrus and slog backends. Global logger instance with interface-based design for flexibility.
- **Collections (`collections/`)**: Generic utilities for maps, slices, sets, and priority queues. Heavy use of Go generics.
- **Dependency Management (`deps/`)**: Template-based system for downloading and installing external binaries. Supports cross-platform paths and verification.
- **Properties (`properties/`)**: Global configuration management with environment variable support and structured properties.

### Key Patterns

1. **Global State Pattern**: Logger and Properties packages use global instances initialized via `Init()` functions. Always check if initialization is needed before using these packages.

2. **Middleware Pattern**: HTTP client uses layered middleware for auth, tracing, and logging:
   ```go
   client := http.NewClient().
       WithLogger(logger).
       WithAuth(username, password).
       WithTrace(true)
   ```

3. **Interface-First Design**: Most packages expose interfaces (e.g., `Logger` interface) allowing for mock implementations in tests.

4. **Builder Pattern**: Used extensively in HTTP client and dependency configurations for fluent API design.

### Testing Approach

- Uses testify for assertions in most tests
- Some modules (like `diff/`) use Ginkgo/Gomega for BDD-style tests
- Test files are colocated with source files following `*_test.go` convention
- No global test setup/teardown - each test is self-contained

### Important Development Notes

1. **Go Version**: Requires Go 1.23.0+ (uses newer generic features)
2. **Linting**: The project has custom golangci-lint rules that disable certain checks for HTTP-related naming
3. **Imports**: When adding new functionality, prefer using existing utilities from within the project (e.g., use `collections` package for slice operations)
4. **Error Handling**: The codebase uses explicit error returns rather than panics - maintain this pattern
5. **Logging**: Use the `logger` package for all logging needs, avoid fmt.Print statements

### Common Tasks

To add a new utility module:
1. Create a new directory at the root level
2. Follow the existing pattern of having a main file with the package name
3. Include comprehensive tests in `*_test.go` files
4. Update imports in other packages if the utility is widely useful

To modify the HTTP client:
1. Changes go in `http/client.go` or `http/middleware.go`
2. Authentication providers are in `http/auth_*.go` files
3. Always maintain backward compatibility with the builder pattern

To work with the dependency installer:
1. Dependencies are defined in `deps/binary.go`
2. Binary definitions include version, download URLs, and install paths
3. Use templates for cross-platform path handling