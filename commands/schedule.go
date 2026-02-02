package commands

import (
	"fmt"
	"strings"
	"time"

	"twooms/storage"
)

func init() {
	Register(&Command{
		Name:        "/today",
		Description: "List tasks due today",
		Params: []Param{
			{Name: "project_id", Type: ParamTypeString, Description: "Optional project ID to filter by", Required: false},
		},
		Handler: func(args []string) bool {
			var projectID string
			if len(args) > 0 {
				projectID = args[0]
			}

			today := dateOnly(time.Now())
			tomorrow := today.AddDate(0, 0, 1)

			listTasksInRange("today", today, tomorrow, projectID)
			return false
		},
	})

	Register(&Command{
		Name:        "/tomorrow",
		Description: "List tasks due tomorrow",
		Params: []Param{
			{Name: "project_id", Type: ParamTypeString, Description: "Optional project ID to filter by", Required: false},
		},
		Handler: func(args []string) bool {
			var projectID string
			if len(args) > 0 {
				projectID = args[0]
			}

			today := dateOnly(time.Now())
			tomorrow := today.AddDate(0, 0, 1)
			dayAfter := today.AddDate(0, 0, 2)

			listTasksInRange("tomorrow", tomorrow, dayAfter, projectID)
			return false
		},
	})

	Register(&Command{
		Name:        "/week",
		Description: "List tasks due this week (Monday through Sunday)",
		Params: []Param{
			{Name: "project_id", Type: ParamTypeString, Description: "Optional project ID to filter by", Required: false},
		},
		Handler: func(args []string) bool {
			var projectID string
			if len(args) > 0 {
				projectID = args[0]
			}

			today := dateOnly(time.Now())
			weekStart := startOfWeek(today)
			weekEnd := weekStart.AddDate(0, 0, 7)

			listTasksInRange("this week", weekStart, weekEnd, projectID)
			return false
		},
	})
}

// dateOnly extracts just the year, month, day as a comparable date in local timezone
// This ignores the time-of-day and original timezone, treating the date as a calendar date
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// startOfWeek returns the Monday of the week containing the given time
func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday is day 7
	}
	return t.AddDate(0, 0, -(weekday - 1))
}

// listTasksInRange lists tasks with due dates in the given range [start, end)
func listTasksInRange(label string, start, end time.Time, projectID string) {
	var tasks []*storage.Task
	var err error

	if projectID != "" {
		// Verify project exists
		project, err := GetStore().GetProject(projectID)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		tasks, err = GetStore().ListTasks(projectID)
		if err != nil {
			fmt.Printf("Error listing tasks: %v\n", err)
			return
		}
		fmt.Printf("Tasks due %s in %s:\n", label, project.Name)
	} else {
		tasks, err = GetStore().ListAllTasks()
		if err != nil {
			fmt.Printf("Error listing tasks: %v\n", err)
			return
		}
		fmt.Printf("Tasks due %s:\n", label)
	}

	// Filter tasks by due date range and incomplete status
	var filtered []*storage.Task
	for _, t := range tasks {
		if t.Done {
			continue
		}
		if t.DueDate == nil {
			continue
		}
		due := dateOnly(*t.DueDate)
		if !due.Before(start) && due.Before(end) {
			filtered = append(filtered, t)
		}
	}

	if len(filtered) == 0 {
		fmt.Println("  No tasks due")
		return
	}

	// Build project name lookup for display
	projectNames := make(map[string]string)
	if projectID == "" {
		projects, _ := GetStore().ListProjects()
		for _, p := range projects {
			projectNames[p.ID] = p.Name
		}
	}

	for _, t := range filtered {
		var extras []string
		if t.Duration != "" {
			extras = append(extras, string(t.Duration))
		}
		extras = append(extras, "due "+t.DueDate.Format("2006-01-02"))
		if projectID == "" {
			if name, ok := projectNames[t.ProjectID]; ok {
				extras = append(extras, name)
			}
		}

		extraStr := ""
		if len(extras) > 0 {
			extraStr = " (" + strings.Join(extras, ", ") + ")"
		}

		fmt.Printf("  [ ] [%s] %s%s\n", t.ID, t.Name, extraStr)
	}

	// Show total duration
	totalMinutes := storage.TotalDuration(filtered)
	if totalMinutes > 0 {
		fmt.Printf("\nTotal: %s\n", storage.FormatMinutes(totalMinutes))
	}
}
