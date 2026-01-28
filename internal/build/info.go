package build

import "fmt"

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
	BuiltBy = "local"
)

// String renders a single-line version string.
func String() string {
	return fmt.Sprintf("%s (commit %s) built %s by %s", Version, Commit, Date, BuiltBy)
}
