# innominatus Product Backlog

This document tracks user stories, improvements, bug reports, and technical debt for the innominatus project.

**Status Legend:**
- `[ ]` - Not started
- `[~]` - In progress
- `[x]` - Completed

---

## High Priority

### User Stories

#### US-001: Improved Error Messages for CLI Users
- **As a** developer using innominatus-ctl
- **I want** clear, actionable error messages when deployments fail
- **So that** I can quickly understand and fix issues without contacting the platform team
- **Acceptance Criteria:**
  - Error messages include context (app name, step that failed)
  - Suggestions for common fixes are provided
  - Error codes are consistent and documented
- **Status:** `[ ]`
- **Priority:** High
- **Effort:** Medium (3-5 days)
- **Related Files:** `internal/errors/`, `internal/cli/commands.go`

#### US-002: Workflow Progress Tracking in Web UI
- **As a** platform user
- **I want** to see real-time progress of my workflow execution in the Web UI
- **So that** I don't have to keep running CLI commands to check status
- **Acceptance Criteria:**
  - Dashboard shows running workflows with progress indicators
  - Each workflow step shows current status (pending/running/completed/failed)
  - Auto-refresh updates without manual page reload
  - Estimated time remaining is displayed
- **Status:** `[ ]`
- **Priority:** High
- **Effort:** Large (5-8 days)
- **Related Files:** `web-ui/src/app/workflows/page.tsx`, `internal/server/handlers.go`

#### US-003: API Key Management UI
- **As a** platform user
- **I want** to manage my API keys through the Web UI
- **So that** I can easily create, rotate, and revoke keys without using the CLI
- **Acceptance Criteria:**
  - Profile page shows list of active API keys
  - Users can generate new keys with custom names and expiry
  - Users can revoke keys with confirmation dialog
  - Key creation shows the key only once with copy button
- **Status:** `[~]` (Partially implemented - see `web-ui/src/components/profile/security-tab.tsx`)
- **Priority:** High
- **Effort:** Small (1-2 days)
- **Related Files:** `web-ui/src/app/profile/page.tsx`, `internal/server/auth_handlers.go`

#### US-004: Golden Path Parameter Validation
- **As a** platform engineer
- **I want** golden path parameters to be validated before workflow execution
- **So that** users get immediate feedback on invalid inputs
- **Acceptance Criteria:**
  - Parameter types are validated (string, number, boolean)
  - Required parameters are enforced
  - Default values are applied correctly
  - Clear validation error messages
- **Status:** `[x]` (Implemented in `internal/goldenpaths/parameter_validator.go`)
- **Priority:** High
- **Effort:** Completed
- **Related Files:** `internal/goldenpaths/parameter_validator.go`

#### US-005: Product Team Workflows - Multi-Tier Executor Activation
- **As a** product team (e.g., e-commerce, analytics, ML)
- **I want** my product-specific workflows to execute when applications using my product are deployed
- **So that** I can automatically configure infrastructure and services for consuming applications
- **Acceptance Criteria:**
  - Multi-tier workflow executor activated in production server
  - Product workflows from `workflows/products/` execute automatically
  - Workflows trigger based on app metadata (e.g., `product: ecommerce`)
  - Platform, product, and application workflows execute in correct phases
- **Status:** `[ ]` (Implementation exists but not wired - see docs/PRODUCT_WORKFLOW_GAPS.md GAP-1)
- **Priority:** High
- **Effort:** Small (1 hour code change + 8 hours testing)
- **Related Files:** `internal/server/handlers.go:226`, `internal/workflow/executor.go:96-126`
- **Blockers:** None - implementation complete, just needs wiring

#### US-006: Product Workflow Policy Enforcement
- **As a** platform engineer
- **I want** product workflow policies to be enforced before execution
- **So that** only approved product workflows can run in production
- **Acceptance Criteria:**
  - `allowedProductWorkflows` policy from admin-config.yaml enforced
  - Unauthorized workflows blocked with clear error message
  - Audit log of policy violations
  - Platform team notified of violation attempts
- **Status:** `[ ]` (Validation function exists but not called - see docs/PRODUCT_WORKFLOW_GAPS.md GAP-3)
- **Priority:** High
- **Effort:** Small (2 hours)
- **Related Files:** `internal/workflow/resolver.go:369`, `internal/server/handlers.go`
- **Security Impact:** High - currently any workflow in `workflows/products/` can execute
- **Depends On:** US-005 (Multi-tier executor activation)

### Bug Reports

#### BUG-001: Debug Print Statements in Production Code
- **Description:** Multiple `fmt.Println` and debug print statements found in production code
- **Impact:** Clutters logs, makes production debugging harder
- **Severity:** Medium
- **Steps to Reproduce:**
  1. Start server with standard configuration
  2. Check logs for "DEBUG:" prefixed messages
  3. Notice console output mixed with structured logging
- **Expected Behavior:** All logging should use structured logger (zerolog)
- **Actual Behavior:** Mix of `fmt.Println`, `log.Println`, and structured logging
- **Status:** `[ ]`
- **Priority:** High
- **Effort:** Small (1 day)
- **Files to Fix:**
  - `cmd/server/main.go:67,85,87,89`
  - `internal/database/database.go:50`
  - `workflow-demo.go` (entire file)
- **Solution:** Replace all print statements with `log.Info()`, `log.Debug()`, etc.

#### BUG-002: TODO Comments Indicate Incomplete Features
- **Description:** Several TODO comments indicate incomplete implementations
- **Impact:** Features may not work as expected, potential production issues
- **Severity:** Medium
- **Status:** `[ ]`
- **Priority:** High
- **Locations:**
  - `internal/server/handlers.go:1532,1555` - Demo API integration incomplete
  - `internal/cli/commands.go:622` - Parameter substitution not implemented
  - `web-ui/src/app/ai-assistant/page.tsx:166` - Deployment not implemented
  - `web-ui/src/components/navigation.tsx:194` - User role hardcoded
- **Effort:** Medium (3-5 days total)

#### BUG-003: Missing Test Coverage for Critical Paths
- **Description:** Some critical code paths lack test coverage
- **Impact:** Potential bugs in production, harder to refactor safely
- **Severity:** Medium
- **Status:** `[ ]`
- **Priority:** Medium
- **Areas Needing Tests:**
  - OIDC callback flow (`internal/server/auth_handlers.go`)
  - Session management edge cases
  - Workflow executor error handling
  - Database migration rollback scenarios
- **Effort:** Large (5-8 days)

---

## Medium Priority

### User Stories

#### US-007: Product Workflow Discovery API
- **As a** product team developer
- **I want** API endpoints to discover and query product workflows
- **So that** I can understand what workflows are available and their requirements
- **Acceptance Criteria:**
  - `GET /api/workflows/products` - List all products with workflows
  - `GET /api/workflows/products/{product}` - List workflows for specific product
  - `GET /api/workflows/products/{product}/{workflow}` - Get workflow details
  - `POST /api/workflows/products/validate` - Validate workflow YAML
  - Response includes metadata (owner, phase, triggers, step count)
- **Status:** `[ ]` (No API endpoints exist - see docs/PRODUCT_WORKFLOW_GAPS.md GAP-2)
- **Priority:** Medium
- **Effort:** Medium (8 hours)
- **Related Files:** `internal/server/handlers.go`, `internal/workflow/resolver.go`
- **Depends On:** US-005 (Multi-tier executor activation)

#### US-008: Product Team CLI Commands
- **As a** product team developer
- **I want** CLI commands to manage and test product workflows
- **So that** I can develop workflows efficiently without platform team involvement
- **Acceptance Criteria:**
  - `innominatus-ctl list-products` - List products with workflow count
  - `innominatus-ctl list-product-workflows [product]` - List workflows for product
  - `innominatus-ctl validate-product-workflow <file>` - Validate workflow YAML
  - `innominatus-ctl test-product-workflow <workflow> <score-spec> --dry-run` - Test execution
  - Clear error messages for validation failures
- **Status:** `[ ]` (No CLI commands exist - see docs/PRODUCT_WORKFLOW_GAPS.md GAP-4)
- **Priority:** Medium
- **Effort:** Medium (8 hours)
- **Related Files:** `cmd/cli/main.go`, `internal/cli/commands.go`
- **Depends On:** US-005 (Multi-tier executor activation)

#### US-009: Workflow Visualization Export
- **As a** platform engineer
- **I want** to export workflow graphs as images (PNG/SVG)
- **So that** I can include them in documentation and presentations
- **Acceptance Criteria:**
  - Export button on workflow graph page
  - Support for PNG, SVG, and PDF formats
  - Configurable image size and resolution
  - Download with meaningful filename
- **Status:** `[~]` (Mermaid export exists, see `internal/server/handlers.go:1160`)
- **Priority:** Medium
- **Effort:** Medium (2-3 days)
- **Related Files:** `web-ui/src/components/graph-visualization.tsx`, `internal/graph/adapter.go`

#### US-010: Bulk Operations for Applications
- **As a** platform engineer
- **I want** to perform bulk operations (delete, deprovision) on multiple apps
- **So that** I can manage applications efficiently during cleanup or migration
- **Acceptance Criteria:**
  - Multi-select checkboxes on applications list
  - Bulk delete with confirmation
  - Bulk deprovision with confirmation
  - Progress indicator for bulk operations
  - Rollback capability if operations fail
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Medium (3-4 days)
- **Related Files:** `web-ui/src/app/dashboard/page.tsx`, `internal/server/handlers.go`

#### US-011: Workflow Step Retry Mechanism
- **As a** developer
- **I want** to retry failed workflow steps without restarting the entire workflow
- **So that** I can recover from transient failures efficiently
- **Acceptance Criteria:**
  - Retry button on failed workflow steps
  - Configurable retry attempts (1-5)
  - Exponential backoff between retries
  - Step-level retry history tracked
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Large (5-7 days)
- **Related Files:** `internal/workflow/executor.go`, `internal/queue/queue.go`

#### US-012: Resource Usage Dashboard
- **As a** platform engineer
- **I want** a dashboard showing resource utilization across all deployments
- **So that** I can identify cost optimization opportunities
- **Acceptance Criteria:**
  - Show CPU/memory requests and limits per app
  - Show storage usage (PVCs)
  - Cost estimates based on resource definitions
  - Sortable and filterable by team/environment
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Large (6-8 days)
- **Related Files:** `internal/resources/manager.go`, `web-ui/src/app/resources/page.tsx`

### Technical Improvements

#### TECH-001: Consolidate Logging Strategy
- **Description:** Migrate all logging to structured logging with consistent log levels
- **Benefits:**
  - Easier log parsing and analysis
  - Better production debugging
  - Consistent log format across all components
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Medium (3-4 days)
- **Tasks:**
  - Replace all `fmt.Println` with `log.Info/Debug`
  - Replace all `log.Println` with structured logging
  - Add log levels to all log statements
  - Document logging conventions in CONTRIBUTING.md
- **Related Files:** All `.go` files

#### TECH-002: Database Connection Pooling Optimization
- **Description:** Review and optimize database connection pool settings for HA
- **Benefits:**
  - Better performance under load
  - Reduced connection exhaustion
  - Faster query response times
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Small (2-3 days)
- **Tasks:**
  - Add connection pool metrics
  - Tune MaxOpenConns and MaxIdleConns
  - Add connection lifecycle logging
  - Document pool settings in operations guide
- **Related Files:** `internal/database/database.go`

#### TECH-003: API Response Caching
- **Description:** Implement caching for frequently accessed read-only endpoints
- **Benefits:**
  - Reduced database load
  - Faster API response times
  - Better scalability
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Medium (4-5 days)
- **Endpoints to Cache:**
  - `GET /api/goldenpaths` (5 min TTL)
  - `GET /api/stats` (30 sec TTL)
  - `GET /api/environments` (2 min TTL)
- **Related Files:** `internal/server/handlers.go`, add `internal/cache/`

#### TECH-004: Improve Build Script Error Handling
- **Description:** Add better error handling and validation to `build-web-ui.sh`
- **Benefits:**
  - Clearer build failure messages
  - Faster troubleshooting
  - Better CI/CD reliability
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Small (1 day)
- **Tasks:**
  - Check for Node.js version requirements
  - Validate npm dependencies before build
  - Add cleanup on failure
  - Better error messages for common issues
- **Related Files:** `scripts/build-web-ui.sh`

---

## Low Priority

### User Stories

#### US-013: Dark Mode Toggle Persistence
- **As a** Web UI user
- **I want** my dark mode preference to persist across sessions
- **So that** I don't have to toggle it every time I log in
- **Acceptance Criteria:**
  - Dark mode setting saved to user preferences
  - Setting syncs across browser tabs
  - Setting persists after logout/login
- **Status:** `[ ]`
- **Priority:** Low
- **Effort:** Small (1 day)
- **Related Files:** `web-ui/src/contexts/theme-context.tsx`

#### US-014: Keyboard Shortcuts for Common Actions
- **As a** power user
- **I want** keyboard shortcuts for common CLI operations
- **So that** I can work more efficiently
- **Acceptance Criteria:**
  - `Ctrl+L` to list applications
  - `Ctrl+D` to deploy last spec
  - `Ctrl+S` to check status
  - `/?` to show help
  - Shortcuts documented in CLI help
- **Status:** `[ ]`
- **Priority:** Low
- **Effort:** Small (2 days)
- **Related Files:** `cmd/cli/main.go`

#### US-015: Workflow Templates Gallery
- **As a** developer
- **I want** a gallery of pre-built workflow templates
- **So that** I can quickly start with common deployment patterns
- **Acceptance Criteria:**
  - Template gallery in Web UI
  - Templates for common stacks (Node.js, Python, Go, Java)
  - One-click template download
  - Templates include documentation
- **Status:** `[ ]`
- **Priority:** Low
- **Effort:** Medium (3-4 days)
- **Related Files:** `web-ui/src/app/workflows/page.tsx`, add `templates/` directory

### Technical Improvements

#### TECH-005: Improve CLI Test Coverage
- **Description:** Increase test coverage for CLI commands
- **Current Coverage:** ~60% (estimated)
- **Target Coverage:** 85%+
- **Status:** `[ ]`
- **Priority:** Low
- **Effort:** Large (5-7 days)
- **Focus Areas:**
  - Command flag parsing
  - Authentication flow
  - Error handling
  - Output formatting
- **Related Files:** `internal/cli/commands_test.go`

#### TECH-006: Add OpenTelemetry Tracing
- **Description:** Enhance tracing with OpenTelemetry for better observability
- **Benefits:**
  - Distributed tracing across services
  - Better performance debugging
  - Integration with observability platforms
- **Status:** `[ ]`
- **Priority:** Low
- **Effort:** Large (6-8 days)
- **Tasks:**
  - Replace custom tracing with OpenTelemetry
  - Add span annotations to critical paths
  - Add trace context propagation
  - Document tracing setup
- **Related Files:** `internal/tracing/tracer.go`

#### TECH-007: Helm Chart Improvements
- **Description:** Enhance Helm chart with advanced features
- **Status:** `[ ]`
- **Priority:** Low
- **Effort:** Medium (3-4 days)
- **Improvements:**
  - Add NetworkPolicy templates
  - Add PodDisruptionBudget
  - Add ServiceMonitor for Prometheus Operator
  - Add Grafana dashboard ConfigMaps
  - Improve values.yaml documentation
- **Related Files:** `charts/innominatus/`

---

## Technical Debt

#### DEBT-001: Hardcoded User Roles in Web UI
- **Description:** User roles hardcoded instead of fetched from API
- **Location:** `web-ui/src/components/navigation.tsx:194`
- **Impact:** Admin features shown to all users
- **Status:** `[ ]`
- **Priority:** High
- **Effort:** Small (1 day)
- **Solution:** Add `/api/auth/me` endpoint, fetch user role on login

#### DEBT-002: Incomplete Demo API Handlers
- **Description:** Demo handlers marked as TODO, not integrated with CLI demo functionality
- **Location:** `internal/server/handlers.go:1532,1555`
- **Impact:** Web UI demo controls don't work
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Medium (2-3 days)
- **Solution:** Integrate with `internal/demo/installer.go` functions

#### DEBT-003: Missing Parameter Substitution
- **Description:** Golden path parameter substitution not fully implemented
- **Location:** `internal/cli/commands.go:622`
- **Impact:** Parameters passed to CLI not substituted in workflow steps
- **Status:** `[ ]`
- **Priority:** Medium
- **Effort:** Medium (2-3 days)
- **Solution:** Implement template variable replacement in workflow executor

#### DEBT-004: Unused TypeScript Definitions
- **Description:** TypeScript warnings for unused types/variables in Web UI
- **Location:** `web-ui/src/app/ai-assistant/page.tsx:11,31,113,158,165`
- **Impact:** Code bloat, potential confusion
- **Status:** `[ ]`
- **Priority:** Low
- **Effort:** Small (< 1 day)
- **Solution:** Remove unused imports and variables

---

## Feature Requests

### FEAT-001: Product Provisioner Registry
- **Description:** Allow product teams to register custom provisioners for product-specific resource types
- **Benefits:**
  - Product teams can define custom resources (e.g., payment-gateway, data-lake)
  - Extends innominatus without modifying core code
  - Enables product-specific infrastructure automation
- **Status:** `[ ]` (SDK interface exists, no registry - see docs/PRODUCT_WORKFLOW_GAPS.md GAP-5)
- **Priority:** Medium (after US-005, US-007, US-008)
- **Effort:** Large (16 hours)
- **Requirements:**
  - Provisioner registry for loading and managing provisioners
  - Support for Go plugins or gRPC-based provisioners
  - Admin config schema for provisioner registration
  - Product provisioner discovery API
  - Example provisioners (payment-gateway, data-lake)
- **Related Files:** `pkg/sdk/provisioner.go`, `internal/platform/`, `admin-config.yaml`
- **Examples:**
  - E-commerce team registers `payment-gateway` provisioner
  - Analytics team registers `data-lake` provisioner
  - ML team registers `model-serving` provisioner

### FEAT-002: Multi-Tenancy Support
- **Description:** Add full multi-tenancy with tenant isolation
- **Benefits:**
  - Support multiple organizations in single deployment
  - Improved security through tenant isolation
  - Cost efficiency through shared infrastructure
- **Status:** `[ ]`
- **Priority:** Future
- **Effort:** X-Large (3-4 weeks)
- **Requirements:**
  - Tenant-scoped database queries
  - Tenant-based RBAC
  - Tenant-scoped workflows and resources
  - Tenant management UI

### FEAT-003: Audit Log Viewer
- **Description:** Web UI for viewing and filtering audit logs
- **Benefits:**
  - Compliance and security monitoring
  - Easier troubleshooting
  - Better accountability
- **Status:** `[ ]`
- **Priority:** Future
- **Effort:** Large (1-2 weeks)
- **Requirements:**
  - Audit log collection in database
  - Web UI with filtering and search
  - Export to CSV/JSON
  - Retention policy configuration

### FEAT-004: Workflow Approval Gates
- **Description:** Add manual approval steps to workflows
- **Benefits:**
  - Production deployment safety
  - Compliance requirements
  - Change management integration
- **Status:** `[ ]`
- **Priority:** Future
- **Effort:** X-Large (3-4 weeks)
- **Requirements:**
  - Approval step type in workflows
  - Notification system for approvers
  - Approval UI in Web interface
  - Approval history and audit trail

### FEAT-005: Cost Estimation and Tracking
- **Description:** Estimate and track infrastructure costs
- **Benefits:**
  - Budget management
  - Cost optimization
  - Chargeback to teams
- **Status:** `[ ]`
- **Priority:** Future
- **Effort:** X-Large (4-5 weeks)
- **Requirements:**
  - Cost models for resources
  - Cost estimation before deployment
  - Cost tracking dashboard
  - Export cost reports

---

## Bugs

### [P1] Integrate demo API handlers with CLI functionality (ID: BL-BUG-002)
- **Description**: Demo handlers marked as TODO (internal/server/handlers.go:1893, 1916). Web UI demo controls don't work because API endpoints are not integrated with CLI demo functionality. Wire up demo API endpoints to call internal/demo/installer functions for demo-time, demo-status, demo-nuke. Related to existing DEBT-002.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: TODO Scanner
- **Added**: 2025-10-19
- **Files**: internal/server/handlers.go, internal/demo/installer.go
### [P1] Complete parameter substitution in golden paths (ID: BL-BUG-001)
- **Description**: Golden path parameter substitution not fully implemented (internal/cli/commands.go:622). Parameters passed via CLI are not being substituted into workflow steps, breaking golden path workflows with parameters. Implement template variable replacement in workflow executor using golden path parameters. Related to existing BUG-002.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: TODO Scanner
- **Added**: 2025-10-19
- **Files**: internal/cli/commands.go, internal/workflow/executor.go
*Items added automatically by autopilot backlog scanner - categorized by bug type*

---

## Improvements

### [P1] Implement workflow retry dialog in Web UI (ID: BL-IMP-003)
- **Description**: Retry workflow dialog marked as TODO (web-ui/src/app/workflows/page.tsx:115). Users cannot retry failed workflows from Web UI. Create dialog to upload updated workflow spec and call retry API endpoint to enable self-service workflow recovery.
- **Priority**: P1 (High)
- **Effort**: S (Small, <2h)
- **Source**: TODO Scanner
- **Added**: 2025-10-19
- **Files**: web-ui/src/app/workflows/page.tsx
### [P1] Implement AI deployment functionality in Web UI (ID: BL-IMP-002)
- **Description**: AI deployment marked as TODO (web-ui/src/app/ai-assistant/page.tsx:166). AI assistant can generate specs but cannot deploy them. Implement deployment via API call to POST /api/specs or golden path execution to complete the AI-assisted deployment workflow.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: TODO Scanner
- **Added**: 2025-10-19
- **Files**: web-ui/src/app/ai-assistant/page.tsx, web-ui/src/lib/api.ts
### [P0] Implement AI chat functionality (ID: BL-IMP-001)
- **Description**: AI chat command marked as TODO at cmd/cli/main.go:327. User-facing feature that appears incomplete despite being advertised in CLI help. Complete AI chat implementation or remove from CLI if not planned for current release. Integration with AI assistant API, conversation history management, and Score spec generation required.
- **Priority**: P0 (Critical)
- **Effort**: L (Large, 1-3d)
- **Source**: TODO Scanner
- **Added**: 2025-10-19
- **Files**: cmd/cli/main.go, cmd/cli/chat.go, internal/ai/
*Items added automatically by autopilot backlog scanner - categorized by improvement type*

---

## Maintenance

### [P3] Update metrics pusher logging (ID: BL-MNT-030)
- **Description**: internal/metrics/pusher.go line 77 uses log.Println instead of structured logging. For consistency with logging standards, replace with zerolog structured logging using appropriate log level (log.Info or log.Debug).
- **Priority**: P3 (Low)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/metrics/pusher.go
### [P3] Refactor admin config display (ID: BL-MNT-029)
- **Description**: internal/admin/config.go contains 10 fmt.Println statements for config display. Should return formatted string instead for better testability and reusability. Add String() method to Config struct, return formatted string instead of printing directly.
- **Priority**: P3 (Low)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/admin/config.go
### [P3] Refactor demo cheatsheet logging (ID: BL-MNT-028)
- **Description**: internal/demo/cheatsheet.go contains 100+ fmt.Println statements for demo credential display. Should be refactored for better testability. Extract display logic into separate function that returns string, then print. Improves testability and separation of concerns.
- **Priority**: P3 (Low)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/demo/cheatsheet.go
### [P3] Refactor CLI chat command logging (ID: BL-MNT-027)
- **Description**: cmd/cli/chat.go contains 80+ fmt.Println statements for interactive chat output. While appropriate for CLI UX, should be reviewed for consistency. Document these as intentional user-facing output (not logging), or extract to UI layer for better separation of concerns.
- **Priority**: P3 (Low)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: cmd/cli/chat.go
### [P3] Clean up test integration files (ID: BL-MNT-026)
- **Description**: tests/integration/test-workflow-retry.go contains many fmt.Println statements for test output. Should use testing.T.Log for better test output management and consistency with Go testing best practices. Replace fmt.Println with t.Log, t.Logf for better test output control.
- **Priority**: P3 (Low)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: tests/integration/test-workflow-retry.go, tests/integration/create-failed-workflow.go
### [P3] Refactor workflow-demo.go logging (ID: BL-MNT-025)
- **Description**: workflow-demo.go contains 15 fmt.Println statements for demo output. While this is a demo file, should follow logging standards for consistency or document as demo-specific output. Replace with structured logging or add comment explaining demo-specific user-facing output.
- **Priority**: P3 (Low)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: workflow-demo.go
### [P3] Replace deprecated Faker dependency (ID: BL-MNT-024)
- **Description**: github.com/bxcodec/faker/v3 v3.8.1 is deprecated. Should migrate to maintained alternative to avoid future security issues and compatibility problems. Migrate to github.com/go-faker/faker or remove if only used in tests and not critical.
- **Priority**: P3 (Low)
- **Effort**: S (Small, <2h)
- **Source**: Dependency Audit
- **Added**: 2025-10-19
- **Files**: go.mod
### [P2] Fix ESLint warnings in useApi hook (ID: BL-MNT-023)
- **Description**: ESLint warning disabled at web-ui/src/hooks/use-api.ts:41 (react-hooks/exhaustive-deps). Indicates potential missing dependency in useEffect hook. Fix dependency array or properly justify why exhaustive-deps can be safely disabled with detailed comment.
- **Priority**: P2 (Medium)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: web-ui/src/hooks/use-api.ts
### [P2] Remove unused TypeScript variables (ID: BL-MNT-022)
- **Description**: Unused TypeScript variables found in web-ui/src/app/ai-assistant/page.tsx (lines 11,31,113,158,165). Code bloat and potential confusion for developers. Remove unused imports and variables. Related to existing DEBT-004.
- **Priority**: P2 (Medium)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: web-ui/src/app/ai-assistant/page.tsx
### [P2] Add test coverage for health, tracing, teams, users, vault packages (ID: BL-MNT-021)
- **Description**: internal/health, internal/tracing, internal/teams, internal/users, internal/vault all have 0% coverage. Add comprehensive unit tests for all functions and error cases to ensure reliability of these critical utility and domain packages.
- **Priority**: P2 (Medium)
- **Effort**: M (Medium, 2-8h)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/health/, internal/tracing/, internal/teams/, internal/users/, internal/vault/
### [P2] Update Bytedance Sonic dependency (ID: BL-MNT-020)
- **Description**: github.com/bytedance/sonic is outdated (v1.14.0, latest v1.14.1). Update to latest version for performance improvements and bug fixes. Run: go get github.com/bytedance/sonic@v1.14.1 && go mod tidy
- **Priority**: P2 (Medium)
- **Effort**: S (Small, <2h)
- **Source**: Dependency Audit
- **Added**: 2025-10-19
- **Files**: go.mod, go.sum
### [P2] Update OpenTelemetry detectors dependency (ID: BL-MNT-019)
- **Description**: github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp is outdated (v1.29.0, latest v1.30.0). Update to latest version for bug fixes and improvements. Run: go get github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp@v1.30.0 && go mod tidy
- **Priority**: P2 (Medium)
- **Effort**: S (Small, <2h)
- **Source**: Dependency Audit
- **Added**: 2025-10-19
- **Files**: go.mod, go.sum
### [P2] Deduplicate database connection pool configuration (ID: BL-MNT-018)
- **Description**: Connection pool settings (SetMaxOpenConns, SetMaxIdleConns, SetConnMaxLifetime) duplicated in database.go. Magic numbers (25, 5 minutes) should be constants. Extract configureConnectionPool(db *sql.DB) function, define constants for pool settings to improve maintainability.
- **Priority**: P2 (Medium)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/database/database.go
### [P2] Deduplicate database connection string building (ID: BL-MNT-017)
- **Description**: Connection string building logic duplicated in internal/database/database.go (lines 43-48, 83-88). DRY violation increases maintenance burden. Extract buildConnectionString(config Config) string function, reuse in both NewDatabase and NewDatabaseWithConfig.
- **Priority**: P2 (Medium)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/database/database.go
### [P2] Refactor installer.go for maintainability (ID: BL-MNT-016)
- **Description**: internal/demo/installer.go is 1,423 lines. Component installation logic should be modularized. Extract each component (Gitea, ArgoCD, Vault, Minio, etc.) into separate installer files for better maintainability and testability.
- **Priority**: P2 (Medium)
- **Effort**: M (Medium, 2-8h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/demo/installer.go
### [P2] Refactor executor.go to reduce complexity (ID: BL-MNT-015)
- **Description**: internal/workflow/executor.go is 1,599 lines. Should be refactored to separate concerns. Split into: executor.go (core), terraform_executor.go, ansible_executor.go, kubernetes_executor.go, step_runner.go for better separation of workflow step execution logic.
- **Priority**: P2 (Medium)
- **Effort**: L (Large, 1-3d)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/workflow/executor.go
### [P2] Refactor commands.go to reduce file size (ID: BL-MNT-014)
- **Description**: internal/cli/commands.go is 2,238 lines, violating SRP. Should be split into focused command modules. Split into: workflow_commands.go, app_commands.go, admin_commands.go, demo_commands.go, login_commands.go, goldenpath_commands.go for better code organization and maintainability.
- **Priority**: P2 (Medium)
- **Effort**: L (Large, 1-3d)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/cli/commands.go
### [P2] Refactor handlers.go to reduce file size (ID: BL-MNT-013)
- **Description**: internal/server/handlers.go is 3,570 lines, violating Single Responsibility Principle (SRP). File should be <500 lines per CLAUDE.md standards. Split into multiple files: auth_handlers.go, workflow_handlers.go, app_handlers.go, admin_handlers.go, demo_handlers.go, stats_handlers.go for better maintainability.
- **Priority**: P2 (Medium)
- **Effort**: L (Large, 1-3d)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: internal/server/handlers.go
### [P1] Replace TypeScript any types with proper interfaces (ID: BL-MNT-012)
- **Description**: 13+ instances of 'any' type found in TypeScript codebase, violating type safety standards. Reduces IDE assistance and increases runtime errors. Define proper TypeScript interfaces for all data structures, API responses, component props. Files affected: workflow-types.ts, api.ts, use-api.ts, graph components, application/resource panes. Related to existing DEBT-004.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: web-ui/src/lib/workflow-types.ts, web-ui/src/lib/api.ts, web-ui/src/hooks/use-api.ts, web-ui/src/components/graph-visualization.tsx, web-ui/src/components/application-details-pane.tsx, web-ui/src/components/workflow-diagram.tsx
### [P1] Add test coverage for validation, logging, security packages (ID: BL-MNT-011)
- **Description**: internal/validation, internal/logging, internal/security all have 0% coverage. These utility packages need comprehensive testing. Add unit tests for all validation rules, logging utilities, security functions, input sanitization, and error cases.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/validation/, internal/logging/, internal/security/
### [P1] Add test coverage for metrics package (ID: BL-MNT-010)
- **Description**: internal/metrics package has only 5.8% coverage. Prometheus metrics export and push functionality inadequately tested. Expand test coverage for metric collection, push gateway integration, metric registration, and error handling in metrics collection.
- **Priority**: P1 (High)
- **Effort**: S (Small, <2h)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/metrics/metrics.go, internal/metrics/pusher.go, internal/metrics/pusher_test.go
### [P1] Add test coverage for resources package (ID: BL-MNT-009)
- **Description**: internal/resources package has only 6.9% coverage. Resource provisioning (Kubernetes, ArgoCD, Gitea) lacks comprehensive testing. Expand tests for all provisioners, error handling, cleanup logic, integration scenarios, and rollback procedures.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/resources/
### [P1] Add test coverage for server handlers (ID: BL-MNT-008)
- **Description**: internal/server package has only 7.7% coverage. HTTP handlers represent the primary API surface with minimal testing. Add comprehensive integration tests for all API endpoints, authentication, authorization, request validation, error handling, and response formatting.
- **Priority**: P1 (High)
- **Effort**: L (Large, 1-3d)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/server/handlers.go, internal/server/handlers_test.go
### [P1] Add test coverage for database package (ID: BL-MNT-007)
- **Description**: internal/database package has only 5.3% coverage. Data persistence layer lacks comprehensive testing. Add tests for connection pooling, migrations, repository methods, transaction handling, error cases, and connection lifecycle management.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/database/, internal/database/database_test.go
### [P1] Add test coverage for auth package (ID: BL-MNT-006)
- **Description**: internal/auth package has only 3.6% coverage. Authentication is security-critical functionality with minimal automated validation. Add tests for OIDC flow, API key validation, session management, token generation, authorization checks, and error cases. Critical for security assurance.
- **Priority**: P1 (High)
- **Effort**: M (Medium, 2-8h)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/auth/, internal/auth/auth_test.go
### [P1] Add test coverage for demo package (ID: BL-MNT-005)
- **Description**: internal/demo package has 0% test coverage (installer, health, reset, git, grafana modules). Demo environment is critical for onboarding but completely untested. Add comprehensive unit and integration tests for demo installation, health checks, cleanup logic, git operations, and Grafana dashboard setup.
- **Priority**: P1 (High)
- **Effort**: L (Large, 1-3d)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: internal/demo/installer.go, internal/demo/health.go, internal/demo/reset.go, internal/demo/git.go, internal/demo/grafana.go
### [P0] Add test coverage for server entry point (ID: BL-MNT-004)
- **Description**: cmd/server/main.go has 0% test coverage. Server initialization, configuration loading, and startup logic untested. Add integration tests for server startup, config loading, database connection, OIDC setup, graceful shutdown, and error handling during initialization.
- **Priority**: P0 (Critical)
- **Effort**: M (Medium, 2-8h)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: cmd/server/main.go, cmd/server/main_test.go
### [P0] Add test coverage for CLI package (ID: BL-MNT-003)
- **Description**: cmd/cli package has 0% test coverage, representing critical user-facing functionality with no automated validation. High risk of regressions. Implement comprehensive test suite for all CLI commands including: authentication, golden paths, workflows, admin commands, chat functionality, login/logout flows, and error handling.
- **Priority**: P0 (Critical)
- **Effort**: L (Large, 1-3d)
- **Source**: Coverage Report
- **Added**: 2025-10-19
- **Files**: cmd/cli/main.go, cmd/cli/chat.go, cmd/cli/main_test.go
### [P0] Remove DEBUG print statements from database.go (ID: BL-MNT-002) ✓
- **Description**: internal/database/database.go contains debug print statements on lines 50, 72, 76 that expose connection details and internal state to console output. This is a security risk in production as connection strings and database names are printed. Remove or replace with conditional debug logging at appropriate level using structured logger.
- **Priority**: P0 (Critical)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Completed**: 2025-10-19
- **Files**: internal/database/database.go
- **Implementation**: Replaced fmt.Printf with structured logging using internal/logging package. Connection strings no longer exposed in logs.
### [P0] Remove debug print statements from production code (ID: BL-MNT-001)
- **Description**: 300+ fmt.Println and log.Println statements found across production code, violating CLAUDE.md logging standards. These clutter logs and make production debugging harder. Major files affected: cmd/server/main.go (30+ instances), cmd/cli/chat.go (80+ instances), internal/demo/cheatsheet.go (100+ instances), internal/database/database.go (3 instances). Replace all with structured zerolog statements (log.Info, log.Debug, log.Error) for consistent, parseable logging.
- **Priority**: P0 (Critical)
- **Effort**: S (Small, <2h)
- **Source**: Code Quality Scanner
- **Added**: 2025-10-19
- **Files**: cmd/server/main.go, cmd/cli/chat.go, internal/demo/cheatsheet.go, internal/database/database.go, internal/admin/config.go, workflow-demo.go, tests/integration/test-workflow-retry.go, internal/metrics/pusher.go
*Items added automatically by autopilot backlog scanner - categorized by maintenance type*

---

## DX

### [P2] Add input validation with helpful feedback (ID: BL-DX-002)
- **Description**: User inputs in CLI lack comprehensive validation with helpful error messages. Users must trial-and-error to discover valid values. Add validation for all user inputs (file paths, enum values, IDs, names) with specific error messages and examples of valid input formats.
- **Priority**: P2 (Medium)
- **Effort**: M (Medium, 2-8h)
- **Source**: Manual Review
- **Added**: 2025-10-19
- **Files**: internal/cli/commands.go, internal/validation/
### [P2] Improve error messages in CLI commands (ID: BL-DX-001)
- **Description**: Many CLI commands return generic error messages like 'invalid input' without specific guidance. Violates CLAUDE.md principle of clear, actionable error messages. Add validation errors with examples: 'Priority must be P0-P3 (got: P5)', 'Effort must be S/M/L/XL (got: MEDIUM)', 'File not found: check path and try again'. Related to existing US-001.
- **Priority**: P2 (Medium)
- **Effort**: M (Medium, 2-8h)
- **Source**: Manual Review
- **Added**: 2025-10-19
- **Files**: internal/cli/commands.go, internal/errors/
*Items added automatically by autopilot backlog scanner - categorized by developer experience type*

---

## Completed Items (Archive)

### BL-MNT-002: Remove DEBUG Print Statements from database.go ✓
- **Completed:** 2025-10-19
- **Implementation:** `internal/database/database.go`
- **Notes:** Security fix - replaced fmt.Printf debug statements with structured logging. Connection strings and sensitive data no longer exposed in logs. Uses internal/logging package for consistent logging.

### US-004: Golden Path Parameter Validation ✓
- **Completed:** 2025-10-XX
- **Implementation:** `internal/goldenpaths/parameter_validator.go`
- **Notes:** Full parameter validation with types, required fields, and defaults

---

## Backlog Grooming Notes

**Last Updated:** 2025-10-19
**Next Review:** 2025-11-02

**Focus Areas for Next Sprint:**
- **🔥 HIGH PRIORITY: Activate product workflows (US-005)** - 1 hour code change unlocks major feature
- **🔥 HIGH PRIORITY: Enable workflow policy enforcement (US-006)** - Security critical
- Fix debug logging issues (BUG-001)
- Complete TODO items (BUG-002)
- Improve API key management UI (US-003)

**Product Workflow Roadmap:**
- **Phase 1 (1 week):** US-005 + US-006 (activate multi-tier, enforce policies)
- **Phase 2 (2 weeks):** US-007 + US-008 (API endpoints + CLI commands)
- **Phase 3 (3 weeks):** FEAT-001 (provisioner registry for custom resources)

**Dependencies:**
- US-007 (Product API) depends on US-005 (activation)
- US-008 (Product CLI) depends on US-005 (activation)
- FEAT-001 (Provisioners) depends on US-005, US-007, US-008
- FEAT-004 (Approval) depends on notification system (not yet implemented)
- FEAT-005 (Cost) depends on resource usage tracking (US-012)
- TECH-006 can replace existing custom tracing

**Technical Priorities:**
1. **Activate product workflows (US-005) + policy enforcement (US-006)** ⭐ NEW - HIGHEST IMPACT
2. Remove debug print statements (DEBT-001 → BUG-001)
3. Consolidate logging (TECH-001)
4. Improve test coverage (BUG-003, TECH-005)
5. Complete incomplete features (BUG-002)

**New Documentation:**
- `docs/PRODUCT_WORKFLOW_GAPS.md` - Technical gap analysis for product workflows
- `docs/product-team-guide/README.md` - Product team user guide (with limitations noted)
- More product team docs in progress (see PRODUCT_WORKFLOW_GAPS.md)

---

## Contributing

To add items to this backlog:

1. Use the template format above
2. Assign a unique ID (US-XXX, BUG-XXX, TECH-XXX, FEAT-XXX, DEBT-XXX)
3. Set priority (High/Medium/Low/Future)
4. Estimate effort (Small/Medium/Large/X-Large)
5. Link related files and issues

See [CONTRIBUTING.md](CONTRIBUTING.md) for more details.
