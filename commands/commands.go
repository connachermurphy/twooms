package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"twooms/llm"
	"twooms/storage"
)

// ParamType defines the type of a command parameter
type ParamType string

const (
	ParamTypeString ParamType = "string"
)

// Param defines a parameter for a command
type Param struct {
	Name        string
	Type        ParamType
	Description string
	Required    bool
}

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Handler     func(args []string) bool // returns true to quit
	Params      []Param                  // parameter definitions for tool generation
	Hidden      bool                     // if true, exclude from tool generation
	Destructive bool                     // if true, requires confirmation when called via tool
}

var (
	registry  = make(map[string]*Command)
	store     storage.Store
	llmClient llm.Client
)

// Register adds a command to the registry
func Register(cmd *Command) {
	registry[strings.ToLower(cmd.Name)] = cmd
}

// SetStore sets the global store for commands to use
func SetStore(s storage.Store) {
	store = s
}

// GetStore returns the global store
func GetStore() storage.Store {
	return store
}

// SetLLMClient sets the global LLM client for commands to use
func SetLLMClient(c llm.Client) {
	llmClient = c
}

// GetLLMClient returns the global LLM client
func GetLLMClient() llm.Client {
	return llmClient
}

// Execute runs a command by name with arguments
func Execute(input string) (bool, error) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false, fmt.Errorf("empty command")
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	cmd, exists := registry[cmdName]
	if !exists {
		return false, fmt.Errorf("unknown command: %s", cmdName)
	}

	return cmd.Handler(args), nil
}

// ExecuteWithOutput runs a command and returns its captured stdout output
func ExecuteWithOutput(input string) (quit bool, output string, err error) {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return false, "", fmt.Errorf("failed to create pipe: %w", pipeErr)
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

	// Run the command
	quit, err = Execute(input)

	// Close the write end of the pipe and wait for read to complete
	w.Close()
	<-done
	r.Close()

	output = strings.TrimSpace(buf.String())
	return quit, output, err
}

// List returns all registered commands
func List() []*Command {
	cmds := make([]*Command, 0, len(registry))
	for _, cmd := range registry {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// GetByName returns a command by name (with or without leading /)
func GetByName(name string) *Command {
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}
	return registry[strings.ToLower(name)]
}

// GenerateToolDefinitions creates Tool definitions from registered commands
func GenerateToolDefinitions() []*llm.Tool {
	var tools []*llm.Tool

	for _, cmd := range registry {
		if cmd.Hidden {
			continue
		}

		// Build properties and required arrays from Params
		properties := make(map[string]*llm.ToolProperty)
		var required []string

		for _, p := range cmd.Params {
			properties[p.Name] = &llm.ToolProperty{
				Type:        "string",
				Description: p.Description,
			}
			if p.Required {
				required = append(required, p.Name)
			}
		}

		// Create Tool
		tool := &llm.Tool{
			Name:        strings.TrimPrefix(cmd.Name, "/"),
			Description: cmd.Description,
		}

		// Only add Parameters if there are any
		if len(properties) > 0 {
			tool.Parameters = &llm.ToolParameters{
				Type:       "object",
				Properties: properties,
				Required:   required,
			}
		}

		tools = append(tools, tool)
	}

	return tools
}
