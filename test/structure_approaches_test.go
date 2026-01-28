package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMonorepoStructure tests a monorepo with multiple services
func TestMonorepoStructure(t *testing.T) {
	bin := buildBinary(t)

	files := map[string]string{
		// Root level
		"go.work":     "go 1.24\n\nuse (\n\t./services/api\n\t./services/worker\n\t./libs/shared\n)",
		"README.md":   "# Monorepo",
		"Makefile":    "all: build",
		".gitignore":  "bin/\n*.log",

		// Service 1: API
		"services/api/go.mod":              "module monorepo/services/api\ngo 1.24",
		"services/api/cmd/main.go":         "package main\nfunc main() {}",
		"services/api/internal/handler.go": "package internal",
		"services/api/Dockerfile":          "FROM golang:1.24",

		// Service 2: Worker
		"services/worker/go.mod":            "module monorepo/services/worker\ngo 1.24",
		"services/worker/cmd/main.go":       "package main\nfunc main() {}",
		"services/worker/internal/worker.go": "package internal",
		"services/worker/Dockerfile":        "FROM golang:1.24",

		// Shared library
		"libs/shared/go.mod":    "module monorepo/libs/shared\ngo 1.24",
		"libs/shared/utils.go":  "package shared",
		"libs/shared/types.go":  "package shared",

		// Infrastructure
		"infra/terraform/main.tf":          "provider \"aws\" {}",
		"infra/k8s/api-deployment.yaml":    "apiVersion: apps/v1",
		"infra/k8s/worker-deployment.yaml": "apiVersion: apps/v1",

		// Documentation
		"docs/architecture.md": "# Architecture",
		"docs/api/openapi.yaml": "openapi: 3.0.0",

		// Scripts
		"scripts/build-all.sh": "#!/bin/bash\necho 'Building...'",
		"scripts/deploy.sh":    "#!/bin/bash\necho 'Deploying...'",
	}

	config := `dir_structure:
  allowedPaths:
    - "."
    - "services/**"
    - "libs/**"
    - "infra/**"
    - "docs/**"
    - "scripts/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
  requiredPaths:
    - "services"
    - "libs"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.mod"
    - "*.sum"
    - "*.yaml"
    - "*.yml"
    - "*.md"
    - "*.sh"
    - "*.tf"
    - "Makefile"
    - "Dockerfile*"
    - ".gitignore"
    - "go.work"
  disallowed:
    - "*.env*"
    - "*.log"
  required:
    - "go.work"
    - "README.md"

ignore:
  - ".git"
  - "vendor"
`

	projectDir := createTestProject(t, files, config)
	reportPath := filepath.Join(t.TempDir(), "monorepo-report.json")

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
		"--json-output", reportPath,
	)

	if err != nil {
		t.Errorf("Monorepo validation failed: %v\nOutput: %s", err, out)
	}

	// Verify JSON report
	data, _ := os.ReadFile(reportPath)
	var report map[string]interface{}
	json.Unmarshal(data, &report)

	if failures := report["failures"].(float64); failures > 0 {
		t.Errorf("Expected 0 failures, got %v", failures)
	}
}

// TestNodeJSProjectStructure tests a typical Node.js/TypeScript project
func TestNodeJSProjectStructure(t *testing.T) {
	bin := buildBinary(t)

	files := map[string]string{
		// Root config
		"package.json":     `{"name": "test-app", "version": "1.0.0"}`,
		"package-lock.json": `{"lockfileVersion": 2}`,
		"tsconfig.json":    `{"compilerOptions": {"target": "ES2020"}}`,
		".eslintrc.json":   `{"extends": ["eslint:recommended"]}`,
		".prettierrc":      `{"semi": true}`,
		"README.md":        "# Node.js App",
		".gitignore":       "node_modules/\ndist/\n.env",

		// Source code
		"src/index.ts":              "export const main = () => {}",
		"src/app.ts":                "import express from 'express'",
		"src/routes/users.ts":       "export const userRoutes = {}",
		"src/routes/products.ts":    "export const productRoutes = {}",
		"src/middleware/auth.ts":    "export const authMiddleware = {}",
		"src/services/user.ts":      "export class UserService {}",
		"src/models/user.ts":        "export interface User {}",
		"src/utils/logger.ts":       "export const logger = {}",

		// Tests
		"tests/unit/user.test.ts":       "describe('User', () => {})",
		"tests/integration/api.test.ts": "describe('API', () => {})",
		"jest.config.js":                "module.exports = {}",

		// Config
		"config/default.json":    `{"port": 3000}`,
		"config/production.json": `{"port": 8080}`,

		// Docker
		"Dockerfile":          "FROM node:20-alpine",
		"docker-compose.yml":  "version: '3'",

		// CI/CD
		".github/workflows/ci.yml": "name: CI",
	}

	config := `dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "tests/**"
    - "config/**"
    - ".github/**"
  disallowedPaths:
    - "node_modules/**"
    - "dist/**"
    - "coverage/**"

file_naming_pattern:
  allowed:
    - "*.ts"
    - "*.js"
    - "*.json"
    - "*.yml"
    - "*.yaml"
    - "*.md"
    - "Dockerfile*"
    - "docker-compose*"
    - ".gitignore"
    - ".eslintrc*"
    - ".prettierrc*"
    - "jest.config.*"
    - "tsconfig*.json"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
  required:
    - "package.json"
    - "README.md"

ignore:
  - "node_modules"
  - "dist"
  - ".git"
`

	projectDir := createTestProject(t, files, config)

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
	)

	if err != nil {
		t.Errorf("Node.js project validation failed: %v\nOutput: %s", err, out)
	}
}

// TestPythonProjectStructure tests a typical Python project
func TestPythonProjectStructure(t *testing.T) {
	bin := buildBinary(t)

	files := map[string]string{
		// Root files
		"pyproject.toml":   "[project]\nname = 'myapp'",
		"setup.py":         "from setuptools import setup",
		"requirements.txt": "fastapi==0.100.0\nuvicorn==0.23.0",
		"README.md":        "# Python App",
		".gitignore":       "__pycache__/\n*.pyc\n.env\nvenv/",
		"Makefile":         "install:\n\tpip install -r requirements.txt",

		// Source package
		"src/myapp/__init__.py":        "",
		"src/myapp/main.py":            "def main(): pass",
		"src/myapp/api/routes.py":      "from fastapi import APIRouter",
		"src/myapp/api/__init__.py":    "",
		"src/myapp/models/user.py":     "class User: pass",
		"src/myapp/models/__init__.py": "",
		"src/myapp/utils/logger.py":    "import logging",
		"src/myapp/utils/__init__.py":  "",

		// Tests
		"tests/__init__.py":          "",
		"tests/conftest.py":          "import pytest",
		"tests/test_main.py":         "def test_main(): pass",
		"tests/unit/__init__.py":     "",
		"tests/unit/test_user.py":    "def test_user(): pass",

		// Config
		"config/settings.yaml": "debug: true",

		// Docker
		"Dockerfile":         "FROM python:3.11-slim",
		"docker-compose.yml": "version: '3'",
	}

	config := `dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "tests/**"
    - "config/**"
    - "scripts/**"
  disallowedPaths:
    - "venv/**"
    - "__pycache__/**"
    - ".pytest_cache/**"
    - "*.egg-info/**"

file_naming_pattern:
  allowed:
    - "*.py"
    - "*.yaml"
    - "*.yml"
    - "*.toml"
    - "*.txt"
    - "*.md"
    - "*.cfg"
    - "*.ini"
    - "Makefile"
    - "Dockerfile*"
    - "docker-compose*"
    - ".gitignore"
    - "setup.py"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.pyc"
  required:
    - "README.md"
    - "*.py"

ignore:
  - "venv"
  - "__pycache__"
  - ".git"
  - ".pytest_cache"
`

	projectDir := createTestProject(t, files, config)

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
	)

	if err != nil {
		t.Errorf("Python project validation failed: %v\nOutput: %s", err, out)
	}
}

// TestDeepNestedStructure tests handling of deeply nested directories
func TestDeepNestedStructure(t *testing.T) {
	bin := buildBinary(t)

	files := map[string]string{
		"level1/level2/level3/level4/level5/deep.go":                       "package deep",
		"level1/level2/level3/level4/level5/level6/level7/verydeep.go":     "package verydeep",
		"a/b/c/d/e/f/g/h/i/j/extreme.go":                                   "package extreme",
		"src/internal/pkg/domain/entity/user/repository/postgres/impl.go": "package postgres",
		"README.md": "# Deep Structure Test",
	}

	config := `dir_structure:
  allowedPaths:
    - "."
    - "level1/**"
    - "a/**"
    - "src/**"
  disallowedPaths: []

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
  disallowed: []

ignore:
  - ".git"
`

	projectDir := createTestProject(t, files, config)

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
	)

	if err != nil {
		t.Errorf("Deep nested structure validation failed: %v\nOutput: %s", err, out)
	}

	// Check that all deep files were found
	if !strings.Contains(out, "passed") || strings.Contains(out, "violation") {
		t.Logf("Output: %s", out)
	}
}

// TestMixedLanguageProject tests a project with multiple languages
func TestMixedLanguageProject(t *testing.T) {
	bin := buildBinary(t)

	files := map[string]string{
		// Go backend
		"backend/go.mod":             "module backend\ngo 1.24",
		"backend/cmd/server/main.go": "package main",
		"backend/internal/api.go":    "package internal",

		// TypeScript frontend
		"frontend/package.json":      `{"name": "frontend"}`,
		"frontend/src/App.tsx":       "export const App = () => {}",
		"frontend/src/index.ts":      "import './App'",
		"frontend/tsconfig.json":     "{}",

		// Python ML service
		"ml/requirements.txt":        "numpy==1.24.0",
		"ml/src/__init__.py":         "",
		"ml/src/model.py":            "class Model: pass",

		// Shared
		"shared/proto/api.proto":     "syntax = \"proto3\";",
		"shared/scripts/build.sh":    "#!/bin/bash",

		// Root
		"docker-compose.yml":         "version: '3'",
		"Makefile":                   "all: build",
		"README.md":                  "# Multi-language Project",
	}

	config := `dir_structure:
  allowedPaths:
    - "."
    - "backend/**"
    - "frontend/**"
    - "ml/**"
    - "shared/**"
  disallowedPaths:
    - "node_modules/**"
    - "vendor/**"
    - "__pycache__/**"
    - "venv/**"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.mod"
    - "*.sum"
    - "*.ts"
    - "*.tsx"
    - "*.js"
    - "*.json"
    - "*.py"
    - "*.txt"
    - "*.proto"
    - "*.yaml"
    - "*.yml"
    - "*.md"
    - "*.sh"
    - "Makefile"
    - "Dockerfile*"
    - "docker-compose*"
  disallowed:
    - "*.env*"
    - "*.log"

ignore:
  - ".git"
  - "node_modules"
  - "vendor"
  - "__pycache__"
  - "venv"
`

	projectDir := createTestProject(t, files, config)

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
	)

	if err != nil {
		t.Errorf("Mixed language project validation failed: %v\nOutput: %s", err, out)
	}
}

// TestStrictVsPermissiveMode tests different strictness configurations
func TestStrictVsPermissiveMode(t *testing.T) {
	bin := buildBinary(t)

	// Same project files for both tests
	files := map[string]string{
		"main.go":          "package main",
		"utils.go":         "package main",
		"README.md":        "# Test",
		"config.yaml":      "key: value",
		"extra.txt":        "extra file",
		"random/file.data": "some data",
	}

	t.Run("permissive", func(t *testing.T) {
		// Permissive: broad patterns that match almost anything
		permissiveConfig := `dir_structure:
  allowedPaths:
    - "."
    - "**"  # Allow all directories
  disallowedPaths:
    - "vendor/**"

file_naming_pattern:
  allowed:
    - "*"      # Allow any file
    - "**/*"   # Allow files in any directory
    - ".*"     # Allow dotfiles
  disallowed:
    - "*.env*"

ignore:
  - ".git"
`
		projectDir := createTestProject(t, files, permissiveConfig)

		out, err := runBinaryInDir(t, bin, projectDir,
			"validate",
			"--config", ".structlint.yaml",
		)

		if err != nil {
			t.Errorf("Permissive mode should pass: %v\nOutput: %s", err, out)
		}
	})

	t.Run("strict", func(t *testing.T) {
		strictConfig := `dir_structure:
  allowedPaths:
    - "."  # Only root allowed
  disallowedPaths:
    - "random/**"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
  disallowed:
    - "*.txt"
    - "*.data"

ignore:
  - ".git"
`
		projectDir := createTestProject(t, files, strictConfig)

		out, err := runBinaryInDir(t, bin, projectDir,
			"validate",
			"--config", ".structlint.yaml",
		)

		// Should fail due to violations
		if err == nil {
			t.Error("Strict mode should fail with violations")
		}

		// Should report the violations
		if !strings.Contains(out, "extra.txt") {
			t.Errorf("Should report extra.txt violation: %s", out)
		}
	})
}

// TestSpecialCharactersInPaths tests handling of special characters in file/dir names
func TestSpecialCharactersInPaths(t *testing.T) {
	bin := buildBinary(t)

	files := map[string]string{
		"file-with-dashes.go":       "package main",
		"file_with_underscores.go":  "package main",
		"file.multiple.dots.go":     "package main",
		"CamelCaseFile.go":          "package main",
		"UPPERCASE.go":              "package main",
		"dir-with-dash/file.go":     "package sub",
		"dir_with_underscore/file.go": "package sub",
		"README.md":                 "# Test",
	}

	config := `dir_structure:
  allowedPaths:
    - "."
    - "dir-with-dash/**"
    - "dir_with_underscore/**"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
  disallowed: []

ignore:
  - ".git"
`

	projectDir := createTestProject(t, files, config)

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
	)

	if err != nil {
		t.Errorf("Special characters validation failed: %v\nOutput: %s", err, out)
	}
}

// TestEmptyDirectories tests handling of empty directories
func TestEmptyDirectories(t *testing.T) {
	bin := buildBinary(t)

	projectDir := t.TempDir()

	// Create empty directories
	emptyDirs := []string{
		"empty1",
		"empty2/nested",
		"src/empty",
	}

	for _, dir := range emptyDirs {
		if err := os.MkdirAll(filepath.Join(projectDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
	}

	// Create minimal files
	writeTestFile(t, projectDir, "main.go", "package main")
	writeTestFile(t, projectDir, "src/app.go", "package src")

	config := `dir_structure:
  allowedPaths:
    - "."
    - "empty1/**"
    - "empty2/**"
    - "src/**"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"

ignore:
  - ".git"
`
	writeTestFile(t, projectDir, ".structlint.yaml", config)

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
	)

	if err != nil {
		t.Errorf("Empty directories validation failed: %v\nOutput: %s", err, out)
	}
}

// TestRequiredFilesWithGlobsComprehensive tests various required file patterns
func TestRequiredFilesWithGlobsComprehensive(t *testing.T) {
	bin := buildBinary(t)

	t.Run("required_met", func(t *testing.T) {
		files := map[string]string{
			"main.go":           "package main",
			"app_test.go":       "package main",
			"README.md":         "# Test",
			"docs/USAGE.md":     "# Usage",
			"internal/util.go":  "package internal",
		}

		config := `dir_structure:
  allowedPaths:
    - "."
    - "docs/**"
    - "internal/**"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
  required:
    - "*.go"        # At least one Go file
    - "*_test.go"   # At least one test file
    - "README.md"   # Exact match
    - "docs/*.md"   # At least one doc

ignore:
  - ".git"
`
		projectDir := createTestProject(t, files, config)

		out, err := runBinaryInDir(t, bin, projectDir,
			"validate",
			"--config", ".structlint.yaml",
		)

		if err != nil {
			t.Errorf("Required files validation failed: %v\nOutput: %s", err, out)
		}
	})

	t.Run("required_not_met", func(t *testing.T) {
		files := map[string]string{
			"main.go": "package main",
			// Missing: test files, README, docs
		}

		config := `dir_structure:
  allowedPaths:
    - "."

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.md"
    - "*.yaml"
  required:
    - "*_test.go"  # Missing!
    - "README.md"  # Missing!

ignore:
  - ".git"
`
		projectDir := createTestProject(t, files, config)

		out, err := runBinaryInDir(t, bin, projectDir,
			"validate",
			"--config", ".structlint.yaml",
		)

		if err == nil {
			t.Error("Should fail when required files are missing")
		}

		if !strings.Contains(out, "test.go") || !strings.Contains(out, "README") {
			t.Logf("Should mention missing required files: %s", out)
		}
	})
}

// TestIgnorePatternComprehensive tests various ignore patterns
func TestIgnorePatternComprehensive(t *testing.T) {
	bin := buildBinary(t)

	files := map[string]string{
		"main.go":                    "package main",
		"vendor/lib/lib.go":          "package lib",
		"node_modules/pkg/index.js":  "module.exports = {}",
		".git/config":                "[core]",
		"__pycache__/cache.pyc":      "bytecode",
		"build/output/app":           "binary",
		".idea/workspace.xml":        "<xml>",
		"tmp/debug.log":              "logs",
		"valid/code.go":              "package valid",
	}

	config := `dir_structure:
  allowedPaths:
    - "."
    - "valid/**"
  disallowedPaths:
    - "tmp/**"

file_naming_pattern:
  allowed:
    - "*.go"
    - "*.yaml"
  disallowed:
    - "*.log"
    - "*.pyc"

ignore:
  - "vendor"
  - "node_modules"
  - ".git"
  - "__pycache__"
  - "build"
  - ".idea"
  - "tmp"
`

	projectDir := createTestProject(t, files, config)

	out, err := runBinaryInDir(t, bin, projectDir,
		"validate",
		"--config", ".structlint.yaml",
	)

	if err != nil {
		t.Errorf("Ignore pattern validation failed: %v\nOutput: %s", err, out)
	}

	// Should not report any violations from ignored paths
	ignoredPaths := []string{"vendor", "node_modules", ".git", "__pycache__", "build", ".idea", "tmp"}
	for _, ignored := range ignoredPaths {
		if strings.Contains(out, ignored) && strings.Contains(out, "violation") {
			t.Errorf("Should not report violations from ignored path %s: %s", ignored, out)
		}
	}
}
