package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"twooms/storage"
)

// setupTestStore creates a temporary store for testing
func setupTestStore(t *testing.T) func() {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.json")

	store, err := storage.NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}

	SetStore(store)

	return func() {
		store.Close()
	}
}

// captureCommandOutput runs a command and captures its stdout
func captureCommandOutput(t *testing.T, input string) string {
	t.Helper()

	output := captureOutput(func() {
		Execute(input)
	})

	return output
}

func TestProjectCommands(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a project
	output := captureCommandOutput(t, "/project My Test Project")
	if !strings.Contains(output, "Created project: My Test Project") {
		t.Errorf("Expected project creation message, got: %s", output)
	}
	if !strings.Contains(output, "proj-1") {
		t.Errorf("Expected project ID proj-1, got: %s", output)
	}

	// List projects
	output = captureCommandOutput(t, "/projects")
	if !strings.Contains(output, "My Test Project") {
		t.Errorf("Expected project in list, got: %s", output)
	}
	if !strings.Contains(output, "proj-1") {
		t.Errorf("Expected project ID in list, got: %s", output)
	}

	// Create another project
	output = captureCommandOutput(t, "/project Second Project")
	if !strings.Contains(output, "proj-2") {
		t.Errorf("Expected project ID proj-2, got: %s", output)
	}

	// Delete first project
	output = captureCommandOutput(t, "/delproject proj-1")
	if !strings.Contains(output, "Deleted project: proj-1") {
		t.Errorf("Expected deletion message, got: %s", output)
	}

	// Verify project is gone
	output = captureCommandOutput(t, "/projects")
	if strings.Contains(output, "My Test Project") {
		t.Errorf("Deleted project should not appear in list, got: %s", output)
	}
	if !strings.Contains(output, "Second Project") {
		t.Errorf("Second project should still exist, got: %s", output)
	}
}

func TestTaskCommands(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a project first
	captureCommandOutput(t, "/project Test Project")

	// Create a task
	output := captureCommandOutput(t, "/task proj-1 Buy groceries")
	if !strings.Contains(output, "Created task: Buy groceries") {
		t.Errorf("Expected task creation message, got: %s", output)
	}
	if !strings.Contains(output, "task-1") {
		t.Errorf("Expected task ID task-1, got: %s", output)
	}

	// List tasks
	output = captureCommandOutput(t, "/tasks proj-1")
	if !strings.Contains(output, "Buy groceries") {
		t.Errorf("Expected task in list, got: %s", output)
	}
	if !strings.Contains(output, "[ ]") {
		t.Errorf("Expected unchecked status, got: %s", output)
	}

	// Mark as done
	output = captureCommandOutput(t, "/done task-1")
	if !strings.Contains(output, "Marked task task-1 as done") {
		t.Errorf("Expected done message, got: %s", output)
	}

	// Verify done status in list
	output = captureCommandOutput(t, "/tasks proj-1")
	if !strings.Contains(output, "[✓]") {
		t.Errorf("Expected checked status, got: %s", output)
	}

	// Mark as undone
	output = captureCommandOutput(t, "/undone task-1")
	if !strings.Contains(output, "Marked task task-1 as not done") {
		t.Errorf("Expected undone message, got: %s", output)
	}

	// Delete task
	output = captureCommandOutput(t, "/deltask task-1")
	if !strings.Contains(output, "Deleted task: task-1") {
		t.Errorf("Expected deletion message, got: %s", output)
	}

	// Verify task is gone
	output = captureCommandOutput(t, "/tasks proj-1")
	if strings.Contains(output, "Buy groceries") {
		t.Errorf("Deleted task should not appear in list, got: %s", output)
	}
}

func TestDueDateCommand(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup: create project and task
	captureCommandOutput(t, "/project Test Project")
	captureCommandOutput(t, "/task proj-1 Important task")

	// Set due date
	output := captureCommandOutput(t, "/due task-1 2025-12-31")
	if !strings.Contains(output, "Set due date for task task-1 to 2025-12-31") {
		t.Errorf("Expected due date set message, got: %s", output)
	}

	// Verify due date in task list
	output = captureCommandOutput(t, "/tasks proj-1")
	if !strings.Contains(output, "due 2025-12-31") {
		t.Errorf("Expected due date in task list, got: %s", output)
	}

	// Clear due date
	output = captureCommandOutput(t, "/due task-1 none")
	if !strings.Contains(output, "Cleared due date for task task-1") {
		t.Errorf("Expected due date cleared message, got: %s", output)
	}

	// Verify due date is cleared (no parentheses means no extra info)
	output = captureCommandOutput(t, "/tasks proj-1")
	if strings.Contains(output, "due 2025-12-31") {
		t.Errorf("Due date should be cleared, got: %s", output)
	}
}

func TestDueDateInvalidFormat(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	captureCommandOutput(t, "/project Test Project")
	captureCommandOutput(t, "/task proj-1 Test task")

	// Try invalid date format
	output := captureCommandOutput(t, "/due task-1 12-31-2025")
	if !strings.Contains(output, "Invalid date format") {
		t.Errorf("Expected invalid date format error, got: %s", output)
	}

	// Try another invalid format
	output = captureCommandOutput(t, "/due task-1 tomorrow")
	if !strings.Contains(output, "Invalid date format") {
		t.Errorf("Expected invalid date format error, got: %s", output)
	}
}

func TestDurationCommand(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	captureCommandOutput(t, "/project Test Project")
	captureCommandOutput(t, "/task proj-1 Quick task")

	// Test all valid durations
	validDurations := []string{"15m", "30m", "1h", "2h", "4h"}
	for _, dur := range validDurations {
		output := captureCommandOutput(t, "/duration task-1 "+dur)
		if !strings.Contains(output, "Set duration for task task-1 to "+dur) {
			t.Errorf("Expected duration set message for %s, got: %s", dur, output)
		}

		// Verify in task list
		output = captureCommandOutput(t, "/tasks proj-1")
		if !strings.Contains(output, "("+dur) {
			t.Errorf("Expected duration %s in task list, got: %s", dur, output)
		}
	}
}

func TestDurationInvalid(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	captureCommandOutput(t, "/project Test Project")
	captureCommandOutput(t, "/task proj-1 Test task")

	// Try invalid durations
	invalidDurations := []string{"10m", "45m", "3h", "1d", "invalid"}
	for _, dur := range invalidDurations {
		output := captureCommandOutput(t, "/duration task-1 "+dur)
		if !strings.Contains(output, "Invalid duration") {
			t.Errorf("Expected invalid duration error for %s, got: %s", dur, output)
		}
	}
}

func TestDueDateAndDurationTogether(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	captureCommandOutput(t, "/project Test Project")
	captureCommandOutput(t, "/task proj-1 Full task")

	// Set both duration and due date
	captureCommandOutput(t, "/duration task-1 2h")
	captureCommandOutput(t, "/due task-1 2025-06-15")

	// Verify both appear in task list
	output := captureCommandOutput(t, "/tasks proj-1")
	if !strings.Contains(output, "2h") {
		t.Errorf("Expected duration in task list, got: %s", output)
	}
	if !strings.Contains(output, "due 2025-06-15") {
		t.Errorf("Expected due date in task list, got: %s", output)
	}
	// Should show as "(2h, due 2025-06-15)"
	if !strings.Contains(output, "(2h, due 2025-06-15)") {
		t.Errorf("Expected combined format (2h, due 2025-06-15), got: %s", output)
	}
}

func TestTaskInNonexistentProject(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Try to create task in nonexistent project
	output := captureCommandOutput(t, "/task proj-999 Orphan task")
	if !strings.Contains(output, "project not found") {
		t.Errorf("Expected project not found error, got: %s", output)
	}
}

func TestCommandUsageMessages(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	tests := []struct {
		command  string
		expected string
	}{
		{"/project", "Usage: /project <name>"},
		{"/delproject", "Usage: /delproject <project-id>"},
		{"/task", "Usage: /task <project-id> <task name>"},
		{"/task proj-1", "Usage: /task <project-id> <task name>"},
		{"/tasks", "Usage: /tasks <project-id>"},
		{"/done", "Usage: /done <task-id>"},
		{"/undone", "Usage: /undone <task-id>"},
		{"/deltask", "Usage: /deltask <task-id>"},
		{"/due", "Usage: /due <task-id>"},
		{"/due task-1", "Usage: /due <task-id>"},
		{"/duration", "Usage: /duration <task-id>"},
		{"/duration task-1", "Usage: /duration <task-id>"},
		{"/chat", "Usage: /chat <message>"},
	}

	for _, tc := range tests {
		t.Run(tc.command, func(t *testing.T) {
			output := captureCommandOutput(t, tc.command)
			if !strings.Contains(output, tc.expected) {
				t.Errorf("Expected usage message %q, got: %s", tc.expected, output)
			}
		})
	}
}

func TestDeleteProjectDeletesTasks(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create project with tasks
	captureCommandOutput(t, "/project Project to delete")
	captureCommandOutput(t, "/task proj-1 Task 1")
	captureCommandOutput(t, "/task proj-1 Task 2")

	// Verify tasks exist
	output := captureCommandOutput(t, "/tasks proj-1")
	if !strings.Contains(output, "Task 1") || !strings.Contains(output, "Task 2") {
		t.Errorf("Expected tasks to exist before deletion, got: %s", output)
	}

	// Delete project
	captureCommandOutput(t, "/delproject proj-1")

	// Create a new project (will get proj-2)
	captureCommandOutput(t, "/project New Project")

	// The old task IDs should not exist
	output = captureCommandOutput(t, "/done task-1")
	if !strings.Contains(output, "task not found") {
		t.Errorf("Expected task not found after project deletion, got: %s", output)
	}
}

func TestEmptyProjectTaskList(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create empty project
	captureCommandOutput(t, "/project Empty Project")

	// List tasks
	output := captureCommandOutput(t, "/tasks proj-1")
	if !strings.Contains(output, "No tasks yet") {
		t.Errorf("Expected no tasks message, got: %s", output)
	}
}

func TestNoProjectsMessage(t *testing.T) {
	// Use a completely fresh temp directory
	tmpDir, err := os.MkdirTemp("", "twooms-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.json")
	store, err := storage.NewJSONStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	defer store.Close()

	SetStore(store)

	output := captureCommandOutput(t, "/projects")
	if !strings.Contains(output, "No projects yet") {
		t.Errorf("Expected no projects message, got: %s", output)
	}
}
