package validator

import (
	"fmt"
	"sort"
	"strings"
)

// ViolationSummary represents grouped violations by type
type ViolationSummary struct {
	Type        string   `json:"type"`
	Count       int      `json:"count"`
	Examples    []string `json:"examples"`
	Description string   `json:"description"`
}

// ValidationSummary provides a comprehensive summary of validation results
type ValidationSummary struct {
	TotalSuccesses int                `json:"total_successes"`
	TotalFailures  int                `json:"total_failures"`
	Violations     []ViolationSummary `json:"violations"`
	AllErrors      []string           `json:"all_errors,omitempty"` // Only included if requested
}

// GroupViolationsByType groups errors by type and provides examples
func (v *Validator) GroupViolationsByType() []ViolationSummary {
	violationMap := make(map[string][]string)

	// Group violations by type
	for _, err := range v.Errors {
		var violationType string
		if strings.Contains(err, "Disallowed directory found:") {
			violationType = "disallowed_directory"
		} else if strings.Contains(err, "Directory not in allowed list:") {
			violationType = "unallowed_directory"
		} else if strings.Contains(err, "Disallowed file naming pattern found:") {
			violationType = "disallowed_file_pattern"
		} else if strings.Contains(err, "File not in allowed naming pattern:") {
			violationType = "unallowed_file_pattern"
		} else if strings.Contains(err, "Required directory missing:") {
			violationType = "missing_required_directory"
		} else if strings.Contains(err, "Required file pattern missing:") {
			violationType = "missing_required_file"
		} else {
			violationType = "other"
		}

		violationMap[violationType] = append(violationMap[violationType], err)
	}

	// Convert to summary format
	var summaries []ViolationSummary
	for violationType, errors := range violationMap {
		summary := ViolationSummary{
			Type:     violationType,
			Count:    len(errors),
			Examples: getExamples(errors, 3), // Show up to 3 examples
		}

		// Add description
		switch violationType {
		case "disallowed_directory":
			summary.Description = "Directories that are explicitly disallowed"
		case "unallowed_directory":
			summary.Description = "Directories not in the allowed list"
		case "disallowed_file_pattern":
			summary.Description = "Files matching disallowed naming patterns"
		case "unallowed_file_pattern":
			summary.Description = "Files not matching any allowed naming pattern"
		case "missing_required_directory":
			summary.Description = "Required directories that are missing"
		case "missing_required_file":
			summary.Description = "Required file patterns that are missing"
		default:
			summary.Description = "Other validation errors"
		}

		summaries = append(summaries, summary)
	}

	// Sort by count (highest first)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Count > summaries[j].Count
	})

	return summaries
}

// getExamples returns up to maxCount examples from the error list
func getExamples(errors []string, maxCount int) []string {
	if len(errors) <= maxCount {
		return errors
	}

	examples := make([]string, maxCount)
	copy(examples, errors[:maxCount])
	return examples
}

// PrintGroupedSummary prints a grouped summary of violations
func (v *Validator) PrintGroupedSummary() {
	if v.Silent {
		return
	}

	fmt.Println("\n--- Validation Summary ---")
	fmt.Printf("✓ %d checks passed\n", v.Successes)
	fmt.Printf("✗ %d checks failed\n", len(v.Errors))

	if len(v.Errors) == 0 {
		return
	}

	// Group violations
	summaries := v.GroupViolationsByType()

	fmt.Println("\n--- Violation Summary ---")
	for _, summary := range summaries {
		fmt.Printf("\n%s (%d violations):\n", summary.Description, summary.Count)

		// Show examples
		for _, example := range summary.Examples {
			fmt.Printf("  - %s\n", example)
		}

		// Show if there are more
		if summary.Count > len(summary.Examples) {
			fmt.Printf("  ... and %d more\n", summary.Count-len(summary.Examples))
		}
	}
}

// GetValidationSummary returns a comprehensive validation summary
func (v *Validator) GetValidationSummary(includeAllErrors bool) ValidationSummary {
	summary := ValidationSummary{
		TotalSuccesses: v.Successes,
		TotalFailures:  len(v.Errors),
		Violations:     v.GroupViolationsByType(),
	}

	if includeAllErrors {
		summary.AllErrors = v.Errors
	}

	return summary
}
