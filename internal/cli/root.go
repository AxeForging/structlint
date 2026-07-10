package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/AxeForging/structlint/internal/logging"
	"github.com/urfave/cli/v3"
)

// Setup wires logging and config before any command runs.
func Setup(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	// Hydrate logging based on resolved flag value.
	level := "info"
	if v := cmd.String("log-level"); v != "" {
		level = v
	}
	noColor := cmd.Bool("no-color")

	lg, err := logging.New(level, noColor)
	if err != nil {
		return ctx, err
	}

	// Attach logger to context for downstream use.
	ctx = logging.With(ctx, lg)
	// Also set as slog default so callers without ctx access (e.g. helpers
	// invoked from within command Actions before context propagation) get
	// the CLI-formatted output.
	slog.SetDefault(lg)
	return ctx, nil
}

// Common error values for consistent exit codes.
var (
	ErrInvalidArgs = errors.New("invalid arguments")
)

// LoadConfigForContext loads the configuration file based on the CLI context.
// When --config was not explicitly set (flag or STRUCTLINT_CONFIG env),
// falls back to Discover() which walks upward from --path/cwd looking for
// .structlint.{yaml,yml,json}, stopping at the first .git-containing dir
// (inclusive) or the filesystem root. See spec 006.
func LoadConfigForContext(cmd *cli.Command) (*config.Config, error) {
	if cmd.IsSet("config") {
		return loadFromExplicitPath(cmd, cmd.String("config"))
	}

	// Try the start dir first, mirroring the legacy behavior when nothing
	// is set: a config right next to us wins with no log noise.
	start := cmd.String("path")
	if start == "" {
		start = "."
	}
	if info, err := os.Stat(".structlint.yaml"); err == nil && !info.IsDir() {
		return loadAndLog(cmd, ".structlint.yaml", false)
	}

	discovered, err := config.Discover(start)
	if err != nil {
		return nil, fmt.Errorf("discover config: %w", err)
	}
	if discovered == "" {
		return nil, fmt.Errorf("configuration file not found: searched upward from %s for %v (stopped at .git or filesystem root)\n\nRun 'structlint init' to generate a starter configuration", start, describeConfigNames())
	}
	return loadAndLog(cmd, discovered, true)
}

func loadFromExplicitPath(cmd *cli.Command, configPath string) (*config.Config, error) {
	if configPath == "" {
		configPath = ".structlint.yaml"
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s\n\nRun 'structlint init' to generate a starter configuration", configPath)
	}
	return loadAndLog(cmd, configPath, false)
}

func loadAndLog(cmd *cli.Command, path string, discovered bool) (*config.Config, error) {
	cfg, err := config.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	if lg := loggerFromCommand(cmd); lg != nil {
		if discovered {
			lg.Info(fmt.Sprintf("using config: %s (discovered)", path))
		} else {
			lg.Debug(fmt.Sprintf("using config: %s (explicit)", path))
		}
	}
	return cfg, nil
}

func loggerFromCommand(_ *cli.Command) *slog.Logger {
	// The Before hook attaches a logger to context; urfave/cli v3 doesn't
	// expose that ctx from cmd here, so callers reaching for the logger
	// go through slog.Default (which the logging package configures).
	return slog.Default()
}

func describeConfigNames() []string {
	return []string{".structlint.yaml", ".structlint.yml", ".structlint.json"}
}
