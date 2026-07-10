package validator

// CodeDescriptions is the canonical registry of violation codes.
// Codes are frozen and append-only; specs 010 (suggest) and 011
// (SKILL.md / violation-codes.md) build on this map.
var CodeDescriptions = map[string]string{
	"disallowed_directory":         "Directories that are explicitly disallowed",
	"unallowed_directory":          "Directories not in the allowed list",
	"disallowed_file_pattern":      "Files matching disallowed naming patterns",
	"unallowed_file_pattern":       "Files not matching any allowed naming pattern",
	"missing_required_directory":   "Required directories that are missing",
	"missing_required_file":        "Required file patterns that are missing",
	"placement_violation":          "Files placed outside their required directories",
	"missing_required_group":       "Required groups with no matching file",
	"missing_required_group_match": "Required group patterns matching no directories",
	"missing_group_file":           "Directories missing files required by their group",
	"boundary_violation":           "Files importing paths forbidden by boundary rules",
	"parse_error":                  "Source files whose imports could not be parsed",
	"walk_error":                   "Filesystem errors encountered during validation",
}

// DescribeCode returns the human description for a violation code, falling
// back to the legacy "Other validation errors" for unknown codes so any
// future rule that forgets to register still surfaces in the summary.
func DescribeCode(code string) string {
	if d, ok := CodeDescriptions[code]; ok {
		return d
	}
	return "Other validation errors"
}
