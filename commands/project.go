package commands

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Name:        "/project",
		Description: "Create a new project",
		Params: []Param{
			{Name: "name", Type: ParamTypeString, Description: "The name of the project to create", Required: true},
		},
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /project <name>")
				return false
			}

			name := strings.Join(args, " ")
			project, err := GetStore().CreateProject(name)
			if err != nil {
				fmt.Printf("Error creating project: %v\n", err)
				return false
			}

			fmt.Printf("Created project: %s (shortcut: %s)\n", project.Name, project.Shortcut)
			return false
		},
	})

	Register(&Command{
		Name:        "/projects",
		Description: "List all projects with their IDs. Use this to find a project's ID when you have the name.",
		Handler: func(args []string) bool {
			projects, err := GetStore().ListProjects()
			if err != nil {
				fmt.Printf("Error listing projects: %v\n", err)
				return false
			}

			if len(projects) == 0 {
				fmt.Println("No projects yet. Create one with /project <name>")
				return false
			}

			fmt.Println("Projects:")
			for _, p := range projects {
				// Count tasks for this project
				tasks, _ := GetStore().ListTasks(p.ID)
				done := 0
				for _, t := range tasks {
					if t.Done {
						done++
					}
				}

				fmt.Printf("  [%s] %s (%d/%d tasks complete)\n",
					p.Shortcut, p.Name, done, len(tasks))
			}

			return false
		},
	})

	Register(&Command{
		Name:        "/delproject",
		Description: "Delete a project and its tasks",
		Destructive: true,
		Params: []Param{
			{Name: "project_id", Type: ParamTypeString, Description: "The ID of the project to delete", Required: true},
		},
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /delproject <project-id>")
				return false
			}

			projectRef := args[0]

			// Resolve project ID
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

			if err := GetStore().DeleteProject(projectID); err != nil {
				fmt.Printf("Error deleting project: %v\n", err)
				return false
			}

			fmt.Printf("Deleted project: %s\n", project.Name)
			return false
		},
	})
}
