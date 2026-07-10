# Specs — structlint improvement roadmap

Four pillars, phased into small PRs. One spec per PR; a spec must be reviewed (or explicitly waived) before implementation. Everything stays backward compatible: existing config keys, flags, and exit codes are untouched.

| # | Spec | Pillar | Status | Depends on |
|---|------|--------|--------|------------|
| 001 | [`--staged` validation mode](001-staged-validation.md) | A — Adoption/DX | PR [#8](https://github.com/AxeForging/structlint/pull/8) | — |
| 002 | [`hook install`](002-hook-install.md) | A — Adoption/DX | PR [#10](https://github.com/AxeForging/structlint/pull/10) | 001 |
| 003 | [pre-commit packaging](003-pre-commit-packaging.md) | A — Adoption/DX | draft | 001 |
| 004 | [summary by code](004-summary-by-code.md) | D — Internals | draft | — |
| 005 | [rule engine + single walk](005-rule-engine.md) | D — Internals | draft | 004 |
| 006 | [config discovery](006-config-discovery.md) | D — Internals | draft | — |
| 007 | [`extends` + presets](007-extends-presets.md) | D — Internals | draft | — |
| 008 | [JSON Schema](008-json-schema.md) | D — Internals | draft | — |
| 009 | [`init --infer`](009-init-infer.md) | B — Suggest | draft | 005 |
| 010 | [`suggest`](010-suggest.md) | B — Suggest | draft | 005, 009 |
| 011 | [agent skill + frozen codes](011-agent-skill.md) | C — Agent surface | draft | 004, 010 |

**Wave order:** 1 (001–003) → 2 (004 first, then 005, then 006–008 in any order) → 3 (009 → 010) → 4 (011).

**Cross-cutting rule:** this repo self-validates via lefthook (`bin/structlint validate --silent`). Any PR adding a new top-level directory (`skills/`, `schema/`) must add it to `.structlint.yaml` `allowedPaths` in the same PR, or CI fails on the PR itself.

Specs 001 and 002 live on their feature branches until those PRs merge; the links above resolve once they land on `main`.
