# structlint Configuration Examples

This document provides comprehensive examples of `.structlint.yaml` configurations for different project types and strictness levels.

## 📋 Table of Contents

- [Strictness Levels](#strictness-levels)
- [Go Projects](#go-projects)
- [Node.js Projects](#nodejs-projects)
- [Python Projects](#python-projects)
- [Microservices](#microservices)
- [Monorepos](#monorepos)
- [AI-Friendly Configurations](#ai-friendly-configurations)
- [CI/CD Integration](#cicd-integration)

---

## Strictness Levels

Choose the strictness level that matches your project's needs:

<details>
<summary><strong>🟢 Permissive Mode</strong> - Flexible, allows most structures</summary>

```yaml
# .structlint.yaml - Permissive Mode
dir_structure:
  allowedPaths:
    - "**"  # Allow everything
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "*.log"
file_naming_pattern:
  allowed:
    - "**"  # Allow all file types
  disallowed:
    - "*.env*"
    - "*.key"
    - "*.pem"
ignore:
  - ".git"
  - "vendor"
  - "node_modules"
```

**Use when:** Starting a new project, prototyping, or working with legacy codebases.
</details>

<details>
<summary><strong>🟡 Balanced Mode</strong> - Reasonable defaults with some restrictions</summary>

```yaml
# .structlint.yaml - Balanced Mode
dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "test/**"
    - "docs/**"
    - "scripts/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.js"
    - "*.ts"
    - "*.py"
    - "*.md"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.sh"
    - "Makefile"
    - ".gitignore"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*.swp"
ignore:
  - ".git"
  - "vendor"
  - "node_modules"
  - "bin"
```

**Use when:** Most production projects, team development, standard applications.
</details>

<details>
<summary><strong>🔴 Strict Mode</strong> - Explicit paths only, no wildcards</summary>

```yaml
# .structlint.yaml - Strict Mode
dir_structure:
  allowedPaths:
    - "."
    - "cmd"
    - "cmd/myapp"
    - "internal"
    - "internal/app"
    - "internal/config"
    - "internal/handler"
    - "test"
    - "test/unit"
    - "test/integration"
    - "docs"
    - "scripts"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
  requiredPaths:
    - "cmd"
    - "internal"
    - "test"
file_naming_pattern:
  allowed:
    - "*.go"
    - "go.mod"
    - "go.sum"
    - "README.md"
    - ".gitignore"
    - "Makefile"
    - ".structlint.yaml"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*.swp"
    - "*.bak"
  required:
    - "go.mod"
    - "README.md"
    - "*.go"
ignore:
  - ".git"
```

**Use when:** Enterprise projects, strict compliance requirements, AI-assisted development.
</details>

---

## Go Projects

<details>
<summary><strong>Standard Go Project</strong> - Following Go conventions</summary>

```yaml
# .structlint.yaml - Standard Go Project
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"           # Command-line applications
    - "internal/**"      # Private application code
    - "pkg/**"           # Library code
    - "api/**"           # API definitions
    - "web/**"           # Web assets
    - "test/**"          # Test files
    - "docs/**"          # Documentation
    - "scripts/**"       # Build scripts
    - "configs/**"       # Configuration files
  disallowedPaths:
    - "vendor/**"        # Go modules (use go mod)
    - "tmp/**"
    - "temp/**"
    - ".git/**"
  requiredPaths:
    - "cmd"              # Must have commands
    - "internal"         # Must have internal packages
file_naming_pattern:
  allowed:
    - "*.go"             # Go source files
    - "*.mod"            # Go module file
    - "*.sum"            # Go checksum file
    - "*.yaml"           # YAML configs
    - "*.yml"
    - "*.json"
    - "*.md"             # Documentation
    - "Makefile"         # Build automation
    - ".gitignore"
    - ".structlint.yaml"
    - "Dockerfile"
    - "*.sh"             # Shell scripts
  disallowed:
    - "*.env*"           # Environment files
    - "*.log"            # Log files
    - "*.tmp"            # Temporary files
    - "*.swp"            # Editor swap files
    - "*.bak"            # Backup files
  required:
    - "go.mod"           # Must have module file
    - "README.md"        # Must have documentation
    - "*.go"             # Must have Go source
ignore:
  - ".git"
  - "vendor"
  - "bin"
```

**Project Structure:**
```
my-go-project/
├── cmd/
│   └── myapp/
│       └── main.go
├── internal/
│   ├── app/
│   ├── config/
│   └── handler/
├── pkg/
│   └── utils/
├── test/
│   ├── unit/
│   └── integration/
├── docs/
├── scripts/
├── go.mod
├── go.sum
├── README.md
└── .structlint.yaml
```
</details>

<details>
<summary><strong>Go Microservice</strong> - Service-oriented architecture</summary>

```yaml
# .structlint.yaml - Go Microservice
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "api/**"            # API definitions
    - "migrations/**"     # Database migrations
    - "deployments/**"    # Deployment configs
    - "test/**"
    - "docs/**"
  disallowedPaths:
    - "vendor/**"
    - "tmp/**"
    - "logs/**"
  requiredPaths:
    - "cmd"
    - "internal"
    - "api"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.mod"
    - "*.sum"
    - "*.proto"           # Protocol buffers
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.md"
    - "Dockerfile"
    - "docker-compose.yml"
    - ".gitignore"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
  required:
    - "go.mod"
    - "README.md"
    - "Dockerfile"
    - "*.go"
ignore:
  - ".git"
  - "vendor"
```

**Project Structure:**
```
user-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── user/
│   ├── database/
│   └── middleware/
├── api/
│   └── user.proto
├── migrations/
│   └── 001_create_users.sql
├── deployments/
│   └── k8s/
├── Dockerfile
├── docker-compose.yml
└── README.md
```
</details>

---

## Node.js Projects

<details>
<summary><strong>Node.js Application</strong> - Standard Node.js structure</summary>

```yaml
# .structlint.yaml - Node.js Application
dir_structure:
  allowedPaths:
    - "."
    - "src/**"            # Source code
    - "lib/**"            # Library code
    - "test/**"           # Test files
    - "docs/**"           # Documentation
    - "scripts/**"        # Build scripts
    - "public/**"         # Static assets
    - "config/**"         # Configuration
  disallowedPaths:
    - "node_modules/**"
    - "dist/**"           # Build output
    - "build/**"
    - "tmp/**"
    - ".git/**"
  requiredPaths:
    - "src"
file_naming_pattern:
  allowed:
    - "*.js"              # JavaScript files
    - "*.ts"              # TypeScript files
    - "*.jsx"             # React JSX
    - "*.tsx"             # React TSX
    - "*.json"            # JSON configs
    - "*.md"              # Documentation
    - "package.json"      # Node package file
    - "package-lock.json"
    - "yarn.lock"
    - ".gitignore"
    - ".env.example"      # Example env file
    - "*.sh"
  disallowed:
    - "*.env"             # Actual env files
    - "*.log"
    - "*.tmp"
    - "*.swp"
  required:
    - "package.json"
    - "README.md"
    - "*.js"              # At least one JS file
ignore:
  - ".git"
  - "node_modules"
  - "dist"
  - "build"
```

**Project Structure:**
```
my-node-app/
├── src/
│   ├── controllers/
│   ├── models/
│   ├── routes/
│   └── utils/
├── test/
│   ├── unit/
│   └── integration/
├── public/
│   └── assets/
├── config/
├── package.json
├── package-lock.json
├── README.md
└── .structlint.yaml
```
</details>

<details>
<summary><strong>React/Next.js Project</strong> - Frontend application</summary>

```yaml
# .structlint.yaml - React/Next.js Project
dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "components/**"     # React components
    - "pages/**"          # Next.js pages
    - "app/**"            # Next.js app directory
    - "public/**"         # Static assets
    - "styles/**"         # CSS/SCSS files
    - "test/**"
    - "docs/**"
  disallowedPaths:
    - "node_modules/**"
    - ".next/**"          # Next.js build output
    - "out/**"            # Next.js export output
    - "tmp/**"
  requiredPaths:
    - "src"
file_naming_pattern:
  allowed:
    - "*.js"
    - "*.ts"
    - "*.jsx"
    - "*.tsx"
    - "*.css"
    - "*.scss"
    - "*.sass"
    - "*.json"
    - "*.md"
    - "package.json"
    - "next.config.js"
    - "tailwind.config.js"
    - ".gitignore"
  disallowed:
    - "*.env"
    - "*.log"
    - "*.tmp"
  required:
    - "package.json"
    - "README.md"
ignore:
  - ".git"
  - "node_modules"
  - ".next"
  - "out"
```

**Project Structure:**
```
my-react-app/
├── src/
│   ├── components/
│   ├── pages/
│   ├── hooks/
│   └── utils/
├── public/
│   └── images/
├── styles/
├── test/
├── package.json
├── next.config.js
└── README.md
```
</details>

---

## Python Projects

<details>
<summary><strong>Python Package</strong> - Standard Python structure</summary>

```yaml
# .structlint.yaml - Python Package
dir_structure:
  allowedPaths:
    - "."
    - "src/**"            # Source code
    - "tests/**"          # Test files
    - "docs/**"           # Documentation
    - "scripts/**"        # Utility scripts
    - "examples/**"       # Example code
  disallowedPaths:
    - "__pycache__/**"
    - "*.egg-info/**"
    - "build/**"
    - "dist/**"
    - ".pytest_cache/**"
    - "tmp/**"
  requiredPaths:
    - "src"
    - "tests"
file_naming_pattern:
  allowed:
    - "*.py"              # Python files
    - "*.pyi"             # Python stub files
    - "*.md"              # Documentation
    - "*.rst"
    - "*.txt"
    - "*.yml"
    - "*.yaml"
    - "*.json"
    - "*.toml"
    - "requirements.txt"
    - "pyproject.toml"
    - "setup.py"
    - "setup.cfg"
    - ".gitignore"
    - "Makefile"
    - "*.sh"
  disallowed:
    - "*.pyc"             # Compiled Python
    - "*.pyo"
    - "*.pyd"
    - "*.env"
    - "*.log"
    - "*.tmp"
  required:
    - "*.py"              # At least one Python file
    - "README.md"
ignore:
  - ".git"
  - "__pycache__"
  - "*.egg-info"
  - "build"
  - "dist"
  - ".pytest_cache"
```

**Project Structure:**
```
my-python-package/
├── src/
│   └── mypackage/
│       ├── __init__.py
│       ├── core.py
│       └── utils.py
├── tests/
│   ├── test_core.py
│   └── test_utils.py
├── docs/
├── examples/
├── pyproject.toml
├── requirements.txt
├── README.md
└── .structlint.yaml
```
</details>

<details>
<summary><strong>Django Project</strong> - Web framework structure</summary>

```yaml
# .structlint.yaml - Django Project
dir_structure:
  allowedPaths:
    - "."
    - "myproject/**"      # Django project
    - "apps/**"           # Django apps
    - "static/**"         # Static files
    - "media/**"          # Media files
    - "templates/**"      # Template files
    - "tests/**"
    - "docs/**"
    - "scripts/**"
  disallowedPaths:
    - "__pycache__/**"
    - "*.egg-info/**"
    - "build/**"
    - "dist/**"
    - "tmp/**"
  requiredPaths:
    - "myproject"         # Django project directory
file_naming_pattern:
  allowed:
    - "*.py"
    - "*.html"            # Django templates
    - "*.css"
    - "*.js"
    - "*.json"
    - "*.md"
    - "*.yml"
    - "requirements.txt"
    - "manage.py"         # Django management script
    - ".gitignore"
  disallowed:
    - "*.pyc"
    - "*.env"
    - "*.log"
    - "*.tmp"
  required:
    - "manage.py"
    - "requirements.txt"
    - "README.md"
ignore:
  - ".git"
  - "__pycache__"
  - "*.egg-info"
```

**Project Structure:**
```
my-django-project/
├── myproject/
│   ├── __init__.py
│   ├── settings.py
│   ├── urls.py
│   └── wsgi.py
├── apps/
│   ├── users/
│   └── blog/
├── static/
├── templates/
├── tests/
├── manage.py
├── requirements.txt
└── README.md
```
</details>

---

## Microservices

<details>
<summary><strong>Microservice Architecture</strong> - Service-oriented design</summary>

```yaml
# .structlint.yaml - Microservice Architecture
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"            # Service entry points
    - "internal/**"       # Private service code
    - "api/**"            # API definitions
    - "pkg/**"            # Shared libraries
    - "migrations/**"     # Database migrations
    - "deployments/**"    # Deployment configs
    - "test/**"
    - "docs/**"
    - "scripts/**"
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "logs/**"
  requiredPaths:
    - "cmd"
    - "internal"
    - "api"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.js"
    - "*.ts"
    - "*.py"
    - "*.proto"           # Protocol buffers
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.md"
    - "Dockerfile"
    - "docker-compose.yml"
    - "k8s.yaml"
    - ".gitignore"
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
  required:
    - "README.md"
    - "Dockerfile"
ignore:
  - ".git"
  - "vendor"
  - "node_modules"
```

**Project Structure:**
```
user-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── user/
│   ├── database/
│   └── middleware/
├── api/
│   └── user.proto
├── migrations/
├── deployments/
│   └── k8s/
├── test/
├── Dockerfile
├── docker-compose.yml
└── README.md
```
</details>

---

## Monorepos

<details>
<summary><strong>Monorepo Structure</strong> - Multiple projects in one repo</summary>

```yaml
# .structlint.yaml - Monorepo Structure
dir_structure:
  allowedPaths:
    - "."
    - "apps/**"           # Applications
    - "packages/**"       # Shared packages
    - "services/**"       # Microservices
    - "tools/**"          # Development tools
    - "docs/**"           # Documentation
    - "scripts/**"        # Build scripts
    - "configs/**"        # Shared configs
  disallowedPaths:
    - "node_modules/**"
    - "vendor/**"
    - "tmp/**"
    - "dist/**"
    - "build/**"
  requiredPaths:
    - "apps"
    - "packages"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.js"
    - "*.ts"
    - "*.py"
    - "*.json"
    - "*.yaml"
    - "*.yml"
    - "*.md"
    - "package.json"
    - "go.mod"
    - "requirements.txt"
    - "Makefile"
    - ".gitignore"
    - "lerna.json"        # Lerna config
    - "nx.json"            # Nx config
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
  required:
    - "README.md"
    - "package.json"      # Root package.json
ignore:
  - ".git"
  - "node_modules"
  - "vendor"
  - "dist"
  - "build"
```

**Project Structure:**
```
my-monorepo/
├── apps/
│   ├── web-app/
│   ├── mobile-app/
│   └── admin-panel/
├── packages/
│   ├── shared-ui/
│   ├── shared-utils/
│   └── shared-types/
├── services/
│   ├── auth-service/
│   └── user-service/
├── tools/
│   └── build-scripts/
├── docs/
├── package.json
├── lerna.json
└── README.md
```
</details>

---

## AI-Friendly Configurations

<details>
<summary><strong>AI Development Assistant</strong> - Optimized for AI code generation</summary>

```yaml
# .structlint.yaml - AI Development Assistant
dir_structure:
  allowedPaths:
    - "."
    - "src/**"
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "test/**"
    - "docs/**"
    - "scripts/**"
    - "examples/**"       # AI can reference examples
    - "templates/**"      # Code templates
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
    - ".git/**"
  requiredPaths:
    - "src"
    - "test"
    - "docs"              # AI needs documentation
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.js"
    - "*.ts"
    - "*.py"
    - "*.md"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.txt"
    - "*.sh"
    - "Makefile"
    - ".gitignore"
    - "README.md"
    - "CONTRIBUTING.md"   # AI guidelines
    - "ARCHITECTURE.md"   # System design docs
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
    - "*.swp"
  required:
    - "README.md"
    - "CONTRIBUTING.md"   # AI needs contribution guidelines
    - "*.go"              # At least one source file
ignore:
  - ".git"
  - "vendor"
  - "node_modules"
```

**AI Benefits:**
- Clear documentation requirements help AI understand project structure
- Example and template directories provide AI with reference material
- Consistent naming patterns help AI generate appropriate file names
- Required paths ensure AI creates files in correct locations
</details>

---

## CI/CD Integration

<details>
<summary><strong>GitHub Actions Integration</strong> - Automated validation</summary>

```yaml
# .github/workflows/validate-structure.yml
name: Validate Project Structure

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Install structlint
      run: go install github.com/youngestaxe/structlint@latest
    
    - name: Validate project structure
      run: |
        structlint validate \
          --config .structlint.yaml \
          --json-output validation-report.json \
          --group-violations
    
    - name: Upload validation report
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: validation-report
        path: validation-report.json
    
    - name: Comment PR with violations
      if: failure() && github.event_name == 'pull_request'
      run: |
        echo "## 🚨 Project Structure Violations" >> $GITHUB_STEP_SUMMARY
        echo "The following structure violations were found:" >> $GITHUB_STEP_SUMMARY
        echo '```json' >> $GITHUB_STEP_SUMMARY
        cat validation-report.json >> $GITHUB_STEP_SUMMARY
        echo '```' >> $GITHUB_STEP_SUMMARY
```

**Configuration for CI/CD:**
```yaml
# .structlint.yaml - CI/CD Optimized
dir_structure:
  allowedPaths:
    - "."
    - "cmd/**"
    - "internal/**"
    - "pkg/**"
    - "test/**"
    - "docs/**"
    - "scripts/**"
    - ".github/**"        # GitHub workflows
  disallowedPaths:
    - "vendor/**"
    - "node_modules/**"
    - "tmp/**"
    - "temp/**"
  requiredPaths:
    - "cmd"
    - "internal"
    - "test"
file_naming_pattern:
  allowed:
    - "*.go"
    - "*.mod"
    - "*.sum"
    - "*.yaml"
    - "*.yml"
    - "*.json"
    - "*.md"
    - "Makefile"
    - ".gitignore"
    - ".github/**"       # GitHub configs
  disallowed:
    - "*.env*"
    - "*.log"
    - "*.tmp"
  required:
    - "go.mod"
    - "README.md"
    - ".github/workflows/**"  # Must have CI/CD
ignore:
  - ".git"
  - "vendor"
```
</details>

---

## 🎯 Choosing the Right Configuration

### **For New Projects:**
Start with **Balanced Mode** and adjust as your project grows.

### **For Existing Projects:**
Begin with **Permissive Mode** and gradually tighten restrictions.

### **For AI-Assisted Development:**
Use **Strict Mode** with comprehensive documentation requirements.

### **For Enterprise Projects:**
Implement **Strict Mode** with CI/CD integration and compliance reporting.

---

## 🔧 Customization Tips

1. **Start Simple**: Begin with basic allowed/disallowed patterns
2. **Add Gradually**: Introduce required paths and files as needed
3. **Test Changes**: Always test configuration changes before committing
4. **Document Decisions**: Add comments explaining why certain rules exist
5. **Team Alignment**: Ensure all team members understand the structure rules

---

## 📚 Additional Resources

- [Main README](../README.md) - Overview and quick start

---

**Need help choosing a configuration?** Start with the [Balanced Mode](#balanced-mode) and customize based on your project's specific needs!
