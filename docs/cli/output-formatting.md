# CLI Output Formatting Standards

## Overview

The innominatus CLI uses standardized output formatting to ensure consistency across all commands. The formatting system is implemented in `internal/cli/output.go` and provides a unified interface for all CLI output.

## OutputFormatter

All CLI commands should use the `OutputFormatter` to display information to users.

### Usage

```go
formatter := NewOutputFormatter()
formatter.PrintSuccess("Operation completed successfully")
formatter.PrintError("Operation failed")
formatter.PrintWarning("This is a warning message")
formatter.PrintInfo("Informational message")
```

## Standard Symbols

The formatter uses consistent symbols/emojis across commands:

| Symbol | Constant | Usage |
|--------|----------|-------|
| ✓ | SymbolSuccess | Success messages |
| ✗ | SymbolError | Error messages |
| ⚠️ | SymbolWarning | Warning messages |
| ℹ️ | SymbolInfo | Informational messages |
| • | SymbolBullet | List items |
| → | SymbolArrow | Direction/flow indicators |
| 🐳 | SymbolContainer | Container-related info |
| 🔧 | SymbolResource | Resource-related info |
| ⚙️ | SymbolWorkflow | Workflow-related info |
| 📦 | SymbolApp | Application-related info |
| 🌍 | SymbolEnv | Environment-related info |
| 🔗 | SymbolLink | Dependencies/links |
| 💡 | SymbolIdea | Recommendations |
| 🔍 | SymbolSearch | Search/discovery operations |
| ⏳ | SymbolRunning | In-progress operations |
| ✅ | SymbolComplete | Completed operations |

## Standard Separators

Three types of separators are available:

```go
SeparatorHeavy  = "═══════════════════════════════════════..."  // Main headers
SeparatorLight  = "───────────────────────────────────────..."  // Sub-sections
SeparatorMedium = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━..."  // Medium emphasis
```

## Formatting Methods

### Headers

```go
formatter.PrintHeader("Main Title")
// Output:
// Main Title
// ═══════════════════════════════════════

formatter.PrintSubHeader("Subsection")
// Output:
//
// Subsection
// ───────────────────────────────────────
```

### Messages

```go
formatter.PrintSuccess("Operation completed")
// Output: ✓ Operation completed

formatter.PrintError("Operation failed")
// Output: ✗ Operation failed

formatter.PrintWarning("Potential issue detected")
// Output: ⚠️ Potential issue detected

formatter.PrintInfo("Additional information")
// Output: ℹ️ Additional information
```

### Structured Output

```go
// Section with icon
formatter.PrintSection(indent, SymbolApp, "Application Details")
// Output: 📦 Application Details

// Key-value pairs
formatter.PrintKeyValue(indent, "Status", "Running")
// Output:    Status: Running

// List items
formatter.PrintItem(indent, SymbolBullet, "Item description")
// Output:    • Item description

// Dividers
formatter.PrintDivider(indent)
// Output:    ───────────────────────────────────────
```

### Empty States

```go
formatter.PrintEmptyState("No items found")
// Output: No items found
```

### Counts

```go
formatter.PrintCount("applications", 5)
// Output:
// Total: 5 applications
```

## Indentation

All formatting methods that support indentation use a consistent pattern:
- Indent level 0: No indentation
- Indent level 1: 3 spaces ("   ")
- Indent level 2: 6 spaces ("      ")
- Indent level 3: 9 spaces ("         ")

## Time and Duration Formatting

```go
// Format timestamps
formatter.FormatTime(time.Now())
// Output: 2025-01-15T10:30:00Z

// Format durations
formatter.FormatDuration(2 * time.Minute)
// Output: 2.0m
```

## Status Badges

```go
formatter.PrintStatusBadge("completed")
// Output: ✓ completed

formatter.PrintStatusBadge("failed")
// Output: ✗ failed

formatter.PrintStatusBadge("pending")
// Output: ⏳ pending
```

## Table Output

```go
columns := []TableColumn{
    {Header: "Name", Width: 20},
    {Header: "Status", Width: 10},
    {Header: "Created", Width: 25},
}

formatter.PrintTableHeader(columns)
formatter.PrintTableRow(columns, []string{"app-1", "running", "2025-01-15"})
formatter.PrintTableRow(columns, []string{"app-2", "stopped", "2025-01-14"})

// Output:
// Name                 Status     Created
// ──────────────────── ────────── ─────────────────────────
// app-1                running    2025-01-15
// app-2                stopped    2025-01-14
```

## Command Examples

### List Command
```bash
$ innominatus-ctl list
Deployed Applications (2):
═══════════════════════════════════════════════════════════════

📦 Application: my-app
   API Version: score.dev/v1b1
   🐳 Containers (1):
      • web: nginx:latest
   🔧 Resources (1):
      • database (postgres)
   ───────────────────────────────────────

📦 Application: another-app
   API Version: score.dev/v1b1
   🐳 Containers (1):
      • api: node:18
   🔧 Resources: None
   ───────────────────────────────────────

Total: 2 application(s) deployed
```

### Validate Command
```bash
$ innominatus-ctl validate score-spec.yaml
✓ Score spec is valid
   Application: my-app
   API Version: score.dev/v1b1
   Containers: 1
   Resources: 1
   Environment: kubernetes (TTL: 1h)

Dependencies detected:
───────────────────────────────────────────────────────────────
   • web → database
```

### Golden Paths Command
```bash
$ innominatus-ctl list-goldenpaths
Available Golden Paths:
═══════════════════════════════════════════════════════════════
   ⚙️ deploy-app → ./workflows/deploy-app.yaml
   ⚙️ ephemeral-env → ./workflows/ephemeral-env.yaml
   ⚙️ db-lifecycle → ./workflows/db-lifecycle.yaml
```

## Best Practices

1. **Always use the OutputFormatter**: Never use raw `fmt.Println()` or `fmt.Printf()` in commands
2. **Consistent indentation**: Use indent levels consistently (0 for top-level, 1-2 for nested content)
3. **Appropriate symbols**: Use the predefined symbols for their intended purposes
4. **Empty states**: Always handle empty states with `PrintEmptyState()`
5. **Success/Error messages**: Use `PrintSuccess()` and `PrintError()` for operation results
6. **Headers for structure**: Use `PrintHeader()` for main sections, `PrintSubHeader()` for nested sections
7. **Key-value for details**: Use `PrintKeyValue()` for structured data
8. **Dividers for separation**: Use `PrintDivider()` to separate related groups of items

## Migration Guide

When updating existing commands:

1. Create formatter at the beginning:
   ```go
   formatter := NewOutputFormatter()
   ```

2. Replace success messages:
   ```go
   // Before:
   fmt.Printf("✓ Operation completed\n")

   // After:
   formatter.PrintSuccess("Operation completed")
   ```

3. Replace headers:
   ```go
   // Before:
   fmt.Println("Active Applications:")
   fmt.Println("═══════════════════")

   // After:
   formatter.PrintHeader("Active Applications:")
   ```

4. Replace key-value pairs:
   ```go
   // Before:
   fmt.Printf("   Status: %s\n", status)

   // After:
   formatter.PrintKeyValue(1, "Status", status)
   ```

5. Replace list items:
   ```go
   // Before:
   fmt.Printf("  • %s\n", item)

   // After:
   formatter.PrintItem(1, SymbolBullet, item)
   ```

## Future Enhancements

Planned improvements to the formatting system:

1. **Color support**: Add terminal color support via flags (e.g., `--color=auto|always|never`)
2. **JSON output**: Add `--output json` flag for machine-readable output
3. **Table output**: Add `--output table` flag for tabular data display
4. **Quiet mode**: Add `--quiet` flag to suppress non-essential output
5. **Verbose mode**: Add `--verbose` flag for detailed output
6. **Custom templates**: Support for Go template-based custom output formats

---

*Last updated: 2025-01-15*
