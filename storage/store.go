package storage

import "time"

// Store defines the interface for task manager storage
// This allows swapping between JSON, bbolt, or other backends
type Store interface {
	// Project operations
	CreateProject(name string) (*Project, error)
	ListProjects() ([]*Project, error)
	GetProject(id string) (*Project, error)
	DeleteProject(id string) error
	SetProjectShortcut(projectID, shortcut string) error

	// ID resolution - resolves shortcuts/prefixes to full UUIDs
	ResolveProjectID(idOrShortcut string) (string, error)
	ResolveTaskID(idOrPrefix string) (string, error)

	// Task operations
	CreateTask(projectID, name string) (*Task, error)
	ListTasks(projectID string) ([]*Task, error)
	ListAllTasks() ([]*Task, error)
	GetTask(id string) (*Task, error)
	UpdateTask(id string, done bool) error
	SetTaskDueDate(id string, dueDate *time.Time) error
	SetTaskDuration(id string, duration Duration) error
	DeleteTask(id string) error

	// Lifecycle
	Close() error
}
