package validator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// PrintJSONReport writes the same machine-readable report used by --json-output.
func (v *Validator) PrintJSONReport() error {
	data, err := json.MarshalIndent(v.report(), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

// PrintGitHubAnnotations writes GitHub Actions workflow command annotations.
func (v *Validator) PrintGitHubAnnotations() {
	for _, violation := range v.sortedViolations() {
		level := "error"
		if violation.Severity == "warning" {
			level = "warning"
		}
		fmt.Printf("::%s file=%s,title=%s::%s\n", level, violation.Path, violation.Code, escapeGitHub(violation.Message))
	}
}

// PrintSARIFReport writes a small SARIF 2.1.0 report for code scanning systems.
func (v *Validator) PrintSARIFReport() error {
	rules := map[string]map[string]string{}
	results := make([]map[string]any, 0, len(v.Violations))
	for _, violation := range v.sortedViolations() {
		rules[violation.Code] = map[string]string{
			"id":   violation.Code,
			"name": violation.Code,
		}
		level := "error"
		if violation.Severity == "warning" {
			level = "warning"
		}
		results = append(results, map[string]any{
			"ruleId":  violation.Code,
			"level":   level,
			"message": map[string]string{"text": violation.Message},
			"locations": []map[string]any{
				{
					"physicalLocation": map[string]any{
						"artifactLocation": map[string]string{"uri": violation.Path},
					},
				},
			},
		})
	}

	ruleList := make([]map[string]string, 0, len(rules))
	for _, rule := range rules {
		ruleList = append(ruleList, rule)
	}
	report := map[string]any{
		"version": "2.1.0",
		"$schema": "https://json.schemastore.org/sarif-2.1.0.json",
		"runs": []map[string]any{
			{
				"tool": map[string]any{
					"driver": map[string]any{
						"name":  "structlint",
						"rules": ruleList,
					},
				},
				"results": results,
			},
		},
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func (v *Validator) report() JSONReport {
	return JSONReport{
		Successes:       v.Successes,
		Failures:        len(v.Errors),
		TotalViolations: len(v.Errors),
		Errors:          append([]string(nil), v.Errors...),
		Violations:      v.sortedViolations(),
		Summary:         v.GetValidationSummary(false),
	}
}

func escapeGitHub(value string) string {
	replacer := strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A")
	return replacer.Replace(value)
}

// SaveJSONReport saves the validation results to a JSON file.
func (v *Validator) SaveJSONReport(path string) error {
	data, err := json.MarshalIndent(v.report(), "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}
