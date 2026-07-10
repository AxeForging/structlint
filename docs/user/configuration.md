# Configuration Reference

structlint uses YAML or JSON configuration files to define validation rules.

## Configuration File Location

Resolution order:

1. `--config` flag value (or `STRUCTLINT_CONFIG` env) — exact path, no discovery.
2. `.structlint.yaml` in the current directory.
3. **Upward search** — starting from `--path` (or cwd), each ancestor directory is checked for `.structlint.yaml`, `.structlint.yml`, then `.structlint.json`. The search stops after checking the first directory containing `.git` (inclusive — a config sitting alongside `.git` is discoverable) or at the filesystem root.

When discovery finds a config, structlint logs `using config: <path> (discovered)` at info level.

### Globs are relative to `--path`, not the config file

Discovered configs behave exactly like explicit ones: their glob patterns match paths relative to the **validation root** (`--path`, defaulting to cwd), not to the directory the config lives in. If you run `structlint validate --path .` from a subdirectory of a monorepo whose config lives at the root, patterns like `internal/**` are matched against paths relative to your subdirectory, not the root — a repo-root config typically won't do the right thing when validating a single subtree.

For monorepos, put a `.structlint.yaml` in each package/service root and run structlint per-package. `extends` (spec 007) lets each package share a common preset.

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

placement: []           # Files that must live under specific directories
requiredGroups: []      # One-of and per-directory required files
boundaries: []          # Go import boundary rules
ignore: []              # Paths to skip during validation
```

Configuration loading is strict. Unknown keys such as `allowed_paths` fail before validation starts, which helps catch CI drift caused by typos.

### Sharing config with `extends`

A config can inherit from one or more parents:

```yaml
# Requires structlint >= v0.6.0 — older binaries reject the `extends` key.
extends: go-standard
dir_structure:
  allowedPaths:
    - "tools/**"     # additional paths on top of the preset
```

`extends` accepts a string or a list. Each entry is either a **built-in preset** or a **filesystem path relative to this file**.

**Built-in presets:**

| Name | Baseline for |
|------|--------------|
| `go-standard` | Go projects (cmd/, internal/, pkg/, ...) |
| `node-standard` | Node.js / TypeScript projects |
| `python-standard` | Python projects (src/, tests/, ...) |
| `generic` | Language-agnostic starter |

**Merge rules:**

- String slices (`ignore`, `dir_structure.*`, `file_naming_pattern.*`) — parent entries first, then child entries not already present. Order stable, exact-string dedup.
- `placement`, `requiredGroups`, `boundaries` — keyed by `id`. Same ID → child rule replaces the parent's wholesale. New IDs append.
- Parents resolve depth-first. Chains are cycle-checked and capped at depth 10.

**Compatibility warning:** the `extends` key requires structlint **v0.6.0 or newer**. Older binaries strict-parse the file and reject it with `field extends not found in type config.Config` — pin your CI action and pre-commit rev to a version that supports it, and add a `# requires structlint >= vX.Y` comment at the top of configs that use `extends`.

### Editor autocomplete via JSON Schema

structlint ships a JSON Schema at [`schema/structlint.schema.json`](https://github.com/AxeForging/structlint/blob/main/schema/structlint.schema.json). Add this modeline to the top of your `.structlint.yaml` for completion, hover docs, and pre-run validation:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/AxeForging/structlint/main/schema/structlint.schema.json
```

The [Red Hat YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml) picks this up in VS Code. JetBrains and Neovim with `yaml-language-server` behave the same way. JSON configs can use a top-level `"$schema"` key:

```json
{
  "$schema": "https://raw.githubusercontent.com/AxeForging/structlint/main/schema/structlint.schema.json",
  "dir_structure": { "allowedPaths": ["."] }
}
```

The schema uses `additionalProperties: false` throughout, mirroring the strict parser — so typos like `placment:` get flagged in the editor with the same rejection the CLI would produce.

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

## Placement Rules

Placement rules ensure files of a given kind live in the expected part of the repository.

```yaml
placement:
  - id: sql-in-migrations
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]

  - id: tests-near-code
    files: ["*_test.go"]
    mustBeUnder: ["internal/**", "pkg/**", "test/**"]
```

| Field | Description |
|-------|-------------|
| `id` | Stable rule identifier used in JSON, SARIF, and GitHub annotations |
| `files` | File name or path globs to match |
| `mustBeUnder` | Directory globs where matching files are allowed |
| `severity` | Optional severity, defaults to `error` |

## Required Groups

Required groups model repository contracts that are more expressive than a single required path.

```yaml
requiredGroups:
  - id: build-entrypoint
    oneOf: ["Makefile", "Taskfile.yml", "justfile"]

  - id: commands-have-main
    eachDirMatching: "cmd/*"
    mustContain: ["main.go"]
    requireMatch: true

  - id: packages-have-docs
    eachDirMatching: "internal/*"
    mustContainOneOf: ["README.md", "doc.go"]
```

| Field | Description |
|-------|-------------|
| `oneOf` | At least one listed path or glob must exist |
| `eachDirMatching` | Directory glob to apply per-directory checks to |
| `mustContain` | Every matching directory must contain each listed file |
| `mustContainOneOf` | Every matching directory must contain at least one listed file |
| `requireMatch` | Fail if `eachDirMatching` finds no directories |

## Boundary Rules

Boundary rules parse imports and block unwanted dependencies between layers. They are language-aware for Go, JavaScript, TypeScript, and Python source files.

```yaml
boundaries:
  - id: domain-no-db
    from: "internal/domain/**"
    cannotImport:
      - "internal/db/**"
      - "internal/http/**"
```

For Go module imports, structlint reads `go.mod` and converts imports like `example.com/app/internal/db` to `internal/db` before matching `cannotImport`. For JS/TS relative imports, paths like `../db/client` are resolved relative to the importing file. For Python, dotted imports like `app.db.client` are normalized to `app/db/client`.

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

placement:
  - id: migrations-only
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]

requiredGroups:
  - id: build-entrypoint
    oneOf: ["Makefile", "Taskfile.yml", "justfile"]
  - id: commands-have-main
    eachDirMatching: "cmd/*"
    mustContain: ["main.go"]

boundaries:
  - id: domain-no-infrastructure
    from: "internal/domain/**"
    cannotImport: ["internal/db/**", "internal/http/**"]

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
