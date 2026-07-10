---
name: structlint
description: >
  Validate and fix project structure — file placement, naming, directory
  layout, import boundaries. Use when the user says a file is in the wrong
  place, when structlint reports violations, when adopting structlint on a
  legacy repo, or when wiring pre-commit / CI checks. Trigger phrases:
  "structlint violation", "where should this file go", "enforce directory
  layout", ".structlint.yaml", "file in the wrong place".
---

# structlint

structlint is a Go CLI that validates project structure against `.structlint.yaml`. This skill teaches you the machine surface (JSON contracts, exit codes) and the fix loop — you almost never need to hand-write configs; you can drive everything from `suggest`.

## When to run

- **Pre-commit (recommended)** — `structlint validate --staged --silent`. Diffs the git index (not HEAD), so a hook only lints what's actually being committed. Wire it with `structlint hook install` (auto-detects lefthook, pre-commit, or raw `.git/hooks`).
- **CI** — `structlint validate --format github` for inline annotations, `--format sarif` for code scanning, `--format json --json-output report.json` for artifact storage. Exit 0 = clean, 1 = violations.
- **Adopting on a legacy repo** — `structlint init --infer` walks the tree and writes a baseline that `validate` passes on. Tighten incrementally. Alternative: `structlint validate --json-output baseline.json` then commit and use `--baseline baseline.json` to grandfather existing drift while catching new drift.

## Violation codes and how to fix

The 13 codes below are **frozen**: names never change and never disappear. New rules add new codes. Full detail in [docs/user/violation-codes.md](../../docs/user/violation-codes.md).

| Code | Emitted by | Typical fix |
|------|------------|-------------|
| `disallowed_directory` | `dir_structure.disallowedPaths` | **Tree.** Remove or move the directory — the prohibition is deliberate. |
| `unallowed_directory` | `dir_structure.allowedPaths` | **Config** when the dir is intentional (add generalized glob); **tree** if it's stray. |
| `disallowed_file_pattern` | `file_naming_pattern.disallowed` | **Tree.** Deliberate prohibition (secrets, backups, OS junk). Never loosen the rule. |
| `unallowed_file_pattern` | `file_naming_pattern.allowed` | **Config** — add `*.ext` (or exact name for extensionless files) if intentional. |
| `missing_required_directory` | `dir_structure.requiredPaths` | **Tree.** Create the directory. |
| `missing_required_file` | `file_naming_pattern.required` | **Tree.** Create the file (or first pattern-matching file). |
| `placement_violation` | `placement[]` | **Tree.** `git mv` the file under the rule's `mustBeUnder` root. |
| `missing_required_group` | `requiredGroups[].oneOf` | **Tree.** Create at least one of the listed files somewhere. |
| `missing_required_group_match` | `requiredGroups[].eachDirMatching` with `requireMatch: true` | **Config or tree** — the pattern matched no directories; either the pattern is wrong or the expected dirs are missing. |
| `missing_group_file` | `requiredGroups[].mustContain` / `mustContainOneOf` | **Tree.** Add the file to the matching directory. |
| `boundary_violation` | `boundaries[]` | **Tree.** Refactor the import — no mechanical fix; needs human judgment. |
| `parse_error` | Boundary rules parsing a source file | **Operational.** Fix the source file (or exclude it via `ignore`). |
| `walk_error` | Filesystem walk failure | **Operational.** Permissions, missing dir, disk error. |

## Machine contracts

### `validate --format json`

```json
{
  "successes": 42,
  "failures": 2,
  "total_violations": 2,
  "errors": ["Directory not in allowed list: tmp", "..."],
  "violations": [
    {
      "code": "unallowed_directory",
      "severity": "error",
      "path": "tmp",
      "rule": "dir_structure.allowedPaths",
      "message": "Directory not in allowed list: tmp"
    }
  ],
  "summary": {
    "total_successes": 42,
    "total_failures": 2,
    "violations": [{"type": "unallowed_directory", "count": 1, "examples": [...], "description": "Directories not in the allowed list"}]
  }
}
```

The `violations[]` array is the stable contract to key on. Group by `.code`, not by parsing `.message`.

### `suggest --format json` (v1)

```json
{
  "version": 1,
  "configPath": ".structlint.yaml",
  "proposals": [
    {"kind": "config_add", "section": "dir_structure.allowedPaths", "value": "tools/**", "reason": "unallowed_directory: ...", "paths": ["tools"]},
    {"kind": "move", "from": "stray.sql", "to": "migrations/stray.sql", "command": "git mv stray.sql migrations/stray.sql", "reason": "...", "paths": ["stray.sql"]},
    {"kind": "create", "path": "README.md", "reason": "missing_required_file: ...", "paths": ["README.md"]},
    {"kind": "note", "reason": "boundary_violation: ... — no mechanical fix", "paths": ["..."]}
  ],
  "configDiff": "--- a/.structlint.yaml\n+++ b/.structlint.yaml\n@@ ..."
}
```

`configDiff` is a real unified diff built by inserting into the original config text — so it preserves comments/order/quoting and applies cleanly with `patch -p1`. **`suggest` never proposes loosening `disallowed` / `disallowedPaths`.**

### Exit codes

- `validate`: **0** clean, **1** violations found, **2** config error, **3** runtime error.
- `suggest`: **0** always when it ran (even with proposals present — advisory), **non-zero** only on operational error (no config, unreadable tree, bad flag).

## The fix loop

```bash
structlint suggest --format json > /tmp/report.json

# 1. Config changes: apply the diff.
jq -r .configDiff /tmp/report.json | patch -p1

# 2. Placement violations: run the git mv commands.
jq -r '.proposals[] | select(.kind=="move") | .command' /tmp/report.json | sh

# 3. Missing paths: create them.
jq -r '.proposals[] | select(.kind=="create") | .path' /tmp/report.json | xargs -I{} sh -c 'mkdir -p $(dirname "{}") && touch "{}"'

# 4. Verify.
structlint validate
```

Iterate until `validate` exits 0. `note` proposals are for a human — read them, don't blindly automate.

## Setup recipes

**Pre-commit hook (any framework)**:
```bash
structlint hook install                      # auto-detects lefthook / pre-commit / git
structlint hook install --type git --dry-run # preview
```

**pre-commit framework (`.pre-commit-config.yaml`)**:
```yaml
repos:
  - repo: https://github.com/AxeForging/structlint
    rev: v0.6.0
    hooks:
      - id: structlint
```

**GitHub Actions**:
```yaml
- uses: AxeForging/structlint@main
  with:
    config: .structlint.yaml
    comment-on-pr: "true"
```

**Sharing config across repos (`extends`)** — requires structlint ≥ v0.6.0:
```yaml
# requires structlint >= v0.6.0
extends: go-standard          # or node-standard, python-standard, generic
dir_structure:
  allowedPaths:
    - "tools/**"              # additions on top of the preset
```

**Editor autocomplete** — add to the top of `.structlint.yaml`:
```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/AxeForging/structlint/main/schema/structlint.schema.json
```

## Gotchas

- **Config parse is strict.** `allowed_paths:` (typo) fails at load time, not silently. Unknown keys → error. This is intentional — it catches CI drift.
- **`extends` requires a newer binary.** An old structlint reading a config with `extends` fails with `field extends not found in type config.Config`. Pin your CI action / pre-commit `rev` to a version that supports it.
- **`--changed-only` diffs HEAD; `--staged` diffs the index.** Pre-commit hooks want `--staged` — otherwise you lint the working tree, not the commit.
- **`ignore` ≠ `disallowed`.** `ignore` says "don't look here"; `disallowed` says "look here and complain". Vendor / node_modules → `ignore`. `.env` / secrets → `disallowed`.
- **Globs are relative to `--path`, not to the config file.** A repo-root config discovered from a subdirectory still evaluates patterns against `--path`.
