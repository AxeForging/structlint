package validator

import (
	"bufio"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	jsImportPattern      = regexp.MustCompile(`^\s*import(?:\s+[^'"]+\s+from\s+)?['"]([^'"]+)['"]`)
	jsRequirePattern     = regexp.MustCompile(`require\(\s*['"]([^'"]+)['"]\s*\)`)
	pythonImportPattern  = regexp.MustCompile(`^\s*import\s+([A-Za-z0-9_./]+)`)
	pythonFromPattern    = regexp.MustCompile(`^\s*from\s+([A-Za-z0-9_./]+)\s+import\s+`)
	supportedBoundaryExt = map[string]bool{
		".go":  true,
		".js":  true,
		".jsx": true,
		".ts":  true,
		".tsx": true,
		".mjs": true,
		".cjs": true,
		".py":  true,
	}
)

func cleanRoot(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return abs
}

func relativePath(root, currentPath string) string {
	abs, err := filepath.Abs(currentPath)
	if err != nil {
		return normalizePath(currentPath)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return normalizePath(currentPath)
	}
	return normalizePath(rel)
}

func normalizePath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	if path == "" || path == "." {
		return "."
	}
	return strings.TrimPrefix(path, "./")
}

func pathMatches(path, pattern string) bool {
	path = normalizePath(path)
	pattern = normalizePath(pattern)
	if pattern == "." {
		return path == "."
	}
	if strings.HasSuffix(pattern, "/**") {
		base := strings.TrimSuffix(pattern, "/**")
		if path == base || strings.HasPrefix(path, base+"/") {
			return true
		}
	}
	return matches(path, pattern)
}

func isParentOfPattern(path, pattern string) bool {
	path = normalizePath(path)
	pattern = normalizePath(pattern)
	if path == "." {
		return true
	}
	base := patternRoot(pattern)
	return base == path || (base != "" && strings.HasPrefix(base, path+"/"))
}

func patternRoot(pattern string) string {
	pattern = normalizePath(pattern)
	cut := len(pattern)
	for _, marker := range []string{"*", "?", "[", "{"} {
		if idx := strings.Index(pattern, marker); idx >= 0 && idx < cut {
			cut = idx
		}
	}
	root := strings.TrimSuffix(pattern[:cut], "/")
	if idx := strings.LastIndex(root, "/"); idx >= 0 {
		root = root[:idx]
	}
	if root == "" {
		return "."
	}
	return root
}

func matchesAnyFile(relPath, fileName string, patterns []string) bool {
	for _, pattern := range patterns {
		if pathMatches(fileName, pattern) || pathMatches(relPath, pattern) {
			return true
		}
	}
	return false
}

func underAny(relPath string, roots []string) bool {
	for _, root := range roots {
		if pathMatches(relPath, root) || pathMatches(filepath.ToSlash(filepath.Dir(relPath)), root) {
			return true
		}
	}
	return false
}

func severity(value string) string {
	if value == "" {
		return "error"
	}
	return value
}

func existsAny(root string, patterns []string, ignores []string) bool {
	found := false
	_ = filepath.Walk(root, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil || found {
			return nil
		}
		rel := relativePath(root, currentPath)
		for _, ignored := range ignores {
			if pathMatches(rel, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		for _, pattern := range patterns {
			if pathMatches(rel, pattern) || (!info.IsDir() && pathMatches(info.Name(), pattern)) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

func existsAt(root, relPath string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(relPath)))
	return err == nil
}

func matchingDirs(root, pattern string, ignores []string) []string {
	var dirs []string
	_ = filepath.Walk(root, func(currentPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel := relativePath(root, currentPath)
		for _, ignored := range ignores {
			if pathMatches(rel, ignored) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if info.IsDir() && pathMatches(rel, pattern) {
			dirs = append(dirs, rel)
		}
		return nil
	})
	return dirs
}

func readGoModule(root string) string {
	file, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func sourceImports(path, relPath string) ([]string, error) {
	switch filepath.Ext(relPath) {
	case ".go":
		return goImports(path)
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return jsImports(path)
	case ".py":
		return pythonImports(path)
	default:
		return nil, nil
	}
}

func goImports(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}
	imports := make([]string, 0, len(file.Imports))
	for _, imp := range file.Imports {
		imports = append(imports, strings.Trim(imp.Path.Value, `"`))
	}
	return imports, nil
}

func jsImports(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var imports []string
	for _, line := range strings.Split(string(data), "\n") {
		if match := jsImportPattern.FindStringSubmatch(line); len(match) == 2 {
			imports = append(imports, match[1])
		}
		for _, match := range jsRequirePattern.FindAllStringSubmatch(line, -1) {
			if len(match) == 2 {
				imports = append(imports, match[1])
			}
		}
	}
	return imports, nil
}

func pythonImports(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	var imports []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if match := pythonImportPattern.FindStringSubmatch(line); len(match) == 2 {
			imports = append(imports, strings.ReplaceAll(match[1], ".", "/"))
		}
		if match := pythonFromPattern.FindStringSubmatch(line); len(match) == 2 {
			imports = append(imports, strings.ReplaceAll(match[1], ".", "/"))
		}
	}
	return imports, scanner.Err()
}

func importToLocalPath(modulePath, importPath, fromPath string) string {
	if modulePath == "" {
		return resolveRelativeImport(importPath, fromPath)
	}
	if importPath == modulePath {
		return "."
	}
	prefix := modulePath + "/"
	if strings.HasPrefix(importPath, prefix) {
		return strings.TrimPrefix(importPath, prefix)
	}
	return resolveRelativeImport(importPath, fromPath)
}

func resolveRelativeImport(importPath, fromPath string) string {
	if strings.HasPrefix(importPath, ".") {
		return normalizePath(filepath.ToSlash(filepath.Join(filepath.Dir(fromPath), importPath)))
	}
	return strings.ReplaceAll(importPath, ".", "/")
}

func isSupportedBoundaryFile(path string) bool {
	return supportedBoundaryExt[filepath.Ext(path)]
}

func violationKey(v Violation) string {
	return v.Code + "\x00" + v.Path + "\x00" + v.Rule
}
