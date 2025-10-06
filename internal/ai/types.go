package ai

import "time"

// ChatRequest represents a chat message from the user
type ChatRequest struct {
	Message             string    `json:"message"`
	Context             string    `json:"context,omitempty"`              // Optional context (e.g., workflow ID, app name)
	ConversationHistory []Message `json:"conversation_history,omitempty"` // Previous messages in the conversation
	AuthToken           string    `json:"-"`                              // Not sent from client, populated by handler from Authorization header
}

// ChatResponse represents the AI's response
type ChatResponse struct {
	Message       string    `json:"message"`
	GeneratedSpec string    `json:"generated_spec,omitempty"` // YAML spec if generated
	Citations     []string  `json:"citations,omitempty"`      // Document sources used
	TokensUsed    int       `json:"tokens_used,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
}

// GenerateSpecRequest represents a request to generate a Score spec
type GenerateSpecRequest struct {
	Description string            `json:"description"`
	Metadata    map[string]string `json:"metadata,omitempty"` // Optional metadata (team, environment, etc.)
}

// GenerateSpecResponse contains the generated Score spec
type GenerateSpecResponse struct {
	Spec        string   `json:"spec"`        // YAML Score specification
	Explanation string   `json:"explanation"` // AI explanation of the spec
	Citations   []string `json:"citations"`   // Knowledge base sources
	TokensUsed  int      `json:"tokens_used"`
}

// StatusResponse indicates if AI is enabled and configured
type StatusResponse struct {
	Enabled         bool   `json:"enabled"`
	LLMProvider     string `json:"llm_provider"`
	EmbeddingModel  string `json:"embedding_model"`
	DocumentsLoaded int    `json:"documents_loaded"`
	Status          string `json:"status"` // "ready", "not_configured", "error"
	Message         string `json:"message,omitempty"`
}

// Message represents a chat message with role
type Message struct {
	Role      string    `json:"role"` // "user" or "assistant"
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Spec      string    `json:"spec,omitempty"`       // If message contains a generated spec
	ToolCalls []string  `json:"tool_calls,omitempty"` // Tool calls made by assistant
}

// ChatHistory stores conversation history
type ChatHistory struct {
	SessionID string    `json:"session_id"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
