# Contributing Guide for AI

## Code Style

- Go 1.24+
- Use `gofmt` and `goimports`
- Follow standard Go project layout
- Use `internal/` for private packages
- Structured logging with `log/slog`

## Adding a New Feature

### 1. New CLI Command

Create `internal/cli/newcmd.go`:

```go
package cli

import (
    "context"
    "github.com/urfave/cli/v3"
)

func newCommand() *cli.Command {
    return &cli.Command{
        Name:  "newcmd",
        Usage: "Description of command",
        Flags: []cli.Flag{
            // flags here
        },
        Action: func(ctx context.Context, cmd *cli.Command) error {
            // implementation
            return nil
        },
    }
}
```

Register in `internal/app/app.go`:

```go
Commands: []*cli.Command{
    cli.ValidateCommand(),
    cli.VersionCommand(),
    cli.CompletionCommand(),
    cli.NewCommand(),  // Add here
},
```

### 2. New Validation Rule

Add to config struct in `internal/config/config.go`:

```go
type Config struct {
    DirStructure      DirStructureConfig
    FileNamingPattern FileNamingConfig
    NewRule           NewRuleConfig  // Add new rule
    Ignore            []string
}

type NewRuleConfig struct {
    SomeOption string   `yaml:"someOption" json:"someOption"`
    Patterns   []string `yaml:"patterns" json:"patterns"`
}
```

Add validation in `internal/validator/validator.go`:

```go
func ValidateNewRule(cfg *config.Config, rootPath string) (int, int, []string) {
    var successes, failures int
    var errors []string
    // validation logic
    return successes, failures, errors
}
```

Call from `internal/cli/validate.go`.

### 3. New CLI Flag

Add to command's `Flags` slice:

```go
Flags: []cli.Flag{
    &cli.BoolFlag{
        Name:    "new-flag",
        Usage:   "Enable new feature",
        Sources: cli.EnvVars("STRUCTLINT_NEW_FLAG"),
    },
},
```

Access in action:

```go
if cmd.Bool("new-flag") {
    // handle flag
}
```

## Testing

### Writing Tests

Use binary-first approach:

```go
func TestNewFeature(t *testing.T) {
    bin := buildBinary(t)

    projectFiles := map[string]string{
        "file.go": "package main",
    }
    configContent := `...`

    projectDir := createTestProject(t, projectFiles, configContent)

    out, err := runBinaryInDir(t, bin, projectDir,
        "validate",
        "--config", ".structlint.yaml",
    )

    if err != nil {
        t.Errorf("Failed: %v\nOutput: %s", err, out)
    }
}
```

### Running Tests

```bash
go test ./... -v           # All tests
go test ./test/... -v      # Integration tests only
go test -run TestName      # Specific test
```

## Build System

### Makefile Targets

- `make build` - Build for current platform
- `make build-all` - Build for all platforms
- `make test` - Run tests
- `make lint` - Run linter
- `make clean` - Remove build artifacts

### Version Injection

Version info injected via LDFLAGS:

```go
// internal/build/info.go
var (
    Version = "dev"
    Commit  = "none"
    Date    = "unknown"
    BuiltBy = "local"
)
```

Set at build time:
```bash
go build -ldflags "-X .../build.Version=v1.0.0"
```

## Error Handling

- Return errors, don't panic
- Use structured logging for debug info
- Exit codes: 0=pass, 1=fail, 2=config error, 3=runtime error

## Pull Request Checklist

1. [ ] Tests pass: `go test ./...`
2. [ ] Lint passes: `make lint`
3. [ ] Self-validates: `make test-self`
4. [ ] Documentation updated if needed
5. [ ] Conventional commit message (feat/fix/docs/etc)
