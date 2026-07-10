# Getting Started with structlint

structlint validates directory structures and file naming patterns against configurable rules.

## Installation

### Using Go (Recommended)

```bash
go install github.com/AxeForging/structlint@latest
```

<details>
<summary><strong>From Binary Downloads</strong></summary>

Download from [Releases](https://github.com/AxeForging/structlint/releases):

**Linux:**
```bash
curl -LO https://github.com/AxeForging/structlint/releases/latest/download/structlint-linux-amd64.tar.gz
tar -xzf structlint-linux-amd64.tar.gz
sudo mv structlint /usr/local/bin/
```

**macOS:**
```bash
# Intel
curl -LO https://github.com/AxeForging/structlint/releases/latest/download/structlint-darwin-amd64.tar.gz

# Apple Silicon
curl -LO https://github.com/AxeForging/structlint/releases/latest/download/structlint-darwin-arm64.tar.gz

tar -xzf structlint-darwin-*.tar.gz
sudo mv structlint /usr/local/bin/
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri "https://github.com/AxeForging/structlint/releases/latest/download/structlint-windows-amd64.zip" -OutFile structlint.zip
Expand-Archive structlint.zip -DestinationPath .
Move-Item structlint.exe C:\Windows\System32\
```

</details>

<details>
<summary><strong>From Source</strong></summary>

```bash
git clone https://github.com/AxeForging/structlint.git
cd structlint
make build
./bin/structlint version
```

</details>

## Quick Start

### 1. Create a Configuration File

For an existing codebase, the fastest path is:

```bash
structlint init --infer
```

`init --infer` walks the actual tree and generates a `.structlint.yaml` that describes what is really there — `validate` passes on the same tree out of the box. Then tighten rules incrementally instead of drowning in violations on day one. Or start from a canned template with `structlint init --type go|node|python|generic`, or hand-write:

```yaml
# .structlint.yaml
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "test/**"
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
    - ".gitignore"
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

### 2. Run Validation

```bash
structlint validate
```

### 3. Wire into your pre-commit hooks

```bash
structlint hook install
```

Auto-detects lefthook, pre-commit, or a raw git hook and merges a `structlint validate --staged --silent` invocation into it. Idempotent — safe to re-run. Use `--dry-run` to preview.

### 4. View Results

**Passing:**
```
--- Validation Summary ---
✓ 42 files/directories passed validation
✗ 0 violations found
🎉 All files and directories comply with the rules!
```

**Failing:**
```
✗ Directory not in allowed list: tmp
✗ Disallowed file naming pattern found: .env.local
✗ Disallowed file naming pattern found: debug.log

--- Validation Summary ---
✓ 39 files/directories passed validation
✗ 3 violations found
```

## Common Use Cases

<details>
<summary><strong>Validate with Specific Config</strong></summary>

```bash
structlint validate --config custom-config.yaml
```

</details>

<details>
<summary><strong>Generate JSON Report</strong></summary>

```bash
structlint validate --json-output report.json
```

Output:
```json
{
  "successes": 42,
  "failures": 0,
  "errors": []
}
```

</details>

<details>
<summary><strong>Use in CI/CD Pipeline</strong></summary>

```bash
# Exit code 0 = pass, 1 = fail
structlint validate || exit 1
```

</details>

<details>
<summary><strong>Verbose/Debug Output</strong></summary>

```bash
structlint validate --log-level debug
```

</details>

<details>
<summary><strong>Silent Mode (Scripts)</strong></summary>

```bash
if structlint validate --silent; then
  echo "Structure OK"
else
  echo "Structure violations found"
fi
```

</details>

## Understanding Configuration

### Directory Rules

| Field | Purpose | Example |
|-------|---------|---------|
| `allowedPaths` | Only these directories allowed | `["cmd/**", "internal/**"]` |
| `disallowedPaths` | These directories forbidden | `["vendor/**", "tmp/**"]` |
| `requiredPaths` | These must exist | `["cmd", "internal"]` |

### File Rules

| Field | Purpose | Example |
|-------|---------|---------|
| `allowed` | Only these files allowed | `["*.go", "*.md"]` |
| `disallowed` | These files forbidden | `["*.env*", "*.log"]` |
| `required` | At least one must exist | `["go.mod", "README.md"]` |

### Ignore

Paths in `ignore` are completely skipped:
```yaml
ignore:
  - ".git"
  - "vendor"
  - "node_modules"
```

## Next Steps

- [Configuration Reference](configuration.md) - Complete config options
- [CLI Reference](cli-reference.md) - All commands and flags
- [CI/CD Integration](ci-cd-integration.md) - Pipeline examples
