package main

import (
	"context"
	"os"

	"github.com/AxeForging/structlint/internal/app"
)

// main.go serves as a backward compatibility entry point
// The new entry point is in cmd/structlint/main.go
func main() {
	if err := app.New().Run(context.Background(), os.Args); err != nil {
		// keep startup errors quiet and consistent
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
