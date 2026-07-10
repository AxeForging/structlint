package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/AxeForging/structlint/internal/suggest"
	"github.com/urfave/cli/v3"
)

// NewSuggestCmd builds the `structlint suggest` command.
func NewSuggestCmd() *cli.Command {
	return &cli.Command{
		Name:  "suggest",
		Usage: "propose config changes and file moves that would resolve current violations",
		Description: "Runs the same engine as validate, then maps each violation to a proposal. " +
			"Print-only — never writes; exit 0 even when proposals exist (advisory tool).",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Value: ".",
				Usage: "directory to analyze",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "text",
				Usage: "output format: text or json",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			cfg, err := LoadConfigForContext(cmd)
			if err != nil {
				return err
			}
			path := cmd.String("path")
			if path == "" {
				path = "."
			}

			// Discover the actual config path the same way LoadConfigForContext
			// did (so the diff header names the file the user sees).
			configPath := cmd.String("config")

			report, err := suggest.Analyze(cfg, configPath, path)
			if err != nil {
				return err
			}

			switch cmd.String("format") {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				if err := enc.Encode(report); err != nil {
					return err
				}
			default:
				renderText(report)
			}
			return nil
		},
	}
}

func renderText(report *suggest.Report) {
	if len(report.Proposals) == 0 {
		fmt.Println("No proposals — nothing structlint suggest can help with.")
		return
	}
	fmt.Printf("Suggestions for %s\n", report.ConfigPath)

	var (
		adds    []suggest.Proposal
		moves   []suggest.Proposal
		creates []suggest.Proposal
		notes   []suggest.Proposal
	)
	for _, p := range report.Proposals {
		switch p.Kind {
		case suggest.KindConfigAdd:
			adds = append(adds, p)
		case suggest.KindMove:
			moves = append(moves, p)
		case suggest.KindCreate:
			creates = append(creates, p)
		default:
			notes = append(notes, p)
		}
	}
	if len(adds) > 0 {
		fmt.Println("\n== Config additions ==")
		for _, p := range adds {
			fmt.Printf("  + %s: %q  (%s)\n", p.Section, p.Value, p.Reason)
		}
	}
	if len(moves) > 0 {
		fmt.Println("\n== File moves ==")
		for _, p := range moves {
			fmt.Printf("  %s\n    reason: %s\n", p.Command, p.Reason)
		}
	}
	if len(creates) > 0 {
		fmt.Println("\n== Create ==")
		for _, p := range creates {
			fmt.Printf("  touch %s   (%s)\n", p.Path, p.Reason)
		}
	}
	if len(notes) > 0 {
		fmt.Println("\n== Review ==")
		for _, p := range notes {
			fmt.Printf("  %s\n    paths: %v\n", p.Reason, p.Paths)
		}
	}
	if report.ConfigDiff != "" {
		fmt.Println("\n== Config diff (apply with `patch -p1`) ==")
		fmt.Print(report.ConfigDiff)
	}
}
