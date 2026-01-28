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

Validate directory structure and file naming patterns.

```bash
structlint validate [options] [path]
```

**Options:**

| Option | Description |
|--------|-------------|
| `--config, -c` | Path to config file |
| `--json-output` | Path to write JSON report |
| `--silent` | Suppress output (exit code only) |
| `--strict` | Treat warnings as errors |

**Examples:**

```bash
# Validate current directory
structlint validate

# Validate with specific config
structlint validate --config .structlint.yaml

# Validate and generate JSON report
structlint validate --json-output report.json

# Validate specific directory
structlint validate /path/to/project

# Silent mode (for scripts)
structlint validate --silent && echo "Valid"
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
  "errors": [
    "Directory not in allowed list: tmp",
    "Disallowed file found: .env.local"
  ],
  "summary": {
    "directories_checked": 15,
    "files_checked": 27,
    "violations_by_type": {
      "dir_not_allowed": 1,
      "file_disallowed": 1
    }
  }
}
```
