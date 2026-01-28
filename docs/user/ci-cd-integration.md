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
          go-version: '1.24'

      - name: Install structlint
        run: go install github.com/AxeForging/structlint@latest

      - name: Validate structure
        run: structlint validate --config .structlint.yaml
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

Add to `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: local
    hooks:
      - id: structlint
        name: structlint
        entry: structlint validate
        language: system
        pass_filenames: false
        always_run: true
```

Or with lefthook (`.lefthook.yml`):

```yaml
pre-commit:
  commands:
    structlint:
      run: structlint validate --silent
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

2. **Use strict mode in CI** - Add `--strict` flag to fail on warnings

3. **Generate reports** - Always generate JSON reports for debugging

4. **Cache the binary** - In GitHub Actions, Go install is cached automatically with `actions/setup-go`

5. **Fail fast** - Put structlint before tests to catch structure issues immediately
