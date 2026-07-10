# Spec 004 — summary grouping keyed on `Violation.Code`

## Problem

`GroupViolationsByType` (`internal/validator/summary.go`) classifies violations by **string-matching the message text** (`strings.Contains(err, "Disallowed directory found:")` …). Two problems:

1. It only knows 6 of the 13 violation codes. Everything from placement, required-group, boundary, parse and walk failures collapses into an undifferentiated `other` bucket with the description "Other validation errors" — exactly the rules users most need summarized.
2. It couples the summary to human-facing message wording. Rewording a message (or a rule emitting a slightly different phrase) silently reclassifies violations. The structured `Violation.Code` field already exists and is emitted by every rule — the summary just ignores it.

## Approach

Rewrite `GroupViolationsByType` to iterate `v.Violations` (not `v.Errors`) and group on `Violation.Code`. Introduce one canonical, exported `code → description` map covering all 13 codes. This map becomes the single registry of violation codes that specs 010 (`suggest`) and 011 (SKILL.md / `violation-codes.md`) reuse — codes are frozen/append-only from here on.

This is safe because `addViolation` appends to `Violations` and `Errors` in lockstep, and `ApplyBaseline` filters both in lockstep — the two slices are always parallel.

## Non-goals

- No change to the text summary format, JSON report shape, exit codes, or `ViolationSummary` struct.
- No new codes, no renames — this spec freezes the existing 13.
- Not building the `suggest` mapping (spec 010) — only the shared code registry it will key on.

## Backward compatibility

- `ViolationSummary`/`ValidationSummary` JSON shapes unchanged; `PrintGroupedSummary` layout unchanged.
- Grouping **improves** for the 7 codes previously lumped into `other`: they now appear as named groups with real descriptions. The 6 already-recognized codes keep their exact current description strings, so existing output for those groups is byte-identical.
- `other` remains as the fallback for any unknown code (defensive; nothing emits one today).
- Tie-ordering fix: groups are sorted by count descending as today, with a new deterministic tie-break on code (ascending). Today equal-count groups appear in random map order — fixing this is required groundwork for spec 005's golden tests.

## Design

### New file `internal/validator/codes.go`

```go
// CodeDescriptions is the canonical registry of violation codes.
// Codes are frozen and append-only; specs 010/011 build on this map.
var CodeDescriptions = map[string]string{
    "disallowed_directory":         "Directories that are explicitly disallowed",
    "unallowed_directory":          "Directories not in the allowed list",
    "disallowed_file_pattern":      "Files matching disallowed naming patterns",
    "unallowed_file_pattern":       "Files not matching any allowed naming pattern",
    "missing_required_directory":   "Required directories that are missing",
    "missing_required_file":        "Required file patterns that are missing",
    "placement_violation":          "Files placed outside their required directories",
    "missing_required_group":       "Required groups with no matching file",
    "missing_required_group_match": "Required group patterns matching no directories",
    "missing_group_file":           "Directories missing files required by their group",
    "boundary_violation":           "Files importing paths forbidden by boundary rules",
    "parse_error":                  "Source files whose imports could not be parsed",
    "walk_error":                   "Filesystem errors encountered during validation",
}

// DescribeCode returns the description for a code, falling back to the
// legacy "Other validation errors" for unknown codes.
func DescribeCode(code string) string
```

The first six descriptions are copied **verbatim** from the current `switch` in `summary.go` so recognized groups render identically.

### `internal/validator/summary.go` rewrite

- `GroupViolationsByType` builds `map[code][]string` from `for _, violation := range v.Violations`, appending `violation.Message` (same example strings as today). Empty/unknown codes map to `"other"`.
- Description comes from `DescribeCode`; the old `strings.Contains` chain and the description `switch` are deleted.
- Sort: count descending, then `Type` ascending on ties (deterministic).
- `getExamples`, `PrintGroupedSummary`, `GetValidationSummary` unchanged.

### CLI behavior

No flag changes. `validate --group-violations` (default) and `--format json` (summary block) simply produce better-classified groups.

## Implementation steps

1. Add `internal/validator/codes.go` with `CodeDescriptions` + `DescribeCode`, and a package-level completeness test pinning all 13 keys.
2. Rewrite `GroupViolationsByType` to iterate `v.Violations` keyed on `.Code` with the deterministic tie-break; delete the string-matching chain.
3. Add `test/summary_by_code_test.go` binary tests (see Tests).

## Checklist

- [ ] `internal/validator/codes.go` with all 13 codes + `DescribeCode` fallback
- [ ] `internal/validator/codes_test.go` completeness test (exact 13 keys, no more, no fewer)
- [ ] `GroupViolationsByType` iterates `Violations` by `.Code`; string-matching removed
- [ ] Deterministic tie-break sort
- [ ] `test/summary_by_code_test.go` binary tests
- [ ] `docs/user/reports` docs mention grouping is code-based (touch `docs/user/cli-reference.md` if it describes grouping)
- [ ] Self-validation passes (`make build && ./bin/structlint validate`) — no `.structlint.yaml` change expected

## Tests

`test/summary_by_code_test.go` (binary-based, per team convention — build via `buildBinary`, drive fixtures with `createTestProject`):

- `TestSummary_PlacementViolationsGetOwnGroup` — fixture with a `placement` rule violated twice; run `validate`; assert the grouped summary shows "Files placed outside their required directories (2 violations)" and NOT "Other validation errors".
- `TestSummary_BoundaryAndParseGroupedSeparately` — fixture triggering one `boundary_violation` and one `parse_error` (unparseable `.go` file matched by a boundary `from`); assert both descriptions appear as distinct groups.
- `TestSummary_LegacyGroupsUnchanged` — fixture triggering `disallowed_directory`, `unallowed_file_pattern`, `missing_required_file`; run `validate --format json`; assert `summary.violations[].type` equals the code strings and `description` equals the exact legacy strings.
- `TestSummary_TieOrderDeterministic` — fixture producing two groups with equal counts; run `validate` twice; assert identical stdout bytes.

Package-level (`internal/validator/codes_test.go`):

- `TestCodeDescriptions_CoversAllEmittedCodes` — asserts the map has exactly the 13 expected keys; grep-proof against a rule adding a code without registering it (spec 011's `violation-codes.md` test builds on this).

## Verification

- `go test -race ./...` (root `validator_test.go` must still pass untouched).
- `make build && ./bin/structlint validate` — self-dogfood; grouped output identical to before on this repo (it only emits already-recognized codes when violations are introduced).
- Manual: temporarily add a `placement` rule to a scratch config, violate it, confirm the summary names the group instead of "Other validation errors".
