package commands

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Name:        "/project",
		Description: "Create a new project",
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

			fmt.Printf("Created project: %s (ID: %s)\n", project.Name, project.ID)
			return false
		},
	})

	Register(&Command{
		Name:        "/projects",
		Description: "List all projects",
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
					p.ID, p.Name, done, len(tasks))
			}

			return false
		},
	})

	Register(&Command{
		Name:        "/delproject",
		Description: "Delete a project and its tasks",
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /delproject <project-id>")
				return false
			}

			projectID := args[0]
			if err := GetStore().DeleteProject(projectID); err != nil {
				fmt.Printf("Error deleting project: %v\n", err)
				return false
			}

			fmt.Printf("Deleted project: %s\n", projectID)
			return false
		},
	})
}
