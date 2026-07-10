# CI and Self-Validation Hardening

## Context

Structlint's current `main` branch is substantially ahead of the latest `v0.5.0`
release, but its release workflow is manual and its pull-request checks do not
fully reproduce the supported development toolchain. GitHub Actions still use Go
1.24 even though the project pins Go 1.25, the lint configuration is not
compatible with golangci-lint v2, and the PR self-validation action resolves the
latest published Structlint binary instead of validating with the code under
review.

The repository's `.structlint.yaml` also checks broad path and naming rules but
does not enforce the package boundaries, required groups, or placement guarantees
introduced by recent Structlint features. Contributors and maintainers are
affected because a green pull request may not prove that the candidate source,
release packaging, and architectural policy all agree.

## Requirements

1. The Go module, documentation examples, and all Go GitHub Actions workflows
   MUST use Go 1.25.
2. Pull requests MUST run the race-enabled Go test suite and build the real
   `structlint` binary used for self-validation.
3. Pull requests MUST fail when Go modules are untidy or Go source is not
   formatted according to the repository's configured formatter.
4. Pull requests MUST run a reproducibly pinned golangci-lint version with a
   configuration compatible with that major version.
5. Pull requests MUST validate the repository using a binary built from the pull
   request source, not the latest published release.
6. Pull requests MUST validate GoReleaser configuration or perform an equivalent
   non-publishing snapshot/package check.
7. PR-title validation SHOULD use AxeForging's shared action pinned to an immutable
   commit SHA, following the established Gauntlet pattern.
8. Existing advisory AI review behavior MUST remain non-blocking.
9. `.structlint.yaml` MUST enforce meaningful package dependency boundaries for
   orchestration, leaf, validation, and support packages.
10. `.structlint.yaml` MUST enforce required project groups and placement rules
    that protect command entry points, presets, schemas, tests, and shipped skills
    where supported by the current configuration schema.
11. The stronger self-policy MUST be validated by the locally built binary in CI
    and by a regression test that invokes a built binary where practical.
12. The manual release workflow MUST remain manually invokable and MUST use Go
    1.25; publishing a release is out of scope for this change.

## Non-goals

- Automatically publishing a release from every merge to `main`.
- Running the manual release workflow or creating `v0.6.0` as part of this change.
- Changing Structlint's CLI, validation semantics, or public configuration schema.
- Migrating every GitHub Action in the repository to a shared reusable workflow.
- Adding `extends: go-standard` before the self-hosting bootstrap path can safely
  support the `v0.6.0` version pragma.

## Design

Update the test, PR, and release workflows to use Go 1.25. Consolidate the
blocking pull-request quality gates around a source checkout that runs module
tidiness verification, formatting verification, the race suite, a real binary
build, local-binary self-validation, lint, and GoReleaser validation. Pin tools
and shared actions so the same revision is used until deliberately upgraded.

Replace the inline PR-title shell script with the immutable shared
`pr-title-lint` action revision already exercised by Gauntlet. Keep Reviewforge
advisory and non-blocking.

Expand `.structlint.yaml` using existing supported rule types. Boundary rules
will describe the current dependency direction rather than forcing a speculative
refactor: `app` orchestrates `cli`; `cli` may use feature packages; low-level
packages cannot import orchestration packages; and leaf/support packages remain
independent. Required-group and placement rules will assert stable repository
contracts without matching generated output or deliberately invalid fixtures.

The PR workflow will invoke the binary built from the checked-out commit directly.
This avoids the current `AxeForging/structlint@main` behavior, which resolves the
latest published release and therefore cannot prove that changes to the action or
validator under review work together.

### Alternatives considered

- **Release first, then strengthen self-validation:** rejected because it would
  publish before the release gates are reproducible.
- **Keep using `AxeForging/structlint@main` for dogfooding:** rejected because the
  composite action downloads the latest release when referenced as `main`.
- **Adopt `extends: go-standard` immediately:** deferred because released v0.5.0
  cannot parse `extends`; this should follow the v0.6.0 bootstrap release.
- **Automatically release on every merge:** rejected because the requested and
  existing release model is deliberately manual.

## Data & API changes

There are no runtime data, API, CLI, or schema changes. Changes are limited to
GitHub Actions configuration, lint configuration, and repository-local
`.structlint.yaml` policy.

## Test plan

1. Verify every workflow Go setup specifies Go 1.25.
2. Run `go test -race ./...` with Go 1.25.
3. Build `./cmd/structlint` and use that binary to validate `.structlint.yaml`.
4. Run the pinned golangci-lint version against its updated configuration.
5. Verify `go mod tidy` produces no diff.
6. Verify the configured formatter produces no diff.
7. Run GoReleaser's configuration check and/or snapshot build without publishing.
8. Add or update tests that build the binary and prove the stronger self-policy
   accepts the repository and rejects representative boundary/policy violations.
9. Inspect the resulting workflow YAML and, after push, confirm all `main` checks
   complete successfully.

## Rollout

Land the workflow, lint configuration, tests, and `.structlint.yaml` changes as one
coherent commit on `main`. Observe the resulting push workflow. If a new gate is
incorrect, revert the commit; the manual release workflow remains available and
no release/tag is created by this rollout.

After the checks are green, manually dispatch the release workflow for `v0.6.0`.
A follow-up may migrate the self-policy to `extends: go-standard` with the required
minimum-version pragma once v0.6.0 is published.
