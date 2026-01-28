# structlint - AI Context Overview

> **For AI assistants:** This document provides the context you need to understand, use, and modify structlint.

## What is structlint?

structlint is a CLI tool that validates project directory structures and file naming patterns against configurable rules. It helps enforce consistent project organization.

## Quick Reference

```bash
# Validate current directory
structlint validate

# With config file
structlint validate --config .structlint.yaml

# JSON output for parsing
structlint validate --json-output report.json

# Check version
structlint version
```

## Configuration Structure

```yaml
# .structlint.yaml
dir_structure:
  allowedPaths: []      # Glob patterns - directories allowed
  disallowedPaths: []   # Glob patterns - directories forbidden
  requiredPaths: []     # Exact paths - must exist

file_naming_pattern:
  allowed: []           # Glob patterns - files allowed
  disallowed: []        # Glob patterns - files forbidden
  required: []          # Glob patterns - must have at least one match

ignore: []              # Paths to skip entirely
```

## Key Behaviors

<details>
<summary><strong>How allowedPaths works</strong></summary>

If `allowedPaths` is non-empty, ONLY directories matching those patterns are allowed. Everything else is a violation.

```yaml
# Only cmd/, internal/, and their subdirectories allowed
allowedPaths:
  - "."          # Root is always needed
  - "cmd/**"
  - "internal/**"
```

If `allowedPaths` is empty or omitted, all directories are allowed (no restrictions).

</details>

<details>
<summary><strong>How disallowedPaths works</strong></summary>

Any directory matching `disallowedPaths` patterns is a violation.

```yaml
# These directories are forbidden
disallowedPaths:
  - "vendor/**"
  - "node_modules/**"
  - "tmp/**"
```

</details>

<details>
<summary><strong>How ignore works</strong></summary>

Paths in `ignore` are completely skipped - not validated at all.

```yaml
# These paths won't be checked
ignore:
  - ".git"
  - "vendor"
  - "bin"
```

</details>

<details>
<summary><strong>Glob pattern syntax</strong></summary>

| Pattern | Meaning |
|---------|---------|
| `*` | Any chars except `/` |
| `**` | Any chars including `/` |
| `?` | Single char |
| `[abc]` | Char in set |
| `{a,b}` | Either a or b |

Examples:
- `*.go` - Go files in current dir
- `**/*.go` - Go files anywhere
- `cmd/**` - cmd/ and all subdirs
- `test/*_test.go` - Test files in test/

</details>

## Architecture

```
structlint/
├── cmd/structlint/main.go      # Entry point
├── internal/
│   ├── app/app.go              # Root CLI command
│   ├── cli/
│   │   ├── root.go             # Global flags, setup
│   │   ├── validate.go         # validate command
│   │   ├── version.go          # version command
│   │   └── completion.go       # shell completions
│   ├── config/config.go        # Config loading
│   ├── validator/
│   │   ├── validator.go        # Core validation
│   │   ├── types.go            # JSONReport struct
│   │   └── summary.go          # Output formatting
│   ├── build/info.go           # Version info
│   └── logging/logging.go      # Logging setup
└── test/                       # Integration tests
```

## Common Tasks for AI

<details>
<summary><strong>Adding a new CLI flag</strong></summary>

Edit `internal/cli/validate.go`:

```go
// Add to Flags slice in validateCommand()
&cli.BoolFlag{
    Name:    "new-flag",
    Usage:   "Description of flag",
    Sources: cli.EnvVars("STRUCTLINT_NEW_FLAG"),
},

// Use in action function
if cmd.Bool("new-flag") {
    // handle flag
}
```

</details>

<details>
<summary><strong>Adding a new validation rule</strong></summary>

1. Add to config struct in `internal/config/config.go`:
```go
type Config struct {
    DirStructure      DirStructureConfig
    FileNamingPattern FileNamingPatternConfig
    NewRule           NewRuleConfig  // Add this
    Ignore            []string
}

type NewRuleConfig struct {
    Patterns []string `yaml:"patterns" json:"patterns"`
}
```

2. Add validation in `internal/validator/validator.go`:
```go
func (v *Validator) ValidateNewRule(rootPath string) {
    // Implementation
}
```

3. Call from `internal/cli/validate.go`

4. Add tests in `test/`

</details>

<details>
<summary><strong>Modifying JSON report</strong></summary>

Edit `internal/validator/types.go`:

```go
type JSONReport struct {
    Successes int      `json:"successes"`
    Failures  int      `json:"failures"`
    Errors    []string `json:"errors"`
    NewField  string   `json:"new_field"`  // Add fields here
}
```

</details>

<details>
<summary><strong>Writing tests</strong></summary>

Tests use binary-first approach - build the CLI and run it:

```go
func TestNewFeature(t *testing.T) {
    bin := buildBinary(t)  // Compiles CLI

    files := map[string]string{
        "main.go": "package main",
    }
    config := `dir_structure:
  allowedPaths: ["."]
`
    projectDir := createTestProject(t, files, config)

    out, err := runBinaryInDir(t, bin, projectDir,
        "validate", "--config", ".structlint.yaml")

    if err != nil {
        t.Errorf("Failed: %v\nOutput: %s", err, out)
    }
}
```

</details>

## Dependencies

Only 3 direct dependencies (minimal):

| Package | Purpose |
|---------|---------|
| `github.com/gobwas/glob` | Glob pattern matching |
| `github.com/urfave/cli/v3` | CLI framework |
| `gopkg.in/yaml.v2` | YAML parsing |

## Build Commands

```bash
make build        # Build for current platform → bin/structlint
make build-all    # Build all platforms → dist/
make test         # Run all tests
make lint         # Run linter
make clean        # Remove build artifacts
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Validation passed |
| 1 | Validation failed |
| 2 | Config error |
| 3 | Runtime error |
