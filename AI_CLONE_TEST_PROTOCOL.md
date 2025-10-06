# AI Feature Testing Protocol: test-graph-app Cloning

**Version:** 1.0
**Date:** 2025-10-06
**Purpose:** Evaluate the AI assistant's ability to clone/replicate the test-graph-app Score specification

---

## Overview

This protocol tests how well the innominatus AI assistant can:
1. **Understand** an existing Score specification (test-graph-app)
2. **Generate** a similar specification from natural language prompts
3. **Maintain accuracy** of key components (containers, resources, workflows)
4. **Provide helpful context** and explanations

---

## Prerequisites

### Environment Setup
```bash
# 1. Ensure AI service is configured
export OPENAI_API_KEY="your-openai-api-key"
export ANTHROPIC_API_KEY="your-anthropic-api-key"

# 2. Verify innominatus server is running
./innominatus
# Server should be healthy at http://localhost:8081

# 3. Verify CLI is built
go build -o innominatus-ctl cmd/cli/main.go
```

### Reference Specification
The test-graph-app specification (`score-test-graph.yaml`):
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: test-graph-app
containers:
  main:
    image: nginx:latest
    ports:
      - name: http
        port: 80
        protocol: TCP
resources:
  db:
    type: postgres
    params:
      size: small
environment:
  type: kubernetes
  ttl: 1h
workflow:
  steps:
    - name: validate-spec
      type: validation
      description: Validate the Score specification
```

**Key Components to Test:**
- Application name: `test-graph-app`
- Container: `nginx:latest` on port 80
- Resource: PostgreSQL database (`small` size)
- Environment: Kubernetes with 1h TTL
- Workflow: Single validation step

---

## Test Scenarios

### Scenario 1: Exact Clone Request

**Objective:** Test if AI can replicate the spec when given the original as context

**Test Steps:**
```bash
# 1. Start interactive chat
./innominatus-ctl chat

# 2. Send the following prompt:
"I have a Score specification for test-graph-app that uses nginx:latest,
connects to a small postgres database, runs on kubernetes with 1h TTL,
and has a validation workflow step. Please generate an identical Score spec."

# 3. Verify generated output matches original
```

**Success Criteria:**
- ✅ Application name is `test-graph-app`
- ✅ Container uses `nginx:latest` image
- ✅ Port 80 is configured for HTTP
- ✅ PostgreSQL resource with `size: small` parameter
- ✅ Environment type is `kubernetes`
- ✅ TTL is set to `1h`
- ✅ Workflow contains validation step
- ✅ YAML structure is valid Score v1b1 format

**Scoring:**
- **Perfect Match (100%):** All 8 criteria met
- **Good Match (80-99%):** 6-7 criteria met, minor differences
- **Partial Match (60-79%):** 4-5 criteria met
- **Poor Match (<60%):** 3 or fewer criteria met

---

### Scenario 2: Natural Language Clone

**Objective:** Test generation from natural language without technical details

**Test Steps:**
```bash
./innominatus-ctl chat

# Prompt:
"Create a Score spec for a simple web app called test-graph-app that uses
nginx, needs a database, runs on Kubernetes for 1 hour, and validates itself."
```

**Success Criteria:**
- ✅ Uses nginx container (any version acceptable)
- ✅ Includes database resource (postgres preferred)
- ✅ Kubernetes environment specified
- ✅ TTL around 1 hour (accepts: 1h, 60m, 3600s)
- ✅ Contains validation workflow
- ✅ Valid Score specification structure

**Scoring:**
- **Excellent (90-100%):** All criteria met with sensible defaults
- **Good (75-89%):** 5/6 criteria met
- ✅ Fair (60-74%):** 4/6 criteria met
- **Poor (<60%):** 3 or fewer criteria met

---

### Scenario 3: Component-by-Component Build

**Objective:** Test iterative spec building through conversation

**Test Steps:**
```bash
./innominatus-ctl chat

# Prompt 1: Basic structure
"Generate a Score spec for test-graph-app"

# Prompt 2: Add container
"Add an nginx container on port 80"

# Prompt 3: Add database
"Add a small PostgreSQL database resource"

# Prompt 4: Configure environment
"Make it run on Kubernetes with 1 hour TTL"

# Prompt 5: Add workflow
"Add a validation workflow step"
```

**Success Criteria:**
- ✅ AI maintains conversation context across prompts
- ✅ Each addition correctly modifies previous spec
- ✅ Final spec matches test-graph-app components
- ✅ No information loss between iterations
- ✅ Valid YAML structure maintained throughout

**Scoring:**
- **Perfect (100%):** All modifications applied correctly, full context retention
- **Good (80-99%):** Minor issues with 1 prompt, context mostly retained
- **Fair (60-79%):** 2 prompts had issues, some context loss
- **Poor (<60%):** Multiple failures or context loss

---

### Scenario 4: Clone with Variations

**Objective:** Test AI's understanding by requesting variations

**Test Steps:**
```bash
./innominatus-ctl chat

# Test 4a: Different container
"Generate a spec like test-graph-app but use Apache instead of nginx"

# Test 4b: Different database size
"Generate a spec like test-graph-app but use a medium-sized database"

# Test 4c: Different TTL
"Generate a spec like test-graph-app but with 4 hour TTL"

# Test 4d: Additional workflow steps
"Generate a spec like test-graph-app but add provisioning and deployment steps"
```

**Success Criteria:**
- ✅ Maintains base structure while applying changes
- ✅ Requested modifications are correct
- ✅ Unchanged components remain accurate
- ✅ Explanations clarify what changed

**Scoring:**
- **Excellent (90-100%):** All variations correct, clear explanations
- **Good (75-89%):** 3/4 variations correct
- **Fair (60-74%):** 2/4 variations correct
- **Poor (<60%):** 1 or fewer variations correct

---

### Scenario 5: Error Detection and Fixing

**Objective:** Test AI's validation and correction capabilities

**Test Steps:**
```bash
./innominatus-ctl chat

# Test 5a: Generate invalid spec
"Generate a spec like test-graph-app but with an invalid resource type 'mysql-cluster'"

# Test 5b: Ask AI to validate
"Can you validate this spec and suggest corrections?"

# Test 5c: Request fix
"Fix the issues you found"
```

**Success Criteria:**
- ✅ AI identifies the invalid resource type
- ✅ Suggests correct alternatives (postgres, redis, etc.)
- ✅ Corrected spec is valid
- ✅ Provides explanation of the fix

**Scoring:**
- **Perfect (100%):** Identifies all issues, provides correct fixes
- **Good (80-99%):** Identifies most issues, fixes work
- **Fair (60-79%):** Partial identification, some fixes work
- **Poor (<60%):** Fails to identify or fix issues

---

## Test Execution Workflow

### Phase 1: Preparation (5 minutes)
1. ✅ Verify environment variables set
2. ✅ Start innominatus server
3. ✅ Confirm server health: `curl http://localhost:8081/health`
4. ✅ Review reference spec: `cat score-test-graph.yaml`
5. ✅ Build CLI: `go build -o innominatus-ctl cmd/cli/main.go`

### Phase 2: Execute Scenarios (30 minutes)
1. **Scenario 1:** Exact clone (5 min)
2. **Scenario 2:** Natural language (5 min)
3. **Scenario 3:** Iterative building (10 min)
4. **Scenario 4:** Variations (5 min)
5. **Scenario 5:** Error detection (5 min)

### Phase 3: Validation (10 minutes)
For each generated spec:
```bash
# Save the generated spec
/save generated-spec-scenario-X.yaml

# Validate with CLI
./innominatus-ctl validate generated-spec-scenario-X.yaml --explain

# Compare with original
diff score-test-graph.yaml generated-spec-scenario-X.yaml

# Test deployment (optional)
./innominatus-ctl run deploy-app generated-spec-scenario-X.yaml
```

### Phase 4: Documentation (15 minutes)
1. Record test results in table format
2. Note any unexpected behaviors
3. Capture example conversations
4. Calculate overall scores

---

## Results Template

### Test Execution Summary

**Date:** YYYY-MM-DD
**Tester:** [Name]
**AI Model:** [OpenAI/Anthropic version]
**Test Duration:** [minutes]

| Scenario | Score | Status | Notes |
|----------|-------|--------|-------|
| 1. Exact Clone | X% | ✅/⚠️/❌ | |
| 2. Natural Language | X% | ✅/⚠️/❌ | |
| 3. Iterative Build | X% | ✅/⚠️/❌ | |
| 4. Variations | X% | ✅/⚠️/❌ | |
| 5. Error Detection | X% | ✅/⚠️/❌ | |
| **Overall** | **X%** | **✅/⚠️/❌** | |

**Legend:**
- ✅ Pass (≥80%)
- ⚠️ Partial (60-79%)
- ❌ Fail (<60%)

### Detailed Findings

#### Scenario 1: Exact Clone
```yaml
# Paste generated spec here
```
**Analysis:**
- Component accuracy: [details]
- Structure correctness: [details]
- Issues found: [list]

#### Scenario 2: Natural Language
```yaml
# Paste generated spec here
```
**Analysis:**
- Interpretation quality: [details]
- Default choices: [details]
- Issues found: [list]

#### Scenario 3: Iterative Build
**Conversation Log:**
```
User: [prompt 1]
AI: [response 1]

User: [prompt 2]
AI: [response 2]
...
```
**Analysis:**
- Context retention: [details]
- Modification accuracy: [details]
- Issues found: [list]

#### Scenario 4: Variations
**Test 4a (Apache):**
- Result: [Pass/Fail]
- Issues: [list]

**Test 4b (Medium DB):**
- Result: [Pass/Fail]
- Issues: [list]

**Test 4c (4h TTL):**
- Result: [Pass/Fail]
- Issues: [list]

**Test 4d (Extra Steps):**
- Result: [Pass/Fail]
- Issues: [list]

#### Scenario 5: Error Detection
**Validation Accuracy:** [details]
**Fix Quality:** [details]
**Explanation Clarity:** [details]

---

## Quality Metrics

### Accuracy Metrics
- **Component Completeness:** % of required components present
- **Configuration Accuracy:** % of parameters matching expected values
- **Structure Validity:** Pass/Fail for YAML and Score schema compliance
- **Semantic Correctness:** % of components that make sense together

### User Experience Metrics
- **Response Time:** Average time to generate spec
- **Explanation Quality:** Clarity and helpfulness (1-5 scale)
- **Conversation Flow:** Naturalness of interaction (1-5 scale)
- **Error Messages:** Helpfulness when issues occur (1-5 scale)

### Reliability Metrics
- **First-Try Success Rate:** % scenarios passing without retries
- **Context Retention:** % of conversation history maintained correctly
- **Consistency:** Variation in results across multiple runs
- **Error Recovery:** % of errors successfully corrected when prompted

---

## Common Issues and Troubleshooting

### AI Service Not Enabled
```
Error: AI service is not enabled
```
**Fix:**
```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

### Invalid Generated Spec
**Symptoms:** Validation fails with schema errors

**Actions:**
1. Check Score API version (should be `score.dev/v1b1`)
2. Verify required fields (metadata.name, containers)
3. Validate resource types against supported list
4. Check YAML syntax

### Context Loss in Chat
**Symptoms:** AI forgets previous conversation

**Actions:**
1. Verify conversation history is tracked (check `cmd/cli/chat.go:58`)
2. Use `/clear` to reset if needed
3. Re-provide context in new prompt

### Spec Differences from Original
**Not necessarily a failure** - check:
- Are differences semantic (e.g., `1h` vs `60m`)?
- Are defaults reasonable?
- Does spec still achieve same goal?

---

## Acceptance Criteria

### Minimum Viable Performance
- **Overall Score:** ≥70%
- **Critical Components:** 100% accuracy (name, container, resource type)
- **No Breaking Errors:** All generated specs must be valid YAML and Score format

### Production-Ready Performance
- **Overall Score:** ≥85%
- **Exact Clone:** ≥90%
- **Natural Language:** ≥80%
- **Context Retention:** ≥90%
- **Error Detection:** ≥85%

### Exceptional Performance
- **Overall Score:** ≥95%
- **All Scenarios:** ≥90%
- **First-Try Success:** ≥80%
- **User Experience:** ≥4.5/5 average

---

## Test Data Storage

### Directory Structure
```
/Users/philipsahli/projects/innominatus/
├── score-test-graph.yaml                    # Original reference spec
├── AI_CLONE_TEST_PROTOCOL.md                # This protocol
└── test-results/
    └── ai-clone-test-YYYY-MM-DD/
        ├── scenario-1-exact-clone.yaml
        ├── scenario-2-natural-language.yaml
        ├── scenario-3-iterative.yaml
        ├── scenario-4a-apache-variation.yaml
        ├── scenario-4b-medium-db-variation.yaml
        ├── scenario-4c-4h-ttl-variation.yaml
        ├── scenario-4d-extra-steps-variation.yaml
        ├── scenario-5-error-detection.yaml
        ├── conversation-logs.txt
        └── test-results-summary.md
```

### Result File Naming
- Format: `scenario-N-description-YYYYMMDD.yaml`
- Example: `scenario-1-exact-clone-20251006.yaml`

---

## Continuous Testing

### Regression Testing
Run this protocol:
- **After AI service changes** (code modifications to `internal/ai/`)
- **After prompt template updates**
- **After Score schema updates**
- **Monthly** for baseline performance tracking

### Performance Tracking
Maintain a log:
```
Date       | Overall Score | Scenarios Passed | Notes
-----------|---------------|------------------|-------
2025-10-06 | 87%          | 4/5              | Initial test
2025-10-15 | 92%          | 5/5              | Prompt improvements
...
```

---

## Appendix

### A. Quick Reference Commands

```bash
# Start interactive chat
./innominatus-ctl chat

# Generate spec directly
./innominatus-ctl generate-spec "description" output.yaml

# Validate generated spec
./innominatus-ctl validate output.yaml --explain

# Compare specs
diff score-test-graph.yaml output.yaml

# Deploy generated spec
./innominatus-ctl run deploy-app output.yaml
```

### B. Expected AI Capabilities

The AI assistant should be able to:
- ✅ Generate valid Score specifications
- ✅ Understand Score schema and constraints
- ✅ Provide helpful explanations
- ✅ Maintain conversation context
- ✅ Suggest improvements and best practices
- ✅ Validate and fix specifications
- ✅ Handle variations and customizations

### C. Reference Documentation

- **Score Specification:** https://score.dev/docs/
- **innominatus AI Service:** `internal/ai/service.go`
- **CLI Chat Interface:** `cmd/cli/chat.go`
- **Golden Paths:** `goldenpaths.yaml`
- **Workflow Templates:** `workflows/`

---

## Version History

| Version | Date | Changes | Author |
|---------|------|---------|--------|
| 1.0 | 2025-10-06 | Initial protocol creation | Claude Code |

---

**Next Steps:**
1. Execute this protocol to establish baseline performance
2. Document results in `test-results/` directory
3. Identify improvement areas
4. Schedule regression testing
5. Update protocol based on findings

---

*Protocol designed for comprehensive AI feature validation*
*Focus: test-graph-app cloning accuracy and user experience*
