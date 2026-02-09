# Contributing to Dotfiles Management System

First off, thank you for considering contributing to this project! It's people like you that make this tool better for everyone.

## Code of Conduct

This project adheres to a code of conduct that all contributors are expected to follow. Please be respectful and constructive in all interactions.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to avoid duplicates. When creating a bug report, include as many details as possible:

**Bug Report Template:**
```markdown
**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Run command '...'
2. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- OS: [e.g., macOS 13.0, Ubuntu 22.04]
- Architecture: [e.g., amd64, arm64]
- Dotfiles version: [e.g., git commit hash]
- Go version: [run `go version`]

**Additional context**
- Output with `-v` flag
- State files: `cat ~/.dotfiles/.state/module.json`
- Any other relevant information
```

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

- **Clear title and description** of the enhancement
- **Use case** - Why would this be useful?
- **Proposed solution** - How would it work?
- **Alternatives considered** - What other approaches did you think about?

### Pull Requests

1. **Fork the repository** and create your branch from `main`
2. **Make your changes** following the coding standards
3. **Add tests** if you're adding functionality
4. **Update documentation** if needed
5. **Run the test suite** to ensure everything passes
6. **Submit a pull request**

## Development Setup

### Prerequisites

- Go 1.22 or later
- Bash 4.0 or later
- Docker (for integration tests)
- Git

### Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/dotfiles.git
cd dotfiles

# Build the project
make build

# Run tests
make test

# Run integration tests (requires Docker)
make test-integration
```

### Project Structure

```
.
├── cmd/dotfiles/           # CLI commands
├── internal/               # Internal packages
│   ├── config/            # Configuration
│   ├── module/            # Module system
│   ├── secrets/           # Secrets providers
│   ├── state/             # State tracking
│   ├── sysinfo/           # System detection
│   ├── template/          # Template rendering
│   └── ui/                # User interface
├── modules/               # Module definitions
├── lib/                   # Shell helpers
├── test/integration/      # Integration tests
└── docs/                  # Documentation
```

## Coding Standards

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` to format code
- Run `go vet` to catch common issues
- Write meaningful commit messages

**Example:**
```go
// Good: Clear function name and comment
// LoadConfig reads the configuration file from the dotfiles directory
func LoadConfig(dotfilesDir string) (*Config, error) {
    // Implementation
}

// Bad: Unclear naming
func get(d string) (*C, error) {
    // Implementation
}
```

### Shell Scripts

- Use `shellcheck` to lint scripts
- Always use `set -euo pipefail`
- Quote variables: `"$VAR"` not `$VAR`
- Use helper functions from `lib/helpers.sh`

**Example:**
```bash
# Good
if pkg_installed git; then
    log_info "Git already installed"
fi

# Bad: Direct package manager calls
if dpkg -l | grep -q git; then
    echo "Git installed"
fi
```

### Testing

- Write unit tests for Go code
- Add integration test assertions for new modules
- Ensure tests pass locally before submitting PR
- Aim for high test coverage on new code

**Running Tests:**
```bash
# Unit tests
go test ./...

# Integration tests
make test-integration-ubuntu
make test-integration-arch

# All tests
make test-all
```

## Adding a New Module

See [Creating Modules](docs/creating-modules.md) for a detailed guide.

**Quick checklist:**
1. Create `modules/NAME/` directory
2. Add `module.yml` with metadata
3. Write `install.sh` script
4. (Optional) Add OS-specific scripts in `os/`
5. (Optional) Add files to deploy in `files/`
6. Test with `dotfiles install NAME --dry-run`
7. Add integration test assertions
8. Update documentation

## Adding Documentation

- Documentation lives in `docs/`
- Use clear, concise language
- Include code examples
- Update the docs README with links to new pages
- Use proper Markdown formatting

## Commit Messages

Write clear commit messages following these guidelines:

**Format:**
```
Short summary (50 chars or less)

More detailed explanatory text if needed. Wrap at 72 characters.
Explain the problem that this commit is solving and why this approach
was chosen.

- Bullet points are okay
- Use present tense: "Add feature" not "Added feature"
- Reference issues: "Fixes #123"
```

**Examples:**
```
Add support for custom package managers

Implement a pluggable package manager system that allows users to
define custom package managers in their config. This is useful for
enterprise environments with internal package repositories.

Fixes #42
```

## Review Process

1. **Automated Checks**: CI must pass (unit tests, integration tests, linting)
2. **Code Review**: At least one maintainer will review your PR
3. **Changes Requested**: Address feedback and push updates
4. **Approval**: Once approved, a maintainer will merge

### What Reviewers Look For

- **Correctness**: Does it work as intended?
- **Tests**: Are there appropriate tests?
- **Documentation**: Is it documented?
- **Style**: Does it follow project conventions?
- **Scope**: Is the PR focused on one thing?

## Release Process

Releases are managed by maintainers:

1. Version bump in relevant files
2. Update CHANGELOG.md
3. Create GitHub release with notes
4. Tag release: `git tag v1.2.3`

## Getting Help

- **Documentation**: Check [docs/](docs/) first
- **GitHub Issues**: Search existing issues or create a new one
- **Discussions**: Use GitHub Discussions for questions

## Recognition

Contributors are recognized in:
- Git commit history
- GitHub contributors page
- Release notes for significant contributions

Thank you for contributing! 🎉
