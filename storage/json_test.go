package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMigration(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.json")

	// Write old-style data directly to file
	oldData := &jsonData{
		Projects: []*Project{
			{ID: "proj-1", Name: "Work", CreatedAt: time.Now()},
			{ID: "proj-2", Name: "Personal", CreatedAt: time.Now()},
		},
		Tasks: []*Task{
			{ID: "task-1", ProjectID: "proj-1", Name: "Task A", CreatedAt: time.Now()},
			{ID: "task-2", ProjectID: "proj-1", Name: "Task B", CreatedAt: time.Now()},
			{ID: "task-3", ProjectID: "proj-2", Name: "Task C", CreatedAt: time.Now()},
		},
		NextProjID: 3,
		NextTaskID: 4,
		Migrated:   false,
	}

	data, err := json.MarshalIndent(oldData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal old data: %v", err)
	}
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatalf("Failed to write old data: %v", err)
	}

	// Load the store, which should trigger migration
	store, err := NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Verify projects were migrated
	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("Failed to list projects: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	for _, p := range projects {
		// IDs should be UUIDs (contain hyphens, not "proj-")
		if strings.HasPrefix(p.ID, "proj-") {
			t.Errorf("Project ID should be UUID, got: %s", p.ID)
		}
		if !strings.Contains(p.ID, "-") {
			t.Errorf("Project ID should be UUID format, got: %s", p.ID)
		}
		// Shortcut should be set (first 8 chars of UUID)
		if len(p.Shortcut) != 8 {
			t.Errorf("Project shortcut should be 8 chars, got: %s", p.Shortcut)
		}
		if !strings.HasPrefix(p.ID, p.Shortcut) {
			t.Errorf("Shortcut should be prefix of ID: %s vs %s", p.Shortcut, p.ID)
		}
	}

	// Verify tasks were migrated
	allTasks, err := store.ListAllTasks()
	if err != nil {
		t.Fatalf("Failed to list all tasks: %v", err)
	}
	if len(allTasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(allTasks))
	}

	for _, task := range allTasks {
		// Task IDs should be UUIDs
		if strings.HasPrefix(task.ID, "task-") {
			t.Errorf("Task ID should be UUID, got: %s", task.ID)
		}
		// ProjectID should reference UUID
		if strings.HasPrefix(task.ProjectID, "proj-") {
			t.Errorf("Task ProjectID should be UUID, got: %s", task.ProjectID)
		}
	}

	// Verify project-task relationships are preserved
	workProject := projects[0]
	if workProject.Name != "Work" {
		workProject = projects[1]
	}
	workTasks, err := store.ListTasks(workProject.ID)
	if err != nil {
		t.Fatalf("Failed to list work tasks: %v", err)
	}
	if len(workTasks) != 2 {
		t.Errorf("Expected 2 tasks in Work project, got %d", len(workTasks))
	}

	// Verify Migrated flag is set
	if !store.data.Migrated {
		t.Error("Migrated flag should be true after migration")
	}
}

func TestResolveProjectID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.json")

	store, err := NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create a project
	project, err := store.CreateProject("Test Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Resolve by exact UUID
	resolved, err := store.ResolveProjectID(project.ID)
	if err != nil {
		t.Errorf("Failed to resolve by UUID: %v", err)
	}
	if resolved != project.ID {
		t.Errorf("Expected %s, got %s", project.ID, resolved)
	}

	// Resolve by shortcut
	resolved, err = store.ResolveProjectID(project.Shortcut)
	if err != nil {
		t.Errorf("Failed to resolve by shortcut: %v", err)
	}
	if resolved != project.ID {
		t.Errorf("Expected %s, got %s", project.ID, resolved)
	}

	// Resolve by UUID prefix (6+ chars)
	resolved, err = store.ResolveProjectID(project.ID[:6])
	if err != nil {
		t.Errorf("Failed to resolve by 6-char prefix: %v", err)
	}
	if resolved != project.ID {
		t.Errorf("Expected %s, got %s", project.ID, resolved)
	}

	// Should fail with 5-char prefix
	_, err = store.ResolveProjectID(project.ID[:5])
	if err == nil {
		t.Error("Should fail to resolve with 5-char prefix")
	}

	// Should fail with nonexistent ID
	_, err = store.ResolveProjectID("nonexistent")
	if err == nil {
		t.Error("Should fail to resolve nonexistent project")
	}
}

func TestResolveTaskID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.json")

	store, err := NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create project and task
	project, err := store.CreateProject("Test Project")
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}
	task, err := store.CreateTask(project.ID, "Test Task")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Resolve by exact UUID
	resolved, err := store.ResolveTaskID(task.ID)
	if err != nil {
		t.Errorf("Failed to resolve by UUID: %v", err)
	}
	if resolved != task.ID {
		t.Errorf("Expected %s, got %s", task.ID, resolved)
	}

	// Resolve by UUID prefix (8 chars - display format)
	resolved, err = store.ResolveTaskID(task.ID[:8])
	if err != nil {
		t.Errorf("Failed to resolve by 8-char prefix: %v", err)
	}
	if resolved != task.ID {
		t.Errorf("Expected %s, got %s", task.ID, resolved)
	}

	// Resolve by 6-char prefix (minimum)
	resolved, err = store.ResolveTaskID(task.ID[:6])
	if err != nil {
		t.Errorf("Failed to resolve by 6-char prefix: %v", err)
	}
	if resolved != task.ID {
		t.Errorf("Expected %s, got %s", task.ID, resolved)
	}

	// Should fail with 5-char prefix
	_, err = store.ResolveTaskID(task.ID[:5])
	if err == nil {
		t.Error("Should fail to resolve with 5-char prefix")
	}

	// Should fail with nonexistent ID
	_, err = store.ResolveTaskID("nonexistent")
	if err == nil {
		t.Error("Should fail to resolve nonexistent task")
	}
}

func TestSetProjectShortcut(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.json")

	store, err := NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create two projects
	project1, err := store.CreateProject("Project 1")
	if err != nil {
		t.Fatalf("Failed to create project 1: %v", err)
	}
	project2, err := store.CreateProject("Project 2")
	if err != nil {
		t.Fatalf("Failed to create project 2: %v", err)
	}

	// Set custom shortcut
	err = store.SetProjectShortcut(project1.ID, "work")
	if err != nil {
		t.Errorf("Failed to set shortcut: %v", err)
	}

	// Verify shortcut was set
	updated, err := store.GetProject(project1.ID)
	if err != nil {
		t.Fatalf("Failed to get project: %v", err)
	}
	if updated.Shortcut != "work" {
		t.Errorf("Expected shortcut 'work', got '%s'", updated.Shortcut)
	}

	// Should fail with conflicting shortcut
	err = store.SetProjectShortcut(project2.ID, "work")
	if err == nil {
		t.Error("Should fail with conflicting shortcut")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Errorf("Expected 'already in use' error, got: %v", err)
	}

	// Should fail with invalid shortcut
	err = store.SetProjectShortcut(project1.ID, "invalid!")
	if err == nil {
		t.Error("Should fail with invalid shortcut")
	}
	if !strings.Contains(err.Error(), "invalid shortcut") {
		t.Errorf("Expected 'invalid shortcut' error, got: %v", err)
	}

	// Should fail with too long shortcut
	err = store.SetProjectShortcut(project1.ID, "123456789012345678901")
	if err == nil {
		t.Error("Should fail with too long shortcut")
	}
}

func TestUUIDGeneration(t *testing.T) {
	// Generate multiple UUIDs and verify they're unique and properly formatted
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uuid := generateUUID()

		// Check format: 8-4-4-4-12 characters
		parts := strings.Split(uuid, "-")
		if len(parts) != 5 {
			t.Errorf("UUID should have 5 parts, got %d: %s", len(parts), uuid)
		}
		if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 ||
			len(parts[3]) != 4 || len(parts[4]) != 12 {
			t.Errorf("UUID parts have wrong lengths: %s", uuid)
		}

		// Check uniqueness
		if seen[uuid] {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		seen[uuid] = true
	}
}
