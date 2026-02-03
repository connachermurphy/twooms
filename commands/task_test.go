package commands

import (
	"os"
	"path/filepath"
	"regexp"
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

// extractShortcut extracts the shortcut from project creation output
func extractShortcut(output string) string {
	re := regexp.MustCompile(`\(shortcut: ([a-f0-9]+)\)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// extractTaskID extracts the task ID from task creation output
func extractTaskID(output string) string {
	re := regexp.MustCompile(`\(ID: ([a-f0-9]+)\)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func TestProjectCommands(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a project
	output := captureCommandOutput(t, "/project My Test Project")
	if !strings.Contains(output, "Created project: My Test Project") {
		t.Errorf("Expected project creation message, got: %s", output)
	}
	if !strings.Contains(output, "shortcut:") {
		t.Errorf("Expected shortcut in output, got: %s", output)
	}
	shortcut1 := extractShortcut(output)
	if shortcut1 == "" {
		t.Fatalf("Could not extract shortcut from: %s", output)
	}

	// List projects
	output = captureCommandOutput(t, "/projects")
	if !strings.Contains(output, "My Test Project") {
		t.Errorf("Expected project in list, got: %s", output)
	}
	if !strings.Contains(output, "["+shortcut1+"]") {
		t.Errorf("Expected shortcut in list, got: %s", output)
	}

	// Create another project
	output = captureCommandOutput(t, "/project Second Project")
	shortcut2 := extractShortcut(output)
	if shortcut2 == "" {
		t.Fatalf("Could not extract second shortcut from: %s", output)
	}
	if shortcut1 == shortcut2 {
		t.Errorf("Shortcuts should be unique, got: %s and %s", shortcut1, shortcut2)
	}

	// Delete first project using shortcut
	output = captureCommandOutput(t, "/delproject "+shortcut1)
	if !strings.Contains(output, "Deleted project: My Test Project") {
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
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)

	// Create a task using the shortcut
	output = captureCommandOutput(t, "/task "+shortcut+" Buy groceries")
	if !strings.Contains(output, "Created task: Buy groceries") {
		t.Errorf("Expected task creation message, got: %s", output)
	}
	taskID := extractTaskID(output)
	if taskID == "" {
		t.Fatalf("Could not extract task ID from: %s", output)
	}

	// List tasks using shortcut
	output = captureCommandOutput(t, "/tasks "+shortcut)
	if !strings.Contains(output, "Buy groceries") {
		t.Errorf("Expected task in list, got: %s", output)
	}
	if !strings.Contains(output, "[ ]") {
		t.Errorf("Expected unchecked status, got: %s", output)
	}

	// Mark as done using task ID prefix
	output = captureCommandOutput(t, "/done "+taskID)
	if !strings.Contains(output, "Marked task Buy groceries as done") {
		t.Errorf("Expected done message, got: %s", output)
	}

	// Verify done status in list
	output = captureCommandOutput(t, "/tasks "+shortcut)
	if !strings.Contains(output, "[✓]") {
		t.Errorf("Expected checked status, got: %s", output)
	}

	// Mark as undone
	output = captureCommandOutput(t, "/undone "+taskID)
	if !strings.Contains(output, "Marked task Buy groceries as not done") {
		t.Errorf("Expected undone message, got: %s", output)
	}

	// Delete task
	output = captureCommandOutput(t, "/deltask "+taskID)
	if !strings.Contains(output, "Deleted task: Buy groceries") {
		t.Errorf("Expected deletion message, got: %s", output)
	}

	// Verify task is gone
	output = captureCommandOutput(t, "/tasks "+shortcut)
	if strings.Contains(output, "Buy groceries") {
		t.Errorf("Deleted task should not appear in list, got: %s", output)
	}
}

func TestDueDateCommand(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup: create project and task
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Important task")
	taskID := extractTaskID(output)

	// Set due date
	output = captureCommandOutput(t, "/due "+taskID+" 2025-12-31")
	if !strings.Contains(output, "Set due date for task Important task to 2025-12-31") {
		t.Errorf("Expected due date set message, got: %s", output)
	}

	// Verify due date in task list
	output = captureCommandOutput(t, "/tasks "+shortcut)
	if !strings.Contains(output, "due 2025-12-31") {
		t.Errorf("Expected due date in task list, got: %s", output)
	}

	// Clear due date
	output = captureCommandOutput(t, "/due "+taskID+" none")
	if !strings.Contains(output, "Cleared due date for task Important task") {
		t.Errorf("Expected due date cleared message, got: %s", output)
	}

	// Verify due date is cleared (no parentheses means no extra info)
	output = captureCommandOutput(t, "/tasks "+shortcut)
	if strings.Contains(output, "due 2025-12-31") {
		t.Errorf("Due date should be cleared, got: %s", output)
	}
}

func TestDueDateInvalidFormat(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Test task")
	taskID := extractTaskID(output)

	// Try invalid date format
	output = captureCommandOutput(t, "/due "+taskID+" 12-31-2025")
	if !strings.Contains(output, "Invalid date format") {
		t.Errorf("Expected invalid date format error, got: %s", output)
	}

	// Try another invalid format
	output = captureCommandOutput(t, "/due "+taskID+" tomorrow")
	if !strings.Contains(output, "Invalid date format") {
		t.Errorf("Expected invalid date format error, got: %s", output)
	}
}

func TestDurationCommand(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Quick task")
	taskID := extractTaskID(output)

	// Test all valid durations
	validDurations := []string{"15m", "30m", "1h", "2h", "4h"}
	for _, dur := range validDurations {
		output := captureCommandOutput(t, "/duration "+taskID+" "+dur)
		if !strings.Contains(output, "Set duration for task Quick task to "+dur) {
			t.Errorf("Expected duration set message for %s, got: %s", dur, output)
		}

		// Verify in task list
		output = captureCommandOutput(t, "/tasks "+shortcut)
		if !strings.Contains(output, "("+dur) {
			t.Errorf("Expected duration %s in task list, got: %s", dur, output)
		}
	}
}

func TestDurationInvalid(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Test task")
	taskID := extractTaskID(output)

	// Try invalid durations
	invalidDurations := []string{"10m", "45m", "3h", "1d", "invalid"}
	for _, dur := range invalidDurations {
		output := captureCommandOutput(t, "/duration "+taskID+" "+dur)
		if !strings.Contains(output, "Invalid duration") {
			t.Errorf("Expected invalid duration error for %s, got: %s", dur, output)
		}
	}
}

func TestDueDateAndDurationTogether(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Setup
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Full task")
	taskID := extractTaskID(output)

	// Set both duration and due date
	captureCommandOutput(t, "/duration "+taskID+" 2h")
	captureCommandOutput(t, "/due "+taskID+" 2025-06-15")

	// Verify both appear in task list
	output = captureCommandOutput(t, "/tasks "+shortcut)
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
	output := captureCommandOutput(t, "/task nonexistent Orphan task")
	if !strings.Contains(output, "project not found") {
		t.Errorf("Expected project not found error, got: %s", output)
	}
}

func TestCommandUsageMessages(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a project and task for tests that need valid IDs
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Test task")
	taskID := extractTaskID(output)

	tests := []struct {
		command  string
		expected string
	}{
		{"/project", "Usage: /project <name>"},
		{"/delproject", "Usage: /delproject <project-id>"},
		{"/task", "Usage: /task <project-id> <task name>"},
		{"/task " + shortcut, "Usage: /task <project-id> <task name>"},
		{"/tasks", "Usage: /tasks <project-id>"},
		{"/done", "Usage: /done <task-id>"},
		{"/undone", "Usage: /undone <task-id>"},
		{"/deltask", "Usage: /deltask <task-id>"},
		{"/due", "Usage: /due <task-id>"},
		{"/due " + taskID, "Usage: /due <task-id>"},
		{"/duration", "Usage: /duration <task-id>"},
		{"/duration " + taskID, "Usage: /duration <task-id>"},
		{"/chat", "Usage: /chat <message>"},
		{"/shortcut", "Usage: /shortcut <project-id> <new-shortcut>"},
		{"/shortcut " + shortcut, "Usage: /shortcut <project-id> <new-shortcut>"},
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
	output := captureCommandOutput(t, "/project Project to delete")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Task 1")
	taskID1 := extractTaskID(output)
	captureCommandOutput(t, "/task "+shortcut+" Task 2")

	// Verify tasks exist
	output = captureCommandOutput(t, "/tasks "+shortcut)
	if !strings.Contains(output, "Task 1") || !strings.Contains(output, "Task 2") {
		t.Errorf("Expected tasks to exist before deletion, got: %s", output)
	}

	// Delete project
	captureCommandOutput(t, "/delproject "+shortcut)

	// Create a new project
	output = captureCommandOutput(t, "/project New Project")

	// The old task IDs should not exist
	output = captureCommandOutput(t, "/done "+taskID1)
	if !strings.Contains(output, "task not found") {
		t.Errorf("Expected task not found after project deletion, got: %s", output)
	}
}

func TestEmptyProjectTaskList(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create empty project
	output := captureCommandOutput(t, "/project Empty Project")
	shortcut := extractShortcut(output)

	// List tasks
	output = captureCommandOutput(t, "/tasks "+shortcut)
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

func TestShortcutCommand(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a project
	output := captureCommandOutput(t, "/project My Project")
	shortcut := extractShortcut(output)

	// Set a custom shortcut
	output = captureCommandOutput(t, "/shortcut "+shortcut+" work")
	if !strings.Contains(output, "Set shortcut for My Project to: work") {
		t.Errorf("Expected shortcut set message, got: %s", output)
	}

	// Verify new shortcut appears in list
	output = captureCommandOutput(t, "/projects")
	if !strings.Contains(output, "[work]") {
		t.Errorf("Expected new shortcut in list, got: %s", output)
	}

	// Create a task using new shortcut
	output = captureCommandOutput(t, "/task work Test task")
	if !strings.Contains(output, "Created task: Test task") {
		t.Errorf("Expected task creation with shortcut, got: %s", output)
	}

	// List tasks using new shortcut
	output = captureCommandOutput(t, "/tasks work")
	if !strings.Contains(output, "Test task") {
		t.Errorf("Expected task in list using shortcut, got: %s", output)
	}
}

func TestShortcutValidation(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a project
	output := captureCommandOutput(t, "/project My Project")
	shortcut := extractShortcut(output)

	// Try invalid shortcuts
	invalidShortcuts := []string{
		"abc!",                         // special char
		"123456789012345678901",        // too long (21 chars)
		"test@name",                    // @ symbol
	}
	for _, invalid := range invalidShortcuts {
		output = captureCommandOutput(t, "/shortcut "+shortcut+" "+invalid)
		if !strings.Contains(output, "invalid shortcut") {
			t.Errorf("Expected invalid shortcut error for %q, got: %s", invalid, output)
		}
	}

	// Valid shortcuts should work
	validShortcuts := []string{
		"a",           // single char
		"abc123",      // alphanumeric
		"my-project",  // with hyphen
		"12345678901234567890", // 20 chars (max)
	}
	for _, valid := range validShortcuts {
		output = captureCommandOutput(t, "/shortcut "+shortcut+" "+valid)
		if strings.Contains(output, "Error") {
			t.Errorf("Valid shortcut %q should work, got: %s", valid, output)
		}
		shortcut = valid // Update shortcut for next iteration
	}
}

func TestShortcutConflict(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create two projects
	output := captureCommandOutput(t, "/project First Project")
	shortcut1 := extractShortcut(output)
	output = captureCommandOutput(t, "/project Second Project")
	shortcut2 := extractShortcut(output)

	// Set first project shortcut
	captureCommandOutput(t, "/shortcut "+shortcut1+" work")

	// Try to set same shortcut on second project
	output = captureCommandOutput(t, "/shortcut "+shortcut2+" work")
	if !strings.Contains(output, "shortcut already in use") {
		t.Errorf("Expected shortcut conflict error, got: %s", output)
	}
}

func TestUUIDPrefixMatching(t *testing.T) {
	cleanup := setupTestStore(t)
	defer cleanup()

	// Create a project and task
	output := captureCommandOutput(t, "/project Test Project")
	shortcut := extractShortcut(output)
	output = captureCommandOutput(t, "/task "+shortcut+" Test task")
	taskID := extractTaskID(output)

	// Task operations should work with the 8-char prefix
	output = captureCommandOutput(t, "/done "+taskID)
	if !strings.Contains(output, "Marked task Test task as done") {
		t.Errorf("Expected done message with 8-char prefix, got: %s", output)
	}

	// 6-char prefix should also work (minimum)
	output = captureCommandOutput(t, "/undone "+taskID[:6])
	if !strings.Contains(output, "Marked task Test task as not done") {
		t.Errorf("Expected undone message with 6-char prefix, got: %s", output)
	}

	// 5-char prefix should fail (too short)
	output = captureCommandOutput(t, "/done "+taskID[:5])
	if !strings.Contains(output, "task not found") {
		t.Errorf("Expected task not found with 5-char prefix, got: %s", output)
	}
}
