package commands

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

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

// commandContextPrefix identifies command context messages in history
const commandContextPrefix = "[Command executed]"

// getSystemPrompt returns the system prompt with today's date and tool-use instructions
func getSystemPrompt() string {
	today := time.Now().Format("2006-01-02") // YYYY-MM-DD format
	weekday := time.Now().Weekday().String()

	return fmt.Sprintf(`You are a helpful task management assistant for Twooms, a terminal-based task manager.

TODAY'S DATE: %s (%s)

IMPORTANT RULES:
1. When a user refers to a project by NAME (not ID), FIRST call "projects" to find the ID, then use that ID.
2. When a user refers to a task by NAME, FIRST call the listing tool to find the task's ID.
3. NEVER ask the user for an ID. Always look it up using available tools.
4. When users refer to "that task" or "the project I just created", use context from [Command executed] messages.
5. When setting due dates: "today" = %s, "tomorrow" = the next day, etc.
6. Tool outputs are ALREADY shown to the user. After using tools, just say "Done." or give a one-sentence summary. Do NOT repeat or list the tool output.
7. Be concise since this is a terminal application.`, today, weekday, today)
}

// ensureSystemPrompt adds the system prompt if chat history is empty
func ensureSystemPrompt() {
	if len(chatHistory) == 0 {
		chatHistory = append(chatHistory, &llm.Message{
			Role:    "system",
			Content: getSystemPrompt(),
		})
	}
}

// AddCommandContext adds a direct command and its output to the chat history
// so the LLM has context about recent user actions.
func AddCommandContext(command string, output string) {
	ensureSystemPrompt()

	contextMsg := fmt.Sprintf("%s %s\nResult: %s", commandContextPrefix, command, output)
	chatHistory = append(chatHistory, &llm.Message{
		Role:    "user",
		Content: contextMsg,
	})
	// Add assistant acknowledgment to maintain proper conversation flow
	chatHistory = append(chatHistory, &llm.Message{
		Role:    "assistant",
		Content: "Noted.",
	})

	// Trim old context entries to avoid unbounded growth
	// Keep only the most recent command context entries
	trimCommandContext()
}

// trimCommandContext removes old command context entries if there are too many
func trimCommandContext() {
	// Count messages that are command context
	var contextCount int
	for _, msg := range chatHistory {
		if strings.HasPrefix(msg.Content, commandContextPrefix) {
			contextCount++
		}
	}

	// Remove oldest context entries if over limit
	if contextCount > maxCommandContextEntries {
		toRemove := contextCount - maxCommandContextEntries
		var newHistory []*llm.Message
		skipNext := false
		for _, msg := range chatHistory {
			if skipNext {
				skipNext = false
				continue
			}
			if toRemove > 0 && strings.HasPrefix(msg.Content, commandContextPrefix) {
				toRemove--
				skipNext = true // Also skip the following "Noted." acknowledgment
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
			} else {
				fmt.Println("  Total cost:    no data")
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

			// Ensure system prompt is present
			ensureSystemPrompt()

			message := strings.Join(args, " ")
			tools := GenerateToolDefinitions()

			// Create the tool executor that runs commands and captures output
			executor := func(name string, fnArgs map[string]any) string {
				// Check if command is destructive and requires confirmation
				cmd := GetByName(name)
				if cmd != nil && cmd.Destructive {
					// Get description of what will be deleted
					description := getDestructiveDescription(name, fnArgs)
					if !confirmDestructiveAction(name, description) {
						return "Action cancelled by user."
					}
				}

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

				// Print output immediately so user sees progress
				if output != "" {
					fmt.Println(output)
				}

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

			// Only print response text if non-empty (tool outputs already printed)
			if strings.TrimSpace(response.Text) != "" {
				fmt.Println(response.Text)
			}

			// Display usage statistics
			printUsageStats(response)
			return false
		},
	})
}

// printUsageStats displays token usage and cost information and updates session totals
func printUsageStats(response *llm.Response) {
	// Only count if we have actual token data
	if response.InputTokens > 0 || response.OutputTokens > 0 {
		sessionInputTokens += response.InputTokens
		sessionOutputTokens += response.OutputTokens
		sessionCost += response.Cost
		sessionPromptCount++
	}

	// Always show token info (helps debug silent failures)
	fmt.Printf("\n[Tokens: %d in / %d out", response.InputTokens, response.OutputTokens)

	// Display cost if available
	if response.Cost > 0 {
		// Format cost appropriately based on magnitude
		if response.Cost < 0.01 {
			fmt.Printf(" | Cost: $%.6f", response.Cost)
		} else {
			fmt.Printf(" | Cost: $%.4f", response.Cost)
		}
	} else {
		fmt.Printf(" | Cost: no data")
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
	defer func() { os.Stdout = oldStdout }()

	// Read in a goroutine to prevent pipe buffer deadlock
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	// Run the function
	fn()

	// Close the write end of the pipe and wait for read to complete
	w.Close()
	<-done
	r.Close()

	return strings.TrimSpace(buf.String())
}

// getDestructiveDescription returns a human-readable description of what will be deleted
func getDestructiveDescription(cmdName string, args map[string]any) string {
	store := GetStore()
	if store == nil {
		return ""
	}

	switch cmdName {
	case "delproject":
		if projectRef, ok := args["project_id"].(string); ok {
			projectID, err := store.ResolveProjectID(projectRef)
			if err != nil {
				return fmt.Sprintf("project '%s'", projectRef)
			}
			project, err := store.GetProject(projectID)
			if err != nil {
				return fmt.Sprintf("project '%s'", projectRef)
			}
			tasks, _ := store.ListTasks(projectID)
			if len(tasks) > 0 {
				return fmt.Sprintf("project '%s' and its %d task(s)", project.Name, len(tasks))
			}
			return fmt.Sprintf("project '%s'", project.Name)
		}
	case "deltask":
		if taskRef, ok := args["task_id"].(string); ok {
			taskID, err := store.ResolveTaskID(taskRef)
			if err != nil {
				return fmt.Sprintf("task '%s'", taskRef)
			}
			task, err := store.GetTask(taskID)
			if err != nil {
				return fmt.Sprintf("task '%s'", taskRef)
			}
			return fmt.Sprintf("task '%s'", task.Name)
		}
	}
	return ""
}

// confirmDestructiveAction prompts the user to confirm a destructive action
func confirmDestructiveAction(cmdName string, description string) bool {
	if description == "" {
		description = "this item"
	}

	fmt.Printf("\nConfirm delete %s? [y/N]: ", description)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response == "y" || response == "yes" {
			return true
		}
	}

	fmt.Println("Cancelled.")
	return false
}
