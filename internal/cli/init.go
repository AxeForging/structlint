package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

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
				Name:  "force",
				Usage: "overwrite existing configuration file",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			configPath := cmd.Root().String("config")
			if configPath == "" {
				configPath = ".structlint.yaml"
			}

			// Check if config already exists
			if _, err := os.Stat(configPath); err == nil && !cmd.Bool("force") {
				return fmt.Errorf("configuration file already exists: %s (use --force to overwrite)", configPath)
			}

			// Determine project type
			projectType := cmd.String("type")
			if projectType == "" {
				projectType = detectProjectType(".")
			}

			template, ok := projectTemplates[projectType]
			if !ok {
				return fmt.Errorf("unknown project type: %s (available: go, node, python, generic)", projectType)
			}

			if err := os.WriteFile(configPath, []byte(template), 0o644); err != nil {
				return fmt.Errorf("failed to write configuration: %w", err)
			}

			fmt.Printf("Created %s for %s project\n", configPath, projectType)
			fmt.Println("Run 'structlint validate' to check your project structure.")
			return nil
		},
	}
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

var projectTemplates = map[string]string{
	"go": `# structlint configuration for Go projects
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "api/**"
    - "test/**"
    - "docs/**"
    - "scripts/**"
    - ".github/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
  requiredPaths:
    - "cmd"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.mod"
    - "*.sum"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.md"
    - "*.txt"
    - "Makefile"
    - "Dockerfile*"
    - "*.sh"
    - ".gitignore"
    - ".goreleaser.yaml"
    - "LICENSE*"
  disallowed:
    - "*.env*"
    - "*.key"
    - "*.pem"
    - ".DS_Store"
    - "*.log"
    - "*.tmp"
    - "*~"
    - "*.swp"
  required:
    - "go.mod"
    - "README.md"
    - ".gitignore"

ignore:
  - ".git"
  - "vendor"
  - "bin"
  - "dist"
`,

	"node": `# structlint configuration for Node.js projects
dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "lib/**"
    - "test/**"
    - "tests/**"
    - "__tests__/**"
    - "docs/**"
    - "scripts/**"
    - "public/**"
    - "config/**"
    - ".github/**"
  disallowedPaths:
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
  requiredPaths:
    - "src"

file_naming_pattern:
  allowed:
    - "*.js"
    - "*.ts"
    - "*.jsx"
    - "*.tsx"
    - "*.json"
    - "*.yaml"
    - "*.yml"
    - "*.md"
    - "*.css"
    - "*.scss"
    - "*.html"
    - "*.svg"
    - "*.png"
    - "*.jpg"
    - ".gitignore"
    - ".eslintrc*"
    - ".prettierrc*"
    - "*.config.*"
    - "Dockerfile*"
    - "LICENSE*"
  disallowed:
    - "*.env*"
    - "*.key"
    - "*.pem"
    - ".DS_Store"
    - "*.log"
    - "*~"
    - "*.swp"
  required:
    - "package.json"
    - "README.md"

ignore:
  - ".git"
  - "node_modules"
  - "dist"
  - "build"
  - "coverage"
`,

	"python": `# structlint configuration for Python projects
dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "tests/**"
    - "test/**"
    - "docs/**"
    - "scripts/**"
    - ".github/**"
  disallowedPaths:
    - "__pycache__/**"
    - ".tox/**"
    - "*.egg-info/**"
    - "node_modules/**"
  requiredPaths: []

file_naming_pattern:
  allowed:
    - "*.py"
    - "*.pyi"
    - "*.toml"
    - "*.cfg"
    - "*.ini"
    - "*.txt"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.md"
    - "*.rst"
    - "Makefile"
    - "Dockerfile*"
    - "*.sh"
    - ".gitignore"
    - ".flake8"
    - "LICENSE*"
    - "MANIFEST.in"
  disallowed:
    - "*.env*"
    - "*.key"
    - "*.pem"
    - ".DS_Store"
    - "*.log"
    - "*~"
    - "*.swp"
    - "*.pyc"
  required:
    - "README.md"

ignore:
  - ".git"
  - "__pycache__"
  - ".tox"
  - ".venv"
  - "venv"
  - "dist"
  - "build"
  - "*.egg-info"
`,

	"generic": `# structlint configuration
dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "lib/**"
    - "test/**"
    - "tests/**"
    - "docs/**"
    - "scripts/**"
    - ".github/**"
  disallowedPaths:
    - "tmp/**"
    - "temp/**"
    - "node_modules/**"
  requiredPaths: []

file_naming_pattern:
  allowed:
    - "*.*"
  disallowed:
    - "*.env*"
    - "*.key"
    - "*.pem"
    - ".DS_Store"
    - "*.log"
    - "*.tmp"
    - "*~"
    - "*.swp"
  required:
    - "README.md"

ignore:
  - ".git"
  - "node_modules"
  - "vendor"
  - "dist"
  - "build"
`,
}
