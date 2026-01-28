# structlint

**A powerful CLI tool for validating and enforcing directory structure and file naming patterns in your projects.**

## 🎯 What Problems Does This Tool Solve?

### **For Development Teams:**
- **Enforces consistent project structure** across team members
- **Prevents accidental violations** of coding standards
- **Catches structural issues early** in development
- **Ensures required files/directories exist** (like `README.md`, `go.mod`, etc.)
- **Blocks forbidden patterns** (like `.env` files in wrong places)

### **For AI Assistants & Code Generation:**
- **Provides clear project structure expectations** for AI tools
- **Helps AI understand where to place new files** and directories
- **Enables AI to follow your project's conventions** automatically
- **Reduces AI-generated code that violates your standards**

### **For CI/CD Pipelines:**
- **Validates project structure** before deployment
- **Fails builds** when structure violations are detected
- **Generates structured reports** for compliance tracking
- **Integrates seamlessly** with existing workflows

## 🚀 Quick Start

```bash
# Install
go install github.com/AxeForging/structlint@latest

# Create a basic config
cat > .structlint.yaml << 'EOF'
dir_structure:
  allowedPaths: ["cmd/**", "internal/**", "test/**"]
  disallowedPaths: ["vendor/**", "tmp/**"]
file_naming_pattern:
  allowed: ["*.go", "*.md", "*.yaml"]
  disallowed: ["*.env*", "*.log"]
ignore: [".git", "bin"]
EOF

# Validate your project
structlint validate
```

## 📋 Features

### **Directory Structure Validation**
- ✅ **Allowed paths**: Define what directories are permitted
- ❌ **Disallowed paths**: Explicitly forbid certain directories
- 🔒 **Required paths**: Enforce that essential directories exist
- 🚫 **Ignore patterns**: Skip validation for specific paths

### **File Naming Pattern Validation**
- ✅ **Allowed patterns**: Define permitted file naming conventions
- ❌ **Disallowed patterns**: Block forbidden file types/names
- 🔒 **Required files**: Ensure essential files exist (README, configs, etc.)
- 🌐 **Glob support**: Use patterns like `*.go`, `src/**/*.ts`, `tests/*_test.go`

### **Smart Reporting**
- 📊 **Grouped violations**: Similar issues grouped together for readability
- 📈 **Summary statistics**: Quick overview of validation results
- 📄 **JSON reports**: Machine-readable output for CI/CD integration
- 🔇 **Silent mode**: Quiet operation for automated environments

### **Flexible Configuration**
- 📝 **YAML/JSON configs**: Human-readable configuration files
- 🎛️ **Environment variables**: Override settings via env vars
- 🏗️ **Multiple strictness levels**: From permissive to ultra-strict
- 🔧 **Project-specific rules**: Tailored for Go, Node.js, Python, etc.

## 🎨 Configuration Examples

### **Basic Go Project**
```yaml
dir_structure:
  allowedPaths: ["cmd/**", "internal/**", "pkg/**", "test/**"]
  disallowedPaths: ["vendor/**", "tmp/**"]
file_naming_pattern:
  allowed: ["*.go", "*.mod", "*.sum", "README.md", ".gitignore"]
  disallowed: ["*.env*", "*.log"]
ignore: [".git", "vendor"]
```

### **Strict Enforcement Mode**
```yaml
dir_structure:
  allowedPaths: 
    - "."
    - "cmd"
    - "cmd/myapp"  # Explicit paths only
    - "internal"
    - "internal/app"
  requiredPaths: ["cmd", "internal", "test"]
file_naming_pattern:
  allowed: ["*.go", "go.mod", "README.md", ".gitignore"]
  required: ["go.mod", "README.md", "*.go"]
ignore: [".git"]
```

## 🛠️ Usage

### **Command Line Options**
```bash
structlint validate [options]

Options:
  --path string         Path to validate (default: ".")
  --config string       Config file path (default: ".structlint.yaml")
  --json-output string  Save JSON report to file
  --silent              Suppress output except JSON report
  --group-violations    Group violations by type (default: true)
  --verbose             Show all allowed files (default: false)
```

### **Environment Variables**
```bash
export STRUCTLINT_CONFIG=".structlint.yaml"
export STRUCTLINT_PATH="/path/to/project"
export STRUCTLINT_JSON_OUTPUT="report.json"
export STRUCTLINT_SILENT="true"
```

### **CI/CD Integration**
```yaml
# GitHub Actions example
- name: Validate Project Structure
  run: |
    structlint validate --config .structlint.yaml --json-output validation-report.json
    if [ $? -ne 0 ]; then
      echo "Project structure validation failed"
      exit 1
    fi
```

## 📚 Documentation

For comprehensive examples and advanced configurations, see the [docs/](./docs/) directory:

- **[Configuration Examples](./docs/README.md)** - Detailed examples for different project types
- **[Go Projects](./docs/README.md#go-projects)** - Go-specific configurations
- **[Node.js Projects](./docs/README.md#nodejs-projects)** - JavaScript/TypeScript configurations  
- **[Python Projects](./docs/README.md#python-projects)** - Python-specific configurations
- **[Strictness Levels](./docs/README.md#strictness-levels)** - From permissive to ultra-strict

## 🤖 AI Integration Benefits

### **For AI Code Generation:**
When AI tools understand your project structure, they can:
- **Place files in correct directories** automatically
- **Follow your naming conventions** without guidance
- **Respect your project boundaries** and organization
- **Generate code that fits your existing structure**

### **Example AI Prompt Enhancement:**
```
Before: "Create a new API handler"
After: "Create a new API handler following our .structlint.yaml structure:
- Place in internal/api/handlers/
- Use snake_case naming
- Include proper tests in test/api/handlers/"
```

## 🏗️ Building from Source

```bash
# Clone and build
git clone https://github.com/AxeForging/structlint.git
cd structlint
make build

# Run tests
make test

# Self-validate
make test-self
```

## 📄 License

MIT License - see [LICENSE](./LICENSE) for details.

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass: `make test`
5. Submit a pull request

---

**Ready to enforce structure standards in your projects?** Start with the [configuration examples](./docs/README.md) and customize for your needs!
