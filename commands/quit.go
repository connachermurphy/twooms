package commands

import "fmt"

func init() {
	Register(&Command{
		Name:        "/quit",
		Description: "Exit Twooms",
		Hidden:      true,
		Handler: func(args []string) bool {
			fmt.Println("Goodbye!")
			return true
		},
	})

	// Alias
	Register(&Command{
		Name:        "/exit",
		Description: "Exit Twooms",
		Hidden:      true,
		Handler: func(args []string) bool {
			fmt.Println("Goodbye!")
			return true
		},
	})
}
