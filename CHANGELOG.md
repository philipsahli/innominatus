# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Pre-commit hooks for automated code quality checks (golangci-lint, gosec, go-fmt, go-vet, go-imports, Prettier)
- Comprehensive CONTRIBUTING.md with contribution guidelines, workflow, and coding standards
- Enhanced README.md with production setup guide, security best practices, and complete examples
- GitHub Actions workflows for automated security scanning and dependency updates
- Support for JWT authentication and OIDC integration in production deployments
- RBAC (Role-Based Access Control) configuration for multi-tenant environments
- Prometheus metrics endpoints for observability
- Health check endpoints (/health, /ready) for Kubernetes liveness and readiness probes

### Changed
- Restructured README.md with improved sections: Installation, Quickstart, Production Setup, Architecture
- Updated documentation to clearly separate demo environment (local development) from production deployment
- Enhanced API documentation with complete endpoint reference and examples
- Improved error handling with rich error context and suggestions

### Fixed
- Dependabot npm directory configuration (corrected from /v2 to /web-ui)
- 40+ golangci-lint errors (errcheck and staticcheck issues)
- Multiple logBuffer.Write error checking issues in handlers.go
- Pre-commit hook compatibility issues

### Security
- Added gosec security scanning to CI/CD pipeline
- Implemented comprehensive input validation and sanitization
- Added security best practices documentation for production deployments
- Enhanced secret management guidelines with Vault and Kubernetes Secrets integration

---

## Release Guidelines

### Version Format
This project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html):
- **MAJOR** version for incompatible API changes
- **MINOR** version for backwards-compatible functionality additions
- **PATCH** version for backwards-compatible bug fixes

### Release Process
1. Update CHANGELOG.md with release version and date
2. Create annotated git tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
3. Push tag to trigger release workflow: `git push origin vX.Y.Z`
4. GitHub Actions will automatically build and publish release artifacts

---

**Note**: This project is currently in pre-release (v0.x.x) development. The first stable release will be v1.0.0.

[Unreleased]: https://github.com/philipsahli/innominatus/compare/HEAD
