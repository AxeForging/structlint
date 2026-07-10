# Spec 007 — `extends` + built-in presets

## Problem

Every consuming repo re-pastes the same 60-line config that `structlint init` generated, then drifts from it. There is no way to share a baseline across repos or to say "the Go standard, plus these two extra dirs". Config reuse needs an `extends` mechanism, and the templates already living in `internal/cli/init.go` are the obvious built-in baselines.

## Approach

Add an `extends` key to `Config` accepting a string or a list. Each entry is either a **built-in preset name** (`go-standard`, `node-standard`, `python-standard`, `generic`) embedded in the binary via `go:embed` from `internal/config/presets/`, or a **path relative to the extending file**. Resolution is depth-first, parents applied first; cycles are detected; chain depth is capped at 10. Merging: plain string slices become parent slice + child slice with exact-string dedup; the ID'd rule lists (`placement`, `requiredGroups`, `boundaries`) merge by ID with child replacing parent wholesale. Strict parsing is retained for all other unknown keys.

## Non-goals

- No remote extends (URLs, registry fetches) — filesystem paths and embedded presets only.
- No per-key merge strategy syntax (`!override`, `null`-to-clear, etc.). Child cannot REMOVE a parent's slice entry in v1; if a preset is too strict, extend a narrower one or copy it.
- Presets themselves never use `extends` (kept flat; enforced by review, not code).
- Not refactoring `init` to source its templates from the presets (worthwhile follow-up, separate PR).

## Backward compatibility

- Configs without `extends` parse and behave byte-identically. Additive field; strict parse for everything else unchanged.
- **Compat trap (unfixable, state it plainly): an OLD structlint binary strict-parsing a NEW config containing `extends` fails with a yaml field error; unfixable retroactively.** Mitigations: release notes, a `# requires structlint >= vX.Y` comment convention documented next to `extends`, pinned action versions in CI, and `init` never emits `extends` by default.

## Design

### Config field (`internal/config/config.go`)

```go
type Config struct {
    Extends ExtendsList `yaml:"extends" json:"extends"`
    // ... existing fields unchanged
}

// ExtendsList accepts a scalar string or a sequence of strings.
type ExtendsList []string // custom UnmarshalYAML + UnmarshalJSON
```

### Presets (`internal/config/presets/`, new)

`go-standard.yaml`, `node-standard.yaml`, `python-standard.yaml`, `generic.yaml` — content copied from the `projectTemplates` map in `internal/cli/init.go`, minus comments that only make sense in a user's file.

```go
//go:embed presets/*.yaml
var presetFS embed.FS

var presetNames = map[string]string{
    "go-standard":     "presets/go-standard.yaml",
    "node-standard":   "presets/node-standard.yaml",
    "python-standard": "presets/python-standard.yaml",
    "generic":         "presets/generic.yaml",
}
```

### Resolution (`internal/config/merge.go`, new)

```go
const maxExtendsDepth = 10

// loadResolved parses path, resolves its extends chain, and returns the merged config.
// key(entry): preset name as-is; paths as filepath.Abs relative to dir(extendingFile).
func loadResolved(path string, visited map[string]bool, depth int) (*Config, error)

// merge overlays child onto parent per the rules below, returning a new Config.
func merge(parent, child *Config) *Config
```

Algorithm: `LoadConfig(path)` → `loadResolved(path, {}, 0)`. Parse the file (strict); for each entry of `Extends` in order: error if `depth+1 > maxExtendsDepth` ("extends chain too deep (max 10)") or if `visited[key]` ("extends cycle detected: <chain>"); recursively resolve the parent (preset → parse embedded bytes, path → `loadResolved`); fold parents left-to-right into an accumulator with `merge`; finally `merge(accumulator, child)`. `Config.Validate()` runs once, on the final merged config only.

Merge rules:
- String slices (`ignore`, `dir_structure.*`, `file_naming_pattern.*`): parent entries first, then child entries not already present (exact string match). Order stable, no glob-awareness.
- `placement` / `requiredGroups` / `boundaries`: keyed by `id`. Same ID → child rule replaces the parent rule entirely (no field-level merge). New child IDs append after parent rules.
- `Extends` itself is consumed during resolution and empty in the result.

## Implementation steps

1. Add `ExtendsList` type with string-or-list unmarshalling (yaml.v2 `UnmarshalYAML` + `UnmarshalJSON`); add the `Extends` field to `Config`.
2. Add `internal/config/presets/*.yaml` (copied from `init.go` templates) with the `go:embed` FS and `presetNames` map.
3. Add `internal/config/merge.go`: `merge` with slice-dedup + ID-replacement rules.
4. Add `loadResolved` (cycle detection, depth cap, preset/path resolution relative to extending file); route `LoadConfig` through it.
5. Add `test/extends_test.go` (binary-based, see Tests).
6. Docs: `README.md` + config docs — `extends` syntax, preset table, merge semantics, and a boxed warning with the `# requires structlint >= vX.Y` comment convention; note in release notes / CHANGELOG.

## Checklist

- [ ] `Extends ExtendsList` field, string-or-list, strict parse otherwise intact
- [ ] Presets embedded from `internal/config/presets/` (4 files, content matches `init.go` templates)
- [ ] `merge.go`: slice dedup + child-replaces-by-ID for placement/requiredGroups/boundaries
- [ ] `loadResolved`: depth-first parents-first, cycle detection, depth cap 10, paths relative to extending file
- [ ] `Config.Validate()` runs on the merged result only
- [ ] `test/extends_test.go` passing under `go test -race ./...`
- [ ] Docs: syntax, presets, merge rules, old-binary compat warning + version-comment convention
- [ ] `init` output audited: never emits `extends`

## Tests

`test/extends_test.go` (binary-based via `buildBinary`, fixtures in `t.TempDir()`):

- `TestExtends_StringForm` — `extends: go-standard` scalar; tree matching the Go preset validates clean.
- `TestExtends_PresetPlusOverride` — extend `go-standard`, add an extra `allowedPaths` entry; a dir allowed only by the child passes; a dir allowed by neither still fails.
- `TestExtends_RelativePath` — child in `sub/` extending `../base.yaml`; run from a different cwd to prove resolution is relative to the extending file, not cwd.
- `TestExtends_ListMergeOrder` — two parents adding distinct + overlapping slice entries; assert parent-then-child order and exact-string dedup (single occurrence).
- `TestExtends_ChildReplacesRuleByID` — parent and child both define `placement` id `x`; assert only the child's `mustBeUnder` is enforced (parent's variant no longer fires).
- `TestExtends_CycleDetected` — `a.yaml` extends `b.yaml` extends `a.yaml`; expect non-zero exit and "cycle" in the error.
- `TestExtends_DepthCapExceeded` — chain of 11 files; expect "too deep" error.
- `TestExtends_UnknownPresetErrors` — `extends: rust-standard`; error names the entry and lists valid presets.
- `TestExtends_UnknownKeyStillStrict` — config with `extends` AND a typo key (`placment:`); strict parse still fails.
- `TestExtends_InitNeverEmitsExtends` — run `structlint init --type go` (all types); assert generated file contains no `extends` key.

## Verification

- `go test -race ./...`; `make build && ./bin/structlint validate` (self-dogfood — this repo's config does not adopt `extends`, protecting older local binaries in the lefthook flow).
- Manual: scratch repo with `extends: go-standard` + one override; `validate` behaves as merged; add a cycle, confirm the error message shows the chain.
- Compat check: run a pre-`extends` release binary against the scratch config and confirm the failure mode is the documented yaml field error (this is what the release notes must warn about).
