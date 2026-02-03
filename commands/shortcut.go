package commands

import "fmt"

func init() {
	Register(&Command{
		Name:        "/shortcut",
		Description: "Set a custom shortcut for a project",
		Params: []Param{
			{Name: "project_id", Type: ParamTypeString, Description: "The ID or current shortcut of the project", Required: true},
			{Name: "new_shortcut", Type: ParamTypeString, Description: "The new shortcut (alphanumeric + hyphens, max 20 chars)", Required: true},
		},
		Handler: func(args []string) bool {
			if len(args) < 2 {
				fmt.Println("Usage: /shortcut <project-id> <new-shortcut>")
				return false
			}

			projectRef := args[0]
			newShortcut := args[1]

			// Resolve the project ID
			projectID, err := GetStore().ResolveProjectID(projectRef)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			// Get project for display
			project, err := GetStore().GetProject(projectID)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			// Set the new shortcut
			if err := GetStore().SetProjectShortcut(projectID, newShortcut); err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			fmt.Printf("Set shortcut for %s to: %s\n", project.Name, newShortcut)
			return false
		},
	})
}
