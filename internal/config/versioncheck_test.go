package config

import (
	"strings"
	"testing"

	"github.com/AxeForging/structlint/internal/build"
)

// withBuildVersion temporarily overrides build.Version for the duration
// of fn, restoring it via t.Cleanup so parallel tests don't leak state.
func withBuildVersion(t *testing.T, v string) {
	t.Helper()
	prev := build.Version
	build.Version = v
	t.Cleanup(func() { build.Version = prev })
}

func TestEnforceRequiresComment_NoPragmaIsFine(t *testing.T) {
	withBuildVersion(t, "v0.5.0")
	err := enforceRequiresComment("test.yaml", []byte("dir_structure:\n  allowedPaths: [\".\"]\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnforceRequiresComment_SatisfiedVersion(t *testing.T) {
	withBuildVersion(t, "v0.6.1")
	err := enforceRequiresComment("test.yaml", []byte(`# requires structlint >= v0.6.0
extends: go-standard
`))
	if err != nil {
		t.Fatalf("v0.6.1 should satisfy >= v0.6.0, got: %v", err)
	}
}

func TestEnforceRequiresComment_ExactVersion(t *testing.T) {
	withBuildVersion(t, "v0.6.0")
	err := enforceRequiresComment("test.yaml", []byte("# requires structlint >= v0.6.0\n"))
	if err != nil {
		t.Fatalf("v0.6.0 should satisfy >= v0.6.0, got: %v", err)
	}
}

func TestEnforceRequiresComment_TooOld(t *testing.T) {
	withBuildVersion(t, "v0.5.0")
	err := enforceRequiresComment("path.yaml", []byte("# requires structlint >= v0.6.0\n"))
	if err == nil {
		t.Fatal("expected error when running version is older than required")
	}
	msg := err.Error()
	if !strings.Contains(msg, "requires structlint >= v0.6.0") {
		t.Errorf("error should quote the required version, got: %s", msg)
	}
	if !strings.Contains(msg, "running version is v0.5.0") {
		t.Errorf("error should quote the running version, got: %s", msg)
	}
	if !strings.Contains(msg, "path.yaml") {
		t.Errorf("error should reference the config path, got: %s", msg)
	}
}

func TestEnforceRequiresComment_DevVersionSkipsCheck(t *testing.T) {
	withBuildVersion(t, "dev")
	err := enforceRequiresComment("test.yaml", []byte("# requires structlint >= v99.0.0\n"))
	if err != nil {
		t.Fatalf("dev binary should never fail the pragma check, got: %v", err)
	}
}

func TestEnforceRequiresComment_UnstampedDirtyVersion(t *testing.T) {
	// Build metadata (e.g. `-3-gabcdef-dirty`) should not confuse the check.
	withBuildVersion(t, "v0.6.0-3-gabcdef-dirty")
	err := enforceRequiresComment("test.yaml", []byte("# requires structlint >= v0.6.0\n"))
	if err != nil {
		t.Fatalf("v0.6.0-with-metadata should satisfy >= v0.6.0, got: %v", err)
	}
}

func TestEnforceRequiresComment_MinorVersionOnly(t *testing.T) {
	// Pragma without patch should still work.
	withBuildVersion(t, "v0.5.9")
	err := enforceRequiresComment("test.yaml", []byte("# requires structlint >= v0.6\n"))
	if err == nil {
		t.Fatal("expected error: v0.5.9 < v0.6.0")
	}
}

func TestEnforceRequiresComment_MalformedPragmaIgnored(t *testing.T) {
	withBuildVersion(t, "v0.5.0")
	// A comment mentioning the phrase but with garbage version → tolerate.
	err := enforceRequiresComment("test.yaml", []byte("# structlint is a linter\n"))
	if err != nil {
		t.Fatalf("comment without a real pragma should not error: %v", err)
	}
}

func TestParseBinaryVersion_DevBuild(t *testing.T) {
	if _, ok := parseBinaryVersion("dev"); ok {
		t.Error("dev version must return ok=false")
	}
	if _, ok := parseBinaryVersion(""); ok {
		t.Error("empty version must return ok=false")
	}
	if _, ok := parseBinaryVersion("unknown"); ok {
		t.Error("unknown version must return ok=false")
	}
}

func TestParseBinaryVersion_WithMetadata(t *testing.T) {
	sv, ok := parseBinaryVersion("v1.2.3-rc.1")
	if !ok {
		t.Fatal("expected parse to succeed")
	}
	if sv.Major != 1 || sv.Minor != 2 || sv.Patch != 3 {
		t.Errorf("got %+v, want 1.2.3", sv)
	}
}
