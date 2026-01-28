# Testing Guide

## Test Strategy

structlint uses a **binary-first** testing approach:
1. Compile the CLI binary once
2. Run tests against the actual binary using `exec.Command`
3. Tests validate real CLI behavior, not internal functions

## Test Files

| File | Purpose |
|------|---------|
| `test/helpers_test.go` | Test utilities (buildBinary, runBinary) |
| `test/integration_test.go` | CLI integration tests |
| `test/project_validation_test.go` | Various project type tests |
| `test/self_validation_test.go` | Self-validation tests |
| `test/required_validation_test.go` | Required paths/files tests |
| `test/performance_test.go` | Performance benchmarks |
| `test/smoke_test.go` | Basic functionality tests |

## Test Helpers

### buildBinary

```go
func buildBinary(t *testing.T) string
```
Compiles the CLI binary once per test run (uses `sync.Once`).
Returns path to the compiled binary.

### runBinary

```go
func runBinary(t *testing.T, bin string, args ...string) (string, error)
```
Executes the binary with given arguments.
Returns combined stdout/stderr and error.

### runBinaryInDir

```go
func runBinaryInDir(t *testing.T, bin, dir string, args ...string) (string, error)
```
Executes binary in a specific directory.

### createTestProject

```go
func createTestProject(t *testing.T, files map[string]string, configContent string) string
```
Creates a temporary project with given files and config.
Returns the temp directory path.

## Writing Tests

### Basic Integration Test

```go
func TestFeature(t *testing.T) {
    bin := buildBinary(t)

    // Define project files
    files := map[string]string{
        "cmd/main.go": "package main",
        "internal/app.go": "package internal",
    }

    // Define config
    config := `
dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**"]
file_naming_pattern:
  allowed: ["*.go", "*.yaml"]
`

    // Create temp project
    projectDir := createTestProject(t, files, config)

    // Run validation
    out, err := runBinaryInDir(t, bin, projectDir,
        "validate",
        "--config", ".structlint.yaml",
    )

    // Assert
    if err != nil {
        t.Errorf("Unexpected failure: %v\nOutput: %s", err, out)
    }
}
```

### Testing for Expected Violations

```go
func TestViolationsDetected(t *testing.T) {
    bin := buildBinary(t)

    files := map[string]string{
        ".env.local": "SECRET=value",  // Should be disallowed
    }

    config := `
file_naming_pattern:
  disallowed: ["*.env*"]
`

    projectDir := createTestProject(t, files, config)

    out, err := runBinaryInDir(t, bin, projectDir,
        "validate", "--config", ".structlint.yaml",
    )

    // Should fail
    if err == nil {
        t.Error("Expected validation to fail")
    }

    // Check error message
    if !strings.Contains(out, ".env.local") {
        t.Errorf("Expected .env.local in output: %s", out)
    }
}
```

### Testing JSON Output

```go
func TestJSONReport(t *testing.T) {
    bin := buildBinary(t)
    projectDir := createTestProject(t, files, config)
    reportPath := filepath.Join(t.TempDir(), "report.json")

    runBinaryInDir(t, bin, projectDir,
        "validate",
        "--config", ".structlint.yaml",
        "--json-output", reportPath,
    )

    // Read and parse report
    data, _ := os.ReadFile(reportPath)
    var report map[string]interface{}
    json.Unmarshal(data, &report)

    // Assert fields
    if report["failures"].(float64) != 0 {
        t.Error("Expected no failures")
    }
}
```

## Running Tests

```bash
# All tests
go test ./... -v

# Integration tests only
go test ./test/... -v

# Specific test
go test -run TestIntegrationSelfValidation ./test/...

# With race detection
go test -race ./...

# With coverage
go test -cover ./...
```

## Test Patterns

1. **Use `t.TempDir()`** - Automatic cleanup, isolated
2. **Use `t.Helper()`** - Better error reporting
3. **Use subtests** - `t.Run("name", func(t *testing.T) {...})`
4. **Check both success and failure cases**
5. **Verify error messages contain useful info**
