package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"twooms/llm"
)

// chatHistory stores the conversation history for the /chat command
var chatHistory []*llm.Message

// Session usage tracking
var (
	sessionInputTokens  int64
	sessionOutputTokens int64
	sessionCost         float64
	sessionPromptCount  int
)

// maxCommandContextEntries limits how many command context entries to keep
const maxCommandContextEntries = 10

// AddCommandContext adds a direct command and its output to the chat history
// so the LLM has context about recent user actions.
func AddCommandContext(command string, output string) {
	contextMsg := fmt.Sprintf("User ran: %s\nOutput: %s", command, output)
	chatHistory = append(chatHistory, &llm.Message{
		Role:    "system",
		Content: contextMsg,
	})

	// Trim old context entries to avoid unbounded growth
	// Keep only the most recent command context entries
	trimCommandContext()
}

// trimCommandContext removes old command context entries if there are too many
func trimCommandContext() {
	// Count system messages that are command context (not the initial system prompt)
	var contextCount int
	for _, msg := range chatHistory {
		if msg.Role == "system" && strings.HasPrefix(msg.Content, "User ran:") {
			contextCount++
		}
	}

	// Remove oldest context entries if over limit
	if contextCount > maxCommandContextEntries {
		toRemove := contextCount - maxCommandContextEntries
		var newHistory []*llm.Message
		for _, msg := range chatHistory {
			if toRemove > 0 && msg.Role == "system" && strings.HasPrefix(msg.Content, "User ran:") {
				toRemove--
				continue
			}
			newHistory = append(newHistory, msg)
		}
		chatHistory = newHistory
	}
}

func init() {
	Register(&Command{
		Name:        "/clearchat",
		Description: "Clear the chat conversation history",
		Hidden:      true,
		Handler: func(args []string) bool {
			chatHistory = nil
			fmt.Println("Chat history cleared.")
			return false
		},
	})

	Register(&Command{
		Name:        "/usage",
		Description: "Show session token usage and cost statistics",
		Hidden:      true,
		Handler: func(args []string) bool {
			if sessionPromptCount == 0 {
				fmt.Println("No chat usage in this session yet.")
				return false
			}

			fmt.Println("Session Usage Statistics:")
			fmt.Printf("  Prompts:       %d\n", sessionPromptCount)
			fmt.Printf("  Input tokens:  %d\n", sessionInputTokens)
			fmt.Printf("  Output tokens: %d\n", sessionOutputTokens)
			fmt.Printf("  Total tokens:  %d\n", sessionInputTokens+sessionOutputTokens)
			if sessionCost > 0 {
				if sessionCost < 0.01 {
					fmt.Printf("  Total cost:    $%.6f\n", sessionCost)
				} else {
					fmt.Printf("  Total cost:    $%.4f\n", sessionCost)
				}
			}
			return false
		},
	})

	Register(&Command{
		Name:        "/chat",
		Description: "Chat with the AI assistant",
		Hidden:      true, // Exclude from tool generation
		Params: []Param{
			{Name: "message", Type: ParamTypeString, Description: "The message to send to the assistant", Required: true},
		},
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /chat <message>")
				return false
			}

			client := GetLLMClient()
			if client == nil {
				fmt.Println("Error: LLM client not available. Set OPENROUTER_API_KEY environment variable.")
				return false
			}

			message := strings.Join(args, " ")
			tools := GenerateToolDefinitions()

			// Create the tool executor that runs commands and captures output
			executor := func(name string, fnArgs map[string]any) string {
				// Convert function arguments to command args slice
				cmdArgs := convertArgsToSlice(name, fnArgs)

				// Build the full command string
				cmdStr := "/" + name
				if len(cmdArgs) > 0 {
					cmdStr += " " + strings.Join(cmdArgs, " ")
				}

				// Capture stdout while executing the command
				output := captureOutput(func() {
					Execute(cmdStr)
				})

				return output
			}

			ctx := context.Background()
			response, newHistory, err := client.ChatWithTools(ctx, message, chatHistory, tools, executor)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			// Update conversation history
			chatHistory = newHistory

			fmt.Println(response.Text)

			// Display usage statistics
			printUsageStats(response)
			return false
		},
	})
}

// printUsageStats displays token usage and cost information and updates session totals
func printUsageStats(response *llm.Response) {
	// Update session totals
	sessionInputTokens += response.InputTokens
	sessionOutputTokens += response.OutputTokens
	sessionCost += response.Cost
	sessionPromptCount++

	// Only display if we have token data
	if response.TokensUsed == 0 && response.InputTokens == 0 && response.OutputTokens == 0 {
		return
	}

	fmt.Printf("\n[Tokens: %d in / %d out", response.InputTokens, response.OutputTokens)

	// Display cost if available
	if response.Cost > 0 {
		// Format cost appropriately based on magnitude
		if response.Cost < 0.01 {
			fmt.Printf(" | Cost: $%.6f", response.Cost)
		} else {
			fmt.Printf(" | Cost: $%.4f", response.Cost)
		}
	}

	fmt.Println("]")
}

// convertArgsToSlice converts Gemini function call arguments to a string slice
// in the order expected by the command handler
func convertArgsToSlice(cmdName string, args map[string]any) []string {
	// Define the argument order for each command
	argOrder := map[string][]string{
		"project":    {"name"},
		"projects":   {},
		"delproject": {"project_id"},
		"task":       {"project_id", "task_name"},
		"tasks":      {"project_id"},
		"done":       {"task_id"},
		"undone":     {"task_id"},
		"deltask":    {"task_id"},
		"due":        {"task_id", "date"},
		"duration":   {"task_id", "duration"},
	}

	order, exists := argOrder[cmdName]
	if !exists {
		return nil
	}

	var result []string
	for _, key := range order {
		if val, ok := args[key]; ok {
			result = append(result, fmt.Sprintf("%v", val))
		}
	}

	return result
}

// captureOutput captures stdout during execution of a function
func captureOutput(fn func()) string {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Sprintf("Error capturing output: %v", err)
	}

	// Redirect stdout to the pipe
	os.Stdout = w

	// Run the function
	fn()

	// Close the write end of the pipe
	w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	return strings.TrimSpace(buf.String())
}
