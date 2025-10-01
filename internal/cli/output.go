package cli

import (
	"fmt"
	"strings"
	"time"
)

// OutputFormatter provides standardized formatting for CLI output
type OutputFormatter struct {
	useEmojis bool
	useColors bool
}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{
		useEmojis: true,
		useColors: false, // Can be enabled when color support is added
	}
}

// Symbols for consistent output formatting
const (
	SymbolSuccess   = "✓"
	SymbolError     = "✗"
	SymbolWarning   = "⚠️"
	SymbolInfo      = "ℹ️"
	SymbolBullet    = "•"
	SymbolArrow     = "→"
	SymbolContainer = "🐳"
	SymbolResource  = "🔧"
	SymbolWorkflow  = "⚙️"
	SymbolApp       = "📦"
	SymbolEnv       = "🌍"
	SymbolLink      = "🔗"
	SymbolIdea      = "💡"
	SymbolSearch    = "🔍"
	SymbolRunning   = "⏳"
	SymbolComplete  = "✅"
)

// Separators for consistent formatting
const (
	SeparatorHeavy  = "═══════════════════════════════════════════════════════════════"
	SeparatorLight  = "───────────────────────────────────────────────────────────────"
	SeparatorMedium = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

// PrintHeader prints a standard header with separator
func (f *OutputFormatter) PrintHeader(title string) {
	fmt.Println(title)
	fmt.Println(SeparatorHeavy)
}

// PrintSubHeader prints a sub-header with lighter separator
func (f *OutputFormatter) PrintSubHeader(title string) {
	fmt.Println()
	fmt.Println(title)
	fmt.Println(SeparatorLight)
}

// PrintSuccess prints a success message
func (f *OutputFormatter) PrintSuccess(message string) {
	if f.useEmojis {
		fmt.Printf("%s %s\n", SymbolSuccess, message)
	} else {
		fmt.Printf("[SUCCESS] %s\n", message)
	}
}

// PrintError prints an error message
func (f *OutputFormatter) PrintError(message string) {
	if f.useEmojis {
		fmt.Printf("%s %s\n", SymbolError, message)
	} else {
		fmt.Printf("[ERROR] %s\n", message)
	}
}

// PrintWarning prints a warning message
func (f *OutputFormatter) PrintWarning(message string) {
	if f.useEmojis {
		fmt.Printf("%s %s\n", SymbolWarning, message)
	} else {
		fmt.Printf("[WARNING] %s\n", message)
	}
}

// PrintInfo prints an info message
func (f *OutputFormatter) PrintInfo(message string) {
	if f.useEmojis {
		fmt.Printf("%s %s\n", SymbolInfo, message)
	} else {
		fmt.Printf("[INFO] %s\n", message)
	}
}

// PrintItem prints a bulleted list item
func (f *OutputFormatter) PrintItem(indent int, icon string, message string) {
	indentStr := strings.Repeat("   ", indent)
	if f.useEmojis && icon != "" {
		fmt.Printf("%s%s %s\n", indentStr, icon, message)
	} else if icon != "" {
		fmt.Printf("%s%s %s\n", indentStr, SymbolBullet, message)
	} else {
		fmt.Printf("%s%s %s\n", indentStr, SymbolBullet, message)
	}
}

// PrintKeyValue prints a key-value pair with consistent formatting
func (f *OutputFormatter) PrintKeyValue(indent int, key string, value interface{}) {
	indentStr := strings.Repeat("   ", indent)
	fmt.Printf("%s%s: %v\n", indentStr, key, value)
}

// PrintDivider prints a divider line
func (f *OutputFormatter) PrintDivider(indent int) {
	indentStr := strings.Repeat("   ", indent)
	fmt.Printf("%s%s\n", indentStr, SeparatorLight)
}

// PrintEmpty prints an empty line
func (f *OutputFormatter) PrintEmpty() {
	fmt.Println()
}

// PrintSection prints a section with icon and title
func (f *OutputFormatter) PrintSection(indent int, icon string, title string) {
	indentStr := strings.Repeat("   ", indent)
	if f.useEmojis && icon != "" {
		fmt.Printf("%s%s %s\n", indentStr, icon, title)
	} else {
		fmt.Printf("%s%s\n", indentStr, title)
	}
}

// PrintEmptyState prints a message for when no items exist
func (f *OutputFormatter) PrintEmptyState(message string) {
	fmt.Println(message)
}

// PrintCount prints a count summary
func (f *OutputFormatter) PrintCount(item string, count int) {
	fmt.Printf("\nTotal: %d %s\n", count, item)
}

// FormatDuration formats a duration consistently
func (f *OutputFormatter) FormatDuration(duration time.Duration) string {
	if duration < time.Second {
		return fmt.Sprintf("%dms", duration.Milliseconds())
	}
	if duration < time.Minute {
		return fmt.Sprintf("%.1fs", duration.Seconds())
	}
	if duration < time.Hour {
		return fmt.Sprintf("%.1fm", duration.Minutes())
	}
	return fmt.Sprintf("%.1fh", duration.Hours())
}

// FormatTime formats a timestamp consistently
func (f *OutputFormatter) FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// PrintTable prints data in a simple table format
type TableColumn struct {
	Header string
	Width  int
}

func (f *OutputFormatter) PrintTableHeader(columns []TableColumn) {
	var headerParts []string
	for _, col := range columns {
		headerParts = append(headerParts, fmt.Sprintf("%-*s", col.Width, col.Header))
	}
	fmt.Println(strings.Join(headerParts, " "))

	var separatorParts []string
	for _, col := range columns {
		separatorParts = append(separatorParts, strings.Repeat("─", col.Width))
	}
	fmt.Println(strings.Join(separatorParts, " "))
}

func (f *OutputFormatter) PrintTableRow(columns []TableColumn, values []string) {
	var rowParts []string
	for i, col := range columns {
		if i < len(values) {
			rowParts = append(rowParts, fmt.Sprintf("%-*s", col.Width, values[i]))
		} else {
			rowParts = append(rowParts, fmt.Sprintf("%-*s", col.Width, ""))
		}
	}
	fmt.Println(strings.Join(rowParts, " "))
}

// PrintStatusBadge prints a status with appropriate emoji
func (f *OutputFormatter) PrintStatusBadge(status string) string {
	if !f.useEmojis {
		return status
	}

	switch strings.ToLower(status) {
	case "completed", "success", "healthy", "running":
		return fmt.Sprintf("%s %s", SymbolSuccess, status)
	case "failed", "error", "unhealthy":
		return fmt.Sprintf("%s %s", SymbolError, status)
	case "pending", "in_progress", "deploying":
		return fmt.Sprintf("%s %s", SymbolRunning, status)
	case "warning":
		return fmt.Sprintf("%s %s", SymbolWarning, status)
	default:
		return status
	}
}
