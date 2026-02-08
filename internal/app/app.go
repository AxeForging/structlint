package app

import (
	"context"

	clilib "github.com/AxeForging/structlint/internal/cli"
	"github.com/urfave/cli/v3"
)

// New constructs the root command for the application.
// Keep all cross-cutting concerns (global flags, before/after hooks) here.
func New() *cli.Command {
	validateCmd := clilib.NewValidateCmd()

	return &cli.Command{
		Name:  "structlint",
		Usage: "A tool for validating directory structure and file naming patterns",
		// Default action: run validate when no subcommand is given
		Action: validateCmd.Action,
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			return clilib.Setup(ctx, cmd) // initialize logging/config
		},
		Commands: []*cli.Command{
			validateCmd,
			clilib.NewInitCmd(),
			clilib.NewVersionCmd(),
			clilib.NewCompletionCmd(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Usage:   "path to the configuration file",
				Value:   ".structlint.yaml",
				Sources: cli.EnvVars("STRUCTLINT_CONFIG"),
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "logging level: debug|info|warn|error",
				Value:   "info",
				Sources: cli.EnvVars("STRUCTLINT_LOG_LEVEL"),
			},
			&cli.BoolFlag{
				Name:    "no-color",
				Usage:   "disable colored output",
				Sources: cli.EnvVars("STRUCTLINT_NO_COLOR"),
			},
			// Legacy flags for backward compatibility
			&cli.StringFlag{
				Name:    "json-output",
				Usage:   "path to save the JSON report",
				Sources: cli.EnvVars("STRUCTLINT_JSON_OUTPUT"),
			},
			&cli.BoolFlag{
				Name:    "silent",
				Usage:   "suppress all output except for the JSON report",
				Sources: cli.EnvVars("STRUCTLINT_SILENT"),
			},
		},
	}
}
