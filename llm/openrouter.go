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
	debug      bool
}

func NewOpenRouterClient(ctx context.Context) (*OpenRouterClient, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	return &OpenRouterClient{
		apiKey:     apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
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
	var messages []openRouterMessage

	// Add history (which should include a system prompt from the caller)
	for _, msg := range history {
		messages = append(messages, convertMessageToOpenRouter(msg))
	}

	// Add new user message
	messages = append(messages, openRouterMessage{Role: "user", Content: message})

	// Update history with new user message
	newHistory := append(history, &Message{Role: "user", Content: message})

	if c.debug {
		fmt.Printf("[DEBUG] Request: %d messages, %d tools\n", len(messages), len(orTools))
	}

	var totalTokens, totalInputTokens, totalOutputTokens int64
	var totalCost float64
	var accumulatedContent strings.Builder
	var toolResults []string // Track tool results for fallback response

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

		if c.debug {
			fmt.Printf("[DEBUG] Response: finish_reason=%s, tool_calls=%d\n", choice.FinishReason, len(choice.Message.ToolCalls))
		}

		// Accumulate any content from this response
		if choice.Message.Content != "" {
			if accumulatedContent.Len() > 0 {
				accumulatedContent.WriteString(" ")
			}
			accumulatedContent.WriteString(choice.Message.Content)
		}

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

				if c.debug {
					fmt.Printf("[DEBUG] Tool call: %s\n", tc.Function.Name)
					fmt.Printf("[DEBUG]   Arguments: %s\n", tc.Function.Arguments)
				}

				result := executor(tc.Function.Name, args)

				if c.debug {
					// Truncate long outputs for readability
					debugResult := result
					if len(debugResult) > 200 {
						debugResult = debugResult[:200] + "..."
					}
					fmt.Printf("[DEBUG]   Output: %s\n", debugResult)
				}

				toolResults = append(toolResults, result)

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

		// No tool calls - return the accumulated text response
		finalContent := strings.TrimSpace(accumulatedContent.String())

		// If no text content but tools were called, provide a simple confirmation
		// (The actual tool outputs are printed by the executor as they happen)
		if finalContent == "" && len(toolResults) > 0 {
			finalContent = "Done."
		}

		// If we got no content at all (no text, no tool calls), the API likely
		// returned an empty or malformed response
		if finalContent == "" && len(toolResults) == 0 && totalInputTokens == 0 {
			return nil, newHistory, fmt.Errorf("received empty response from API (no content or tool calls)")
		}

		assistantMsg := &Message{
			Role:    "assistant",
			Content: finalContent,
		}
		newHistory = append(newHistory, assistantMsg)

		return &Response{
			Text:         finalContent,
			FinishReason: choice.FinishReason,
			TokensUsed:   totalTokens,
			InputTokens:  totalInputTokens,
			OutputTokens: totalOutputTokens,
			Cost:         totalCost,
		}, newHistory, nil
	}
}

func (c *OpenRouterClient) SetDebug(enabled bool) {
	c.debug = enabled
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
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for error in response body (some APIs return 200 with error in body)
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s (code: %s)", result.Error.Message, result.Error.Code)
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

