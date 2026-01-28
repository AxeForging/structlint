package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// NewCompletionCmd provides dynamic shell completion scripts.
func NewCompletionCmd() *cli.Command {
	return &cli.Command{
		Name:  "completion",
		Usage: "generate shell completion scripts",
		Commands: []*cli.Command{
			BashCompleteCommand(),
			ZshCompleteCommand(),
			FishCompleteCommand(),
		},
	}
}

func BashCompleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "bash",
		Usage: "print bash completion script",
		Action: func(ctx context.Context, c *cli.Command) error {
			// urfave/cli v3 doesn't have the same completion system
			// For now, just print a basic message
			fmt.Println("# Bash completion not implemented in this version")
			return nil
		},
	}
}

func ZshCompleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "zsh",
		Usage: "print zsh completion script",
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Println("# Zsh completion not implemented in this version")
			return nil
		},
	}
}

func FishCompleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "fish",
		Usage: "print fish completion script",
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Println("# Fish completion not implemented in this version")
			return nil
		},
	}
}
