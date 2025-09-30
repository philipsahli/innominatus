# Orphaned Files Analysis

This document lists files that appear to be orphaned or no longer used in the current project structure.

## 🗑️ Definitely Orphaned Files

### Root Level Package Main Files (Likely Old/Unused)
These appear to be old monolithic files that have been superseded by the modular `internal/` structure:

- **`graph.go`** - Contains `package main` and `BuildGraph` function. Superseded by `internal/graph/graph.go`
- **`types.go`** - Contains `package main` and type definitions. Superseded by `internal/types/types.go`
- **`web.go`** - Contains `package main` and web functions. Superseded by `internal/server/web.go`
- **`workflow.go`** - Contains `package main` and workflow functions. Superseded by `internal/workflow/workflow.go`

### Backup/Temporary Files
- **`example-tfe-main.go___`** - Backup file with triple underscore suffix
- **`test-generate-main.go__`** - Backup file with double underscore suffix

### Generated/Compiled Binaries (Test Artifacts)
- **`test-build`** - Compiled binary (14MB)
- **`test-lifecycle`** - Compiled binary (14MB)
- **`test-server`** - Compiled binary (14MB)

### Temporary/Cache Files
- **`cookies.txt`** - Empty curl cookie file
- **`data/sessions.json`** - Runtime session data (should be in .gitignore)
- **`data/storage.json`** - Runtime storage data (should be in .gitignore)
- **`data/workflows.json`** - Runtime workflow data (should be in .gitignore)

## 🤔 Possibly Orphaned Files

### Example/Test YAML Files
These might be examples or test files that could be consolidated:

- **`example-tfe-workflow.yaml`** - TFE workflow example
- **`test-app.yaml`** - Test application spec
- **`test-terraform-generate.yaml`** - Test terraform generation spec
- **`test-unique-app.yaml`** - Test unique application spec

### Generated Web Assets (Next.js Build Output)
All files under `v2/.next/` and `v2/out/` are build artifacts and should be in .gitignore:

- **`v2/.next/`** - Next.js build cache (numerous files)
- **`v2/out/`** - Next.js static export output

### Terraform Cache/State Files
These should be in .gitignore:

- **All `.terraform/` directories**
- **`terraform/*/terraform.tfstate`** files
- **`terraform/*/*.txt` provision status files**

### Generated Provider Files
- **`terraform/*/.terraform/providers/`** - Terraform provider binaries
- **`workspaces/*/terraform/`** - Workspace terraform cache

## ✅ Files That Should Stay

### Documentation
- **`README.md`** - Project documentation
- **`CLAUDE.md`** - AI assistant instructions
- **`TFE-WORKFLOW-README.md`** - TFE workflow documentation
- **`v2/README.md`** - Next.js app documentation

### Configuration Files
- **`admin-config.yaml`** - Admin configuration
- **`goldenpaths.yaml`** - Golden paths configuration
- **`users.yaml`** - User configuration
- **`swagger.yaml`** - API documentation
- **`score-spec*.yaml`** - Score specification examples
- **`v2/package.json`**, **`v2/tsconfig.json`**, **`v2/components.json`** - Next.js config

### Workflow Definitions
- **`workflows/`** directory - All workflow YAML files

### Ansible Playbooks
- **`ansible/`** directory - All ansible playbooks

## 🔧 Recommended Actions

### Immediate Cleanup (Safe to Delete)
```bash
# Remove old package main files
rm graph.go types.go web.go workflow.go

# Remove backup files
rm example-tfe-main.go___ test-generate-main.go__

# Remove test binaries
rm test-build test-lifecycle test-server

# Remove temporary files
rm cookies.txt

# Remove runtime data (add to .gitignore)
rm -rf data/

# Remove Next.js build artifacts (add to .gitignore)
rm -rf v2/.next/ v2/out/

# Remove terraform cache (add to .gitignore)
find . -name ".terraform" -type d -exec rm -rf {} +
find . -name "terraform.tfstate*" -delete
find . -name "*_provisioned.txt" -delete
```

### Update .gitignore
Add these patterns to prevent future orphaned files:
```
# Runtime data
data/
cookies.txt

# Build artifacts
test-build
test-lifecycle
test-server
idp-o
innominatus-ctl
main

# Next.js
v2/.next/
v2/out/

# Terraform
.terraform/
*.tfstate*
*_provisioned.txt

# Backup files
*___
*__
```

### Review Test Files
These files should be reviewed to determine if they're still needed:
- `example-tfe-workflow.yaml`
- `test-app.yaml`
- `test-terraform-generate.yaml`
- `test-unique-app.yaml`

If they're examples, consider moving them to an `examples/` directory.

## 📊 Summary

- **Definitely Orphaned**: 11 files (~42MB with binaries)
- **Possibly Orphaned**: ~200+ build artifacts
- **Should Review**: 4 test/example YAML files

Cleaning up these files will reduce repository size significantly and improve project clarity.