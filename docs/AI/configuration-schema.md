# Configuration Schema

## YAML Structure

```yaml
dir_structure:
  allowedPaths: []      # []string - glob patterns
  disallowedPaths: []   # []string - glob patterns
  requiredPaths: []     # []string - exact paths

file_naming_pattern:
  allowed: []           # []string - glob patterns
  disallowed: []        # []string - glob patterns
  required: []          # []string - glob patterns

placement: []           # []PlacementRule - file placement contracts
requiredGroups: []      # []RequiredGroup - one-of and per-directory requirements
boundaries: []          # []BoundaryRule - import boundary contracts
ignore: []              # []string - exact paths or patterns
```

## Go Structs

```go
// internal/config/config.go

type Config struct {
    DirStructure      DirStructureConfig      `yaml:"dir_structure" json:"dir_structure"`
    FileNamingPattern FileNamingPatternConfig `yaml:"file_naming_pattern" json:"file_naming_pattern"`
    Ignore            []string                `yaml:"ignore" json:"ignore"`
    Placement         []PlacementRule         `yaml:"placement" json:"placement"`
    RequiredGroups    []RequiredGroup         `yaml:"requiredGroups" json:"requiredGroups"`
    Boundaries        []BoundaryRule          `yaml:"boundaries" json:"boundaries"`
}

type DirStructureConfig struct {
    AllowedPaths    []string `yaml:"allowedPaths" json:"allowedPaths"`
    DisallowedPaths []string `yaml:"disallowedPaths" json:"disallowedPaths"`
    RequiredPaths   []string `yaml:"requiredPaths" json:"requiredPaths"`
}

type FileNamingPatternConfig struct {
    Allowed    []string `yaml:"allowed" json:"allowed"`
    Disallowed []string `yaml:"disallowed" json:"disallowed"`
    Required   []string `yaml:"required" json:"required"`
}

type PlacementRule struct {
    ID          string   `yaml:"id" json:"id"`
    Files       []string `yaml:"files" json:"files"`
    MustBeUnder []string `yaml:"mustBeUnder" json:"mustBeUnder"`
    Severity    string   `yaml:"severity" json:"severity"`
}

type RequiredGroup struct {
    ID               string   `yaml:"id" json:"id"`
    OneOf            []string `yaml:"oneOf" json:"oneOf"`
    EachDirMatching  string   `yaml:"eachDirMatching" json:"eachDirMatching"`
    MustContain      []string `yaml:"mustContain" json:"mustContain"`
    MustContainOneOf []string `yaml:"mustContainOneOf" json:"mustContainOneOf"`
    RequireMatch     bool     `yaml:"requireMatch" json:"requireMatch"`
    Severity         string   `yaml:"severity" json:"severity"`
}

type BoundaryRule struct {
    ID           string   `yaml:"id" json:"id"`
    From         string   `yaml:"from" json:"from"`
    CannotImport []string `yaml:"cannotImport" json:"cannotImport"`
    Severity     string   `yaml:"severity" json:"severity"`
}
```

## Glob Pattern Syntax

Uses `github.com/gobwas/glob`:

| Pattern | Matches |
|---------|---------|
| `*` | Any sequence except `/` |
| `**` | Any sequence including `/` |
| `?` | Any single character |
| `[abc]` | Any char in set |
| `[!abc]` | Any char not in set |
| `{a,b}` | Either a or b |

## Validation Rules

### Directory Structure

**allowedPaths**: If non-empty, only directories matching these patterns are allowed.

```yaml
allowedPaths:
  - "."           # Root only
  - "cmd/**"      # cmd and all descendants
  - "src/*"       # Direct children of src
```

**disallowedPaths**: Directories matching these are violations.

```yaml
disallowedPaths:
  - "vendor/**"
  - "node_modules/**"
  - "tmp/**"
```

**requiredPaths**: These directories must exist.

```yaml
requiredPaths:
  - "cmd"
  - "internal"
```

### File Naming

**allowed**: If non-empty, only files matching these patterns are allowed.

```yaml
allowed:
  - "*.go"
  - "*.yaml"
  - "Makefile"
```

**disallowed**: Files matching these are violations.

```yaml
disallowed:
  - "*.env*"
  - "*.log"
```

**required**: At least one file matching each pattern must exist.

```yaml
required:
  - "go.mod"
  - "*.go"  # At least one Go file
```

### Ignore

Paths to skip completely during validation.

```yaml
ignore:
  - ".git"
  - "vendor"
  - "bin"
```

### Placement

**placement**: Matching files must be under one of the configured roots.

```yaml
placement:
  - id: sql-in-migrations
    files: ["*.sql"]
    mustBeUnder: ["migrations/**"]
```

### Required Groups

**requiredGroups**: Higher-level required-file contracts.

```yaml
requiredGroups:
  - id: build-entrypoint
    oneOf: ["Makefile", "Taskfile.yml", "justfile"]
  - id: commands-have-main
    eachDirMatching: "cmd/*"
    mustContain: ["main.go"]
```

### Boundaries

**boundaries**: Import boundary rules for Go, JS/TS, and Python source files.

```yaml
boundaries:
  - id: domain-no-db
    from: "internal/domain/**"
    cannotImport: ["internal/db/**"]
```

## JSON Report Schema

```go
// internal/validator/types.go

type JSONReport struct {
    Successes int               `json:"successes"`
    Failures  int               `json:"failures"`
    TotalViolations int         `json:"total_violations"`
    Errors    []string          `json:"errors"`
    Violations []Violation      `json:"violations"`
    Summary   ValidationSummary `json:"summary,omitempty"`
}

type Violation struct {
    Code     string `json:"code"`
    Severity string `json:"severity"`
    Path     string `json:"path"`
    Rule     string `json:"rule"`
    Message  string `json:"message"`
}

type ValidationSummary struct {
    DirsChecked      int            `json:"directories_checked"`
    FilesChecked     int            `json:"files_checked"`
    ViolationsByType map[string]int `json:"violations_by_type"`
}
```

Example output:

```json
{
  "successes": 42,
  "failures": 2,
  "errors": [
    "Directory not in allowed list: tmp",
    "Disallowed file: .env.local"
  ],
  "summary": {
    "directories_checked": 15,
    "files_checked": 27,
    "violations_by_type": {
      "dir_not_allowed": 1,
      "file_disallowed": 1
    }
  }
}
```

## Example Configurations

### Minimal Go Project

```yaml
dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**", "pkg/**"]

file_naming_pattern:
  allowed: ["*.go", "*.mod", "*.sum", "*.yaml", "*.md"]
  disallowed: ["*.env*"]

ignore: [".git", "vendor"]
```

### Strict Mode

```yaml
dir_structure:
  allowedPaths: [".", "cmd/**", "internal/**"]
  disallowedPaths: ["vendor/**", "tmp/**", "node_modules/**"]
  requiredPaths: ["cmd", "internal"]

file_naming_pattern:
  allowed: ["*.go", "*.mod", "*.sum", "*.yaml", "*.md", "Makefile"]
  disallowed: ["*.env*", "*.log", "*.tmp", "*~", ".DS_Store"]
  required: ["go.mod", "README.md", ".gitignore"]

ignore: [".git", "vendor", "bin", "dist"]
```
