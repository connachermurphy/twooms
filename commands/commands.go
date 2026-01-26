package commands

import (
	"fmt"
	"strings"

	"twooms/storage"
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Handler     func(args []string) bool // returns true to quit
}

var (
	registry = make(map[string]*Command)
	store    storage.Store
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

// List returns all registered commands
func List() []*Command {
	cmds := make([]*Command, 0, len(registry))
	for _, cmd := range registry {
		cmds = append(cmds, cmd)
	}
	return cmds
}
