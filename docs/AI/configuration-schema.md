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

ignore: []              # []string - exact paths or patterns
```

## Go Structs

```go
// internal/config/config.go

type Config struct {
    DirStructure      DirStructureConfig      `yaml:"dir_structure" json:"dir_structure"`
    FileNamingPattern FileNamingPatternConfig `yaml:"file_naming_pattern" json:"file_naming_pattern"`
    Ignore            []string                `yaml:"ignore" json:"ignore"`
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

## JSON Report Schema

```go
// internal/validator/types.go

type JSONReport struct {
    Successes int               `json:"successes"`
    Failures  int               `json:"failures"`
    Errors    []string          `json:"errors"`
    Summary   ValidationSummary `json:"summary,omitempty"`
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
