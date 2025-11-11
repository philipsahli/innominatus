# Innominatus Code & Documentation Review for Demo Readiness

This document provides focused prompts for Claude to analyze code, documentation, and architecture to identify bugs and gaps before the demo.

---

## How to Use

Copy each prompt section and ask Claude to execute it. Claude will analyze the codebase and report findings.

---

## Prompt 1: Application Developer Critical Path Analysis

```
Please analyze the application developer's critical path from Score spec submission to resource provisioning:

ANALYZE:
1. **API Request Flow** (web-ui/src/lib/api.ts, internal/server/handlers.go)
   - Trace: deployApplication() → POST /api/applications → handleDeploySpec()
   - Check: Is Content-Type correctly set to 'application/yaml'? (Recent fix)
   - Verify: Body is raw YAML, not JSON-wrapped (Recent fix)
   - Find: Any missing error handling in the chain

2. **Score Spec Validation** (internal/server/handlers.go:537-556)
   - Check: Is metadata.name validated? (Recent fix at line 552)
   - Look for: Other required fields that should be validated but aren't
   - Verify: Resource type validation against provider capabilities
   - Find: Edge cases that could bypass validation

3. **Resource Creation Flow** (internal/resources/manager.go)
   - Trace: CreateResourceInstance() → database storage → orchestration engine
   - Check: Are all state transitions properly handled?
   - Verify: workflow_execution_id is set correctly
   - Find: Race conditions between resource creation and workflow execution

4. **Orchestration Engine** (internal/orchestration/engine.go)
   - Check: Polling logic for resources with state='requested'
   - Verify: Provider resolution works for all resource types
   - Find: Edge cases where resources might get stuck in 'provisioning'
   - Check: Error handling when workflows fail

5. **Web UI Integration** (web-ui/src/app/dev/applications/[name]/page.tsx)
   - Verify: Application detail page correctly displays resources (Recent addition)
   - Check: Add resource form properly calls api.createResource() (Recent addition)
   - Find: TypeScript type mismatches or any type assertions
   - Check: Loading and error states are handled

REPORT:
- Missing error handling
- Validation gaps
- Race conditions
- Dead code or incomplete features
- Documentation inconsistencies with implementation
```

---

## Prompt 2: Provider System Architecture Review

```
Please analyze the provider system architecture for correctness and completeness:

ANALYZE:
1. **Provider Loading** (internal/providers/loader.go)
   - Check: How are filesystem vs Git providers loaded differently?
   - Verify: Is provider validation thorough? (YAML schema, required fields)
   - Find: Potential issues with provider hot-reload
   - Check: Conflict detection between providers claiming same resource types

2. **Capability Resolution** (internal/orchestration/resolver.go)
   - Trace: How resource types map to providers
   - Check: CRUD operations support (create/update/delete workflows)
   - Verify: Tag-based workflow disambiguation works
   - Find: Edge cases where wrong workflow might be selected

3. **Provider Examples** (providers/*/provider.yaml)
   - Compare: database-team, storage-team, container-team provider definitions
   - Check: Are all using consistent capability declaration format?
   - Verify: resourceTypeCapabilities vs resourceTypes (old vs new format)
   - Find: Providers missing CRUD operations

4. **Workflow Execution** (internal/workflow/executor.go)
   - Check: Are all step types properly registered? (terraform, kubernetes, ansible, etc.)
   - Verify: Variable interpolation ${var} works correctly
   - Find: Missing executors for declared step types
   - Check: Error propagation from failed steps

5. **Documentation Consistency** (CLAUDE.md sections on providers)
   - Compare: Documentation examples vs actual provider implementations
   - Check: Are all provider capabilities documented?
   - Verify: Examples in docs actually work
   - Find: Outdated or missing documentation

REPORT:
- Architecture flaws or inconsistencies
- Missing CRUD workflow implementations
- Provider registration issues
- Documentation gaps or inaccuracies
```

---

## Prompt 3: Authentication & Security Analysis

```
Please analyze authentication flows and identify security issues:

ANALYZE:
1. **OIDC Integration** (internal/auth/oidc.go, internal/cli/oidc_helpers.go)
   - Check: CLI callback port is fixed at 8082 (Recent fix)
   - Verify: Redirect URIs registered in demo installer (internal/demo/installer.go:1042)
   - Find: Token validation and expiry checking
   - Check: PKCE flow implementation for CLI
   - Verify: Token storage security

2. **Session Management** (internal/server/handlers.go, internal/auth/)
   - Check: Session cookie security flags (HttpOnly, Secure, SameSite)
   - Verify: Session expiry and cleanup
   - Find: Potential session fixation vulnerabilities
   - Check: Logout properly clears sessions

3. **API Key Authentication** (internal/database/models.go, internal/server/handlers.go)
   - Check: API key generation uses sufficient entropy
   - Verify: Keys are hashed in database (not stored plaintext)
   - Find: API key expiry enforcement
   - Check: Last used timestamp updates correctly

4. **Authorization/RBAC** (internal/server/middleware.go or similar)
   - Check: Admin vs user role enforcement
   - Verify: Team isolation if implemented
   - Find: Endpoints missing authorization checks
   - Check: Can users access other teams' resources?

5. **Input Validation** (across all handlers)
   - Find: SQL injection vulnerabilities (check database queries)
   - Check: Command injection in workflow executors (especially shell steps)
   - Verify: YAML/JSON parsing is safe
   - Find: Path traversal vulnerabilities in file operations

REPORT:
- Security vulnerabilities (HIGH priority)
- Authentication bypass scenarios
- Missing authorization checks
- Insecure defaults
- Token/session management issues
```

---

## Prompt 4: Database & State Management Review

```
Please analyze database schema, migrations, and state management:

ANALYZE:
1. **Database Migrations** (internal/database/migrations/*.sql)
   - Check: Sequential numbering (001, 002, ..., 010)
   - Verify: Foreign key constraints are correct
   - Find: Missing indexes on frequently queried columns
   - Check: Data type choices (INT vs BIGINT, VARCHAR sizes)
   - Verify: Rollback/down migrations exist (if supported)

2. **Model Definitions** (internal/database/models.go)
   - Compare: Go structs vs database schema
   - Check: GORM tags match database columns
   - Verify: Relationships (foreign keys) properly defined
   - Find: Missing or incorrect database constraints

3. **Resource State Machine** (internal/resources/manager.go)
   - Check: Valid state transitions (requested → provisioning → active)
   - Verify: Invalid transitions are prevented
   - Find: Edge cases where resources could get stuck
   - Check: State transition logging and history

4. **Graph Database** (internal/graph/graph.go, adapter.go)
   - Check: Node creation before edge creation (foreign key integrity)
   - Verify: Graph queries are efficient (no N+1 problems)
   - Find: Orphaned edges or nodes
   - Check: Cascade deletes work correctly

5. **Transaction Handling** (throughout database code)
   - Find: Missing transaction boundaries for multi-step operations
   - Check: Proper rollback on errors
   - Verify: No potential deadlocks
   - Check: Connection pool configuration

REPORT:
- Database integrity issues
- Migration problems
- State machine edge cases
- Performance bottlenecks (missing indexes, N+1 queries)
- Transaction safety issues
```

---

## Prompt 5: Web UI Implementation Review

```
Please analyze the Web UI implementation for bugs and UX issues:

ANALYZE:
1. **Recent Fixes Verification**
   - Check: api.ts deployApplication() sends raw YAML (line 361-369)
   - Verify: Application detail page exists at /dev/applications/[name]/page.tsx
   - Check: createResource() method properly implemented (line 585-600)
   - Verify: generateStaticParams() added for dynamic routes

2. **Component Architecture** (web-ui/src/)
   - Find: Components with missing error boundaries
   - Check: Proper loading states for async operations
   - Verify: TypeScript types are correct (no excessive 'any' usage)
   - Find: Unused imports or dead code

3. **Form Validation** (deploy-wizard/, resources forms)
   - Check: Client-side validation matches backend validation
   - Verify: Error messages are user-friendly
   - Find: Forms that submit without validation
   - Check: Required fields properly marked

4. **API Integration** (lib/api.ts)
   - Check: All API endpoints have corresponding client methods
   - Verify: Error responses are properly handled
   - Find: Missing Content-Type headers
   - Check: Authorization headers included where needed

5. **Route Configuration** (app/ directory structure)
   - Check: Dynamic routes have proper generateStaticParams()
   - Verify: Protected routes check authentication
   - Find: Broken links or 404 pages
   - Check: Next.js 15 compatibility (app router)

REPORT:
- UI bugs or broken functionality
- TypeScript type errors
- Missing validation
- Broken routes or links
- API integration issues
```

---

## Prompt 6: Documentation Completeness Check

```
Please analyze documentation for accuracy and completeness:

ANALYZE:
1. **CLAUDE.md vs Implementation**
   - Compare: Documented CLI commands vs actual commands in internal/cli/commands.go
   - Check: Provider examples in docs vs actual providers/*/provider.yaml
   - Verify: API endpoints documented vs routes in cmd/server/main.go
   - Find: Features documented but not implemented (or vice versa)

2. **README.md vs Current State**
   - Check: Quick start commands actually work
   - Verify: Build instructions are up-to-date
   - Find: Environment variables documented but not used
   - Check: Version numbers and release info current

3. **Code Comments vs Implementation**
   - Find: Comments that don't match code behavior
   - Check: TODOs that should be completed before demo
   - Verify: Function signatures match documentation
   - Find: Complex code without explanatory comments

4. **Examples vs Reality** (examples/ directory)
   - Check: dev-team-app.yaml uses valid resource types
   - Verify: Example workflows actually execute
   - Find: Examples with outdated syntax
   - Check: Referenced providers actually exist

5. **User Guide Gaps** (docs/user-guide/ if exists)
   - Find: Missing documentation for new features (app detail page, add resource)
   - Check: Troubleshooting section covers common errors
   - Verify: Screenshots/examples are current
   - Find: Broken internal links

REPORT:
- Documentation inaccuracies
- Missing documentation for features
- Outdated examples
- Confusing or unclear explanations
- Dead links or references
```

---

## Prompt 7: Error Handling & Edge Cases

```
Please analyze error handling and identify unhandled edge cases:

ANALYZE:
1. **Error Response Consistency** (across all handlers)
   - Check: Do all endpoints return consistent error formats?
   - Verify: HTTP status codes are appropriate (400 vs 404 vs 500)
   - Find: Errors that expose stack traces or sensitive info
   - Check: Error messages are actionable for users

2. **Nil Pointer Safety** (Go code analysis)
   - Find: Potential nil pointer dereferences
   - Check: Proper checks before accessing map/slice elements
   - Verify: Optional fields handled correctly
   - Find: Missing null checks in database queries

3. **Concurrent Access** (goroutines, shared state)
   - Find: Shared state without mutex protection
   - Check: Goroutine leaks (started but never stopped)
   - Verify: Channel operations won't deadlock
   - Find: Race conditions (use go race detector findings if available)

4. **Resource Cleanup** (defer, cleanup functions)
   - Check: Database connections properly closed
   - Verify: HTTP response bodies closed
   - Find: File handles not closed
   - Check: Temporary files/directories cleaned up

5. **Edge Case Scenarios**
   - Find: What happens if all providers are disabled?
   - Check: Empty database/no applications deployed
   - Verify: Very large Score specs (100+ resources)
   - Find: Handling of special characters in names
   - Check: Extremely long workflow executions
   - Verify: Network timeouts and retries

REPORT:
- Critical error handling gaps
- Nil pointer vulnerabilities
- Resource leaks
- Concurrency issues
- Unhandled edge cases
```

---

## Prompt 8: Demo-Critical Path Verification

```
Please verify the critical demo path works end-to-end by analyzing code:

TRACE THE HAPPY PATH:
1. **User Deploys via Web UI**
   - Start: User clicks "Deploy New" in /apps
   - Trace: DeployWizard component → generateScoreYaml() → api.deployApplication()
   - Verify: POST /api/applications with Content-Type: application/yaml
   - Follow: handleDeploySpec() → validation → db.AddApplication()
   - Check: Application node created in graph
   - End: Success response to UI

2. **Resources Auto-Provision**
   - Start: Resources created with state='requested'
   - Trace: Orchestration engine polling (internal/orchestration/engine.go)
   - Verify: Resolver maps resource type to provider
   - Follow: Workflow execution starts → steps execute → state updates
   - Check: Resource transitions to 'active'
   - End: Workflow completes successfully

3. **User Views Application**
   - Start: Click "View" on application
   - Trace: Navigation to /dev/applications/[name]
   - Verify: Page loads application details
   - Follow: api.getResources() → displays resource list
   - Check: All resources shown with correct state
   - End: User sees complete application status

4. **User Adds Resource**
   - Start: Click "Add Resource" button
   - Trace: Form submission → api.createResource()
   - Verify: POST /api/resources → handleCreateResource()
   - Follow: ResourceManager.CreateResourceInstance() → DB insert
   - Check: Orchestration engine picks up new resource
   - End: Resource appears in list, provisioning starts

IDENTIFY FAILURE POINTS:
- Where could the flow break?
- What assumptions are made?
- Missing error handling?
- Race conditions?
- Database constraints that could fail?

REPORT:
- Critical path vulnerabilities
- Assumptions that might not hold
- Missing validation or checks
- Points where system could hang
- Database transaction issues
```

---

## Summary Prompt: Overall Demo Readiness

```
Please provide an overall assessment of demo readiness:

SYNTHESIZE FINDINGS:
1. Review all previous analysis results
2. Categorize issues by severity:
   - CRITICAL (will break demo)
   - HIGH (might cause issues)
   - MEDIUM (UX problems)
   - LOW (polish items)

3. Identify TOP 5 RISKS for demo:
   - What's most likely to fail?
   - What would be most embarrassing?
   - What has least test coverage?

4. Recommend PRE-DEMO FIXES:
   - What MUST be fixed before demo?
   - What can be worked around?
   - What should be avoided/not shown?

5. Create DEMO SCRIPT:
   - Suggest safest demo path
   - Identify features to highlight
   - Recommend preparation steps
   - Suggest backup plans

DELIVERABLE:
Provide a concise "Demo Readiness Report" with:
- Executive summary (demo ready? yes/no/conditional)
- Critical issues to fix (prioritized list)
- Demo script (step-by-step safe path)
- Risk mitigation strategies
- Post-demo improvement backlog
```

---

## Usage Instructions

**Run prompts in this order:**

1. **Prompt 8** (Demo Critical Path) - Start here to verify the main flow
2. **Prompt 1** (Application Developer) - Core functionality
3. **Prompt 3** (Security) - Must be safe for demo
4. **Prompt 5** (Web UI) - Most visible during demo
5. **Prompt 7** (Error Handling) - Prevent embarrassing crashes
6. **Summary Prompt** - Get final readiness assessment

**Optional (if time permits):**
7. Prompt 2 (Provider System)
8. Prompt 4 (Database)
9. Prompt 6 (Documentation)

**Expected Time:** 30-60 minutes for critical prompts

**Outcome:** Clear list of must-fix issues before demo with prioritization.
