package llm

type Response struct {
	Text         string
	FinishReason string
	TokensUsed   int64
}

type Config struct {
	Model       string
	MaxTokens   int32
	Temperature float32
	System      string
}

func DefaultConfig() *Config {
	return &Config{
		Model:       "anthropic/claude-3.5-sonnet",
		MaxTokens:   8192,
		Temperature: 0.7,
		System:      "",
	}
}

// Message represents a chat message in the conversation
type Message struct {
	Role       string     // "user", "assistant", "system", "tool"
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string // For tool response messages
}

// ToolCall represents a function call made by the model
type ToolCall struct {
	ID        string
	Name      string
	Arguments map[string]any
}

// Tool represents a function that can be called by the model
type Tool struct {
	Name        string
	Description string
	Parameters  *ToolParameters
}

// ToolParameters defines the parameters schema for a tool
type ToolParameters struct {
	Type       string
	Properties map[string]*ToolProperty
	Required   []string
}

// ToolProperty defines a single parameter property
type ToolProperty struct {
	Type        string
	Description string
}
