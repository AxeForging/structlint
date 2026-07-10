package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/AxeForging/structlint/internal/hooks"
	"github.com/urfave/cli/v3"
)

// NewHookCmd builds the `structlint hook` command group.
func NewHookCmd() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "manage git hook integration",
		Commands: []*cli.Command{
			newHookInstallCmd(),
		},
	}
}

func newHookInstallCmd() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "install structlint into the repository's pre-commit chain",
		Description: "Auto-detects lefthook, pre-commit, or a raw git hook and merges " +
			"a structlint invocation without overwriting existing content. Idempotent.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "type",
				Usage: "force target: lefthook | pre-commit | git (default: auto-detect)",
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: "repository directory to install into",
				Value: ".",
			},
			&cli.BoolFlag{
				Name:  "dry-run",
				Usage: "print the resulting file without writing",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			dir := cmd.String("path")
			if dir == "" {
				dir = "."
			}
			t, err := chooseHookType(cmd, dir)
			if err != nil {
				return err
			}
			res, err := hooks.Install(dir, t, cmd.Bool("dry-run"))
			if err != nil {
				return err
			}
			return printHookResult(res, cmd.Bool("dry-run"))
		},
	}
}

func chooseHookType(cmd *cli.Command, dir string) (hooks.Type, error) {
	if raw := cmd.String("type"); raw != "" {
		switch hooks.Type(raw) {
		case hooks.TypeLefthook, hooks.TypePreCommit, hooks.TypeGit:
			return hooks.Type(raw), nil
		default:
			return "", fmt.Errorf("unknown --type %q; expected lefthook, pre-commit, or git", raw)
		}
	}
	t, err := hooks.Detect(dir)
	if err != nil && !errors.Is(err, hooks.ErrNoTargetDetected) {
		return "", err
	}
	return t, nil
}

func printHookResult(res hooks.Result, dryRun bool) error {
	switch res.Action {
	case hooks.ActionInstalled:
		if dryRun {
			fmt.Printf("[dry-run] would update %s (%s)\n", res.File, res.Type)
			if res.Preview != "" {
				fmt.Println("---")
				fmt.Print(res.Preview)
				fmt.Println("---")
			}
			return nil
		}
		fmt.Printf("installed structlint hook: %s (%s)\n", res.File, res.Type)
		return nil
	case hooks.ActionAlreadyInstalled:
		fmt.Printf("already installed: %s (%s)\n", res.File, res.Type)
		return nil
	case hooks.ActionRefused:
		fmt.Fprintln(os.Stderr, "structlint hook install refused:")
		fmt.Fprintln(os.Stderr, res.Reason)
		return errors.New("hook install refused; see message above")
	default:
		return fmt.Errorf("unknown hook install action: %s", res.Action)
	}
}
