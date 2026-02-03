package storage

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
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
	Migrated   bool       `json:"migrated"`
}

// generateUUID generates a UUID v4 using crypto/rand
func generateUUID() string {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// Fallback to a timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	// Set version (4) and variant (RFC 4122)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// shortcutRegex validates shortcut format: alphanumeric and hyphens, 1-20 chars
var shortcutRegex = regexp.MustCompile(`^[a-zA-Z0-9-]{1,20}$`)

// NewJSONStore creates or opens a JSON-backed store
func NewJSONStore(filename string) (*JSONStore, error) {
	store := &JSONStore{
		filename: filename,
		data: &jsonData{
			Projects:   []*Project{},
			Tasks:      []*Task{},
			NextProjID: 1,
			NextTaskID: 1,
			Migrated:   true, // New stores are already "migrated"
		},
	}

	// Try to load existing file
	if _, err := os.Stat(filename); err == nil {
		// Create fresh data struct for loading (Migrated defaults to false)
		store.data = &jsonData{}
		if err := store.load(); err != nil {
			return nil, fmt.Errorf("failed to load store: %w", err)
		}
		// Migrate old-style IDs to UUIDs
		if err := store.migrate(); err != nil {
			return nil, fmt.Errorf("failed to migrate store: %w", err)
		}
	}

	return store, nil
}

// migrate converts old proj-N/task-N IDs to UUIDs
func (s *JSONStore) migrate() error {
	if s.data.Migrated {
		return nil
	}

	// Build mapping from old project IDs to new UUIDs
	projectIDMap := make(map[string]string)

	for _, p := range s.data.Projects {
		if strings.HasPrefix(p.ID, "proj-") {
			newID := generateUUID()
			projectIDMap[p.ID] = newID
			p.ID = newID
			// Set shortcut to first 8 chars of UUID
			p.Shortcut = newID[:8]
		}
	}

	// Update tasks: new UUIDs and update ProjectID references
	for _, t := range s.data.Tasks {
		if strings.HasPrefix(t.ID, "task-") {
			t.ID = generateUUID()
		}
		// Update ProjectID if it was remapped
		if newProjectID, ok := projectIDMap[t.ProjectID]; ok {
			t.ProjectID = newProjectID
		}
	}

	s.data.Migrated = true
	return s.save()
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

	id := generateUUID()
	project := &Project{
		ID:        id,
		Name:      name,
		Shortcut:  id[:8], // Default shortcut is first 8 chars of UUID
		CreatedAt: time.Now(),
	}
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
		ID:        generateUUID(),
		ProjectID: projectID,
		Name:      name,
		Done:      false,
		CreatedAt: time.Now(),
	}
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

// ListAllTasks returns all tasks across all projects
func (s *JSONStore) ListAllTasks() ([]*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, len(s.data.Tasks))
	copy(tasks, s.data.Tasks)
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

// ResolveProjectID resolves a project identifier to its full UUID
// It checks: exact UUID match → shortcut match → UUID prefix (min 6 chars)
func (s *JSONStore) ResolveProjectID(idOrShortcut string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// First, try exact UUID match
	for _, p := range s.data.Projects {
		if p.ID == idOrShortcut {
			return p.ID, nil
		}
	}

	// Second, try shortcut match
	for _, p := range s.data.Projects {
		if p.Shortcut == idOrShortcut {
			return p.ID, nil
		}
	}

	// Third, try UUID prefix match (min 6 chars)
	if len(idOrShortcut) >= 6 {
		var matches []*Project
		for _, p := range s.data.Projects {
			if strings.HasPrefix(p.ID, idOrShortcut) {
				matches = append(matches, p)
			}
		}
		if len(matches) == 1 {
			return matches[0].ID, nil
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("ambiguous project ID prefix: %s (matches %d projects)", idOrShortcut, len(matches))
		}
	}

	return "", fmt.Errorf("project not found: %s", idOrShortcut)
}

// ResolveTaskID resolves a task identifier to its full UUID
// It checks: exact UUID match → UUID prefix (min 6 chars)
func (s *JSONStore) ResolveTaskID(idOrPrefix string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// First, try exact UUID match
	for _, t := range s.data.Tasks {
		if t.ID == idOrPrefix {
			return t.ID, nil
		}
	}

	// Second, try UUID prefix match (min 6 chars)
	if len(idOrPrefix) >= 6 {
		var matches []*Task
		for _, t := range s.data.Tasks {
			if strings.HasPrefix(t.ID, idOrPrefix) {
				matches = append(matches, t)
			}
		}
		if len(matches) == 1 {
			return matches[0].ID, nil
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("ambiguous task ID prefix: %s (matches %d tasks)", idOrPrefix, len(matches))
		}
	}

	return "", fmt.Errorf("task not found: %s", idOrPrefix)
}

// SetProjectShortcut sets a custom shortcut for a project
func (s *JSONStore) SetProjectShortcut(projectID, shortcut string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate shortcut format
	if !shortcutRegex.MatchString(shortcut) {
		return fmt.Errorf("invalid shortcut: must be 1-20 alphanumeric characters or hyphens")
	}

	// Check for shortcut conflicts
	for _, p := range s.data.Projects {
		if p.ID != projectID && p.Shortcut == shortcut {
			return fmt.Errorf("shortcut already in use by project: %s", p.Name)
		}
	}

	// Find and update the project
	for _, p := range s.data.Projects {
		if p.ID == projectID {
			p.Shortcut = shortcut
			return s.save()
		}
	}

	return fmt.Errorf("project not found: %s", projectID)
}

// Close closes the store
func (s *JSONStore) Close() error {
	// JSON store doesn't need cleanup, but interface requires it
	return nil
}
