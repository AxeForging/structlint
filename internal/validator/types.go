package validator

// Violation is a stable, machine-readable validation failure for CI systems.
type Violation struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

// JSONReport represents the structure of the JSON report.
type JSONReport struct {
	Successes       int               `json:"successes"`
	Failures        int               `json:"failures"`
	TotalViolations int               `json:"total_violations"`
	Errors          []string          `json:"errors"`
	Violations      []Violation       `json:"violations"`
	Summary         ValidationSummary `json:"summary,omitempty"`
}
