package storage

import (
	"fmt"
	"time"
)

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

// ToMinutes converts a Duration to minutes
func (d Duration) ToMinutes() int {
	switch d {
	case Duration15m:
		return 15
	case Duration30m:
		return 30
	case Duration1h:
		return 60
	case Duration2h:
		return 120
	case Duration4h:
		return 240
	default:
		return 0
	}
}

// FormatMinutes formats a number of minutes as a human-readable string (e.g., "2h 30m")
func FormatMinutes(minutes int) string {
	if minutes == 0 {
		return "0m"
	}
	hours := minutes / 60
	mins := minutes % 60
	if hours == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

// TotalDuration calculates the total duration in minutes for a slice of tasks
func TotalDuration(tasks []*Task) int {
	total := 0
	for _, t := range tasks {
		total += t.Duration.ToMinutes()
	}
	return total
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
