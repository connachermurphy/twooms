package llm

import (
	"context"
	"errors"
)

var (
	ErrMissingAPIKey = errors.New("OPENROUTER_API_KEY environment variable not set")
	ErrEmptyPrompt   = errors.New("prompt cannot be empty")
	ErrNoResponse    = errors.New("no response from model")
)

// ToolExecutor is called when the LLM wants to execute a tool.
// It receives the function name and arguments, and returns the result string.
type ToolExecutor func(name string, args map[string]any) string

type Client interface {
	Chat(ctx context.Context, prompt string) (*Response, error)
	ChatWithConfig(ctx context.Context, prompt string, config *Config) (*Response, error)
	ChatWithTools(ctx context.Context, message string, history []*Message, tools []*Tool, executor ToolExecutor) (*Response, []*Message, error)
	Close() error
}
