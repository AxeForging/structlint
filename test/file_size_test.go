package test

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// fileSizeConfig wraps a file_size block in a permissive base config so the
// only rule family that can fail is file_size.
func fileSizeConfig(fileSize string) string {
	return `dir_structure:
  allowedPaths: [".", "**"]
file_naming_pattern:
  allowed: ["*"]
ignore: [".git"]
` + fileSize
}

type fileSizeViolation struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Rule     string `json:"rule"`
	Message  string `json:"message"`
}

type fileSizeReport struct {
	Violations []fileSizeViolation `json:"violations"`
}

// runFileSizeJSON runs `validate --format json` in dir and returns the
// decoded report plus the exit error (nil on exit 0).
func runFileSizeJSON(t *testing.T, dir string) (fileSizeReport, error) {
	t.Helper()
	bin := buildBinary(t)
	cmd := exec.Command(bin, "validate", "--format", "json")
	cmd.Dir = dir
	stdout, err := cmd.Output()
	var report fileSizeReport
	if jerr := json.Unmarshal(stdout, &report); jerr != nil {
		t.Fatalf("decode json report: %v\nstdout:\n%s", jerr, stdout)
	}
	return report, err
}

func fileTooLong(report fileSizeReport) []fileSizeViolation {
	var out []fileSizeViolation
	for _, v := range report.Violations {
		if v.Code == "file_too_long" {
			out = append(out, v)
		}
	}
	return out
}

func lines(n int, eol string) string {
	return strings.Repeat("x"+eol, n)
}

const pyLimit3 = `file_size:
  rules:
    - pattern: "**/*.py"
      maxLines: 3
`

func TestFileSize_ExactlyMaxLinesPasses(t *testing.T) {
	dir := createTestProject(t, map[string]string{"src/a.py": lines(3, "\n")}, fileSizeConfig(pyLimit3))
	report, err := runFileSizeJSON(t, dir)
	if err != nil {
		t.Fatalf("expected exit 0 at exactly maxLines, got %v: %+v", err, report.Violations)
	}
	if got := fileTooLong(report); len(got) != 0 {
		t.Fatalf("expected no file_too_long, got %+v", got)
	}
}

func TestFileSize_OneOverMaxLinesFails(t *testing.T) {
	dir := createTestProject(t, map[string]string{"src/a.py": lines(4, "\n")}, fileSizeConfig(pyLimit3))
	report, err := runFileSizeJSON(t, dir)
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() == 0 {
		t.Fatalf("expected non-zero exit, got %v", err)
	}
	got := fileTooLong(report)
	if len(got) != 1 {
		t.Fatalf("expected one file_too_long, got %+v", report.Violations)
	}
	want := fileSizeViolation{
		Code:     "file_too_long",
		Severity: "error",
		Path:     "src/a.py",
		Rule:     "**/*.py",
		Message:  "File too long: src/a.py has 4 lines (max 3)",
	}
	if got[0] != want {
		t.Fatalf("violation mismatch:\n got %+v\nwant %+v", got[0], want)
	}
}

func TestFileSize_LineCounting(t *testing.T) {
	// Counting follows `wc -l`: the number of "\n" bytes.
	cases := []struct {
		name      string
		content   string
		wantLines int // 0 means "must pass" with maxLines 3
	}{
		{name: "empty file passes", content: ""},
		{name: "no trailing newline is not counted", content: "a\nb\nc\nd"},
		{name: "trailing newline counted", content: "a\nb\nc\nd\n", wantLines: 4},
		{name: "CRLF counts once per line", content: lines(4, "\r\n"), wantLines: 4},
		{name: "CRLF at limit passes", content: lines(3, "\r\n")},
		{name: "lone CR is not a line break", content: "a\rb\rc\rd\re\r"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := createTestProject(t, map[string]string{"a.py": tc.content}, fileSizeConfig(pyLimit3))
			report, err := runFileSizeJSON(t, dir)
			got := fileTooLong(report)
			if tc.wantLines == 0 {
				if err != nil || len(got) != 0 {
					t.Fatalf("expected pass, got err=%v violations=%+v", err, got)
				}
				return
			}
			if err == nil || len(got) != 1 {
				t.Fatalf("expected one violation and failure, got err=%v violations=%+v", err, got)
			}
			if !strings.Contains(got[0].Message, " has "+itoa(tc.wantLines)+" lines (max 3)") {
				t.Fatalf("unexpected message %q", got[0].Message)
			}
		})
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

func TestFileSize_ExcludedFilesOverLimitPass(t *testing.T) {
	dir := createTestProject(t, map[string]string{
		"app/service_test.py":               lines(10, "\n"),
		"app/migrations/versions/0001_x.py": lines(10, "\n"),
		"migrations/versions/0002_root.py":  lines(10, "\n"),
		"app/service.py":                    lines(10, "\n"),
		"poetry.lock":                       lines(10, "\n"),
	}, fileSizeConfig(`file_size:
  rules:
    - pattern: "**/*.py"
      maxLines: 3
    - pattern: "*.lock"
      maxLines: 3
  exclude: ["**/*_test.py", "**/migrations/versions/*.py", "**/*.lock"]
`))
	report, _ := runFileSizeJSON(t, dir)
	got := fileTooLong(report)
	if len(got) != 1 || got[0].Path != "app/service.py" {
		t.Fatalf("expected only app/service.py to fail, got %+v", got)
	}
}

func TestFileSize_IgnoredDirectorySkipped(t *testing.T) {
	dir := createTestProject(t, map[string]string{
		"vendor/big.py": lines(50, "\n"),
		"ok.py":         lines(1, "\n"),
	}, `dir_structure:
  allowedPaths: [".", "**"]
file_naming_pattern:
  allowed: ["*"]
ignore: [".git", "vendor"]
`+pyLimit3)
	report, err := runFileSizeJSON(t, dir)
	if err != nil || len(fileTooLong(report)) != 0 {
		t.Fatalf("ignored dir must be skipped, got err=%v violations=%+v", err, report.Violations)
	}
}

func TestFileSize_FirstMatchingRuleWins(t *testing.T) {
	files := map[string]string{
		"legacy/old.py": lines(5, "\n"),
		"src/new.py":    lines(5, "\n"),
	}

	t.Run("specific rule first", func(t *testing.T) {
		dir := createTestProject(t, files, fileSizeConfig(`file_size:
  rules:
    - pattern: "legacy/**"
      maxLines: 10
    - pattern: "**/*.py"
      maxLines: 3
`))
		report, _ := runFileSizeJSON(t, dir)
		got := fileTooLong(report)
		if len(got) != 1 || got[0].Path != "src/new.py" || got[0].Rule != "**/*.py" {
			t.Fatalf("expected only src/new.py via **/*.py, got %+v", got)
		}
	})

	t.Run("broad rule first shadows later rule", func(t *testing.T) {
		dir := createTestProject(t, files, fileSizeConfig(`file_size:
  rules:
    - pattern: "**/*.py"
      maxLines: 3
    - pattern: "legacy/**"
      maxLines: 10
`))
		report, _ := runFileSizeJSON(t, dir)
		got := fileTooLong(report)
		if len(got) != 2 {
			t.Fatalf("expected both files to fail under the first rule, got %+v", got)
		}
		for _, v := range got {
			if v.Rule != "**/*.py" {
				t.Errorf("%s matched %q, want first rule **/*.py", v.Path, v.Rule)
			}
		}
	})
}

func TestFileSize_DoubleStarPrefixMatchesRootFiles(t *testing.T) {
	dir := createTestProject(t, map[string]string{"main.py": lines(4, "\n")}, fileSizeConfig(pyLimit3))
	report, _ := runFileSizeJSON(t, dir)
	got := fileTooLong(report)
	if len(got) != 1 || got[0].Path != "main.py" {
		t.Fatalf("expected root main.py to be checked by **/*.py, got %+v", got)
	}
}

func TestFileSize_UnmatchedFilesNotChecked(t *testing.T) {
	dir := createTestProject(t, map[string]string{"notes.md": lines(100, "\n")}, fileSizeConfig(pyLimit3))
	report, err := runFileSizeJSON(t, dir)
	if err != nil || len(fileTooLong(report)) != 0 {
		t.Fatalf("files matching no rule must not be checked, got err=%v %+v", err, report.Violations)
	}
}

func TestFileSize_BinaryFileSkipped(t *testing.T) {
	binary := "\x89PNG\x00\x00" + lines(50, "\n")
	dir := createTestProject(t, map[string]string{"assets/logo.py": binary}, fileSizeConfig(pyLimit3))
	report, err := runFileSizeJSON(t, dir)
	if err != nil || len(fileTooLong(report)) != 0 {
		t.Fatalf("binary file must be skipped, got err=%v %+v", err, report.Violations)
	}
}

func TestFileSize_InvalidConfigRejected(t *testing.T) {
	cases := []struct {
		name    string
		block   string
		wantMsg string
	}{
		{
			name:    "zero maxLines",
			block:   "file_size:\n  rules:\n    - pattern: \"*.py\"\n      maxLines: 0\n",
			wantMsg: `file_size.rules[0] (*.py): maxLines must be greater than 0, got 0`,
		},
		{
			name:    "negative maxLines",
			block:   "file_size:\n  rules:\n    - pattern: \"*.py\"\n      maxLines: -5\n",
			wantMsg: `file_size.rules[0] (*.py): maxLines must be greater than 0, got -5`,
		},
		{
			name:    "missing maxLines",
			block:   "file_size:\n  rules:\n    - pattern: \"*.py\"\n",
			wantMsg: `file_size.rules[0] (*.py): maxLines must be greater than 0, got 0`,
		},
		{
			name:    "missing pattern",
			block:   "file_size:\n  rules:\n    - maxLines: 10\n",
			wantMsg: `file_size.rules[0]: pattern is required`,
		},
		{
			name:    "bad rule glob",
			block:   "file_size:\n  rules:\n    - pattern: \"src/[a.py\"\n      maxLines: 10\n",
			wantMsg: `file_size.rules[0]: invalid pattern "src/[a.py"`,
		},
		{
			name:    "bad exclude glob",
			block:   "file_size:\n  rules:\n    - pattern: \"*.py\"\n      maxLines: 10\n  exclude: [\"vendor/[ab\"]\n",
			wantMsg: `file_size.exclude[0]: invalid pattern "vendor/[ab"`,
		},
		{
			name:    "bad severity",
			block:   "file_size:\n  rules:\n    - pattern: \"*.py\"\n      maxLines: 10\n      severity: fatal\n",
			wantMsg: `file_size.rules[0] (*.py): severity must be "error" or "warning", got "fatal"`,
		},
		{
			name:    "snake_case key is rejected by strict parsing",
			block:   "file_size:\n  rules:\n    - pattern: \"*.py\"\n      max_lines: 10\n",
			wantMsg: "max_lines",
		},
	}
	bin := buildBinary(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := createTestProject(t, map[string]string{"a.py": "x\n"}, fileSizeConfig(tc.block))
			out, err := runBinaryInDir(t, bin, dir, "validate")
			if err == nil {
				t.Fatalf("expected config error, got exit 0:\n%s", out)
			}
			if !strings.Contains(out, tc.wantMsg) {
				t.Fatalf("output missing %q:\n%s", tc.wantMsg, out)
			}
		})
	}
}

func TestFileSize_OutputFormats(t *testing.T) {
	bin := buildBinary(t)
	dir := createTestProject(t, map[string]string{"a.py": lines(4, "\n")}, fileSizeConfig(pyLimit3))

	t.Run("github", func(t *testing.T) {
		out, err := runBinaryInDir(t, bin, dir, "validate", "--format", "github")
		if err == nil {
			t.Fatal("expected non-zero exit")
		}
		want := "::error file=a.py,title=file_too_long::File too long: a.py has 4 lines (max 3)"
		if !strings.Contains(out, want) {
			t.Fatalf("missing annotation %q in:\n%s", want, out)
		}
	})

	t.Run("sarif", func(t *testing.T) {
		out, err := runBinaryInDir(t, bin, dir, "validate", "--format", "sarif", "--silent")
		if err == nil {
			t.Fatal("expected non-zero exit")
		}
		if !strings.Contains(out, `"ruleId": "file_too_long"`) || !strings.Contains(out, `"uri": "a.py"`) {
			t.Fatalf("sarif missing file_too_long result:\n%s", out)
		}
	})

	t.Run("text", func(t *testing.T) {
		out, err := runBinaryInDir(t, bin, dir, "validate")
		if err == nil {
			t.Fatal("expected non-zero exit")
		}
		for _, want := range []string{"Files exceeding their maximum line count", "File too long: a.py has 4 lines (max 3)"} {
			if !strings.Contains(out, want) {
				t.Fatalf("text output missing %q:\n%s", want, out)
			}
		}
	})
}

func TestFileSize_WarningSeverity(t *testing.T) {
	bin := buildBinary(t)
	dir := createTestProject(t, map[string]string{"a.go": lines(4, "\n")}, fileSizeConfig(`file_size:
  rules:
    - pattern: "*.go"
      maxLines: 3
      severity: warning
`))
	out, err := runBinaryInDir(t, bin, dir, "validate", "--format", "github")
	if err == nil {
		t.Fatal("warnings fail validation like every other rule family")
	}
	if !strings.Contains(out, "::warning file=a.go,title=file_too_long::") {
		t.Fatalf("expected warning annotation:\n%s", out)
	}
}

func TestFileSize_StagedOnlyChecksStagedFiles(t *testing.T) {
	dir := createTestProject(t, map[string]string{"old.py": lines(10, "\n")}, fileSizeConfig(pyLimit3))
	initGitRepo(t, dir)
	gitCommit(t, dir, "init")
	writeTestFile(t, dir, "new.py", lines(10, "\n"))
	gitStage(t, dir, "new.py")

	bin := buildBinary(t)
	cmd := exec.Command(bin, "validate", "--staged", "--format", "json")
	cmd.Dir = dir
	stdout, _ := cmd.Output()
	var report fileSizeReport
	if err := json.Unmarshal(stdout, &report); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout)
	}
	got := fileTooLong(report)
	if len(got) != 1 || got[0].Path != "new.py" {
		t.Fatalf("expected only staged new.py, got %+v", got)
	}
}

func TestFileSize_ExtendsChildOverridesParentLimit(t *testing.T) {
	files := map[string]string{
		"base.yaml": fileSizeConfig(pyLimit3),
		"a.py":      lines(5, "\n"),
	}

	t.Run("inherits parent rule", func(t *testing.T) {
		dir := createTestProject(t, files, "extends: ./base.yaml\n")
		report, err := runFileSizeJSON(t, dir)
		if err == nil || len(fileTooLong(report)) != 1 {
			t.Fatalf("expected inherited limit to fail a.py, got err=%v %+v", err, report.Violations)
		}
	})

	t.Run("child rule for same pattern wins", func(t *testing.T) {
		dir := createTestProject(t, files, `extends: ./base.yaml
file_size:
  rules:
    - pattern: "**/*.py"
      maxLines: 10
`)
		report, err := runFileSizeJSON(t, dir)
		if err != nil || len(fileTooLong(report)) != 0 {
			t.Fatalf("expected child limit 10 to win, got err=%v %+v", err, report.Violations)
		}
	})

	t.Run("child exclude appends to parent", func(t *testing.T) {
		dir := createTestProject(t, files, `extends: ./base.yaml
file_size:
  exclude: ["a.py"]
`)
		report, err := runFileSizeJSON(t, dir)
		if err != nil || len(fileTooLong(report)) != 0 {
			t.Fatalf("expected child exclude to apply, got err=%v %+v", err, report.Violations)
		}
	})
}

func TestFileSize_NoConfigBlockNoBehaviorChange(t *testing.T) {
	dir := createTestProject(t, map[string]string{"a.py": lines(5000, "\n")}, fileSizeConfig(""))
	report, err := runFileSizeJSON(t, dir)
	if err != nil || len(report.Violations) != 0 {
		t.Fatalf("no file_size block must mean no size checks, got err=%v %+v", err, report.Violations)
	}
}

// uncommentFileSize turns the commented `# file_size:` example shipped in
// init templates into live YAML, so the example is proven to stay valid.
func uncommentFileSize(t *testing.T, cfg string) string {
	t.Helper()
	var out []string
	inBlock := false
	for _, line := range strings.Split(cfg, "\n") {
		if strings.HasPrefix(line, "# file_size:") {
			inBlock = true
		} else if inBlock && !strings.HasPrefix(line, "#  ") {
			inBlock = false
		}
		if inBlock {
			line = strings.TrimPrefix(line, "# ")
		}
		out = append(out, line)
	}
	if !strings.Contains(strings.Join(out, "\n"), "\nfile_size:\n") {
		t.Fatalf("no commented file_size example found in:\n%s", cfg)
	}
	return strings.Join(out, "\n")
}

func TestFileSize_InitTemplateExamplesAreValid(t *testing.T) {
	bin := buildBinary(t)
	cases := []struct {
		typ      string
		tooLong  string
		excluded string
		limit    int
	}{
		{typ: "python", tooLong: "src/app.py", excluded: "tests/test_app.py", limit: 300},
		{typ: "go", tooLong: "internal/app.go", excluded: "internal/app_test.go", limit: 500},
		{typ: "node", tooLong: "src/app.ts", excluded: "src/app.test.ts", limit: 400},
	}
	for _, tc := range cases {
		t.Run(tc.typ, func(t *testing.T) {
			dir := createTestProject(t, map[string]string{
				tc.tooLong:  lines(tc.limit+1, "\n"),
				tc.excluded: lines(tc.limit*2, "\n"),
			}, "")
			if out, err := runBinaryInDir(t, bin, dir, "init", "--type", tc.typ); err != nil {
				t.Fatalf("init failed: %v\n%s", err, out)
			}
			cfgPath := dir + "/.structlint.yaml"
			data := readFile(t, cfgPath)
			writeTestFile(t, dir, ".structlint.yaml", uncommentFileSize(t, data))

			report, _ := runFileSizeJSON(t, dir)
			got := fileTooLong(report)
			if len(got) != 1 || got[0].Path != tc.tooLong {
				t.Fatalf("expected only %s over the template limit, got %+v", tc.tooLong, got)
			}
		})
	}
}

func TestFileSize_InferOutputHintIsValid(t *testing.T) {
	bin := buildBinary(t)
	dir := createTestProject(t, map[string]string{
		"app/big.py":      lines(301, "\n"),
		"app/big_test.py": lines(900, "\n"),
		"README.md":       "# x\n",
	}, "")
	if out, err := runBinaryInDir(t, bin, dir, "init", "--infer"); err != nil {
		t.Fatalf("init --infer failed: %v\n%s", err, out)
	}
	if out, err := runBinaryInDir(t, bin, dir, "validate", "--silent"); err != nil {
		t.Fatalf("inferred baseline must pass as-is (hint is commented): %v\n%s", err, out)
	}
	data := readFile(t, dir+"/.structlint.yaml")
	writeTestFile(t, dir, ".structlint.yaml", uncommentFileSize(t, data))
	report, _ := runFileSizeJSON(t, dir)
	got := fileTooLong(report)
	if len(got) != 1 || got[0].Path != "app/big.py" {
		t.Fatalf("expected uncommented hint to flag app/big.py only, got %+v", got)
	}
}
