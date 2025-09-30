package validation

import (
	"fmt"
	"innominatus/internal/errors"
	"strings"
)

// ExplanationFormatter formats validation results with detailed explanations
type ExplanationFormatter struct {
	errors   []*errors.RichError
	warnings []*errors.RichError
	info     []*errors.RichError
}

// NewExplanationFormatter creates a new explanation formatter
func NewExplanationFormatter(validationErrors []*errors.RichError) *ExplanationFormatter {
	ef := &ExplanationFormatter{}

	for _, err := range validationErrors {
		switch err.Severity {
		case errors.SeverityError, errors.SeverityFatal:
			ef.errors = append(ef.errors, err)
		case errors.SeverityWarning:
			ef.warnings = append(ef.warnings, err)
		case errors.SeverityInfo:
			ef.info = append(ef.info, err)
		}
	}

	return ef
}

// Format formats the validation results with detailed explanations
func (ef *ExplanationFormatter) Format() string {
	var b strings.Builder

	// Header
	b.WriteString("\n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n")
	b.WriteString("  Score Specification Validation Report\n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n\n")

	// Summary
	total := len(ef.errors) + len(ef.warnings) + len(ef.info)
	if total == 0 {
		b.WriteString("✅ No validation issues found!\n")
		b.WriteString("\nYour Score specification is valid and ready to deploy.\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Found %d issue(s):\n", total))
	if len(ef.errors) > 0 {
		b.WriteString(fmt.Sprintf("  ❌ %d error(s) (must be fixed)\n", len(ef.errors)))
	}
	if len(ef.warnings) > 0 {
		b.WriteString(fmt.Sprintf("  ⚠️  %d warning(s) (should be addressed)\n", len(ef.warnings)))
	}
	if len(ef.info) > 0 {
		b.WriteString(fmt.Sprintf("  ℹ️  %d info message(s)\n", len(ef.info)))
	}
	b.WriteString("\n")

	// Errors
	if len(ef.errors) > 0 {
		b.WriteString("═══ ERRORS (Must Fix) ═══\n\n")
		for i, err := range ef.errors {
			b.WriteString(ef.formatError(i+1, err))
			b.WriteString("\n")
		}
	}

	// Warnings
	if len(ef.warnings) > 0 {
		b.WriteString("═══ WARNINGS (Should Address) ═══\n\n")
		for i, err := range ef.warnings {
			b.WriteString(ef.formatError(i+1, err))
			b.WriteString("\n")
		}
	}

	// Info
	if len(ef.info) > 0 {
		b.WriteString("═══ INFORMATION ═══\n\n")
		for i, err := range ef.info {
			b.WriteString(ef.formatError(i+1, err))
			b.WriteString("\n")
		}
	}

	// Footer with next steps
	b.WriteString("═══════════════════════════════════════════════════════════════\n")
	b.WriteString("\n📋 Next Steps:\n\n")

	if len(ef.errors) > 0 {
		b.WriteString("1. Fix all errors listed above\n")
		b.WriteString("2. Run validation again: innominatus-ctl validate <file> --explain\n")
		b.WriteString("3. Once errors are fixed, address warnings for best practices\n")
	} else if len(ef.warnings) > 0 {
		b.WriteString("1. Review warnings and consider addressing them\n")
		b.WriteString("2. Your spec is valid but could be improved\n")
		b.WriteString("3. Deploy when ready: innominatus-ctl deploy <file>\n")
	}

	b.WriteString("\n💡 Need Help?\n")
	b.WriteString("   • Score Specification: https://score.dev\n")
	b.WriteString("   • innominatus Docs: https://docs.innominatus.dev\n")
	b.WriteString("   • Example Specs: innominatus-ctl examples\n\n")

	return b.String()
}

// formatError formats a single validation error with details
func (ef *ExplanationFormatter) formatError(num int, err *errors.RichError) string {
	var b strings.Builder

	// Error number and icon
	icon := err.Severity.Icon()
	b.WriteString(fmt.Sprintf("%s Issue #%d: %s\n", icon, num, err.Message))

	// Location
	if err.Location != nil {
		b.WriteString(fmt.Sprintf("\n   📍 Location: %s:%d:%d\n", err.Location.File, err.Location.Line, err.Location.Column))

		if err.Location.Source != "" {
			// Show the source line with context
			b.WriteString("\n   │\n")
			b.WriteString(fmt.Sprintf("   │ %4d │ %s\n", err.Location.Line, err.Location.Source))

			// Show error indicator
			if err.Location.Column > 0 {
				padding := strings.Repeat(" ", err.Location.Column+10)
				b.WriteString(fmt.Sprintf("   │      │ %s^\n", padding))
				b.WriteString(fmt.Sprintf("   │      │ %s└─── Issue here\n", padding))
			}
			b.WriteString("   │\n")
		}
	}

	// Context information
	if len(err.Context) > 0 {
		b.WriteString("\n   📋 Context:\n")
		for key, value := range err.Context {
			if key != "trace_id" && key != "request_id" { // Skip internal IDs
				b.WriteString(fmt.Sprintf("      • %s: %v\n", key, value))
			}
		}
	}

	// Suggestions
	if len(err.Suggestions) > 0 {
		b.WriteString("\n   💡 How to Fix:\n")
		for i, suggestion := range err.Suggestions {
			// Wrap long suggestions
			wrapped := wrapText(suggestion, 60)
			lines := strings.Split(wrapped, "\n")
			for j, line := range lines {
				if j == 0 {
					b.WriteString(fmt.Sprintf("      %d. %s\n", i+1, line))
				} else {
					b.WriteString(fmt.Sprintf("         %s\n", line))
				}
			}
		}
	}

	// Retriable indicator
	if err.Retriable {
		b.WriteString("\n   🔄 This may be a transient error. Try running the validation again.\n")
	}

	return b.String()
}

// FormatSimple provides a simple, non-detailed output
func (ef *ExplanationFormatter) FormatSimple() string {
	var b strings.Builder

	total := len(ef.errors) + len(ef.warnings)
	if total == 0 {
		b.WriteString("✅ Validation passed\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Validation found %d issue(s):\n", total))

	for _, err := range ef.errors {
		location := ""
		if err.Location != nil {
			location = fmt.Sprintf(" (%s:%d)", err.Location.File, err.Location.Line)
		}
		b.WriteString(fmt.Sprintf("  ❌ %s%s\n", err.Message, location))
	}

	for _, err := range ef.warnings {
		location := ""
		if err.Location != nil {
			location = fmt.Sprintf(" (%s:%d)", err.Location.File, err.Location.Line)
		}
		b.WriteString(fmt.Sprintf("  ⚠️  %s%s\n", err.Message, location))
	}

	return b.String()
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	if len(text) <= width {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLength := 0

	for i, word := range words {
		wordLen := len(word)

		if lineLength+wordLen+1 > width {
			result.WriteString("\n")
			result.WriteString(word)
			lineLength = wordLen
		} else {
			if i > 0 {
				result.WriteString(" ")
				lineLength++
			}
			result.WriteString(word)
			lineLength += wordLen
		}
	}

	return result.String()
}

// ExportJSON exports validation results as JSON
func (ef *ExplanationFormatter) ExportJSON() string {
	// Simple JSON representation
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString(fmt.Sprintf("  \"total_issues\": %d,\n", len(ef.errors)+len(ef.warnings)+len(ef.info)))
	b.WriteString(fmt.Sprintf("  \"errors\": %d,\n", len(ef.errors)))
	b.WriteString(fmt.Sprintf("  \"warnings\": %d,\n", len(ef.warnings)))
	b.WriteString(fmt.Sprintf("  \"info\": %d,\n", len(ef.info)))
	b.WriteString("  \"issues\": [\n")

	allIssues := append(append(ef.errors, ef.warnings...), ef.info...)
	for i, err := range allIssues {
		b.WriteString("    {\n")
		b.WriteString(fmt.Sprintf("      \"severity\": \"%s\",\n", err.Severity))
		b.WriteString(fmt.Sprintf("      \"message\": \"%s\",\n", strings.ReplaceAll(err.Message, "\"", "\\\"")))

		if err.Location != nil {
			b.WriteString("      \"location\": {\n")
			b.WriteString(fmt.Sprintf("        \"file\": \"%s\",\n", err.Location.File))
			b.WriteString(fmt.Sprintf("        \"line\": %d,\n", err.Location.Line))
			b.WriteString(fmt.Sprintf("        \"column\": %d\n", err.Location.Column))
			b.WriteString("      },\n")
		}

		if len(err.Suggestions) > 0 {
			b.WriteString("      \"suggestions\": [\n")
			for j, suggestion := range err.Suggestions {
				comma := ","
				if j == len(err.Suggestions)-1 {
					comma = ""
				}
				b.WriteString(fmt.Sprintf("        \"%s\"%s\n", strings.ReplaceAll(suggestion, "\"", "\\\""), comma))
			}
			b.WriteString("      ]\n")
		}

		comma := ","
		if i == len(allIssues)-1 {
			comma = ""
		}
		b.WriteString(fmt.Sprintf("    }%s\n", comma))
	}

	b.WriteString("  ]\n")
	b.WriteString("}\n")

	return b.String()
}