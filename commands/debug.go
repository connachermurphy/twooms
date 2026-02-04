package commands

import "fmt"

var debugMode bool

func init() {
	Register(&Command{
		Name:        "/debug",
		Description: "Toggle debug mode for LLM interactions",
		Hidden:      true,
		Handler: func(args []string) bool {
			debugMode = !debugMode
			if debugMode {
				fmt.Println("Debug mode: ON")
			} else {
				fmt.Println("Debug mode: OFF")
			}
			return false
		},
	})
}

// IsDebugMode returns whether debug mode is enabled
func IsDebugMode() bool {
	return debugMode
}
