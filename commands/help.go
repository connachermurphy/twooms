package commands

import (
	"fmt"
	"sort"
)

func init() {
	Register(&Command{
		Name:        "/help",
		Description: "Show available commands",
		Hidden:      true,
		Handler: func(args []string) bool {
			fmt.Println("Available commands:")

			// Get all commands and sort by name
			cmds := List()
			sort.Slice(cmds, func(i, j int) bool {
				return cmds[i].Name < cmds[j].Name
			})

			for _, cmd := range cmds {
				fmt.Printf("  %-15s - %s\n", cmd.Name, cmd.Description)
			}

			return false
		},
	})
}
