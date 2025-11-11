# Archived Examples

This directory contains examples that demonstrate **outdated patterns** that are no longer supported in the current innominatus architecture.

## Why These Files Are Archived

These files are kept for historical reference but should **NOT be used** as templates for new implementations.

### Deprecated Pattern: Standalone Workflow Files

**Old Approach (Deprecated):**
```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: my-workflow
steps:
  - name: step1
    type: terraform
```

**Current Approach:**
Workflows are defined in **provider directories**, not as standalone files:
```
providers/
  database-team/
    provider.yaml
    workflows/
      provision-postgres.yaml
```

### Why This Changed

1. **Provider-Based Architecture**: Workflows are now owned by providers (teams), not centrally defined
2. **Automatic Resolution**: Resources in Score specs automatically trigger the correct provider workflow
3. **Team Autonomy**: Each team manages their own workflows in their provider directory
4. **CRUD Operations**: Workflows support CREATE, UPDATE, DELETE operations via provider capabilities

## Current Best Practices

### ✅ Correct: Score Specification

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app

containers:
  main:
    image: myapp:latest
    env:
      DATABASE_URL: ${resources.db.connection_string}

resources:
  db:
    type: postgres
    properties:  # Use 'properties', not 'params'
      version: "15"
      size: "medium"
      replicas: 2
```

Deploy with:
```bash
./innominatus-ctl deploy score-spec.yaml -w
```

### Key Principles

1. **Resources declared in Score specs** - NOT as separate files
2. **Use `properties` field** - NOT `params`
3. **Don't hardcode credentials** - Use `${resources.name.attribute}` substitution
4. **Workflows in provider directories** - NOT standalone files
5. **Deploy command**: `./innominatus-ctl deploy score-spec.yaml -w`

## Migration Guide

If you have old Score specs or standalone workflows, here's how to migrate:

### From Standalone Workflow → Provider Workflow

1. Identify which team/provider owns this workflow
2. Move to `providers/<team-name>/workflows/`
3. Update `providers/<team-name>/provider.yaml` to register the workflow
4. Define capabilities (resource types this workflow handles)

### From Old Score Spec → Modern Score Spec

1. Change `params:` to `properties:`
2. Remove hardcoded credentials
3. Use `${resources.name.attribute}` for variable substitution
4. Remove embedded `workflows:` sections (if any)

## Archived Files in This Directory

### Standalone Workflow Files (Old Pattern)
- `parallel-workflow.yaml` - Example of parallel step execution
- `conditional-workflow.yaml` - Example of conditional logic
- `context-workflow.yaml` - Example of variable interpolation
- `resource-interpolation-workflow.yaml` - Example of resource references
- `resource-syntax-example.yaml` - Example of variable syntaxes

### Outdated Score Specs
- `score-with-s3.yaml` - Hardcoded S3 credentials (should use provider)
- `spec.yaml` - Embedded workflows in Score spec
- `score-spec-with-workflow.yaml` - Mixed Score + workflow orchestration
- `example-tfe-workflow.yaml` - Terraform Enterprise integration (not implemented)

## Need Help?

- See `examples/` (parent directory) for current, up-to-date examples
- Check `CLAUDE.md` for full documentation
- Review `providers/database-team/` for example provider structure
- Read `DEMO_PLAYBOOK.md` for detailed usage scenarios

---

**Last Updated:** 2025-11-10
**Reason for Archive:** Migration to provider-based architecture with Score specification pattern
