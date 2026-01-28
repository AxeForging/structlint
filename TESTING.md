# Testing Documentation

This document describes the comprehensive testing strategy for structlint, including validation of our own project structure.

## Test Structure

### 1. Unit Tests (`validator_test.go`)
- **Purpose**: Test core validation logic with controlled scenarios
- **Coverage**: Directory structure validation, file naming patterns, JSON reporting
- **Location**: Root directory (legacy compatibility)

### 2. Smoke Tests (`test/smoke_test.go`)
- **Purpose**: Basic CLI functionality tests
- **Coverage**: Version command, help commands, error handling
- **Location**: `test/` directory

### 3. Project Validation Tests (`test/project_validation_test.go`)
- **Purpose**: Test validation against various project structure standards
- **Coverage**: 
  - Go project standards
  - Microservice architecture
  - Configuration format validation (YAML/JSON)
- **Location**: `test/` directory

### 4. Self-Validation Tests (`test/self_validation_test.go`)
- **Purpose**: Validate our own project against defined standards
- **Coverage**: 
  - Project structure compliance
  - Configuration validation
  - Real-world scenarios
- **Location**: `test/` directory

### 5. Integration Tests (`test/integration_test.go`)
- **Purpose**: End-to-end testing of the CLI tool
- **Coverage**:
  - Self-validation of our project
  - Real project structure validation
  - Violation detection
  - CLI command testing
  - Configuration precedence
- **Location**: `test/` directory

## Project Standards

Our project follows these defined standards (`.structlint.yaml`):

### Directory Structure
**Allowed:**
- `.` (root)
- `cmd/**` (command entry points)
- `internal/**` (internal packages)
- `test/**` (test files)
- `docs/**` (documentation)
- `scripts/**` (build scripts)
- `bin/**` (built binaries)
- `dist/**` (distribution files)

**Disallowed:**
- `vendor/**` (vendor dependencies)
- `node_modules/**` (Node.js dependencies)
- `tmp/**`, `temp/**` (temporary directories)
- `.git/**` (Git internal files)

### File Naming Patterns
**Allowed:**
- `*.go`, `*.mod`, `*.sum` (Go files)
- `*.yaml`, `*.yml`, `*.json`, `*.toml` (config files)
- `*.md`, `*.txt`, `README*`, `LICENSE*` (documentation)
- `Makefile`, `Dockerfile*`, `*.sh` (build files)
- `.gitignore`, `.editorconfig`, `.golangci.yml` (tooling)

**Disallowed:**
- `*.env*`, `.env*` (environment/secrets)
- `*.log`, `*.tmp`, `*.temp` (temporary files)
- `*~`, `*.swp`, `*.swo` (backup/editor files)
- `.DS_Store`, `Thumbs.db` (OS files)

### Ignored Patterns
- `.git`, `.svn`, `.hg` (version control)
- `vendor`, `node_modules` (dependencies)
- `bin`, `dist`, `build` (build outputs)
- `.idea`, `.vscode` (IDE files)
- `*.log`, `*.tmp` (temporary files)

## Running Tests

### All Tests
```bash
go test ./...
```

### Specific Test Categories
```bash
# Unit tests only
go test -run TestValidator

# Integration tests only
go test ./test -run TestIntegration

# Self-validation
make test-self
```

### Self-Validation
```bash
# Using Makefile (recommended)
make test-self

# Direct CLI usage
./bin/structlint validate --config .structlint.yaml --json-output validation-report.json
```

## Test Results

### Current Project Status
✅ **34 checks passed, 0 failures**

Our project structure fully complies with the defined standards:
- All directories follow the allowed structure
- All files match allowed naming patterns
- No disallowed patterns detected
- Proper separation of concerns maintained

### Validation Report
The self-validation generates a JSON report (`validation-report.json`):
```json
{
  "successes": 34,
  "failures": 0,
  "errors": []
}
```

## Test Scenarios Covered

### 1. Minimal Go Project
- Basic `cmd/` and `internal/` structure
- Essential Go files (`*.go`, `go.mod`, `go.sum`)
- Documentation (`README.md`)

### 2. Microservice Architecture
- Multiple commands (`cmd/api`, `cmd/worker`)
- Service layer (`internal/service`)
- Repository layer (`internal/repository`)
- API layer (`internal/api`)
- Utilities (`pkg/utils`)

### 3. Violation Detection
- Disallowed directories (`tmp/`, `vendor/`)
- Disallowed files (`*.env*`, `*.log`, `*.tmp`)
- Proper ignore patterns

### 4. Configuration Formats
- YAML configuration files
- JSON configuration files
- Environment variable precedence

## Continuous Integration

The Makefile provides CI-friendly targets:
```bash
# Full CI pipeline
make ci

# Individual steps
make tidy    # Clean dependencies
make check   # Lint and test
make build   # Build binary
make test-self # Self-validation
```

## Best Practices Demonstrated

1. **Self-Validation**: Our tool validates its own structure
2. **Comprehensive Coverage**: Tests cover unit, integration, and real-world scenarios
3. **Standards Enforcement**: Clear, documented project standards
4. **Automated Validation**: Makefile targets for easy testing
5. **Report Generation**: JSON reports for analysis and CI integration
6. **Real-World Testing**: Tests against realistic project structures

## Future Enhancements

- Add performance benchmarks
- Test with larger codebases
- Add more project structure templates
- Integration with CI/CD pipelines
- Custom rule validation
