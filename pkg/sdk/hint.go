package sdk

// Hint provides contextual information about a resource
// Hints are displayed in the UI as quick-access cards with links, commands, and connection strings
type Hint struct {
	// Type identifies the kind of hint
	// Valid types: "url", "dashboard", "command", "connection_string", "git_clone", "api_endpoint", "docs"
	Type string `json:"type" yaml:"type"`

	// Label is the display name shown in the UI
	// Example: "Repository URL", "Admin Dashboard", "Connection String"
	Label string `json:"label" yaml:"label"`

	// Value is the actual hint value (URL, command, string, etc.)
	// Example: "https://github.com/org/repo", "kubectl get pods -n namespace", "postgres://..."
	Value string `json:"value" yaml:"value"`

	// Icon is the optional icon identifier for UI display
	// Example: "git-branch", "dashboard", "terminal", "database", "lock", "external-link"
	Icon string `json:"icon,omitempty" yaml:"icon,omitempty"`
}

// HintType constants for common hint types
const (
	// HintTypeURL represents a clickable URL (opens in new tab)
	HintTypeURL = "url"

	// HintTypeDashboard represents a dashboard or admin console URL
	HintTypeDashboard = "dashboard"

	// HintTypeCommand represents a CLI command (copied to clipboard)
	HintTypeCommand = "command"

	// HintTypeConnectionString represents a connection string (copied to clipboard)
	HintTypeConnectionString = "connection_string"

	// HintTypeGitClone represents a git clone URL
	HintTypeGitClone = "git_clone"

	// HintTypeAPIEndpoint represents an API endpoint URL
	HintTypeAPIEndpoint = "api_endpoint"

	// HintTypeDocs represents a documentation link
	HintTypeDocs = "docs"
)

// IconType constants for common icons
const (
	IconGitBranch    = "git-branch"
	IconDownload     = "download"
	IconSettings     = "settings"
	IconTerminal     = "terminal"
	IconDatabase     = "database"
	IconLock         = "lock"
	IconExternalLink = "external-link"
	IconServer       = "server"
	IconGlobe        = "globe"
	IconKey          = "key"
	IconBook         = "book"
)

// NewURLHint creates a hint for a clickable URL
func NewURLHint(label, url, icon string) Hint {
	return Hint{
		Type:  HintTypeURL,
		Label: label,
		Value: url,
		Icon:  icon,
	}
}

// NewCommandHint creates a hint for a CLI command
func NewCommandHint(label, command, icon string) Hint {
	return Hint{
		Type:  HintTypeCommand,
		Label: label,
		Value: command,
		Icon:  icon,
	}
}

// NewConnectionStringHint creates a hint for a connection string
func NewConnectionStringHint(label, connectionString string) Hint {
	return Hint{
		Type:  HintTypeConnectionString,
		Label: label,
		Value: connectionString,
		Icon:  IconLock,
	}
}

// NewDashboardHint creates a hint for a dashboard URL
func NewDashboardHint(label, url string) Hint {
	return Hint{
		Type:  HintTypeDashboard,
		Label: label,
		Value: url,
		Icon:  IconExternalLink,
	}
}
