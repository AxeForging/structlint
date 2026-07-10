package validator

import "testing"

// TestCodeDescriptions_CoversAllEmittedCodes pins the exact 13 codes emitted
// by the current rule set. If a new rule adds a code without registering it
// here, this test fails — that is intentional per spec 011's frozen-codes
// contract.
func TestCodeDescriptions_CoversAllEmittedCodes(t *testing.T) {
	want := []string{
		"disallowed_directory",
		"unallowed_directory",
		"disallowed_file_pattern",
		"unallowed_file_pattern",
		"missing_required_directory",
		"missing_required_file",
		"placement_violation",
		"missing_required_group",
		"missing_required_group_match",
		"missing_group_file",
		"boundary_violation",
		"parse_error",
		"walk_error",
	}
	if len(CodeDescriptions) != len(want) {
		t.Fatalf("CodeDescriptions has %d entries, want %d", len(CodeDescriptions), len(want))
	}
	for _, code := range want {
		if _, ok := CodeDescriptions[code]; !ok {
			t.Errorf("missing code: %s", code)
		}
	}
}

func TestDescribeCode_UnknownFallsBackToOther(t *testing.T) {
	if got := DescribeCode("no_such_code"); got != "Other validation errors" {
		t.Errorf("unknown fallback: got %q, want %q", got, "Other validation errors")
	}
	if got := DescribeCode("placement_violation"); got != "Files placed outside their required directories" {
		t.Errorf("known lookup: got %q", got)
	}
}
