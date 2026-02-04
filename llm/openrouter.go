package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const openRouterURL = "https://openrouter.ai/api/v1/chat/completions"

type OpenRouterClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewOpenRouterClient(ctx context.Context) (*OpenRouterClient, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	return &OpenRouterClient{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}, nil
}

func (c *OpenRouterClient) Chat(ctx context.Context, prompt string) (*Response, error) {
	return c.ChatWithConfig(ctx, prompt, DefaultConfig())
}

func (c *OpenRouterClient) ChatWithConfig(ctx context.Context, prompt string, config *Config) (*Response, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, ErrEmptyPrompt
	}

	if config == nil {
		config = DefaultConfig()
	}

	messages := []openRouterMessage{
		{Role: "user", Content: prompt},
	}

	if config.System != "" {
		messages = append([]openRouterMessage{{Role: "system", Content: config.System}}, messages...)
	}

	return c.sendRequest(ctx, config, messages, nil)
}

func (c *OpenRouterClient) ChatWithTools(ctx context.Context, message string, history []*Message, tools []*Tool, executor ToolExecutor) (*Response, []*Message, error) {
	if strings.TrimSpace(message) == "" {
		return nil, history, ErrEmptyPrompt
	}

	config := DefaultConfig()

	// Check for model override
	if modelOverride := os.Getenv("OPENROUTER_MODEL"); modelOverride != "" {
		config.Model = modelOverride
	}

	// Convert tools to OpenRouter format
	orTools := convertToolsToOpenRouter(tools)

	// Build messages from history plus new message
	messages := []openRouterMessage{
		{Role: "system", Content: getToolSystemPrompt()},
	}

	// Add history
	for _, msg := range history {
		messages = append(messages, convertMessageToOpenRouter(msg))
	}

	// Add new user message
	messages = append(messages, openRouterMessage{Role: "user", Content: message})

	// Update history with new user message
	newHistory := append(history, &Message{Role: "user", Content: message})

	var totalTokens, totalInputTokens, totalOutputTokens int64
	var totalCost float64

	// Tool calling loop
	for {
		resp, err := c.sendRequestWithTools(ctx, config, messages, orTools)
		if err != nil {
			return nil, newHistory, err
		}

		totalTokens += resp.usage.TotalTokens
		totalInputTokens += resp.usage.PromptTokens
		totalOutputTokens += resp.usage.CompletionTokens
		totalCost += resp.usage.Cost

		if len(resp.choices) == 0 {
			return nil, newHistory, ErrNoResponse
		}

		choice := resp.choices[0]

		// Check for tool calls
		if len(choice.Message.ToolCalls) > 0 {
			// Add assistant's message with tool calls to messages
			messages = append(messages, choice.Message)

			// Add to history
			assistantMsg := &Message{
				Role:      "assistant",
				Content:   choice.Message.Content,
				ToolCalls: make([]ToolCall, len(choice.Message.ToolCalls)),
			}
			for i, tc := range choice.Message.ToolCalls {
				var args map[string]any
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
				assistantMsg.ToolCalls[i] = ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: args,
				}
			}
			newHistory = append(newHistory, assistantMsg)

			// Execute each tool call and add responses
			for _, tc := range choice.Message.ToolCalls {
				var args map[string]any
				json.Unmarshal([]byte(tc.Function.Arguments), &args)

				result := executor(tc.Function.Name, args)

				// Add tool response to messages
				messages = append(messages, openRouterMessage{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})

				// Add to history
				newHistory = append(newHistory, &Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: tc.ID,
				})
			}

			continue
		}

		// No tool calls - return the text response
		assistantMsg := &Message{
			Role:    "assistant",
			Content: choice.Message.Content,
		}
		newHistory = append(newHistory, assistantMsg)

		return &Response{
			Text:         choice.Message.Content,
			FinishReason: choice.FinishReason,
			TokensUsed:   totalTokens,
			InputTokens:  totalInputTokens,
			OutputTokens: totalOutputTokens,
			Cost:         totalCost,
		}, newHistory, nil
	}
}

func (c *OpenRouterClient) Close() error {
	return nil
}

// Internal types for OpenRouter API

type openRouterMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content"`
	ToolCalls  []openRouterToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

type openRouterToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openRouterTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  any    `json:"parameters,omitempty"`
	} `json:"function"`
}

type openRouterRequest struct {
	Model       string              `json:"model"`
	Messages    []openRouterMessage `json:"messages"`
	MaxTokens   int32               `json:"max_tokens,omitempty"`
	Temperature float32             `json:"temperature,omitempty"`
	Tools       []openRouterTool    `json:"tools,omitempty"`
}

type openRouterResponse struct {
	choices []struct {
		Message      openRouterMessage `json:"message"`
		FinishReason string            `json:"finish_reason"`
	}
	usage struct {
		PromptTokens     int64   `json:"prompt_tokens"`
		CompletionTokens int64   `json:"completion_tokens"`
		TotalTokens      int64   `json:"total_tokens"`
		Cost             float64 `json:"cost"`
	}
}

func (c *OpenRouterClient) sendRequest(ctx context.Context, config *Config, messages []openRouterMessage, tools []openRouterTool) (*Response, error) {
	resp, err := c.sendRequestWithTools(ctx, config, messages, tools)
	if err != nil {
		return nil, err
	}

	if len(resp.choices) == 0 {
		return nil, ErrNoResponse
	}

	return &Response{
		Text:         resp.choices[0].Message.Content,
		FinishReason: resp.choices[0].FinishReason,
		TokensUsed:   resp.usage.TotalTokens,
		InputTokens:  resp.usage.PromptTokens,
		OutputTokens: resp.usage.CompletionTokens,
		Cost:         resp.usage.Cost,
	}, nil
}

func (c *OpenRouterClient) sendRequestWithTools(ctx context.Context, config *Config, messages []openRouterMessage, tools []openRouterTool) (*openRouterResponse, error) {
	reqBody := openRouterRequest{
		Model:       config.Model,
		Messages:    messages,
		MaxTokens:   config.MaxTokens,
		Temperature: config.Temperature,
	}

	if len(tools) > 0 {
		reqBody.Tools = tools
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", openRouterURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/connachermurphy/twooms")
	req.Header.Set("X-Title", "Twooms")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message      openRouterMessage `json:"message"`
			FinishReason string            `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int64   `json:"prompt_tokens"`
			CompletionTokens int64   `json:"completion_tokens"`
			TotalTokens      int64   `json:"total_tokens"`
			Cost             float64 `json:"cost"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &openRouterResponse{
		choices: result.Choices,
		usage:   result.Usage,
	}, nil
}

func convertToolsToOpenRouter(tools []*Tool) []openRouterTool {
	var result []openRouterTool

	for _, t := range tools {
		orTool := openRouterTool{
			Type: "function",
		}
		orTool.Function.Name = t.Name
		orTool.Function.Description = t.Description

		if t.Parameters != nil {
			params := map[string]any{
				"type": t.Parameters.Type,
			}
			if len(t.Parameters.Properties) > 0 {
				props := make(map[string]any)
				for name, prop := range t.Parameters.Properties {
					props[name] = map[string]string{
						"type":        prop.Type,
						"description": prop.Description,
					}
				}
				params["properties"] = props
			}
			if len(t.Parameters.Required) > 0 {
				params["required"] = t.Parameters.Required
			}
			orTool.Function.Parameters = params
		}

		result = append(result, orTool)
	}

	return result
}

func convertMessageToOpenRouter(msg *Message) openRouterMessage {
	orMsg := openRouterMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCallID: msg.ToolCallID,
	}

	if len(msg.ToolCalls) > 0 {
		orMsg.ToolCalls = make([]openRouterToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			args, _ := json.Marshal(tc.Arguments)
			orMsg.ToolCalls[i] = openRouterToolCall{
				ID:   tc.ID,
				Type: "function",
			}
			orMsg.ToolCalls[i].Function.Name = tc.Name
			orMsg.ToolCalls[i].Function.Arguments = string(args)
		}
	}

	return orMsg
}

func getToolSystemPrompt() string {
	today := time.Now().Format("2006-01-02")
	weekday := time.Now().Weekday().String()

	return fmt.Sprintf(`You are a helpful task management assistant for Twooms.

TODAY'S DATE: %s (%s)

IMPORTANT RULES:
1. When a user refers to a project by NAME (not ID), FIRST call "projects" to find the ID, then use that ID.
2. When a user refers to a task by NAME, FIRST call the listing tool to find the task's ID.
3. NEVER ask the user for an ID. Always look it up using available tools.
4. Project IDs look like "proj-1". Task IDs look like "task-1".
5. When setting due dates, use the current date above. "Today" means %s, "tomorrow" means the next day, etc.

EXAMPLES:
- "list tasks in Office" -> call projects, find Office's ID, call tasks with that ID
- "mark documentation task done" -> list projects/tasks to find IDs, then call done
- "due today" -> use date %s`, today, weekday, today, today)
}
