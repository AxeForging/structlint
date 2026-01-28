package test

import (
	"context"
	"testing"

	"github.com/youngestaxe/structlint/internal/app"
)

func TestRootRuns(t *testing.T) {
	app := app.New()
	if err := app.Run(context.Background(), []string{"structlint", "version"}); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestValidateRuns(t *testing.T) {
	app := app.New()
	// This should fail gracefully if no config file exists
	err := app.Run(context.Background(), []string{"structlint", "validate", "--config", "nonexistent.yaml"})
	if err == nil {
		t.Fatal("expected error when config file doesn't exist")
	}
}
