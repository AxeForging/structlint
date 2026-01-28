# Getting Started with structlint

structlint is a CLI tool that validates directory structures and file naming patterns in your projects.

## Installation

### Using Go Install (Recommended)

```bash
go install github.com/AxeForging/structlint@latest
```

### From Source

```bash
git clone https://github.com/AxeForging/structlint.git
cd structlint
make build
# Binary will be at ./bin/structlint
```

### From Releases

Download the appropriate binary for your platform from the [Releases page](https://github.com/AxeForging/structlint/releases).

## Quick Start

1. **Create a configuration file** in your project root:

```yaml
# .structlint.yaml
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
  disallowedPaths:
    - "vendor/**"
    - "tmp/**"
  requiredPaths:
    - "cmd"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"
    - "*.md"
    - "Makefile"
  disallowed:
    - "*.env*"
    - "*.log"
  required:
    - "go.mod"
    - "README.md"

ignore:
  - ".git"
  - "vendor"
  - "bin"
```

2. **Run validation**:

```bash
structlint validate
```

3. **View results**:

```
--- Validation Summary ---
✓ 42 files/directories passed validation
✗ 0 violations found
```

## Common Use Cases

### Validate a Go Project

```bash
structlint validate --config .structlint.yaml
```

### Generate JSON Report

```bash
structlint validate --json-output report.json
```

### Use in CI/CD

```bash
structlint validate || exit 1
```

### Verbose Output

```bash
structlint validate --log-level debug
```

## Next Steps

- [Configuration Reference](configuration.md)
- [CLI Reference](cli-reference.md)
- [CI/CD Integration](ci-cd-integration.md)
