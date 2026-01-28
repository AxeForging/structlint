# Configuration Reference

structlint uses YAML or JSON configuration files to define validation rules.

## Configuration File Location

By default, structlint looks for:
1. `--config` flag value
2. `STRUCTLINT_CONFIG` environment variable
3. `.structlint.yaml` in current directory
4. `.structlint.yml` in current directory
5. `.structlint.json` in current directory

## Configuration Structure

```yaml
dir_structure:
  allowedPaths: []      # Glob patterns for allowed directories
  disallowedPaths: []   # Glob patterns for disallowed directories
  requiredPaths: []     # Directories that must exist

file_naming_pattern:
  allowed: []           # Glob patterns for allowed files
  disallowed: []        # Glob patterns for disallowed files
  required: []          # Files that must exist (supports globs)

ignore: []              # Paths to skip during validation
```

## Glob Pattern Syntax

structlint supports standard glob patterns:

| Pattern | Description |
|---------|-------------|
| `*` | Matches any sequence of characters (not including `/`) |
| `**` | Matches any sequence including `/` (recursive) |
| `?` | Matches any single character |
| `[abc]` | Matches any character in the set |
| `[!abc]` | Matches any character not in the set |

### Examples

```yaml
allowedPaths:
  - "."           # Root directory only
  - "cmd/**"      # cmd/ and all subdirectories
  - "src/*.go"    # Go files directly in src/
  - "test/*"      # Direct children of test/
```

## Directory Structure Rules

### allowedPaths

Directories that are permitted. If specified, any directory not matching these patterns is a violation.

```yaml
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "test/**"
```

### disallowedPaths

Directories that are explicitly forbidden.

```yaml
dir_structure:
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - ".cache/**"
```

### requiredPaths

Directories that must exist.

```yaml
dir_structure:
  requiredPaths:
    - "cmd"
    - "internal"
    - "docs"
```

## File Naming Rules

### allowed

File patterns that are permitted.

```yaml
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.md"
    - "Makefile"
    - "Dockerfile*"
    - ".gitignore"
```

### disallowed

File patterns that are forbidden.

```yaml
file_naming_pattern:
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*~"
    - "*.bak"
```

### required

Files that must exist.

```yaml
file_naming_pattern:
  required:
    - "go.mod"
    - "README.md"
    - ".gitignore"
    - "*.go"  # At least one Go file
```

## Ignore Rules

Paths to completely skip during validation.

```yaml
ignore:
  - ".git"
  - "vendor"
  - "node_modules"
  - "bin"
  - "dist"
  - ".idea"
  - ".vscode"
```

## Complete Example

```yaml
# .structlint.yaml - Go Project Configuration

dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "api/**"
    - "web/**"
    - "configs/**"
    - "scripts/**"
    - "test/**"
    - "docs/**"
    - ".github/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
  requiredPaths:
    - "cmd"
    - "internal"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.mod"
    - "*.sum"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.toml"
    - "*.md"
    - "*.txt"
    - "Makefile"
    - "Dockerfile*"
    - ".gitignore"
    - ".golangci.yml"
    - "go.work"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*~"
    - "*.swp"
    - ".DS_Store"
  required:
    - "go.mod"
    - "README.md"
    - ".gitignore"

ignore:
  - ".git"
  - "vendor"
  - "bin"
  - "dist"
  - ".idea"
  - ".vscode"
```

## Environment Variables

All configuration options can be overridden via environment variables:

| Variable | Description |
|----------|-------------|
| `STRUCTLINT_CONFIG` | Path to configuration file |
| `STRUCTLINT_LOG_LEVEL` | Log level (debug, info, warn, error) |
| `STRUCTLINT_NO_COLOR` | Disable colored output |
