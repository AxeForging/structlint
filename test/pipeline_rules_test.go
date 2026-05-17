package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/AxeForging/structlint/internal/logging"
	"github.com/AxeForging/structlint/internal/validator"
)

func TestPlacementRequiredGroupsAndBoundaries(t *testing.T) {
	project := createPipelineRuleProject(t, map[string]string{
		"go.mod":                          "module example.com/app\n\ngo 1.24\n",
		"README.md":                       "# app",
		"Makefile":                        "test:\n\tgo test ./...",
		"cmd/api/main.go":                 "package main",
		"internal/domain/user.go":         "package domain\n\nimport _ \"example.com/app/internal/db\"\n",
		"internal/db/db.go":               "package db",
		"migrations/001_init.sql":         "create table users(id int);",
		"internal/service/orphan_test.go": "package service",
	})

	cfg, err := config.LoadConfig(filepath.Join(project, ".structlint.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	logger, _ := logging.New("error", true)
	v := validator.New(cfg, logger)
	v.Silent = true
	v.ValidatePlacement(project)
	v.ValidateRequiredGroups(project)
	v.ValidateBoundaries(project)

	if len(v.Violations) != 1 {
		t.Fatalf("expected one boundary violation, got %d: %#v", len(v.Violations), v.Violations)
	}
	got := v.Violations[0]
	if got.Code != "boundary_violation" || got.Rule != "domain-no-db" || got.Path != "internal/domain/user.go" {
		t.Fatalf("unexpected violation: %#v", got)
	}
}

func TestBoundaryRulesSupportJavaScriptAndPython(t *testing.T) {
	project := createTestProject(t, map[string]string{
		"src/domain/user.ts": "import db from '../db/client'\nexport const user = db\n",
		"src/db/client.ts":   "export default {}\n",
		"app/domain/user.py": "from app.db.client import connect\n",
		"app/db/client.py":   "def connect(): pass\n",
		"package.json":       `{"type":"module"}`,
		"pyproject.toml":     "[project]\nname = \"app\"\n",
		"README.md":          "# mixed",
	}, `dir_structure:
  allowedPaths: [".", "src/**", "app/**"]
file_naming_pattern:
  allowed: ["*.ts", "*.py", "*.json", "*.toml", "*.md"]
boundaries:
  - id: ts-domain-no-db
    from: "src/domain/**"
    cannotImport: ["src/db/**"]
  - id: py-domain-no-db
    from: "app/domain/**"
    cannotImport: ["app/db/**"]
ignore: [".git"]
`)

	cfg, err := config.LoadConfig(filepath.Join(project, ".structlint.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	logger, _ := logging.New("error", true)
	v := validator.New(cfg, logger)
	v.Silent = true
	v.ValidateBoundaries(project)

	seen := map[string]bool{}
	for _, violation := range v.Violations {
		seen[violation.Rule] = true
	}
	for _, rule := range []string{"ts-domain-no-db", "py-domain-no-db"} {
		if !seen[rule] {
			t.Fatalf("expected %s boundary violation, got %#v", rule, v.Violations)
		}
	}
}

func TestPlacementAndRequiredGroupViolationsAreStructured(t *testing.T) {
	project := createPipelineRuleProject(t, map[string]string{
		"go.mod":              "module example.com/app\n\ngo 1.24\n",
		"cmd/api/handler.go":  "package main",
		"schema.sql":          "create table users(id int);",
		"internal/app/app.go": "package app",
	})

	cfg, err := config.LoadConfig(filepath.Join(project, ".structlint.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	logger, _ := logging.New("error", true)
	v := validator.New(cfg, logger)
	v.Silent = true
	v.ValidatePlacement(project)
	v.ValidateRequiredGroups(project)

	codes := map[string]bool{}
	for _, violation := range v.Violations {
		codes[violation.Code] = true
		if violation.Path == "" || violation.Rule == "" || violation.Message == "" {
			t.Fatalf("violation is not fully structured: %#v", violation)
		}
	}
	for _, code := range []string{"placement_violation", "missing_required_group", "missing_group_file"} {
		if !codes[code] {
			t.Fatalf("missing violation code %s in %#v", code, v.Violations)
		}
	}
}

func createPipelineRuleProject(t *testing.T, files map[string]string) string {
	t.Helper()
	return createTestProject(t, files, `dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**", "migrations/**"]
file_naming_pattern:
  allowed: ["*.go", "*.mod", "*.md", "Makefile", "*.sql"]
placement:
  - id: sql-in-migrations
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]
  - id: tests-in-test-roots
    files: ["*_test.go"]
    mustBeUnder: ["test/**", "internal/**"]
requiredGroups:
  - id: build-entrypoint
    oneOf: ["Makefile", "Taskfile.yml", "justfile"]
  - id: commands-have-main
    eachDirMatching: "cmd/*"
    mustContain: ["main.go"]
    requireMatch: true
boundaries:
  - id: domain-no-db
    from: "internal/domain/**"
    cannotImport: ["internal/db/**"]
ignore: [".git"]
`)
}

func TestStrictConfigRejectsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".structlint.yaml")
	if err := os.WriteFile(path, []byte(`dir_structure:
  allowed_paths: ["cmd/**"]
file_naming_pattern:
  allowed: ["*.go"]
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := config.LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "allowed_paths") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestJSONReportIncludesTypedViolations(t *testing.T) {
	project := createTestProject(t, map[string]string{
		"main.go": "package main",
		".env":    "SECRET=1",
	}, `dir_structure:
  allowedPaths: ["."]
file_naming_pattern:
  allowed: ["*.go"]
  disallowed: ["*.env*"]
ignore: []
`)

	cfg, err := config.LoadConfig(filepath.Join(project, ".structlint.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	logger, _ := logging.New("error", true)
	v := validator.New(cfg, logger)
	v.Silent = true
	v.ValidateFileNaming(project)

	reportPath := filepath.Join(t.TempDir(), "report.json")
	if err := v.SaveJSONReport(reportPath); err != nil {
		t.Fatalf("save report: %v", err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var report validator.JSONReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if report.TotalViolations != 2 || len(report.Violations) != 2 {
		t.Fatalf("expected typed violations in report, got %#v", report)
	}
}
