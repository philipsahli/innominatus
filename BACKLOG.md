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

#### US-005: Workflow Visualization Export
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

#### US-006: Bulk Operations for Applications
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

#### US-007: Workflow Step Retry Mechanism
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

#### US-008: Resource Usage Dashboard
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

#### US-009: Dark Mode Toggle Persistence
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

#### US-010: Keyboard Shortcuts for Common Actions
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

#### US-011: Workflow Templates Gallery
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

### FEAT-001: Multi-Tenancy Support
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

### FEAT-002: Audit Log Viewer
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

### FEAT-003: Workflow Approval Gates
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

### FEAT-004: Cost Estimation and Tracking
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

## Completed Items (Archive)

### US-004: Golden Path Parameter Validation ✓
- **Completed:** 2025-10-XX
- **Implementation:** `internal/goldenpaths/parameter_validator.go`
- **Notes:** Full parameter validation with types, required fields, and defaults

---

## Backlog Grooming Notes

**Last Updated:** 2025-10-14
**Next Review:** 2025-10-28

**Focus Areas for Next Sprint:**
- Fix debug logging issues (BUG-001)
- Complete TODO items (BUG-002)
- Improve API key management UI (US-003)
- Add workflow progress tracking (US-002)

**Dependencies:**
- FEAT-003 depends on notification system (not yet implemented)
- FEAT-004 depends on resource usage tracking (US-008)
- TECH-006 can replace existing custom tracing

**Technical Priorities:**
1. Remove debug print statements (DEBT-001 → BUG-001)
2. Consolidate logging (TECH-001)
3. Improve test coverage (BUG-003, TECH-005)
4. Complete incomplete features (BUG-002)

---

## Contributing

To add items to this backlog:

1. Use the template format above
2. Assign a unique ID (US-XXX, BUG-XXX, TECH-XXX, FEAT-XXX, DEBT-XXX)
3. Set priority (High/Medium/Low/Future)
4. Estimate effort (Small/Medium/Large/X-Large)
5. Link related files and issues

See [CONTRIBUTING.md](CONTRIBUTING.md) for more details.
