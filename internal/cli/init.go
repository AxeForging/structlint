package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/AxeForging/structlint/internal/infer"
	"github.com/urfave/cli/v3"
)

// projectTypeToPreset maps `init --type` values to preset names in
// internal/config/presets/. Kept as a distinct map (instead of using the
// preset names directly as --type values) so the shorter --type flag is
// still ergonomic and the preset names remain the source of truth.
var projectTypeToPreset = map[string]string{
	"go":      "go-standard",
	"node":    "node-standard",
	"python":  "python-standard",
	"generic": "generic",
}

// projectTypeHeaders is the single-line header prepended to the preset
// content so the generated config file reads like a starter template
// instead of a bare preset dump.
var projectTypeHeaders = map[string]string{
	"go":      "# structlint configuration for Go projects\n",
	"node":    "# structlint configuration for Node.js projects\n",
	"python":  "# structlint configuration for Python projects\n",
	"generic": "# structlint configuration\n",
}

// NewInitCmd creates the init command for generating starter configs.
func NewInitCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "generate a starter .structlint.yaml configuration file",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "type",
				Usage: "project type: go, node, python, generic (auto-detected if omitted)",
			},
			&cli.BoolFlag{
				Name:  "infer",
				Usage: "generate config by inspecting the current tree instead of a template",
			},
			&cli.BoolFlag{
				Name:  "force",
				Usage: "overwrite existing configuration file",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			if cmd.Bool("infer") && cmd.IsSet("type") {
				return errors.New("--infer and --type are mutually exclusive")
			}

			configPath := cmd.Root().String("config")
			if configPath == "" {
				configPath = ".structlint.yaml"
			}

			if _, err := os.Stat(configPath); err == nil && !cmd.Bool("force") {
				return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath)
			}

			if cmd.Bool("infer") {
				data, err := infer.Generate(".")
				if err != nil {
					return fmt.Errorf("infer config: %w", err)
				}
				if err := os.WriteFile(configPath, data, 0o644); err != nil {
					return fmt.Errorf("failed to write configuration: %w", err)
				}
				fmt.Printf("Created %s from tree inspection\n", configPath)
				fmt.Println("Run 'structlint validate' to check your project structure.")
				return nil
			}

			projectType := cmd.String("type")
			if projectType == "" {
				projectType = detectProjectType(".")
			}

			data, err := renderProjectTemplate(projectType)
			if err != nil {
				return err
			}
			if err := os.WriteFile(configPath, data, 0o644); err != nil {
				return fmt.Errorf("failed to write configuration: %w", err)
			}

			fmt.Printf("Created %s for %s project\n", configPath, projectType)
			fmt.Println("Run 'structlint validate' to check your project structure.")
			return nil
		},
	}
}

// renderProjectTemplate builds a starter config for the requested project
// type by reading the corresponding preset (source of truth for the
// baseline rules) and prepending a project-typed comment header. This
// keeps init's output and `extends:` presets from drifting.
func renderProjectTemplate(projectType string) ([]byte, error) {
	presetName, ok := projectTypeToPreset[projectType]
	if !ok {
		return nil, fmt.Errorf("unknown project type: %s (available: %s)",
			projectType, strings.Join(projectTypeList(), ", "))
	}
	body, err := config.ReadPreset(presetName)
	if err != nil {
		return nil, fmt.Errorf("read preset for %s: %w", projectType, err)
	}
	header := projectTypeHeaders[projectType]
	buf := make([]byte, 0, len(header)+len(body))
	buf = append(buf, header...)
	buf = append(buf, body...)
	return buf, nil
}

func projectTypeList() []string {
	out := make([]string, 0, len(projectTypeToPreset))
	for k := range projectTypeToPreset {
		out = append(out, k)
	}
	// Alphabetical for stable error messages.
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1] > out[j] {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out
}

// detectProjectType guesses the project type from files in the directory.
func detectProjectType(dir string) string {
	checks := []struct {
		file     string
		projType string
	}{
		{"go.mod", "go"},
		{"package.json", "node"},
		{"pyproject.toml", "python"},
		{"setup.py", "python"},
		{"requirements.txt", "python"},
	}

	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			return c.projType
		}
	}

	return "generic"
}
