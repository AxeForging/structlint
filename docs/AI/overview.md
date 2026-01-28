# structlint - AI Context Overview

## What is structlint?

structlint is a CLI tool written in Go that validates directory structures and file naming patterns. It helps enforce consistent project organization.

## Key Concepts

1. **Configuration-driven** - All rules defined in `.structlint.yaml`
2. **Glob patterns** - Uses glob syntax for path matching
3. **Three rule types**:
   - `dir_structure` - Directory validation
   - `file_naming_pattern` - File validation
   - `ignore` - Paths to skip

## Architecture

```
cmd/structlint/main.go     Entry point
internal/
  app/app.go               Root command construction
  cli/
    root.go                CLI setup, global flags
    validate.go            Validate command
    version.go             Version command
    completion.go          Shell completion
  config/config.go         Config loading (YAML/JSON)
  validator/
    validator.go           Core validation logic
    types.go               JSONReport struct
    summary.go             Result printing
  build/info.go            Version info (LDFLAGS)
  logging/logging.go       Structured logging
test/                      Integration tests
```

## How Validation Works

1. Load config from file
2. Walk filesystem from target path
3. For each directory: check against `allowedPaths` and `disallowedPaths`
4. For each file: check against file naming patterns
5. Check `requiredPaths` and `required` files exist
6. Skip anything in `ignore` list
7. Return violations

## Common Tasks

### Adding a new CLI flag

Edit `internal/cli/validate.go`, add to `Flags` slice:

```go
&cli.StringFlag{
    Name:  "new-flag",
    Usage: "Description",
},
```

### Adding a new validation rule type

1. Add field to `Config` struct in `internal/config/config.go`
2. Add validation logic in `internal/validator/validator.go`
3. Add tests in `test/`

### Modifying JSON report format

Edit `JSONReport` struct in `internal/validator/types.go`

## Testing

Tests use binary-first approach:
- `buildBinary(t)` compiles the CLI
- Tests run actual binary via `exec.Command`
- Use `t.TempDir()` for isolation

Run tests: `go test ./... -v`

## Build

```bash
make build        # Single platform
make build-all    # All platforms
```

Version info injected via LDFLAGS.
