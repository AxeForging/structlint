# Spec 002 — `structlint hook install`

## Problem

Getting structlint into a repo's pre-commit chain is currently a copy-paste-and-hope exercise: users have to know which hook framework the repo uses, hand-craft a snippet, and re-paste it if they lose their config. This kills adoption. Every consuming repo we've onboarded has needed the same three or four lines wired into `lefthook.yml`, `.pre-commit-config.yaml`, or `.git/hooks/pre-commit` — that's a command.

## Approach

New `structlint hook install [--type lefthook|pre-commit|git] [--dry-run]` command that:

1. Auto-detects the target from files already in the repo (`--type` overrides).
2. Merges a `structlint validate --staged --silent` invocation into the target file **without ever touching content it didn't put there**.
3. Is idempotent — running it twice is a no-op.
4. Is reversible in principle (marker-blocked for raw git hooks; append-only for YAML frameworks, so users can revert with a text editor).

Detection order (when `--type` is omitted, first match wins):
1. `lefthook.yml` or `lefthook.yaml` in cwd → **lefthook**
2. `.pre-commit-config.yaml` in cwd → **pre-commit**
3. else → **raw git hook**

## Non-goals

- Not implementing `hook uninstall` in this spec (it's the marker-block reverse for raw; for YAML frameworks the file the user edited by hand is theirs, we won't diff-and-remove). Follow-up if needed.
- Not touching `commit-msg`, `pre-push`, or other stages — only pre-commit.
- Not adding a global "install everywhere" flag; if you want lefthook + a raw hook, run the command twice with `--type`.

## Backward compatibility

- Purely additive: new command, no changes to existing commands or flags.
- No new dependency beyond `gopkg.in/yaml.v3` for AST-preserving YAML.

## Design

### CLI

```
structlint hook install [--type lefthook|pre-commit|git] [--dry-run]
```

- `--type` — force target instead of auto-detecting.
- `--dry-run` — print the resulting file (or diff summary) to stdout; write nothing to disk.

Exit codes: 0 on success (including "already installed" no-op); 2 on hard refusal (see anchors/aliases below); 3 on I/O errors.

### `internal/hooks/` layout

- `detect.go` — auto-detection + shared marker constants + string constants (`HookRun = "structlint validate --staged --silent"`).
- `lefthook.go` — `InstallLefthook(dir string, dryRun bool) (Result, error)`.
- `precommit.go` — `InstallPreCommit(dir string, dryRun bool) (Result, error)`.
- `githook.go` — `InstallGitHook(dir string, dryRun bool) (Result, error)`.

`Result` = `{Action string /* "installed"|"already-installed"|"refused" */; File string; Reason string; Preview string}`. Preview is populated for `--dry-run`.

### Lefthook merge (yaml.v3 AST)

- Parse `lefthook.yml` into `*yaml.Node`. yaml.v3 round-trips comments and key order for nodes it didn't touch.
- Walk to `pre-commit.commands`. If either doesn't exist, create it (mapping node).
- If a `structlint` key already exists under `commands`, no-op (`already-installed`). We don't try to update in place — if the user hand-edited it we defer to them.
- Otherwise append a mapping node:
  ```yaml
  structlint:
    run: structlint validate --staged --silent
  ```
- Re-encode with `yaml.Encoder{Indent: 2}`.
- **Refusal:** walk the AST for `Kind == yaml.AliasNode` (`*foo`) or non-empty `Anchor` on any node. If present, refuse — the AST round-trip through yaml.v3 loses anchor identity, and rewriting the file could corrupt the user's structure. Print the snippet to stdout instead and return "refused" with reason.

### Pre-commit merge (yaml.v3 AST)

- Parse `.pre-commit-config.yaml`.
- Locate `repos` sequence; if absent, create it.
- If any entry has `repo: https://github.com/AxeForging/structlint` OR any `hooks[].id == structlint`, no-op.
- Else append:
  ```yaml
  - repo: https://github.com/AxeForging/structlint
    rev: <structlint version from internal/build>
    hooks:
      - id: structlint
  ```
- Same anchor/alias refusal rule as lefthook.

### Raw git hook

- Resolve hooks dir: `git rev-parse --git-path hooks` (respects `core.hooksPath`, honors worktrees). If git isn't installed or we're not in a repo, fail with a helpful message.
- Marker block:
  ```
  # >>> structlint hook >>>
  # Managed by `structlint hook install`. Edit inside markers only.
  if command -v structlint >/dev/null 2>&1; then
    structlint validate --staged --silent || exit 1
  fi
  # <<< structlint hook <<<
  ```
- Three cases for `pre-commit`:
  1. Absent → write new file with `#!/bin/sh` + block, chmod 0755.
  2. Present but no markers → append the block (preserving user content byte-for-byte).
  3. Present with markers → replace content between markers with the current block.
- Never touch content outside the markers. Idempotent by construction — case 3 always yields the same output.
- `command -v structlint` guard: prevents committing when structlint isn't installed (e.g., a teammate who hasn't run `go install` yet gets a helpful "not installed" skip instead of a broken hook). Trade-off: an uninstalled tool silently passes. Documented as intentional — pair with the CI `validate` job for enforcement.

### Version resolution for the pre-commit `rev`

- Use `internal/build.Version` (the ldflags-injected value). When it's `dev` (unstamped local build), fall back to `main` with a stderr warning — pre-commit will pin to that tag, which for `dev` builds is the least surprising thing. In released binaries the injected version wins.

## Tests

`test/hook_install_test.go` (binary-based):

For each of {lefthook, pre-commit, git}:

- `TestHook<Kind>_FreshFile` — no file exists, run install, assert file created with expected content.
- `TestHook<Kind>_ExistingWithout` — file exists with user content (comments, ordering, extra keys); run install; assert user content byte-preserved and structlint entry present.
- `TestHook<Kind>_Idempotent` — run install twice; assert second run reports "already-installed" AND the file bytes are identical between run 1 and run 2.
- `TestHook<Kind>_DryRun` — assert file on disk is byte-identical before/after and stdout contains a preview.

Plus:

- `TestHookLefthook_RefusesOnAnchors` — lefthook.yml with `&anchor`/`*alias`; assert refusal + suggestion snippet printed + file unchanged.
- `TestHookGit_ReplaceBetweenMarkers` — existing hook with an outdated marker block (older invocation) — assert the block is replaced, content outside markers preserved.
- `TestHookGit_NoRepo` — no `.git`, assert helpful error and non-zero exit.
- `TestHookDetect_AutoDetect` — table across combinations of files present asserting the right merger is picked.

Uses the `initGitRepo` helper introduced in spec 001; tests run the binary.

## Verification

- `go test -race ./...`
- `make build && ./bin/structlint hook install --dry-run` in the structlint repo itself — should preview appending to `lefthook.yml` without touching it (or say "already-installed" if a previous spec added the entry).
- Manual: in a scratch repo with no hook config, run `hook install`, verify the raw `.git/hooks/pre-commit` runs, `git commit` triggers it.
