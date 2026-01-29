package storage

import "time"

// Duration represents a task duration
type Duration string

const (
	Duration15m Duration = "15m"
	Duration30m Duration = "30m"
	Duration1h  Duration = "1h"
	Duration2h  Duration = "2h"
	Duration4h  Duration = "4h"
)

// ValidDurations lists all valid duration values
var ValidDurations = []Duration{Duration15m, Duration30m, Duration1h, Duration2h, Duration4h}

// IsValidDuration checks if a string is a valid duration
func IsValidDuration(s string) bool {
	for _, d := range ValidDurations {
		if string(d) == s {
			return true
		}
	}
	return false
}

// Project represents a parent container for tasks
type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Task represents a child item within a project
type Task struct {
	ID        string     `json:"id"`
	ProjectID string     `json:"project_id"`
	Name      string     `json:"name"`
	Done      bool       `json:"done"`
	CreatedAt time.Time  `json:"created_at"`
	DueDate   *time.Time `json:"due_date,omitempty"`
	Duration  Duration   `json:"duration,omitempty"`
}
