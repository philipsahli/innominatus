# Contributing to innominatus

Thank you for your interest in contributing to innominatus! 🎉

We're building a focused platform orchestration component for the Score specification ecosystem, and we value contributions that help us maintain clarity of purpose, code quality, and enterprise readiness.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How Can I Contribute?](#how-can-i-contribute)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Pull Request Process](#pull-request-process)
- [Quality Checklist](#quality-checklist)
- [Issue Guidelines](#issue-guidelines)
- [Testing](#testing)
- [Documentation](#documentation)

---

## Code of Conduct

This project adheres to a Code of Conduct that all contributors are expected to follow. We are committed to providing a welcoming and inclusive environment for everyone.

**Our Standards**:
- Be respectful and inclusive in language and actions
- Welcome diverse perspectives and constructive feedback
- Focus on what's best for the community and the project
- Show empathy and kindness toward other community members

**Unacceptable Behavior**:
- Harassment, discrimination, or exclusionary behavior
- Personal attacks or inflammatory comments
- Publishing others' private information without permission
- Any conduct that would be inappropriate in a professional setting

If you experience or witness unacceptable behavior, please report it to the project maintainers.

---

## How Can I Contribute?

There are many ways to contribute to innominatus:

### 1. **Report Bugs**
Found a bug? Please [create an issue](https://github.com/philipsahli/idp-o/issues/new) with:
- Clear description of the problem
- Steps to reproduce
- Expected vs. actual behavior
- Environment details (OS, Go version, Kubernetes version)
- Relevant logs or error messages

### 2. **Suggest Features**
Have an idea for a new feature? Start a discussion by:
- Opening an issue to describe the feature
- Explaining the use case and benefits
- Considering how it fits with the project's focus on Score orchestration

### 3. **Improve Documentation**
Help make our docs better:
- Fix typos or clarify existing documentation
- Add examples and tutorials
- Improve API documentation
- Create guides for specific use cases

### 4. **Write Code**
Contribute new features or bug fixes:
- New workflow step types (e.g., Helm, Flux, Crossplane)
- Platform integrations (Backstage plugins, CNOE components)
- Observability improvements (metrics, tracing, logging)
- Security enhancements (auth, RBAC, encryption)
- Performance optimizations

### 5. **Write Tests**
Help improve test coverage:
- Unit tests for new functionality
- Integration tests for workflow steps
- End-to-end tests for complete workflows
- Performance and load tests

---

## Getting Started

### Prerequisites

- **Go 1.21+**
- **PostgreSQL 13+** (for running tests locally)
- **Docker Desktop** with Kubernetes enabled (for demo environment)
- **kubectl** configured with cluster access
- **Git** for version control

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR_USERNAME/idp-o.git
cd idp-o
```

3. Add the upstream repository:

```bash
git remote add upstream https://github.com/philipsahli/idp-o.git
```

### Build and Run

```bash
# Build the server
go build -o innominatus cmd/server/main.go

# Build the CLI
go build -o innominatus-ctl cmd/cli/main.go

# Run tests
go test ./... -v

# Run linters
golangci-lint run
gosec ./...
```

### Install Development Tools

```bash
# Install pre-commit hooks
brew install pre-commit
pre-commit install

# Install golangci-lint
brew install golangci-lint

# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Install goimports
go install golang.org/x/tools/cmd/goimports@latest
```

---

## Development Workflow

### 1. **Create a Branch**

Always create a new branch for your work:

```bash
# Update your fork
git fetch upstream
git checkout main
git merge upstream/main

# Create a feature branch
git checkout -b feat/add-helm-workflow-step
```

**Branch Naming Conventions**:
- `feat/description` - New features
- `fix/description` - Bug fixes
- `docs/description` - Documentation changes
- `test/description` - Test additions or changes
- `refactor/description` - Code refactoring
- `chore/description` - Maintenance tasks

### 2. **Make Your Changes**

Write clean, well-documented code:

```go
// Good: Clear function with documentation
// ProcessWorkflow executes a workflow for the given Score specification.
// It validates the spec, creates a workflow execution record, and processes
// each step sequentially with proper error handling.
func ProcessWorkflow(spec *types.ScoreSpec) error {
    // Implementation...
}
```

### 3. **Write Tests**

All new code should include tests:

```go
func TestProcessWorkflow(t *testing.T) {
    spec := &types.ScoreSpec{
        Metadata: types.Metadata{
            Name: "test-app",
        },
    }

    err := ProcessWorkflow(spec)
    assert.NoError(t, err)
}
```

### 4. **Run Pre-Commit Checks**

Before committing, ensure all checks pass:

```bash
# Pre-commit will automatically run on git commit
# Or run manually:
pre-commit run --all-files
```

This runs:
- Trailing whitespace removal
- EOF fixer
- YAML validation
- go-fmt, go-vet, go-imports
- golangci-lint
- gosec security scanning
- Prettier (for web-ui)

### 5. **Commit Your Changes**

Follow [Conventional Commits](https://www.conventionalcommits.org/) format:

```bash
git add .
git commit -m "feat: add Helm workflow step type

- Implement HelmStepExecutor for chart deployments
- Add support for values files and inline values
- Include integration tests with local Helm charts
- Update documentation with Helm step examples

Closes #123"
```

### 6. **Push and Create PR**

```bash
# Push to your fork
git push origin feat/add-helm-workflow-step

# Create a pull request on GitHub
```

---

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` for formatting (enforced by pre-commit)
- Run `golangci-lint` and fix all issues
- Ensure `gosec` security scanner passes
- Write clear, descriptive variable and function names
- Add comments for exported functions and types
- Keep functions focused and under 50 lines when possible

### Error Handling

```go
// Good: Contextual error messages
if err := db.SaveWorkflow(workflow); err != nil {
    return fmt.Errorf("failed to save workflow %s: %w", workflow.Name, err)
}

// Bad: Generic error messages
if err != nil {
    return err
}
```

### Logging

Use structured logging with appropriate levels:

```go
log.Info("Starting workflow execution",
    "workflow_id", workflow.ID,
    "app_name", workflow.AppName,
)

log.Error("Workflow step failed",
    "workflow_id", workflow.ID,
    "step", step.Name,
    "error", err,
)
```

### Security

- Never hardcode secrets or credentials
- Use environment variables for sensitive data
- Validate all user input
- Sanitize file paths and commands
- Follow the principle of least privilege
- Document security considerations in code comments

---

## Commit Message Guidelines

We use [Conventional Commits](https://www.conventionalcommits.org/) to maintain a clean, semantic commit history.

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- **feat**: New feature for the user
- **fix**: Bug fix for the user
- **docs**: Documentation changes
- **style**: Code style changes (formatting, no logic change)
- **refactor**: Code refactoring (no feature change)
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Maintenance tasks (dependencies, CI, etc.)
- **ci**: CI/CD pipeline changes
- **build**: Build system or dependency changes

### Examples

```bash
# Feature with scope
feat(workflow): add Helm chart deployment step

# Bug fix with issue reference
fix(api): prevent race condition in workflow status updates

Fixes #234

# Breaking change
feat(auth)!: replace basic auth with JWT-based authentication

BREAKING CHANGE: Basic authentication is no longer supported.
All API clients must use JWT tokens.

# Documentation
docs: add production deployment guide

# Multiple changes
feat: implement RBAC for workflow execution

- Add role-based permission model
- Implement middleware for permission checks
- Add admin endpoints for role management
- Update API documentation

Closes #156
```

### Commit Message Rules

1. **Use imperative mood**: "add feature" not "added feature"
2. **First line under 72 characters**: Keep the summary concise
3. **Reference issues**: Use "Closes #123" or "Fixes #456"
4. **Explain why**: The body should explain why the change was made
5. **One logical change**: Each commit should represent one logical change

---

## Pull Request Process

### Before Submitting

1. ✅ **Update your branch** with the latest upstream changes
2. ✅ **Run all tests** and ensure they pass
3. ✅ **Run linters** (golangci-lint, gosec)
4. ✅ **Update documentation** if needed
5. ✅ **Add tests** for new functionality
6. ✅ **Check commit messages** follow Conventional Commits

### PR Template

When creating a PR, include:

```markdown
## Description
Brief description of what this PR does and why.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## How Has This Been Tested?
Describe the tests you ran and how to reproduce them.

## Checklist
- [ ] My code follows the code style of this project
- [ ] I have performed a self-review of my own code
- [ ] I have commented my code, particularly in hard-to-understand areas
- [ ] I have made corresponding changes to the documentation
- [ ] My changes generate no new warnings or errors
- [ ] I have added tests that prove my fix is effective or that my feature works
- [ ] New and existing unit tests pass locally with my changes
- [ ] Any dependent changes have been merged and published

## Related Issues
Closes #123
```

### Review Process

1. **Automated Checks**: CI will run tests, linters, and security scans
2. **Code Review**: Maintainers will review your code
3. **Feedback**: Address any requested changes
4. **Approval**: Once approved, a maintainer will merge your PR

### After Merge

- Your contribution will be included in the next release
- You'll be added to the contributors list
- Thank you for your contribution! 🎉

---

## Quality Checklist

Before submitting, ensure your contribution meets these quality standards:

### Code Quality

- [ ] Code follows Go best practices and project conventions
- [ ] Functions have clear, descriptive names
- [ ] Complex logic is commented and explained
- [ ] No commented-out code or debug statements
- [ ] No unnecessary dependencies added

### Testing

- [ ] All existing tests pass: `go test ./...`
- [ ] New tests added for new functionality
- [ ] Test coverage maintained or improved
- [ ] Edge cases and error conditions tested
- [ ] Integration tests added where appropriate

### Security

- [ ] `gosec` security scanner passes with no new issues
- [ ] No hardcoded secrets or credentials
- [ ] User input is validated and sanitized
- [ ] Error messages don't leak sensitive information
- [ ] Dependencies scanned for vulnerabilities

### Documentation

- [ ] Public functions have godoc comments
- [ ] README updated if public API changed
- [ ] Examples provided for new features
- [ ] CHANGELOG updated (for maintainers)

### Git Hygiene

- [ ] Commits follow Conventional Commits format
- [ ] Commit history is clean (no "fix typo" commits)
- [ ] Branch is up to date with upstream/main
- [ ] No merge conflicts

---

## Issue Guidelines

### Creating Good Issues

When reporting bugs or requesting features, please provide:

#### Bug Reports

```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Deploy Score spec with '...'
2. Execute workflow with '...'
3. Check status at '...'
4. See error

**Expected behavior**
What you expected to happen.

**Actual behavior**
What actually happened.

**Screenshots/Logs**
If applicable, add screenshots or log output.

**Environment:**
- OS: [e.g., Ubuntu 22.04]
- Go version: [e.g., 1.21.3]
- innominatus version: [e.g., v0.1.0]
- Kubernetes version: [e.g., 1.28.0]
- PostgreSQL version: [e.g., 15.2]

**Additional context**
Any other relevant information.
```

#### Feature Requests

```markdown
**Is your feature request related to a problem?**
A clear description of the problem. Ex. I'm frustrated when [...]

**Describe the solution you'd like**
A clear description of what you want to happen.

**Describe alternatives you've considered**
Other approaches you've considered.

**Use case**
Explain how this feature would be used in practice.

**Additional context**
Any other relevant information, mockups, or examples.
```

---

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/workflow/...

# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...
```

### Writing Tests

#### Unit Tests

```go
func TestWorkflowExecutor_ExecuteStep(t *testing.T) {
    executor := NewWorkflowExecutor(mockDB, mockLogger)
    step := &types.WorkflowStep{
        Name: "test-step",
        Type: "kubernetes",
    }

    err := executor.ExecuteStep(step)
    assert.NoError(t, err)
}
```

#### Integration Tests

```go
func TestKubernetesProvisioner_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Test with real Kubernetes cluster
    provisioner := NewKubernetesProvisioner(realConfig)
    err := provisioner.Provision(testResource)
    assert.NoError(t, err)
}
```

#### Table-Driven Tests

```go
func TestValidateScoreSpec(t *testing.T) {
    tests := []struct {
        name    string
        spec    *types.ScoreSpec
        wantErr bool
    }{
        {
            name: "valid spec",
            spec: &types.ScoreSpec{
                Metadata: types.Metadata{Name: "test"},
            },
            wantErr: false,
        },
        {
            name:    "missing name",
            spec:    &types.ScoreSpec{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateScoreSpec(tt.spec)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

---

## Documentation

### Code Documentation

- Use godoc comments for all exported functions, types, and packages
- Include examples where helpful
- Document edge cases and error conditions

```go
// WorkflowExecutor manages the execution of workflow steps.
// It handles step-by-step processing, error recovery, and status updates.
//
// Example usage:
//   executor := NewWorkflowExecutor(db, logger)
//   err := executor.Execute(workflow)
type WorkflowExecutor struct {
    db     *database.Database
    logger *log.Logger
}
```

### API Documentation

- Update OpenAPI/Swagger specs when changing API endpoints
- Include request/response examples
- Document error codes and responses

### User Documentation

- Update README.md for user-facing changes
- Add tutorials and guides for new features
- Keep examples up to date

---

## Getting Help

Need help with your contribution?

- **Ask Questions**: Open a discussion on GitHub
- **Join Chat**: (Link to Slack/Discord if available)
- **Check Docs**: Review existing documentation
- **Review Code**: Look at similar existing code for patterns

---

## Recognition

Contributors who have their pull requests merged will:

- Be added to the contributors list
- Receive credit in release notes (for significant contributions)
- Build a track record in the Score/IDP ecosystem

---

## License

By contributing to innominatus, you agree that your contributions will be licensed under the [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0).

---

**Thank you for contributing to innominatus!** Your efforts help build better platform engineering tools for the community. 🚀
