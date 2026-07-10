# Spec 010 — `structlint suggest`

## Problem

`validate` tells you *what* is wrong and stops. The human (or agent) then has to reverse-engineer the fix: should the config gain an entry, or should the file move? For the headline use case — AI tools dropping files in the wrong place — that translation step is where adoption dies. We already have everything needed to propose the fix mechanically: the rule engine knows which rule fired (spec 005), and the infer heuristics know how to generalize a path into a config entry (spec 009). Nothing connects them.

## Approach

New print-only advisory command:

```
structlint suggest [--path <dir>] [--format text|json]
```

It runs the same engine as `validate`, then maps each violation to a *proposal* — either a config change (rendered as a ready-to-apply unified diff against the actual config file) or a filesystem action (`git mv`, create). It never writes anything and never prompts.

Exit semantics: **0 even when proposals exist** — suggest is advisory; deciding that violations fail the build is `validate`'s job. Exit 1 only on operational errors (no config found, unreadable tree, bad flag values).

Violation → proposal mapping (keyed on `Violation.Code`, the registry from spec 004):

| Code | Proposal |
|---|---|
| `unallowed_directory` | `config_add`: generalized glob (spec 009's `AllowedPaths` generalizer) into `dir_structure.allowedPaths` |
| `unallowed_file_pattern` | `config_add`: `*.ext` or exact extensionless name into `file_naming_pattern.allowed` |
| `disallowed_directory`, `disallowed_file_pattern` | `note`: "matches an explicit disallowed rule — review the rule or remove the path". **Never** propose loosening `disallowed` — those entries are deliberate prohibitions |
| `placement_violation` | `move`: from/to derived from the placement rule's expected location, with a copy-pasteable `git mv` command |
| `missing_required_directory`, `missing_required_file`, `missing_group_file` | `create`: the missing path |
| everything else (`boundary_violation`, `missing_required_group*`, `parse_error`, `walk_error`) | `note`: no mechanical fix; one-line explanation |

## Non-goals

- No `--write`, no interactive apply, no prompts (user decision: print-only).
- No loosening of `disallowed`/`disallowedPaths` under any circumstance.
- No boundary-violation refactoring proposals (import graphs are not structlint's fix to make).
- Not a replacement for `validate` in CI gates.

## Backward compatibility

- Purely additive command; `validate` untouched.
- The JSON output is a **versioned contract from day one** (`"version": 1`); any breaking change to the shape bumps the version. Depends on specs 005 (engine/Tree) and 009 (`internal/infer` generalizer) being merged first.

## Design

### Files

- `internal/cli/suggest.go` — flag parsing, config discovery (same path as `validate`), output rendering.
- `internal/suggest/suggest.go` — `Analyze(cfg *config.Config, configPath string, tree *validator.Tree, violations []validator.Violation) (*Report, error)`.
- `internal/suggest/proposals.go` — per-code mapping table; types `Proposal{Kind, Section, Value, From, To, Command, Path, Reason, Paths}`.
- `internal/suggest/configdiff.go` — unified-diff builder (below).

### JSON contract (v1)

```json
{
  "version": 1,
  "configPath": ".structlint.yaml",
  "proposals": [
    {
      "kind": "config_add",
      "section": "dir_structure.allowedPaths",
      "value": "skills/**",
      "reason": "unallowed_directory: skills/ exists but is not in allowedPaths",
      "paths": ["skills", "skills/structlint"]
    },
    {
      "kind": "move",
      "from": "internal/util/parser_test.go",
      "to": "test/parser_test.go",
      "command": "git mv internal/util/parser_test.go test/parser_test.go",
      "reason": "placement_violation: tests must live under test/ (rule: tests-location)",
      "paths": ["internal/util/parser_test.go"]
    },
    {
      "kind": "create",
      "path": "README.md",
      "reason": "missing_required_file: README.md is required",
      "paths": ["README.md"]
    }
  ],
  "configDiff": "--- a/.structlint.yaml\n+++ b/.structlint.yaml\n@@ -6,6 +6,7 @@\n     - \"internal/**\"\n+    - \"skills/**\"\n"
}
```

`kind ∈ {config_add, move, create, note}`; `note` carries only `reason` + `paths`. Text format renders the same data as sections (Config additions / Moves / Creates / Review) with the diff last.

### configDiff construction

Built by **line-level insertion into the original config text** — locate the target section's list in the raw file, insert `- "<value>"` lines matching the neighboring indentation, and diff original vs. modified. Never re-marshal the YAML: re-marshal would destroy comments, ordering, and quoting, making the diff unappliable to the user's real file. The contract is that `patch -p1` (or an agent applying the hunks) against the working tree succeeds and a subsequent `validate` no longer reports the `config_add`-mapped violations. Proposals are deduped (many files, one `*.ext` entry) and sorted for deterministic output.

## Implementation steps

1. `internal/suggest/proposals.go` — Proposal types + code→kind mapping table (compile-time coverage of spec 004's registry).
2. `internal/suggest/suggest.go` — `Analyze` wiring engine output through the mapping, reusing `internal/infer` generalizers for `config_add` values.
3. `internal/suggest/configdiff.go` — line-level insertion + unified diff.
4. `internal/cli/suggest.go` + registration in `internal/app/app.go`; text and JSON renderers.
5. `test/suggest_test.go` — table per code + diff round-trip.
6. Docs: `docs/user/cli-reference.md` section + JSON contract documented for agent consumers.

## Checklist

- [ ] `internal/suggest/proposals.go` — types + full code mapping (disallowed_* → note, never loosen)
- [ ] `internal/suggest/suggest.go` — Analyze over engine + infer generalizers
- [ ] `internal/suggest/configdiff.go` — text-insertion unified diff
- [ ] `internal/cli/suggest.go` + `internal/app/app.go` registration
- [ ] `test/suggest_test.go` — per-code table, JSON contract, diff-apply round-trip, exit codes
- [ ] docs: `cli-reference.md` + JSON v1 contract
- [ ] self-validation: `bin/structlint validate` still passes (no new top-level dirs in this spec)

## Tests

`test/suggest_test.go` (binary-based via `buildBinary`):

- `TestSuggest_PerCodeProposals` — table over fixture trees triggering each mapped code; assert text output section and, in `--format json`, parsed `kind`/`section`/`value`/`from`/`to`/`path` per row.
- `TestSuggest_DisallowedNeverLoosened` — fixture hits `disallowed_file_pattern` (`.env.local`); assert proposal kind is `note`, and `configDiff` contains no change to `disallowed`.
- `TestSuggest_ConfigDiffAppliesAndValidatePasses` — the round-trip property: run `suggest --format json`, pipe `configDiff` through `patch`, re-run `validate`, assert the `config_add`-mapped violations are gone (exit 0 for config-only fixtures).
- `TestSuggest_ConfigDiffPreservesCommentsAndOrder` — config with comments/odd spacing; apply diff; assert untouched lines byte-identical.
- `TestSuggest_ExitZeroWithProposals` — violating tree → exit 0 with non-empty proposals.
- `TestSuggest_ExitOneOnOperationalError` — no config file → exit 1.
- `TestSuggest_JSONContractVersion` — `version == 1` and required top-level keys present.
- `TestSuggest_MoveEmitsGitMv` — placement fixture; assert `command` starts with `git mv ` and from/to match the rule.

## Verification

- `go test -race ./...`
- `make build`; in a scratch repo with a deliberately stale config: `./bin/structlint suggest`, apply the printed diff with `patch`, run the printed `git mv` commands, `./bin/structlint validate` → exit 0.
- `./bin/structlint suggest --format json | jq .` parses clean.
