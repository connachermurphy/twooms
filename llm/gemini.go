package llm

import (
	"context"
	"os"
	"strings"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client *genai.Client
}

func NewGeminiClient(ctx context.Context) (*GeminiClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &GeminiClient{client: client}, nil
}

func (g *GeminiClient) Chat(ctx context.Context, prompt string) (*Response, error) {
	return g.ChatWithConfig(ctx, prompt, DefaultConfig())
}

func (g *GeminiClient) ChatWithConfig(ctx context.Context, prompt string, config *Config) (*Response, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, ErrEmptyPrompt
	}

	if config == nil {
		config = DefaultConfig()
	}

	genConfig := &genai.GenerateContentConfig{
		MaxOutputTokens: config.MaxTokens,
		Temperature:     genai.Ptr(config.Temperature),
	}

	if config.System != "" {
		genConfig.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: config.System}},
		}
	}

	result, err := g.client.Models.GenerateContent(ctx, config.Model, genai.Text(prompt), genConfig)
	if err != nil {
		return nil, err
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return nil, ErrNoResponse
	}

	var text string
	for _, part := range result.Candidates[0].Content.Parts {
		if part.Text != "" {
			text += part.Text
		}
	}

	var tokensUsed int64
	if result.UsageMetadata != nil {
		tokensUsed = int64(result.UsageMetadata.TotalTokenCount)
	}

	finishReason := ""
	if result.Candidates[0].FinishReason != "" {
		finishReason = string(result.Candidates[0].FinishReason)
	}

	return &Response{
		Text:         text,
		FinishReason: finishReason,
		TokensUsed:   tokensUsed,
	}, nil
}

func (g *GeminiClient) ChatWithTools(ctx context.Context, message string, history []*genai.Content, tools []*genai.FunctionDeclaration, executor ToolExecutor) (*Response, []*genai.Content, error) {
	if strings.TrimSpace(message) == "" {
		return nil, history, ErrEmptyPrompt
	}

	config := DefaultConfig()

	genConfig := &genai.GenerateContentConfig{
		MaxOutputTokens: config.MaxTokens,
		Temperature:     genai.Ptr(config.Temperature),
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: getToolSystemPrompt()}},
		},
		Tools: []*genai.Tool{
			{FunctionDeclarations: tools},
		},
	}

	// Build conversation contents from history plus new message
	contents := make([]*genai.Content, len(history))
	copy(contents, history)
	contents = append(contents, genai.NewContentFromText(message, genai.RoleUser))

	var totalTokens int64

	// Tool calling loop
	for {
		result, err := g.client.Models.GenerateContent(ctx, config.Model, contents, genConfig)
		if err != nil {
			return nil, contents, err
		}

		if result.UsageMetadata != nil {
			totalTokens += int64(result.UsageMetadata.TotalTokenCount)
		}

		if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
			return nil, contents, ErrNoResponse
		}

		candidate := result.Candidates[0]

		// Check for function calls
		var functionCalls []*genai.FunctionCall
		var textParts []string

		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				functionCalls = append(functionCalls, part.FunctionCall)
			}
			if part.Text != "" {
				textParts = append(textParts, part.Text)
			}
		}

		// If no function calls, return the text response
		if len(functionCalls) == 0 {
			finishReason := ""
			if candidate.FinishReason != "" {
				finishReason = string(candidate.FinishReason)
			}

			// Add model's final response to history
			contents = append(contents, candidate.Content)

			return &Response{
				Text:         strings.Join(textParts, ""),
				FinishReason: finishReason,
				TokensUsed:   totalTokens,
			}, contents, nil
		}

		// Add model's response to history
		contents = append(contents, candidate.Content)

		// Execute function calls and build responses
		var functionResponses []*genai.Part
		for _, fc := range functionCalls {
			result := executor(fc.Name, fc.Args)
			functionResponses = append(functionResponses, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     fc.Name,
					Response: map[string]any{"result": result},
				},
			})
		}

		// Add function responses to history
		contents = append(contents, &genai.Content{
			Role:  genai.RoleUser,
			Parts: functionResponses,
		})
	}
}

func (g *GeminiClient) Close() error {
	// The genai client doesn't have a Close method, but we implement it
	// for the interface to support potential future cleanup needs
	return nil
}

func getToolSystemPrompt() string {
	return `You are a helpful task management assistant for Twooms.

IMPORTANT RULES:
1. When a user refers to a project by NAME (not ID), FIRST call "projects" to find the ID, then use that ID.
2. When a user refers to a task by NAME, FIRST call the listing tool to find the task's ID.
3. NEVER ask the user for an ID. Always look it up using available tools.
4. Project IDs look like "proj-1". Task IDs look like "task-1".

EXAMPLES:
- "list tasks in Office" -> call projects, find Office's ID, call tasks with that ID
- "mark documentation task done" -> list projects/tasks to find IDs, then call done`
}
