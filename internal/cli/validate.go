package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/AxeForging/structlint/internal/logging"
	"github.com/AxeForging/structlint/internal/validator"
	"github.com/urfave/cli/v3"
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
			&cli.StringFlag{
				Name:    "format",
				Usage:   "output format: text|json|sarif|github",
				Value:   "text",
				Sources: cli.EnvVars("STRUCTLINT_FORMAT"),
			},
			&cli.StringFlag{
				Name:    "baseline",
				Usage:   "JSON report with known violations to suppress",
				Sources: cli.EnvVars("STRUCTLINT_BASELINE"),
			},
			&cli.BoolFlag{
				Name:    "changed-only",
				Usage:   "only validate changed files from git diff against HEAD",
				Sources: cli.EnvVars("STRUCTLINT_CHANGED_ONLY"),
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
			if cmd.Bool("changed-only") {
				v.LoadChangedPaths(path)
			}
			v.ValidateDirStructure(path)
			v.ValidateFileNaming(path)
			v.ValidateRequiredPaths(path)
			v.ValidateRequiredFiles(path)
			v.ValidatePlacement(path)
			v.ValidateRequiredGroups(path)
			v.ValidateBoundaries(path)

			if baseline := cmd.String("baseline"); baseline != "" {
				if err := v.ApplyBaseline(baseline); err != nil {
					return err
				}
			}

			// Save JSON report if requested
			jsonOutput := cmd.String("json-output")
			if jsonOutput != "" {
				if err := v.SaveJSONReport(jsonOutput); err != nil {
					return err
				}
			}

			switch format := cmd.String("format"); format {
			case "text", "":
				v.PrintSummary()
			case "json":
				if err := v.PrintJSONReport(); err != nil {
					return err
				}
			case "sarif":
				if err := v.PrintSARIFReport(); err != nil {
					return err
				}
			case "github":
				v.PrintGitHubAnnotations()
			default:
				return fmt.Errorf("unknown output format: %s", format)
			}

			// Return error if validation failed
			if len(v.Errors) > 0 {
				return fmt.Errorf("validation failed with %d errors", len(v.Errors))
			}

			return nil
		},
	}
}
