# CLI Reference

## Global Options

```
--config, -c      Path to configuration file
--log-level       Log level: debug, info, warn, error (default: info)
--no-color        Disable colored output
--help, -h        Show help
--version, -v     Show version
```

## Commands

### validate

Validate directory structure, file naming patterns, placement rules, required groups, and import boundaries.

```bash
structlint validate [options]
```

**Options:**

| Option | Default | Description |
|--------|---------|-------------|
| `--path` / `$STRUCTLINT_PATH` | `.` | Directory to validate |
| `--config` / `$STRUCTLINT_CONFIG` | `.structlint.yaml` | Path to config file |
| `--json-output` / `$STRUCTLINT_JSON_OUTPUT` | — | Path to write JSON report |
| `--format` / `$STRUCTLINT_FORMAT` | `text` | Output format: `text`, `json`, `sarif`, or `github` |
| `--baseline` / `$STRUCTLINT_BASELINE` | — | JSON report with known violations to suppress |
| `--changed-only` / `$STRUCTLINT_CHANGED_ONLY` | false | Validate only files changed in `git diff --name-only HEAD` |
| `--silent` / `$STRUCTLINT_SILENT` | false | Suppress text logging |
| `--group-violations` / `$STRUCTLINT_GROUP_VIOLATIONS` | true | Group text output by violation type |
| `--verbose` / `$STRUCTLINT_VERBOSE` | false | Show successful checks as well as violations |

**Examples:**

```bash
# Validate current directory
structlint validate

# Validate with specific config
structlint validate --config .structlint.yaml

# Validate and generate JSON report
structlint validate --json-output report.json

# Validate specific directory
structlint validate --path /path/to/project

# Silent mode (for scripts)
structlint validate --silent && echo "Valid"

# GitHub Actions annotations
structlint validate --format github

# SARIF for code scanning
structlint validate --format sarif > structlint.sarif

# Suppress known drift while failing on new drift
structlint validate --baseline .structlint-baseline.json
```

### version

Display version information.

```bash
structlint version
```

**Output:**

```
v1.0.0 (commit abc1234) built 2024-01-15T10:30:00Z by user
```

### completion

Generate shell completion scripts.

```bash
structlint completion <shell>
```

**Supported shells:** bash, zsh, fish

**Setup:**

```bash
# Bash
structlint completion bash > /etc/bash_completion.d/structlint

# Zsh
structlint completion zsh > "${fpath[1]}/_structlint"

# Fish
structlint completion fish > ~/.config/fish/completions/structlint.fish
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Validation passed |
| 1 | Validation failed (violations found) |
| 2 | Configuration error |
| 3 | Runtime error |

## JSON Report Format

When using `--json-output`, the report structure is:

```json
{
  "successes": 42,
  "failures": 2,
  "total_violations": 2,
  "errors": [
    "Directory not in allowed list: tmp",
    "Disallowed file naming pattern found: .env.local"
  ],
  "violations": [
    {
      "code": "unallowed_directory",
      "severity": "error",
      "path": "tmp",
      "rule": "dir_structure.allowedPaths",
      "message": "Directory not in allowed list: tmp"
    },
    {
      "code": "disallowed_file_pattern",
      "severity": "error",
      "path": ".env.local",
      "rule": "*.env*",
      "message": "Disallowed file naming pattern found: .env.local"
    }
  ],
  "summary": {
    "total_successes": 42,
    "total_failures": 2,
    "violations": []
  }
}
```

The `violations` array is the stable CI contract. Human-readable `errors` are kept for backward compatibility.
