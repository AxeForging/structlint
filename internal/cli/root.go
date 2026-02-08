package cli

import (
	"context"
	"errors"
	"fmt"
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
	return ctx, nil
}

// Common error values for consistent exit codes.
var (
	ErrInvalidArgs = errors.New("invalid arguments")
)

// LoadConfigForContext loads the configuration file based on the CLI context.
func LoadConfigForContext(cmd *cli.Command) (*config.Config, error) {
	configPath := cmd.String("config")
	if configPath == "" {
		configPath = ".structlint.yaml"
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s\n\nRun 'structlint init' to generate a starter configuration", configPath)
	}

	return config.LoadConfig(configPath)
}
