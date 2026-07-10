# Spec 006 — Config upward discovery

## Problem

structlint only looks for `.structlint.yaml` in the current working directory (or wherever `--config` points). Run it from `internal/foo/` in a repo whose config lives at the root and you get a hard error: `configuration file not found`. Monorepos, editor integrations, and hooks that execute from subdirectories all hit this. Every other linter in the toolchain (golangci-lint, ESLint, Ruff) discovers its config by walking upward; structlint should too.

## Approach

When the user did NOT explicitly choose a config (`!cmd.IsSet("config")` — flag and `STRUCTLINT_CONFIG` env both count as explicit) and no config file exists in the start directory, ascend from the start directory looking for `.structlint.yaml`, `.structlint.yml`, or `.structlint.json` (in that priority order per directory). The start directory is `--path` when set, otherwise cwd. Stop ascending at the first directory containing `.git` (that directory IS checked — inclusive) or at the filesystem root. Log which config file was ultimately used.

This is strictly additive: every case that changes behavior is today a hard "configuration file not found" error, so no existing invocation can start behaving differently — some previously-failing ones start working.

## Non-goals

- No rebasing of globs to the config's location. All patterns keep matching against paths relative to the **validation root** (`--path`), exactly as they do today with an explicit `--config ../.structlint.yaml`. Rebasing would silently change the meaning of existing configs, diverge discovered-config behavior from explicit `--config` behavior, and force a second path-resolution mental model. If a repo-root config is discovered while validating a subdirectory, its globs apply to the subtree as-is — document this, don't "fix" it.
- No downward/sibling search, no `$XDG_CONFIG_HOME` or home-directory fallback.
- No change to `init` (it keeps writing to `--config`/`.structlint.yaml` in cwd).
- No config merging across levels — first match wins (merging is spec 007's `extends`).

## Backward compatibility

- Explicit `--config` / `STRUCTLINT_CONFIG` behavior is byte-identical (discovery never runs).
- A config in the start directory is byte-identical (discovery never ascends).
- Only the current hard-error path gains behavior. Exit codes unchanged.
- New: `.structlint.yml` and `.structlint.json` are now honored without `--config` (they participate in discovery, including in the start dir). Previously they required an explicit flag — additive.

## Design

### `internal/config/discover.go` (new)

```go
// configNames in priority order, checked per directory.
var configNames = []string{".structlint.yaml", ".structlint.yml", ".structlint.json"}

// Discover walks upward from startDir looking for a structlint config file.
// It stops after checking the first directory that contains a .git entry
// (file or dir — worktrees have a .git file) or the filesystem root.
// Returns the absolute path of the first match, or "" when none is found.
func Discover(startDir string) (string, error)
```

Algorithm: `dir = filepath.Abs(startDir)`; loop — for each name in `configNames`, if `dir/name` stats as a regular file, return it; if `dir/.git` exists, return `""` (boundary reached, inclusive check already done); if `dir == filepath.Dir(dir)` (fs root), return `""`; else `dir = filepath.Dir(dir)`.

### `internal/cli/root.go` — `LoadConfigForContext`

```go
func LoadConfigForContext(cmd *cli.Command) (*config.Config, error) {
    if cmd.IsSet("config") {
        return loadAt(cmd, cmd.String("config")) // exact current behavior
    }
    start := cmd.String("path")
    if start == "" {
        start = "."
    }
    found, err := config.Discover(start)
    // found == "" → today's "configuration file not found" error,
    // message now mentions the upward search and the boundary reached.
}
```

Note: the `config` flag lives on the root command with a default of `.structlint.yaml`, so `cmd.String("config")` is never empty — `cmd.IsSet("config")` is the only correct "did the user choose" signal (urfave/cli v3 resolves persistent flags and env sources through `IsSet`).

Log line (info level, respects `--silent` conventions of the logger): `using config` with the resolved absolute path and whether it was discovered vs. start-dir vs. explicit.

## Implementation steps

1. Add `internal/config/discover.go` with `Discover` and `configNames` as above.
2. Wire discovery into `LoadConfigForContext` behind `!cmd.IsSet("config")`; improve the not-found error to mention the upward search; add the `using config` log line.
3. Add `test/config_discovery_test.go` (binary-based, see Tests).
4. Docs: `README.md` + `docs/user/` config section — discovery order, `.git` boundary, explicit-flag override, and the "globs are relative to `--path`, not the config file" caveat with a monorepo example.

## Checklist

- [ ] `internal/config/discover.go` with priority-ordered names and `.git`-inclusive stop
- [ ] `LoadConfigForContext` gated on `cmd.IsSet("config")`; error message mentions upward search
- [ ] `using config` log line shows which file was loaded
- [ ] `test/config_discovery_test.go` passing under `go test -race ./...`
- [ ] Docs updated (discovery order + glob-rebase caveat)
- [ ] Self-dogfood: `make build && ./bin/structlint validate` from repo root AND from `internal/`

## Tests

`test/config_discovery_test.go` (binary-based via `buildBinary`, git repos via the spec 001 `initGitRepo` helper, `t.TempDir()`):

- `TestDiscovery_FindsConfigInParent` — config at repo root, run binary with `--path` pointing at a nested subdir and cwd set to that subdir; expect success and the root config's rules applied.
- `TestDiscovery_StopsAtGitBoundary` — config placed ABOVE a directory containing `.git`; run inside the repo; expect "configuration file not found" (the outer config must not leak in).
- `TestDiscovery_GitDirItselfIsChecked` — config in the same directory as `.git`; run from a subdir; expect it found (inclusive stop).
- `TestDiscovery_StartDirConfigWins` — configs in both start dir and parent; assert the start dir one is used (distinguishable rule sets).
- `TestDiscovery_NamePriority` — `.structlint.yml` and `.structlint.json` in one dir; assert `.yml` wins; then `.yaml` vs `.yml`; assert `.yaml` wins.
- `TestDiscovery_ExplicitConfigDisablesSearch` — parent has a valid config but `--config ./missing.yaml` is passed; expect hard error, not the parent config.
- `TestDiscovery_EnvVarCountsAsExplicit` — same as above via `STRUCTLINT_CONFIG=missing.yaml`.
- `TestDiscovery_NoConfigAnywhereErrors` — bare temp dir with `.git`, no config anywhere; expect exit non-zero and the init hint preserved.
- `TestDiscovery_LogsChosenConfig` — assert stderr/log output names the discovered config path.

## Verification

- `go test -race ./...` (binary suite).
- `make build`, then `cd internal && ../bin/structlint validate --path .` — previously an error, now discovers the root `.structlint.yaml` and logs it.
- `./bin/structlint validate --config .structlint.yaml` from root — output byte-identical to before this change.
