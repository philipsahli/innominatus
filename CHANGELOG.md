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
- **API/CLI/UI Gap Analysis** - Comprehensive consistency analysis identifying 47 gaps across all priority levels (docs/API_CLI_UI_GAP_ANALYSIS.md)
- **Terminology Guide** - Standardized terminology definitions for specs, applications, workflows, and resources (docs/TERMINOLOGY.md)
- **Deprovision Operation in Web UI** - Added application deprovision functionality with confirmation dialog
- **Delete Operation in Web UI** - Added application delete functionality with clear warnings and confirmation
- **Resource Management CLI Commands** - Added `resource` command with subcommands: get, delete, update, transition, health
- **Standardized API Endpoints** - New `/api/applications` endpoints (preferred) with `/api/specs` as deprecated aliases
- **Environments Page in Web UI** - Full environments listing page with statistics and descriptions (`/environments`)
- **User Management API Endpoints** - Admin-only REST API for user CRUD operations (`/api/admin/users`)
- **Admin API Key Management Endpoints** - Admin endpoints to manage any user's API keys (`/api/admin/users/{username}/api-keys`)
- **Golden Paths API Endpoints** - REST API to list and retrieve golden path metadata (`/api/golden-paths`)
- CLI admin commands for managing user API keys: `admin user-api-keys`, `admin user-generate-key`, `admin user-revoke-key`
- API client hook for deprovision operations (`useDeprovisionApplication`)
- Environments API client method (`getEnvironments()`)
- Navigation link for Environments page in Web UI
- Deprecation headers for legacy `/api/specs` endpoints

### Changed
- Restructured README.md with improved sections: Installation, Quickstart, Production Setup, Architecture
- Updated documentation to clearly separate demo environment (local development) from production deployment
- Enhanced API documentation with complete endpoint reference and examples
- Improved error handling with rich error context and suggestions
- **API Client Consistency** - Updated `deleteApplication()` to use `/api/applications/{name}` endpoint (was `/api/apps/{name}`)
- **Application Details Pane** - Added action buttons for deprovision and delete operations with descriptive dialogs
- **Alert Dialogs** - Improved UX with clear distinction between deprovision (keeps audit trail) and delete (permanent)
- **CLI Endpoint Migration** - Updated all CLI methods to use `/api/applications` instead of `/api/specs`
- **UI Endpoint Migration** - Updated Web UI API client to use `/api/applications` for all operations
- **Resource Management** - CLI now supports full resource lifecycle (get, delete, update, transition, health check)

### Fixed
- Dependabot npm directory configuration (corrected from /v2 to /web-ui)
- 40+ golangci-lint errors (errcheck and staticcheck issues)
- Multiple logBuffer.Write error checking issues in handlers.go
- Pre-commit hook compatibility issues
- **Endpoint Inconsistency (P0-1)** - ✅ Fixed Web UI API client to use correct `/applications/` endpoint instead of `/apps/`
- **Endpoint Inconsistency (P0-2)** - ✅ Fixed CLI to use consistent `/api/applications` endpoint across all commands
- **Missing UI Operations (P0-3)** - ✅ Resolved: deprovision operation now available in Web UI
- **Missing CLI Commands (P0-5)** - ✅ Resolved: added full resource management command suite
- **Environment List Not Available in UI (P0-8)** - ✅ Resolved: created `/environments` page with full environment listing
- **User Management API Missing (P0-7)** - ✅ Resolved: CLI now uses API endpoints instead of direct file access
- **API Key Management Access Control (P0-6)** - ✅ Resolved: admins can now manage any user's API keys via API
- **Golden Paths Not Server-Managed (P0-4)** - ✅ Resolved: golden paths now exposed via REST API
- **Missing Health Check (P1-8)** - ✅ Resolved: added resource health command to CLI

### Security
- Added gosec security scanning to CI/CD pipeline
- Implemented comprehensive input validation and sanitization
- Added security best practices documentation for production deployments
- Enhanced secret management guidelines with Vault and Kubernetes Secrets integration

### Documentation
- **Gap Analysis Report** - Identified and categorized 47 inconsistencies:
  - 8 P0 (Critical) - Including endpoint naming, missing operations
  - 12 P1 (High Priority) - Including admin features, resource management
  - 18 P2 (Medium Priority) - Including formatting, filtering
  - 9 P3 (Low Priority) - Including quality-of-life improvements
- **Terminology Standardization** - Defined context-specific usage:
  - "Score Spec" for YAML files
  - "Application" for deployed instances
  - "App" for UI brevity
  - Clear definitions for workflows, golden paths, resources, and environments
- **Coverage Matrix** - Complete API ↔ CLI ↔ UI mapping showing feature parity across interfaces

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
