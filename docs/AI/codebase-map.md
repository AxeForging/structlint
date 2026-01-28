# Codebase Map

## Directory Structure

```
structlint/
├── cmd/structlint/main.go      # CLI entry point (minimal, calls app.New())
├── main.go                     # Alternative entry (backward compat)
├── internal/                   # Private packages
│   ├── app/app.go             # Constructs root CLI command
│   ├── build/info.go          # Version/commit info (LDFLAGS injected)
│   ├── cli/                   # CLI commands
│   │   ├── root.go            # Root command, global flags, setup
│   │   ├── validate.go        # validate command implementation
│   │   ├── version.go         # version command
│   │   └── completion.go      # shell completion
│   ├── config/config.go       # Config struct and loading
│   ├── logging/logging.go     # slog-based logging setup
│   └── validator/             # Core validation
│       ├── validator.go       # ValidateDirStructure, ValidateFileNaming
│       ├── types.go           # JSONReport struct
│       └── summary.go         # PrintSummary, result formatting
├── test/                      # Integration tests
│   ├── helpers_test.go        # buildBinary, runBinary helpers
│   ├── integration_test.go    # CLI integration tests
│   ├── project_validation_test.go
│   ├── self_validation_test.go
│   ├── required_validation_test.go
│   ├── performance_test.go
│   └── smoke_test.go
├── docs/
│   ├── user/                  # User documentation
│   └── AI/                    # AI context documentation
├── .github/workflows/         # CI/CD
│   ├── test.yml              # Test on push/PR
│   └── release.yml           # Manual release workflow
├── .structlint.yaml          # Self-validation config
├── Makefile                  # Build automation
├── go.mod                    # Go module (github.com/AxeForging/structlint)
└── go.sum
```

## Key Files Explained

### cmd/structlint/main.go
Minimal entry point. Just calls `app.New().Run(ctx, os.Args)`.

### internal/app/app.go
Constructs the urfave/cli root command with all subcommands.

### internal/cli/root.go
- Defines global flags (--config, --log-level, --no-color)
- Sets up logging based on flags
- Loads config into context

### internal/cli/validate.go
- Main validation logic
- Calls validator functions
- Handles JSON output
- Returns exit code based on results

### internal/config/config.go
```go
type Config struct {
    DirStructure      DirStructureConfig
    FileNamingPattern FileNamingConfig
    Ignore            []string
}
```
Handles YAML/JSON parsing and file discovery.

### internal/validator/validator.go
Core validation functions:
- `ValidateDirStructure()` - Walks dirs, checks allowedPaths/disallowedPaths
- `ValidateFileNaming()` - Checks file patterns
- `ValidateRequiredPaths()` - Ensures required dirs exist
- `ValidateRequiredFiles()` - Ensures required files exist

Uses `github.com/gobwas/glob` for pattern matching.

### internal/validator/types.go
```go
type JSONReport struct {
    Successes int
    Failures  int
    Errors    []string
    Summary   ValidationSummary
}
```

### test/helpers_test.go
Test utilities:
- `buildBinary(t)` - Compiles CLI binary once per test run
- `runBinary(t, bin, args...)` - Executes binary
- `createTestProject(t, files, config)` - Sets up temp project

## Dependencies

- `github.com/urfave/cli/v3` - CLI framework
- `github.com/gobwas/glob` - Glob pattern matching
- `gopkg.in/yaml.v2` - YAML parsing

## Data Flow

```
User runs: structlint validate --config .structlint.yaml

1. main.go → app.New() → cli.Root()
2. cli/root.go: Parse global flags, setup logging
3. cli/root.go: Load config from file
4. cli/validate.go: Call validator functions
5. validator/validator.go: Walk filesystem, check patterns
6. cli/validate.go: Print summary, write JSON if requested
7. Return exit code (0=pass, 1=fail)
```
