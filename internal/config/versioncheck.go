package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/AxeForging/structlint/internal/build"
)

// requiresRE matches a `# requires structlint >= vX.Y[.Z]` comment
// anywhere in the config file. Whitespace around tokens is tolerated;
// the version literal is left unquoted, in vX.Y or vX.Y.Z form.
var requiresRE = regexp.MustCompile(`(?m)^\s*#\s*requires\s+structlint\s*>=\s*v?(\d+)\.(\d+)(?:\.(\d+))?`)

// enforceRequiresComment inspects raw config bytes for a version pragma
// and fails with a helpful message when the running binary is older than
// the required version. Skips when the running binary is a dev build —
// unstamped local builds can't be meaningfully compared to a tag.
func enforceRequiresComment(configPath string, data []byte) error {
	match := requiresRE.FindSubmatch(data)
	if match == nil {
		return nil
	}
	need, err := parseSemver(string(match[1]), string(match[2]), string(match[3]))
	if err != nil {
		return nil // malformed comment; ignore
	}
	current, ok := parseBinaryVersion(build.Version)
	if !ok {
		// dev build or otherwise unparseable; don't second-guess it.
		return nil
	}
	if semverLess(current, need) {
		return fmt.Errorf(
			"%s requires structlint >= v%s, but running version is v%s.\n"+
				"Upgrade the binary (go install github.com/AxeForging/structlint/cmd/structlint@latest) "+
				"or pin the older feature set by removing the pragma.",
			configPath, semverString(need), semverString(current),
		)
	}
	return nil
}

type semver struct{ Major, Minor, Patch int }

func parseSemver(major, minor, patch string) (semver, error) {
	M, err := strconv.Atoi(major)
	if err != nil {
		return semver{}, err
	}
	m, err := strconv.Atoi(minor)
	if err != nil {
		return semver{}, err
	}
	p := 0
	if patch != "" {
		p, err = strconv.Atoi(patch)
		if err != nil {
			return semver{}, err
		}
	}
	return semver{Major: M, Minor: m, Patch: p}, nil
}

// parseBinaryVersion extracts a semver from build.Version. Accepts the
// leading `v`, tolerates trailing metadata (e.g. `v0.6.0-3-gabcdef-dirty`
// or `v0.6.0+meta`). Returns ok=false for `dev` or unparseable inputs so
// callers can skip the check.
func parseBinaryVersion(v string) (semver, bool) {
	v = strings.TrimSpace(v)
	if v == "" || v == "dev" || v == "unknown" {
		return semver{}, false
	}
	v = strings.TrimPrefix(v, "v")
	// Split on the first hyphen or plus so metadata like `-3-gabc-dirty` or
	// `+meta` doesn't confuse the parse.
	for _, sep := range []string{"-", "+"} {
		if idx := strings.Index(v, sep); idx > 0 {
			v = v[:idx]
		}
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return semver{}, false
	}
	patch := "0"
	if len(parts) >= 3 {
		patch = parts[2]
	}
	sv, err := parseSemver(parts[0], parts[1], patch)
	if err != nil {
		return semver{}, false
	}
	return sv, true
}

func semverLess(a, b semver) bool {
	if a.Major != b.Major {
		return a.Major < b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor < b.Minor
	}
	return a.Patch < b.Patch
}

func semverString(s semver) string {
	return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
}
