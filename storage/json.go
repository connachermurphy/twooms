package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// JSONStore implements Store using a JSON file
type JSONStore struct {
	filename string
	data     *jsonData
	mu       sync.RWMutex
}

type jsonData struct {
	Projects   []*Project `json:"projects"`
	Tasks      []*Task    `json:"tasks"`
	NextProjID int        `json:"next_proj_id"`
	NextTaskID int        `json:"next_task_id"`
}

// NewJSONStore creates or opens a JSON-backed store
func NewJSONStore(filename string) (*JSONStore, error) {
	store := &JSONStore{
		filename: filename,
		data: &jsonData{
			Projects:   []*Project{},
			Tasks:      []*Task{},
			NextProjID: 1,
			NextTaskID: 1,
		},
	}

	// Try to load existing file
	if _, err := os.Stat(filename); err == nil {
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load store: %w", err)
		}
	}

	return store, nil
}

func (s *JSONStore) load() error {
	data, err := os.ReadFile(s.filename)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, s.data)
}

func (s *JSONStore) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filename, data, 0644)
}

// CreateProject creates a new project
func (s *JSONStore) CreateProject(name string) (*Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project := &Project{
		ID:        fmt.Sprintf("proj-%d", s.data.NextProjID),
		Name:      name,
		CreatedAt: time.Now(),
	}
	s.data.NextProjID++
	s.data.Projects = append(s.data.Projects, project)

	if err := s.save(); err != nil {
		return nil, err
	}

	return project, nil
}

// ListProjects returns all projects
func (s *JSONStore) ListProjects() ([]*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy to prevent external modification
	projects := make([]*Project, len(s.data.Projects))
	copy(projects, s.data.Projects)
	return projects, nil
}

// GetProject retrieves a project by ID
func (s *JSONStore) GetProject(id string) (*Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, p := range s.data.Projects {
		if p.ID == id {
			return p, nil
		}
	}

	return nil, fmt.Errorf("project not found: %s", id)
}

// DeleteProject removes a project and its tasks
func (s *JSONStore) DeleteProject(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find and remove project
	found := false
	for i, p := range s.data.Projects {
		if p.ID == id {
			s.data.Projects = append(s.data.Projects[:i], s.data.Projects[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("project not found: %s", id)
	}

	// Remove all tasks in this project
	newTasks := []*Task{}
	for _, t := range s.data.Tasks {
		if t.ProjectID != id {
			newTasks = append(newTasks, t)
		}
	}
	s.data.Tasks = newTasks

	return s.save()
}

// CreateTask creates a new task in a project
func (s *JSONStore) CreateTask(projectID, name string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify project exists
	projectExists := false
	for _, p := range s.data.Projects {
		if p.ID == projectID {
			projectExists = true
			break
		}
	}

	if !projectExists {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	task := &Task{
		ID:        fmt.Sprintf("task-%d", s.data.NextTaskID),
		ProjectID: projectID,
		Name:      name,
		Done:      false,
		CreatedAt: time.Now(),
	}
	s.data.NextTaskID++
	s.data.Tasks = append(s.data.Tasks, task)

	if err := s.save(); err != nil {
		return nil, err
	}

	return task, nil
}

// ListTasks returns all tasks for a project
func (s *JSONStore) ListTasks(projectID string) ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := []*Task{}
	for _, t := range s.data.Tasks {
		if t.ProjectID == projectID {
			tasks = append(tasks, t)
		}
	}

	return tasks, nil
}

// GetTask retrieves a task by ID
func (s *JSONStore) GetTask(id string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.data.Tasks {
		if t.ID == id {
			return t, nil
		}
	}

	return nil, fmt.Errorf("task not found: %s", id)
}

// UpdateTask updates a task's done status
func (s *JSONStore) UpdateTask(id string, done bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.data.Tasks {
		if t.ID == id {
			t.Done = done
			return s.save()
		}
	}

	return fmt.Errorf("task not found: %s", id)
}

// SetTaskDueDate sets or clears a task's due date
func (s *JSONStore) SetTaskDueDate(id string, dueDate *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.data.Tasks {
		if t.ID == id {
			t.DueDate = dueDate
			return s.save()
		}
	}

	return fmt.Errorf("task not found: %s", id)
}

// SetTaskDuration sets a task's duration
func (s *JSONStore) SetTaskDuration(id string, duration Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.data.Tasks {
		if t.ID == id {
			t.Duration = duration
			return s.save()
		}
	}

	return fmt.Errorf("task not found: %s", id)
}

// DeleteTask removes a task
func (s *JSONStore) DeleteTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, t := range s.data.Tasks {
		if t.ID == id {
			s.data.Tasks = append(s.data.Tasks[:i], s.data.Tasks[i+1:]...)
			return s.save()
		}
	}

	return fmt.Errorf("task not found: %s", id)
}

// Close closes the store
func (s *JSONStore) Close() error {
	// JSON store doesn't need cleanup, but interface requires it
	return nil
}
