package commands

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Name:        "/task",
		Description: "Add a task to a project",
		Handler: func(args []string) bool {
			if len(args) < 2 {
				fmt.Println("Usage: /task <project-id> <task name>")
				return false
			}

			projectID := args[0]
			taskName := strings.Join(args[1:], " ")

			task, err := GetStore().CreateTask(projectID, taskName)
			if err != nil {
				fmt.Printf("Error creating task: %v\n", err)
				return false
			}

			fmt.Printf("Created task: %s (ID: %s)\n", task.Name, task.ID)
			return false
		},
	})

	Register(&Command{
		Name:        "/tasks",
		Description: "List tasks in a project",
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /tasks <project-id>")
				return false
			}

			projectID := args[0]

			// Get project info
			project, err := GetStore().GetProject(projectID)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			tasks, err := GetStore().ListTasks(projectID)
			if err != nil {
				fmt.Printf("Error listing tasks: %v\n", err)
				return false
			}

			fmt.Printf("Tasks in %s:\n", project.Name)
			if len(tasks) == 0 {
				fmt.Println("  No tasks yet. Add one with /task <project-id> <name>")
				return false
			}

			for _, t := range tasks {
				status := "[ ]"
				if t.Done {
					status = "[✓]"
				}
				fmt.Printf("  %s [%s] %s\n", status, t.ID, t.Name)
			}

			return false
		},
	})

	Register(&Command{
		Name:        "/done",
		Description: "Mark a task as done",
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /done <task-id>")
				return false
			}

			taskID := args[0]
			if err := GetStore().UpdateTask(taskID, true); err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			fmt.Printf("Marked task %s as done ✓\n", taskID)
			return false
		},
	})

	Register(&Command{
		Name:        "/undone",
		Description: "Mark a task as not done",
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /undone <task-id>")
				return false
			}

			taskID := args[0]
			if err := GetStore().UpdateTask(taskID, false); err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			fmt.Printf("Marked task %s as not done\n", taskID)
			return false
		},
	})

	Register(&Command{
		Name:        "/deltask",
		Description: "Delete a task",
		Handler: func(args []string) bool {
			if len(args) == 0 {
				fmt.Println("Usage: /deltask <task-id>")
				return false
			}

			taskID := args[0]
			if err := GetStore().DeleteTask(taskID); err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			fmt.Printf("Deleted task: %s\n", taskID)
			return false
		},
	})
}
