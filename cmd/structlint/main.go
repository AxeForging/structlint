package main

import (
	"context"
	"os"

	"github.com/AxeForging/structlint/internal/app"
)

func main() {
	if err := app.New().Run(context.Background(), os.Args); err != nil {
		// keep startup errors quiet and consistent
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
