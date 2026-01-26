package storage

// Store defines the interface for task manager storage
// This allows swapping between JSON, bbolt, or other backends
type Store interface {
	// Project operations
	CreateProject(name string) (*Project, error)
	ListProjects() ([]*Project, error)
	GetProject(id string) (*Project, error)
	DeleteProject(id string) error

	// Task operations
	CreateTask(projectID, name string) (*Task, error)
	ListTasks(projectID string) ([]*Task, error)
	GetTask(id string) (*Task, error)
	UpdateTask(id string, done bool) error
	DeleteTask(id string) error

	// Lifecycle
	Close() error
}
