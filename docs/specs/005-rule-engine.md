# Spec 005 — rule engine, registry, and single-walk Tree snapshot

## Problem

Every rule family is a hardcoded `Validate*` method on `Validator`, and four of them (`ValidateDirStructure`, `ValidateFileNaming`, `ValidatePlacement`, `ValidateBoundaries`) each run their own `filepath.Walk` over the whole tree — plus one more walk **per required file pattern** in `ValidateRequiredFiles`. Ignore filtering and changed-only skipping are copy-pasted into each walk with small drift between copies. Adding a rule means adding another walk and another hand-wired call in `internal/cli/validate.go`. Specs 009 (`init --infer`) and 010 (`suggest`) need to reuse the walk and enumerate rules with metadata — impossible in the current shape.

**Ordering:** this spec lands after spec 004 (deterministic summary ordering is a precondition for the golden tests below). Specs 009 and 010 depend on the `Tree`/`Registry` infrastructure introduced here.

## Approach

1. One ignore-filtered walk producing an immutable `Tree` snapshot (`Snapshot(root, ignore)`).
2. A `Rule` interface + `Registry(cfg)` returning the seven rule implementations in the legacy execution order.
3. `Validator.Run(path)` = snapshot once, then run each rule over the snapshot. `internal/cli/validate.go` swaps its seven `v.Validate*` calls for `v.Run(path)`.
4. **Behavior parity is bug-for-bug** and gated by golden tests captured from the pre-refactor binary. Quirks are replicated, not fixed (fixes are separate future PRs where they'd be visible diffs).

Rule-outer / entry-inner iteration: each rule replays `Tree.Entries` in walk order, which is exactly the order its old private `filepath.Walk` visited — so violation order, success counts, and all output stay byte-identical.

## Non-goals

- No behavior changes, no new rules, no config changes, no flag changes.
- No parallelism across rules (ordering guarantees first; revisit later if perf demands).
- Not migrating `test/` fixtures or the root `validator_test.go` — they must pass unmodified.

## Backward compatibility

- All output (text, `--format json|sarif|github`), exit codes, and `Successes` counts byte-identical — enforced by golden tests.
- Exported `Validate*` methods remain as thin wrappers (snapshot + run the single corresponding rule) so the root `validator_test.go` and any external callers keep compiling and passing.
- Known quirks preserved verbatim (see Design → Parity quirks).

## Design

### `internal/validator/walk.go`

```go
type Entry struct {
    RelPath string // slash-normalized, "." for root
    Name    string
    IsDir   bool
}

type Tree struct {
    Root    string
    Entries []Entry // filepath.Walk lexical order, ignore-filtered
    dirs    map[string]Entry
    files   map[string]Entry
    walkErr error // first walk error, replayed per rule (see quirks)
}

// Snapshot walks root once, applying ignore patterns exactly as the
// legacy per-rule walks did (pathMatches + SkipDir on ignored dirs).
func Snapshot(root string, ignore []string) *Tree
```

### `internal/validator/engine.go`

```go
type Rule interface {
    Name() string        // stable id: "dir_structure", "file_naming", ...
    Run(ctx *RunContext)
}

type RunContext struct {
    Cfg      *config.Config
    Tree     *Tree
    Reporter Reporter // satisfied by *Validator
    Skip     func(relPath string, isDir bool) bool
}

// Reporter is the narrow surface rules need from Validator.
type Reporter interface {
    AddViolation(code, severity, path, rule, message string)
    Pass(message string)   // printSuccess + Successes++
    CountSuccess()         // Successes++ only (placement/boundaries/groups quirk)
}

// Registry returns all rules in the legacy execution order:
// dirStructure, fileNaming, requiredPaths, requiredFiles,
// placement, requiredGroups, boundaries.
func Registry(cfg *config.Config) []Rule

func (v *Validator) Run(path string)
```

`Skip` closes over the validator's changed-set state and **absorbs spec 001's staged/changed-only logic**: for files it is today's `shouldSkipChanged`; for dirs it is spec 001's `shouldSkipChangedDir` (equals-or-ancestor-of a changed path). Rules that are global by design (required paths/files/groups) simply never call `Skip` — the spec 001 asymmetry lives in the rules, not the predicate.

### `internal/validator/rules_impl.go`

Seven unexported structs (`dirStructureRule`, `fileNamingRule`, `requiredPathsRule`, `requiredFilesRule`, `placementRule`, `requiredGroupsRule`, `boundariesRule`), each a mechanical port of its `Validate*` body iterating `ctx.Tree.Entries` instead of walking. `Validate*` methods become `Snapshot + single rule Run` wrappers.

### Parity quirks (replicate bug-for-bug)

- **`requiredPathsRule` keeps direct `os.Stat`** — today `ValidateRequiredPaths` stats `root/requiredPath` without ignore filtering, so it finds required paths **inside ignored directories**. `Tree` excludes ignored entries; using it would change behavior. Same reasoning for `requiredGroupsRule`'s `existsAt`/`existsAny` helpers (stat-based, see ignored paths) — keep them as-is.
- **`Successes` counter quirks:** `boundariesRule` increments once per (file, matching rule) pair **even when that pair produced boundary violations** (the legacy `v.Successes++` sits after the imports loop unconditionally). `placementRule` increments once per (file, matching rule) pair that passes — a file matched by N placement rules can add N successes. Both use `Reporter.CountSuccess()` (no verbose print), matching today.
- **`walk_error` replay:** legacy behavior emits one `walk_error` per walking rule with rule-specific `rule` labels (`filesystem`, `placement`, `boundaries`, `file_naming_pattern.required`). `Tree.walkErr` records the failure once; each rule that formerly walked re-reports it with its legacy label and message format.
- **Exported `Validate*` stay as thin wrappers** so the root `validator_test.go` compiles and passes unchanged.

### CLI behavior

`internal/cli/validate.go` Action: the seven `v.Validate*` calls collapse to `v.Run(path)`. Flags, baseline handling, report writing, and exit behavior untouched.

## Implementation steps

1. Add parity fixtures under `test/testdata/parity/` (go-style, node-style, and a full-featured project exercising all seven rule families including a parse error and a placement/boundary/group violation) **and** golden files (`.golden.txt`, `.golden.json`) generated by the current pre-refactor binary, plus `test/engine_parity_test.go` asserting current binary output matches goldens byte-for-byte. This commit is green trivially — it pins the contract.
2. Add `internal/validator/walk.go` (`Entry`, `Tree`, `Snapshot`) with package tests for ignore filtering and walk order.
3. Add `internal/validator/engine.go` (`Rule`, `RunContext`, `Reporter`, `Registry`, `Validator.Run`) — `Reporter` implemented by `*Validator`.
4. Port `dirStructureRule` + `fileNamingRule` to `rules_impl.go`; convert their `Validate*` methods to wrappers.
5. Port `requiredPathsRule` (os.Stat) + `requiredFilesRule`; convert wrappers.
6. Port `placementRule` + `requiredGroupsRule` + `boundariesRule` with the `Successes` quirks; convert wrappers.
7. Swap `internal/cli/validate.go` to `v.Run(path)`; wire `Skip` to the changed/staged state.
8. Extend `test/performance_test.go` with the single-walk budget assertion on a ~5k-file tree.

## Checklist

- [ ] Parity fixtures + goldens captured from pre-refactor binary (commit 1, before any refactor)
- [ ] `walk.go`: `Snapshot`/`Tree`/`Entry` + ignore/order tests
- [ ] `engine.go`: `Rule`, `RunContext`, `Reporter`, `Registry`, `Validator.Run`
- [ ] All seven rules ported to `rules_impl.go`, legacy order preserved
- [ ] Parity quirks preserved: requiredPaths `os.Stat`, boundaries/placement `Successes`, walk_error replay
- [ ] `Validate*` methods are thin wrappers; root `validator_test.go` untouched and green
- [ ] `validate.go` uses `v.Run(path)`; `Skip` absorbs staged/changed-only logic
- [ ] Performance budget test on ~5k-file tree
- [ ] `docs/AI/overview.md` (or architecture doc) updated to describe engine/registry
- [ ] Self-validation passes — no `.structlint.yaml` change expected (all new files under `internal/**` and `test/**`)

## Tests

`test/engine_parity_test.go` (binary-based; goldens are the gate — the refactor may not land while any of these fail):

- `TestEngineParity_TextOutput` — for each fixture in `test/testdata/parity/`, run `validate` (grouped, default flags) and diff stdout byte-for-byte against `<fixture>.golden.txt`.
- `TestEngineParity_JSONOutput` — same fixtures with `--format json`, byte-for-byte against `<fixture>.golden.json` (locks violation order, `successes` counts, and summary grouping).
- `TestEngineParity_ExitCodes` — per fixture, assert the exit code matches the recorded pre-refactor code.
- `TestEngineParity_RequiredPathInsideIgnoredDir` — dedicated fixture where a `requiredPaths` entry only exists inside an ignored directory; must still PASS (locks the `os.Stat` quirk).
- `TestEngineParity_BoundarySuccessQuirk` — fixture where a file violates a boundary rule; assert `--format json` `successes` still counts the (file, rule) pair (locks the counter quirk).

`test/performance_test.go`:

- `TestEnginePerformance_SingleWalkBudget` — generate ~5k files in `t.TempDir()`, run `validate`; assert wall time within budget and not slower than the recorded multi-walk baseline.

## Verification

- `go test -race ./...` — parity goldens, root `validator_test.go`, and the full existing `test/` suite all green with zero fixture edits.
- Golden parity diff must be empty (the hard gate from the roadmap).
- `make build && ./bin/structlint validate` — self-dogfood output identical to pre-refactor binary on this repo (`diff <(old validate) <(new validate)`).
- Perf test shows walk time ≤ previous multi-walk time on the 5k-file tree.
