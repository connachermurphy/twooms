package commands

import (
	"fmt"
	"sort"
)

func init() {
	Register(&Command{
		Name:        "/help",
		Shorthand:   "/h",
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
				nameCol := cmd.Name
				if cmd.Shorthand != "" {
					nameCol = fmt.Sprintf("%s (%s)", cmd.Name, cmd.Shorthand)
				}
				fmt.Printf("  %-22s - %s\n", nameCol, cmd.Description)
			}

			return false
		},
	})
}
