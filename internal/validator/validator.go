package validator

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/AxeForging/structlint/internal/config"
	"github.com/gobwas/glob"
)

// Validator holds the configuration and validation results.
type Validator struct {
	Config          *config.Config
	Errors          []string
	Successes       int
	Logger          *slog.Logger
	Silent          bool
	GroupViolations bool
	Verbose         bool // Show all allowed files, not just violations
}

// New creates a new Validator.
func New(cfg *config.Config, logger *slog.Logger) *Validator {
	return &Validator{
		Config:          cfg,
		Errors:          []string{},
		Successes:       0,
		Logger:          logger,
		Silent:          false,
		GroupViolations: true,  // Default to grouping
		Verbose:         false, // Default to quiet mode - only show violations
	}
}

// ValidateDirStructure validates the directory structure.
func (v *Validator) ValidateDirStructure(path string) {
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the path should be ignored
		for _, ignored := range v.Config.Ignore {
			if matches(currentPath, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if info.IsDir() {
			// Check against disallowed paths
			for _, disallowed := range v.Config.DirStructure.DisallowedPaths {
				if matches(currentPath, disallowed) {
					msg := fmt.Sprintf("Disallowed directory found: %s", currentPath)
					v.printError(msg)
					v.Errors = append(v.Errors, msg)
					return filepath.SkipDir // Skip validating contents of disallowed directories
				}
			}

			// Check against allowed paths
			isAllowed := false
			for _, allowed := range v.Config.DirStructure.AllowedPaths {
				if matches(currentPath, allowed) {
					isAllowed = true
					break
				}
			}

			// Also consider a directory allowed if it's a parent of an allowed path
			if !isAllowed {
				for _, allowed := range v.Config.DirStructure.AllowedPaths {
					if strings.HasPrefix(allowed, currentPath) {
						isAllowed = true
						break
					}
				}
			}

			if isAllowed {
				msg := fmt.Sprintf("Allowed directory found: %s", currentPath)
				v.printSuccess(msg)
				v.Successes++
			} else {
				msg := fmt.Sprintf("Directory not in allowed list: %s", currentPath)
				v.printError(msg)
				v.Errors = append(v.Errors, msg)
			}
		}
		return nil
	})

	if err != nil {
		v.Errors = append(v.Errors, fmt.Sprintf("Error walking directory: %s", err))
	}
}

// matches checks if a path matches a glob pattern.
func matches(path, pattern string) bool {
	g, err := glob.Compile(pattern)
	if err != nil {
		return false
	}
	return g.Match(path)
}

// ValidateFileNaming validates the file naming conventions.
func (v *Validator) ValidateFileNaming(path string) {
	err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if the path should be ignored
		for _, ignored := range v.Config.Ignore {
			if matches(currentPath, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		if !info.IsDir() {
			fileName := info.Name()

			// Check against disallowed patterns
			for _, disallowed := range v.Config.FileNamingPattern.Disallowed {
				if matches(fileName, disallowed) {
					msg := fmt.Sprintf("Disallowed file naming pattern found: %s", currentPath)
					v.printError(msg)
					v.Errors = append(v.Errors, msg)
					return nil
				}
			}

			// Check against allowed patterns
			isAllowed := false
			for _, allowed := range v.Config.FileNamingPattern.Allowed {
				if matches(fileName, allowed) {
					isAllowed = true
					break
				}
			}
			if isAllowed {
				msg := fmt.Sprintf("Allowed file naming pattern found: %s", currentPath)
				v.printSuccess(msg)
				v.Successes++
			} else {
				msg := fmt.Sprintf("File not in allowed naming pattern: %s", currentPath)
				v.printError(msg)
				v.Errors = append(v.Errors, msg)
			}
		}
		return nil
	})

	if err != nil {
		v.Errors = append(v.Errors, fmt.Sprintf("Error walking directory: %s", err))
	}
}

// PrintSummary prints a summary of the validation results.
func (v *Validator) PrintSummary() {
	if v.Silent {
		return
	}

	// Always show a concise summary
	fmt.Println("\n--- Validation Summary ---")
	fmt.Printf("✓ %d files/directories passed validation\n", v.Successes)
	fmt.Printf("✗ %d violations found\n", len(v.Errors))

	if len(v.Errors) == 0 {
		fmt.Println("🎉 All files and directories comply with the rules!")
		return
	}

	// Use grouped summary if enabled or if there are many violations
	if v.GroupViolations || len(v.Errors) > 10 {
		v.PrintGroupedSummary()
		return
	}

	// For few errors, show detailed list
	fmt.Println("\nViolations:")
	for _, err := range v.Errors {
		fmt.Printf("- %s\n", err)
	}
}

// SaveJSONReport saves the validation results to a JSON file.
func (v *Validator) SaveJSONReport(path string) error {
	report := JSONReport{
		Successes: v.Successes,
		Failures:  len(v.Errors),
		Errors:    v.Errors,
		Summary:   v.GetValidationSummary(false), // Don't include all errors in summary to avoid duplication
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ValidateRequiredPaths validates that all required directories exist.
func (v *Validator) ValidateRequiredPaths(path string) {
	for _, requiredPath := range v.Config.DirStructure.RequiredPaths {
		fullPath := filepath.Join(path, requiredPath)

		// Check if the required path exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			v.printError(fmt.Sprintf("Required directory missing: %s", requiredPath))
			v.Errors = append(v.Errors, fmt.Sprintf("Required directory missing: %s", requiredPath))
		} else {
			v.printSuccess(fmt.Sprintf("Required directory found: %s", requiredPath))
			v.Successes++
		}
	}
}

// ValidateRequiredFiles validates that all required files exist.
func (v *Validator) ValidateRequiredFiles(path string) {
	for _, requiredFile := range v.Config.FileNamingPattern.Required {
		// Check if any file matching the pattern exists
		found := false
		err := filepath.Walk(path, func(currentPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Check if the path should be ignored
			for _, ignored := range v.Config.Ignore {
				if matches(currentPath, ignored) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			if !info.IsDir() {
				// For required file patterns, we need to check both the filename and the relative path
				fileName := info.Name()
				relPath, err := filepath.Rel(path, currentPath)
				if err != nil {
					relPath = currentPath // Fallback to full path if relative path fails
				}

				// Check if either the filename or the relative path matches the pattern
				if matches(fileName, requiredFile) || matches(relPath, requiredFile) {
					found = true
					return filepath.SkipAll // Stop walking once we find a match
				}
			}
			return nil
		})

		if err != nil {
			v.Errors = append(v.Errors, fmt.Sprintf("Error checking for required file %s: %s", requiredFile, err))
			continue
		}

		if found {
			v.printSuccess(fmt.Sprintf("Required file pattern found: %s", requiredFile))
			v.Successes++
		} else {
			v.printError(fmt.Sprintf("Required file pattern missing: %s", requiredFile))
			v.Errors = append(v.Errors, fmt.Sprintf("Required file pattern missing: %s", requiredFile))
		}
	}
}

func (v *Validator) printSuccess(message string) {
	if !v.Silent && v.Logger != nil && v.Verbose {
		v.Logger.Info("✓ " + message)
	}
}

func (v *Validator) printError(message string) {
	if !v.Silent && v.Logger != nil {
		v.Logger.Error("✗ " + message)
	}
}
