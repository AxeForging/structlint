package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/urfave/cli/v3"
	"github.com/AxeForging/structlint/internal/logging"
	"github.com/AxeForging/structlint/internal/validator"
)

// NewValidateCmd creates the main validation command.
func NewValidateCmd() *cli.Command {
	return &cli.Command{
		Name:  "validate",
		Usage: "validate directory structure and file naming patterns",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Usage:   "path to validate",
				Value:   ".",
				Sources: cli.EnvVars("STRUCTLINT_PATH"),
			},
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
			&cli.BoolFlag{
				Name:    "group-violations",
				Usage:   "group violations by type for better readability",
				Value:   true,
				Sources: cli.EnvVars("STRUCTLINT_GROUP_VIOLATIONS"),
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Usage:   "show all allowed files and directories (default: only show violations)",
				Sources: cli.EnvVars("STRUCTLINT_VERBOSE"),
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			// Load configuration
			config, err := LoadConfigForContext(cmd)
			if err != nil {
				return err
			}

			// Get logger from context
			logger, ok := ctx.Value(logging.LoggerKey()).(*slog.Logger)
			if !ok || logger == nil {
				logger = slog.Default()
			}

			// Create validator
			v := validator.New(config, logger)

			// Get flags
			v.Silent = cmd.Bool("silent")
			v.GroupViolations = cmd.Bool("group-violations")
			v.Verbose = cmd.Bool("verbose")

			// Run validations
			path := cmd.String("path")
			if path == "" {
				path = "."
			}
			v.ValidateDirStructure(path)
			v.ValidateFileNaming(path)
			v.ValidateRequiredPaths(path)
			v.ValidateRequiredFiles(path)
			v.PrintSummary()

			// Save JSON report if requested
			jsonOutput := cmd.String("json-output")
			if jsonOutput != "" {
				if err := v.SaveJSONReport(jsonOutput); err != nil {
					return err
				}
			}

			// Return error if validation failed
			if len(v.Errors) > 0 {
				return fmt.Errorf("validation failed with %d errors", len(v.Errors))
			}

			return nil
		},
	}
}
