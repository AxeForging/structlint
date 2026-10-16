package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestFileSizeValidate(t *testing.T) {
	cases := []struct {
		name    string
		fs      FileSize
		wantErr string
	}{
		{name: "empty block is valid", fs: FileSize{}},
		{name: "valid rule", fs: FileSize{Rules: []FileSizeRule{{Pattern: "**/*.py", MaxLines: 300}}}},
		{name: "warning severity", fs: FileSize{Rules: []FileSizeRule{{Pattern: "*.go", MaxLines: 1, Severity: "warning"}}}},
		{name: "zero", fs: FileSize{Rules: []FileSizeRule{{Pattern: "*.py"}}}, wantErr: "file_size.rules[0] (*.py): maxLines must be greater than 0, got 0"},
		{name: "negative", fs: FileSize{Rules: []FileSizeRule{{Pattern: "*.py", MaxLines: -1}}}, wantErr: "maxLines must be greater than 0, got -1"},
		{name: "second rule reported by index", fs: FileSize{Rules: []FileSizeRule{{Pattern: "*.py", MaxLines: 1}, {Pattern: "*.go"}}}, wantErr: "file_size.rules[1] (*.go)"},
		{name: "missing pattern", fs: FileSize{Rules: []FileSizeRule{{MaxLines: 3}}}, wantErr: "file_size.rules[0]: pattern is required"},
		{name: "bad glob", fs: FileSize{Rules: []FileSizeRule{{Pattern: "[", MaxLines: 3}}}, wantErr: `file_size.rules[0]: invalid pattern "["`},
		{name: "bad exclude", fs: FileSize{Exclude: []string{"ok", "[a"}}, wantErr: `file_size.exclude[1]: invalid pattern "[a"`},
		{name: "empty exclude", fs: FileSize{Exclude: []string{""}}, wantErr: "file_size.exclude[0]: pattern is required"},
		{name: "bad severity", fs: FileSize{Rules: []FileSizeRule{{Pattern: "*.py", MaxLines: 3, Severity: "info"}}}, wantErr: `severity must be "error" or "warning", got "info"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := (&Config{FileSize: tc.fs}).Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestMergeFileSize_ChildRulesFirstAndOverrideSamePattern(t *testing.T) {
	parent := FileSize{
		Rules: []FileSizeRule{
			{Pattern: "**/*.py", MaxLines: 300},
			{Pattern: "**/*.go", MaxLines: 500},
		},
		Exclude: []string{"**/*_test.py"},
	}
	child := FileSize{
		Rules: []FileSizeRule{
			{Pattern: "legacy/**", MaxLines: 2000},
			{Pattern: "**/*.py", MaxLines: 400},
		},
		Exclude: []string{"**/*.lock", "**/*_test.py"},
	}
	got := mergeFileSize(parent, child)
	want := FileSize{
		Rules: []FileSizeRule{
			{Pattern: "legacy/**", MaxLines: 2000},
			{Pattern: "**/*.py", MaxLines: 400},
			{Pattern: "**/*.go", MaxLines: 500},
		},
		Exclude: []string{"**/*_test.py", "**/*.lock"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merge mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestMergeFileSize_EmptySidesKeepOther(t *testing.T) {
	p := FileSize{Rules: []FileSizeRule{{Pattern: "*.py", MaxLines: 1}}}
	if got := mergeFileSize(p, FileSize{}); !reflect.DeepEqual(got, p) {
		t.Fatalf("empty child changed parent: %+v", got)
	}
	if got := mergeFileSize(FileSize{}, p); !reflect.DeepEqual(got, p) {
		t.Fatalf("empty parent changed child: %+v", got)
	}
	if got := mergeFileSize(FileSize{}, FileSize{}); !reflect.DeepEqual(got, FileSize{}) {
		t.Fatalf("both empty should stay empty: %+v", got)
	}
}
