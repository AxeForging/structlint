# Spec 003 — pre-commit framework packaging

## Problem

Users of the [pre-commit framework](https://pre-commit.com) cannot consume structlint the idiomatic way: pointing `.pre-commit-config.yaml` at this repo fails because there is no `.pre-commit-hooks.yaml` at any tag. This also leaves spec 002 half-finished — `structlint hook install --type pre-commit` appends a repos entry referencing this repo, but the framework refuses to install from a repo that doesn't declare its hooks. One small YAML file closes both gaps.

## Approach

Ship a `.pre-commit-hooks.yaml` (the leading dot is required by the framework) at the repo root declaring a single `structlint` hook that runs `structlint validate --staged --silent`, built by pre-commit itself via `language: golang`. Document the consumer snippet in the CI/CD guide and README.

Depends on spec 001 (`--staged` must exist for the entry to work) and complements spec 002 (its generated repos entry becomes installable once a tag containing this file is cut).

## Non-goals

- No additional hook ids (e.g. a full-tree variant, a `--format github` variant) — follow-up if asked.
- No `language: docker` / prebuilt-binary variants; `golang` is enough and needs no registry.
- Not touching `.pre-commit-config.yaml` merging — that is spec 002's job.

## Backward compatibility

- Purely additive: one new file, no code changes, no flag changes.
- Only tags cut **after** this spec are valid `rev` targets. Document the minimum usable rev in the README so users don't pin an older tag and get a confusing framework error.

## Design

### `.pre-commit-hooks.yaml` (repo root)

```yaml
- id: structlint
  name: structlint
  description: Validate project structure and file naming against .structlint.yaml
  entry: structlint validate --staged --silent
  language: golang
  pass_filenames: false
  always_run: true
```

Key choices:

- `entry` uses `--staged` (spec 001): the hook validates exactly what is being committed, matching the invocation spec 002 wires into lefthook and raw git hooks.
- `pass_filenames: false` — structlint computes its own staged set via `git diff --cached`; receiving filenames as argv would be ignored and is misleading.
- `always_run: true` — deletions and directory-level drift don't reliably surface through pre-commit's filename filtering, and required-file rules are global (spec 001 asymmetry). Always running keeps the hook honest; `--staged` keeps it fast.
- `language: golang` — pre-commit clones the pinned rev and runs `go install ./...` into an isolated GOPATH. Note: this repo has two `main` packages (root `main.go` and `cmd/structlint/`) that both delegate to `internal/app` and both yield a binary named `structlint`. Verification must confirm `go install ./...` succeeds in a clean GOPATH; if the Go toolchain rejects the duplicate binary name, consolidating the entry points becomes a prerequisite commit in this PR.

### Docs

- `docs/user/ci-cd-integration.md` — new "pre-commit framework" section:
  ```yaml
  repos:
    - repo: https://github.com/AxeForging/structlint
      rev: v0.X.Y  # first tag containing .pre-commit-hooks.yaml
      hooks:
        - id: structlint
  ```
- `README.md` — add the same snippet next to the existing hook/CI install options.

### Self-validation

`.pre-commit-hooks.yaml` lives at the root (allowed path `.`) and matches the allowed `*.yaml` file pattern — **no `.structlint.yaml` change needed**. The verification step still runs self-validation to prove it.

## Implementation steps

1. Add `.pre-commit-hooks.yaml` with the content above.
2. Add `test/precommit_packaging_test.go` (see Tests).
3. Add the pre-commit framework section to `docs/user/ci-cd-integration.md` and the README snippet, including the minimum-rev note.

## Checklist

- [ ] `.pre-commit-hooks.yaml` added at repo root
- [ ] `test/precommit_packaging_test.go` — parse test + flag-drift test
- [ ] `docs/user/ci-cd-integration.md` pre-commit section
- [ ] `README.md` consumer snippet + minimum-rev note
- [ ] Clean-GOPATH `go install ./...` verified (two-main-package risk)
- [ ] `make build && ./bin/structlint validate` self-validation passes (no config change expected)

## Tests

`test/precommit_packaging_test.go` (binary-based, per team convention):

- `TestPreCommitHooks_ParsesWithRequiredKeys` — read `.pre-commit-hooks.yaml` from the repo root, unmarshal with `gopkg.in/yaml.v3` into a hook list, assert exactly one hook with `id: structlint`, `language: golang`, `pass_filenames: false`, `always_run: true`, and the exact `entry` string.
- `TestPreCommitHooks_EntryFlagsExistOnBinary` — build the binary via `buildBinary(t)`, run `validate --help`, assert every flag referenced by `entry` (`--staged`, `--silent`) appears in the help output. This is the drift guard: if a future rename breaks the packaged entry, this test fails instead of every consumer's commit.
- `TestPreCommitHooks_EntryFailsOnStagedViolation` — reuse the `initGitRepo` helper from spec 001: stage a forbidden file, run the exact `entry` command (binary + args) in the repo, expect exit 1; unstage, expect exit 0. Proves the packaged invocation works end-to-end without the framework.

## Verification

- `go test -race ./...`
- `GOBIN=$(mktemp -d) go install ./...` from a clean checkout — must succeed and produce a working `structlint` binary (the framework runs exactly this).
- Manual: in a scratch git repo, `pre-commit try-repo /path/to/structlint structlint --verbose` with a staged violation — expect failure; unstage — expect pass.
- `make build && ./bin/structlint validate` (self-dogfood, no config change needed).
