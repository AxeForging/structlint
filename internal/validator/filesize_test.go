package validator

import (
	"bytes"
	"strings"
	"testing"
)

func TestCountLines_MatchesWcL(t *testing.T) {
	cases := []struct {
		name       string
		content    string
		wantLines  int
		wantBinary bool
	}{
		{name: "empty", content: "", wantLines: 0},
		{name: "single newline", content: "\n", wantLines: 1},
		{name: "no trailing newline", content: "a\nb", wantLines: 1},
		{name: "only text no newline", content: "abc", wantLines: 0},
		{name: "trailing newline", content: "a\nb\n", wantLines: 2},
		{name: "crlf", content: "a\r\nb\r\n", wantLines: 2},
		{name: "lone cr", content: "a\rb\r", wantLines: 0},
		{name: "blank lines count", content: "\n\n\n", wantLines: 3},
		{name: "nul in header is binary", content: "ab\x00cd\n", wantBinary: true},
		{name: "nul after sniff window is text", content: strings.Repeat("a\n", binarySniffLen) + "\x00", wantLines: binarySniffLen},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, binary, err := countLines(strings.NewReader(tc.content))
			if err != nil {
				t.Fatalf("countLines error: %v", err)
			}
			if binary != tc.wantBinary {
				t.Fatalf("binary = %v, want %v", binary, tc.wantBinary)
			}
			if !tc.wantBinary && got != tc.wantLines {
				t.Fatalf("lines = %d, want %d", got, tc.wantLines)
			}
		})
	}
}

func TestCountLines_SpansReadChunks(t *testing.T) {
	// Larger than one read buffer so chunk boundaries are exercised.
	const n = 100_000
	var buf bytes.Buffer
	for i := 0; i < n; i++ {
		buf.WriteString("line of text\r\n")
	}
	got, binary, err := countLines(&buf)
	if err != nil || binary {
		t.Fatalf("unexpected err=%v binary=%v", err, binary)
	}
	if got != n {
		t.Fatalf("lines = %d, want %d", got, n)
	}
}

func TestFileSizeMatches_DoubleStarPrefixAlsoMatchesRoot(t *testing.T) {
	cases := []struct {
		rel, pattern string
		want         bool
	}{
		{"main.py", "**/*.py", true},
		{"pkg/mod/a.py", "**/*.py", true},
		{"migrations/versions/1.py", "**/migrations/versions/*.py", true},
		{"app/migrations/versions/1.py", "**/migrations/versions/*.py", true},
		{"app/migrations/1.py", "**/migrations/versions/*.py", false},
		{"tests/foo_test.py", "*_test.py", true},
		{"main.go", "**/*.py", false},
		{"legacy/x/y.py", "legacy/**", true},
	}
	for _, tc := range cases {
		if got := fileSizeMatches(tc.rel, tc.pattern); got != tc.want {
			t.Errorf("fileSizeMatches(%q, %q) = %v, want %v", tc.rel, tc.pattern, got, tc.want)
		}
	}
}
