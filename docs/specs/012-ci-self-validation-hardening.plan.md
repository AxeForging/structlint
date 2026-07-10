# CI and Self-Validation Hardening Plan

- [x] Update GitHub workflows for Go 1.25, reproducible PR gates, local-source self-validation, and safer release validation (`.github/workflows/*.yml`).
- [x] Migrate and pin golangci-lint v2 configuration (`.golangci.yml`, workflow references).
- [x] Strengthen repository self-policy with architectural boundaries, required groups, placement rules, and schema metadata (`.structlint.yaml`).
- [x] Add meaningful built-binary regression coverage for the strengthened self-policy (`test/*_test.go`, fixtures where needed).
- [x] Run Go 1.25 race tests, formatting/tidiness checks, lint, local binary self-validation, and GoReleaser validation.
- [x] Review the final diff and commit using project conventions; push `main` and monitor GitHub checks as the rollout verification.

## Risks

- A self-policy rule can accidentally reject valid repository evolution; rules must encode stable architecture rather than incidental layout.
- Tool/action pin changes can diverge from the versions available on GitHub-hosted runners.
- Release validation must never publish or create a tag during pull-request checks.
- Rollback is a normal revert of the single hardening commit; the release workflow remains manual.

## Done means

- All testable requirements in `012-ci-self-validation-hardening.md` are implemented.
- Go 1.25 `go test -race ./...` passes.
- The pinned linter and formatting/tidiness checks pass without modifying tracked files.
- A binary built from the current source validates the strengthened `.structlint.yaml`.
- GoReleaser configuration/snapshot validation succeeds without publishing.
- The commit is pushed to `main` and the resulting GitHub checks complete successfully.
