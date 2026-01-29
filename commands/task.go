package commands

import (
	"fmt"
	"strings"
	"time"

	"twooms/storage"
)

func init() {
	Register(&Command{
		Name:        "/task",
		Description: "Add a task to a project",
		Params: []Param{
			{Name: "project_id", Type: ParamTypeString, Description: "The ID of the project to add the task to", Required: true},
			{Name: "task_name", Type: ParamTypeString, Description: "The name of the task to create", Required: true},
		},
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
		Params: []Param{
			{Name: "project_id", Type: ParamTypeString, Description: "The ID of the project to list tasks for", Required: true},
		},
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

				// Build extra info string
				var extras []string
				if t.Duration != "" {
					extras = append(extras, string(t.Duration))
				}
				if t.DueDate != nil {
					extras = append(extras, "due "+t.DueDate.Format("2006-01-02"))
				}

				extraStr := ""
				if len(extras) > 0 {
					extraStr = " (" + strings.Join(extras, ", ") + ")"
				}

				fmt.Printf("  %s [%s] %s%s\n", status, t.ID, t.Name, extraStr)
			}

			return false
		},
	})

	Register(&Command{
		Name:        "/done",
		Description: "Mark a task as done",
		Params: []Param{
			{Name: "task_id", Type: ParamTypeString, Description: "The ID of the task to mark as done", Required: true},
		},
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
		Params: []Param{
			{Name: "task_id", Type: ParamTypeString, Description: "The ID of the task to mark as not done", Required: true},
		},
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
		Params: []Param{
			{Name: "task_id", Type: ParamTypeString, Description: "The ID of the task to delete", Required: true},
		},
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

	Register(&Command{
		Name:        "/due",
		Description: "Set a task's due date",
		Params: []Param{
			{Name: "task_id", Type: ParamTypeString, Description: "The ID of the task", Required: true},
			{Name: "date", Type: ParamTypeString, Description: "Due date in YYYY-MM-DD format, or 'none' to clear", Required: true},
		},
		Handler: func(args []string) bool {
			if len(args) < 2 {
				fmt.Println("Usage: /due <task-id> <YYYY-MM-DD|none>")
				return false
			}

			taskID := args[0]
			dateStr := args[1]

			if dateStr == "none" {
				if err := GetStore().SetTaskDueDate(taskID, nil); err != nil {
					fmt.Printf("Error: %v\n", err)
					return false
				}
				fmt.Printf("Cleared due date for task %s\n", taskID)
				return false
			}

			dueDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				fmt.Println("Error: Invalid date format. Use YYYY-MM-DD (e.g., 2024-12-31)")
				return false
			}

			if err := GetStore().SetTaskDueDate(taskID, &dueDate); err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			fmt.Printf("Set due date for task %s to %s\n", taskID, dateStr)
			return false
		},
	})

	Register(&Command{
		Name:        "/duration",
		Description: "Set a task's duration",
		Params: []Param{
			{Name: "task_id", Type: ParamTypeString, Description: "The ID of the task", Required: true},
			{Name: "duration", Type: ParamTypeString, Description: "Duration: 15m, 30m, 1h, 2h, or 4h", Required: true},
		},
		Handler: func(args []string) bool {
			if len(args) < 2 {
				fmt.Println("Usage: /duration <task-id> <15m|30m|1h|2h|4h>")
				return false
			}

			taskID := args[0]
			durationStr := args[1]

			if !storage.IsValidDuration(durationStr) {
				fmt.Println("Error: Invalid duration. Use 15m, 30m, 1h, 2h, or 4h")
				return false
			}

			if err := GetStore().SetTaskDuration(taskID, storage.Duration(durationStr)); err != nil {
				fmt.Printf("Error: %v\n", err)
				return false
			}

			fmt.Printf("Set duration for task %s to %s\n", taskID, durationStr)
			return false
		},
	})
}
