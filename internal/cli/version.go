package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
	"github.com/youngestaxe/structlint/internal/build"
)

// NewVersionCmd prints version metadata injected at build time via ldflags.
func NewVersionCmd() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "print version information",
		Action: func(ctx context.Context, c *cli.Command) error {
			fmt.Println(build.String())
			return nil
		},
	}
}
