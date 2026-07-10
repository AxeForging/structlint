# CI/CD Integration

structlint is designed to work seamlessly in CI/CD pipelines.

## GitHub Actions

```yaml
name: Validate Structure

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25.x'

      - name: Install structlint
        run: go install github.com/AxeForging/structlint@latest

      - name: Validate structure
        run: structlint validate --config .structlint.yaml
```

### GitHub Annotations

Use annotation output when you want violations to appear inline on pull requests.

```yaml
      - name: Validate structure
        run: structlint validate --format github
```

### SARIF Upload

Use SARIF when your pipeline collects code-scanning reports.

```yaml
      - name: Validate structure
        run: structlint validate --format sarif > structlint.sarif

      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        if: always()
        with:
          sarif_file: structlint.sarif
```

### With JSON Report Artifact

```yaml
      - name: Validate structure
        run: structlint validate --json-output report.json

      - name: Upload report
        uses: actions/upload-artifact@v4
        if: always()
        with:
          name: structlint-report
          path: report.json
```

## GitLab CI

```yaml
structlint:
  image: golang:1.24
  stage: test
  script:
    - go install github.com/AxeForging/structlint@latest
    - structlint validate --config .structlint.yaml
  artifacts:
    when: always
    paths:
      - report.json
        expire_in: 1 week
```

## Baselines

Baselines let legacy repositories adopt structlint without blocking every existing violation. First, create a report from the current state:

```bash
structlint validate --json-output .structlint-baseline.json || true
```

Then fail only on new violations:

```bash
structlint validate --baseline .structlint-baseline.json
```

The baseline matches typed violations by `code`, `path`, and `rule`.

## Changed Files

For fast pull-request checks, validate only changed files:

```bash
structlint validate --changed-only
```

This uses `git diff --name-only --diff-filter=ACMRT HEAD`. Repository-wide requirements such as required paths still run, while file-oriented checks are limited to changed files.

## Jenkins

```groovy
pipeline {
    agent any
    stages {
        stage('Validate Structure') {
            steps {
                sh 'go install github.com/AxeForging/structlint@latest'
                sh 'structlint validate --json-output report.json'
            }
            post {
                always {
                    archiveArtifacts artifacts: 'report.json'
                }
            }
        }
    }
}
```

## Pre-commit Hook

### Auto-install (recommended)

```bash
structlint hook install
```

Auto-detects lefthook, pre-commit, or a raw git hook and merges a `structlint validate --staged --silent` invocation. Idempotent.

### pre-commit framework

structlint ships a `.pre-commit-hooks.yaml`, so consumers can pin it directly:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/AxeForging/structlint
    rev: v0.X.Y  # first tag containing .pre-commit-hooks.yaml
    hooks:
      - id: structlint
```

The hook runs `structlint validate --staged --silent` via `language: golang`, so pre-commit clones the pinned rev and installs the binary in an isolated environment — no local install required.

### lefthook

```yaml
# lefthook.yml
pre-commit:
  commands:
    structlint:
      run: structlint validate --staged --silent
```

### Raw `.pre-commit-config.yaml` (local)

```yaml
repos:
  - repo: local
    hooks:
      - id: structlint
        name: structlint
        entry: structlint validate --staged --silent
        language: system
        pass_filenames: false
        always_run: true
```

## Docker

```dockerfile
FROM golang:1.24-alpine AS builder
RUN go install github.com/AxeForging/structlint@latest

FROM alpine:latest
COPY --from=builder /go/bin/structlint /usr/local/bin/
ENTRYPOINT ["structlint"]
```

Usage:

```bash
docker run --rm -v $(pwd):/project -w /project structlint validate
```

## Makefile Integration

```makefile
.PHONY: lint-structure

lint-structure:
	@structlint validate --config .structlint.yaml

ci: lint test lint-structure build
```

## Best Practices

1. **Run early in the pipeline** - Structure validation is fast and catches issues before expensive builds

2. **Use typed output in CI** - Prefer `--format github`, `--format sarif`, or `--json-output`

3. **Generate reports** - Always generate JSON reports for debugging

4. **Cache the binary** - In GitHub Actions, Go install is cached automatically with `actions/setup-go`

5. **Fail fast** - Put structlint before tests to catch structure issues immediately
