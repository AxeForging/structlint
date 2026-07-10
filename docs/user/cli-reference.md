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
| `--changed-only` / `$STRUCTLINT_CHANGED_ONLY` | false | Validate only files changed in `git diff --name-only HEAD` (also filters directory-scope rules) |
| `--staged` / `$STRUCTLINT_STAGED` | false | Validate only staged files (`git diff --cached`); implies `--changed-only`. Use in pre-commit hooks |
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

# Pre-commit hook: only validate what's actually being committed
structlint validate --staged --silent
```

**Scope of `--changed-only` and `--staged`:**

Both flags filter file-level rules (naming, placement, boundaries) to the changed set and prune directory-structure walks so pre-existing drift elsewhere in the repo doesn't block the commit. Existence-based rules (`requiredPaths`, `required` files, `requiredGroups`) are always checked in full — otherwise a commit that deletes `README.md` would silently pass.

### hook install

Merge a `structlint validate --staged --silent` invocation into the repository's pre-commit hook chain. Auto-detects lefthook, pre-commit, or a raw git hook; every merge is idempotent and never overwrites content it did not put there.

```bash
structlint hook install [options]
```

**Options:**

| Option | Default | Description |
|--------|---------|-------------|
| `--type` | auto | Force target: `lefthook`, `pre-commit`, or `git` |
| `--path` | `.` | Repository directory to install into |
| `--dry-run` | false | Print the resulting file, write nothing |

**Detection order** (when `--type` is omitted): `lefthook.yml`/`lefthook.yaml` → lefthook; `.pre-commit-config.yaml` → pre-commit; otherwise a raw git hook under `.git/hooks/pre-commit`.

Running the command twice is a no-op. YAML edits refuse (with a suggested snippet) when the target file uses anchors/aliases, since round-tripping would lose them.

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
