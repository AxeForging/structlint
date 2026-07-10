# Spec 011 — Agent skill + frozen violation-code contract

## Problem

"AI tools placing files incorrectly" is structlint's headline use case, yet an agent encountering this repo has no machine-oriented entry point: it must reverse-engineer flags from `--help`, guess what each violation code means, and discover by trial that `suggest`'s JSON is the intended fix interface. Meanwhile nothing declares the violation codes stable — an agent (or CI parser) keying on `unallowed_directory` today has no promise it won't be renamed tomorrow. We need a shipped skill that teaches agents the full loop, and a document that freezes the contract the loop depends on.

## Approach

1. Ship `skills/structlint/SKILL.md` **in this repo at the repo root** — `skills/`, not `.claude/` — because it is a distributed artifact for *consumers* (installed into their agent setups or read from the release/repo), not local tooling config for structlint's own development.
2. Add `docs/user/violation-codes.md` declaring the 13 violation codes (spec 004's `map[code]description` registry) **frozen and append-only**: codes are never renamed or removed; new rules add new codes.
3. Self-validation lockstep (roadmap risk 5): `skills/` is a new top-level dir, so `.structlint.yaml` `allowedPaths` gains `skills/**` **in the same PR**.
4. Cross-link the skill and codes doc from `README.md` and `docs/AI/overview.md`.

## Non-goals

- No skill marketplace publishing / packaging automation (follow-up if wanted).
- No new CLI behavior — this spec is docs + one config line; the contracts it documents come from specs 001–010.
- No per-agent-vendor variants; one SKILL.md, plain Markdown + frontmatter.

## Backward compatibility

- Purely additive files. The codes doc *creates* a compatibility promise (append-only) rather than breaking one; spec 004's registry becomes the enforcement point.
- Depends on specs 001 (`--staged`), 002 (hook install), 003 (pre-commit), 007 (extends), 008 (schema), 009 (`--infer`), 010 (`suggest`) for the workflows it documents — land last (Wave 4).

## Design

### `skills/structlint/SKILL.md` outline

1. **Frontmatter** — `name: structlint`, `description` with trigger phrases ("validate project structure", "file in the wrong place", "structlint violation", "where should this file go", "enforce directory layout").
2. **When to run** — `validate --staged` in pre-commit (spec 001); CI formats (`--format json`, `--silent`, exit codes); legacy adoption path: `init --infer` to baseline (spec 009) + `--baseline` for grandfathering.
3. **Violation codes** — table of all 13 codes with, per code, the decision rule: *fix the tree or fix the config?* (e.g. `unallowed_directory` → usually config if the dir is intentional, tree if it's stray; `disallowed_file_pattern` → tree, the prohibition is deliberate; `placement_violation` → tree via `git mv`; `missing_required_*` → create the path; `parse_error`/`walk_error` → operational, fix the environment).
4. **Machine contracts** — `validate --format json` shape (`JSONReport`: `successes`, `failures`, `total_violations`, `errors`, `violations[]{code, severity, path, rule, message}`, `summary`); `suggest` JSON v1 (spec 010); exit semantics (`validate`: 0 clean / 1 violations; `suggest`: 0 even with proposals, 1 operational).
5. **The fix loop** — `structlint suggest --format json` → apply `configDiff` (patch/edit) and/or run `git mv` commands → `structlint validate` → repeat until exit 0.
6. **Setup recipes** — `hook install` (spec 002), `.pre-commit-config.yaml` (spec 003), GitHub Action snippet, `extends` presets (spec 007), yaml-language-server schema modeline (spec 008).
7. **Gotchas** — config parse is strict (unknown keys fail); `extends` requires a newer binary (old binaries strict-fail); `--changed-only` diffs HEAD vs `--staged` diffs the index — hooks want `--staged`; `ignore` (invisible) vs `disallowed` (violation) are different tools.

### `docs/user/violation-codes.md`

Header paragraph declaring the freeze: codes are a public contract, append-only, never renamed/removed. Then one entry per code: code, emitting rule/config section, meaning, typical fix — exactly covering spec 004's registry: `disallowed_directory`, `unallowed_directory`, `disallowed_file_pattern`, `unallowed_file_pattern`, `missing_required_directory`, `missing_required_file`, `placement_violation`, `missing_required_group`, `missing_required_group_match`, `missing_group_file`, `boundary_violation`, `parse_error`, `walk_error`.

### Config + cross-links

- `.structlint.yaml`: `dir_structure.allowedPaths` += `"skills/**"` (comment: shipped agent skill).
- `README.md`: short "For AI agents" section linking `skills/structlint/SKILL.md` and `docs/user/violation-codes.md`.
- `docs/AI/overview.md`: same links in the contributor-facing map.

## Implementation steps

1. Add `skills/structlint/SKILL.md` per the outline **and** add `skills/**` to `.structlint.yaml` `allowedPaths` (same commit — self-validation lockstep).
2. Add `docs/user/violation-codes.md` covering all 13 codes.
3. Add `test/skill_contract_test.go` (coverage + self-validation tests below).
4. Cross-link from `README.md` and `docs/AI/overview.md`.

## Checklist

- [ ] `skills/structlint/SKILL.md` — frontmatter, when-to-run, code table with fix-tree-or-config rules, machine contracts, fix loop, setup recipes, gotchas
- [ ] `.structlint.yaml` — `allowedPaths` gains `skills/**` (same PR as the new dir; roadmap risk 5)
- [ ] `docs/user/violation-codes.md` — all 13 codes, frozen/append-only declaration
- [ ] `test/skill_contract_test.go` — registry↔doc coverage + skill file assertions + self-validation
- [ ] `README.md` cross-links
- [ ] `docs/AI/overview.md` cross-links

## Tests

`test/skill_contract_test.go` (binary-based where a binary is exercised):

- `TestViolationCodesDoc_CoversRegistry` — the keystone: iterate spec 004's `map[code]description` (exported registry from `internal/validator`), assert every code appears as an entry in `docs/user/violation-codes.md`; fail with the missing code's name. Append-only by construction: adding a code without documenting it breaks the build.
- `TestViolationCodesDoc_NoUnknownCodes` — the reverse direction: every code-formatted entry in the doc exists in the registry (catches typos/renames in the doc).
- `TestSkillFile_ExistsWithFrontmatter` — `skills/structlint/SKILL.md` exists, starts with `---` frontmatter containing `name:` and `description:`.
- `TestSkillFile_MentionsAllCodes` — the SKILL.md code table covers all 13 registry codes (agents get the full decision table).
- `TestSelfValidation_AllowsSkillsDir` — build the binary, run `validate` against the repo root, assert exit 0 (proves the `.structlint.yaml` update landed with the new dir).

## Verification

- `go test -race ./...`
- `make build && ./bin/structlint validate` — must pass with `skills/` present (the lockstep check, also enforced by the lefthook pre-commit).
- Manual: point a Claude Code session at a violating scratch repo with the skill installed; confirm it runs the suggest → apply → re-validate loop unprompted from a "fix the structure" ask.
- Grep: every code in `docs/user/violation-codes.md` also appears in SKILL.md's table.
