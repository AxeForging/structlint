package validator

import (
	"fmt"
	"sort"
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

// GroupViolationsByType groups violations by their structured Code and
// returns per-group summaries. Grouping is keyed on Violation.Code (not
// on message text): every rule already tags its violations, and messages
// are for humans. Unknown codes bucket into "other" as a defensive
// fallback so a new rule that forgets to register a code still surfaces.
func (v *Validator) GroupViolationsByType() []ViolationSummary {
	violationMap := make(map[string][]string, len(CodeDescriptions))
	for _, viol := range v.Violations {
		code := viol.Code
		if code == "" {
			code = "other"
		}
		violationMap[code] = append(violationMap[code], viol.Message)
	}

	summaries := make([]ViolationSummary, 0, len(violationMap))
	for code, messages := range violationMap {
		summaries = append(summaries, ViolationSummary{
			Type:        code,
			Count:       len(messages),
			Examples:    getExamples(messages, 3),
			Description: DescribeCode(code),
		})
	}

	// Sort by count descending, tie-break on Type ascending so equal-count
	// groups render deterministically (required by spec 005's goldens).
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Count != summaries[j].Count {
			return summaries[i].Count > summaries[j].Count
		}
		return summaries[i].Type < summaries[j].Type
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
