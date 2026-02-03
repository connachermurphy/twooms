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
			return false
		},
	})
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
