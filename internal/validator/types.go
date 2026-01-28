package validator

// JSONReport represents the structure of the JSON report.
type JSONReport struct {
	Successes int               `json:"successes"`
	Failures  int               `json:"failures"`
	Errors    []string          `json:"errors"`
	Summary   ValidationSummary `json:"summary,omitempty"`
}
