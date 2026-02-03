package commands

import (
	"testing"
)

func TestGenerateToolDefinitions(t *testing.T) {
	tools := GenerateToolDefinitions()

	// Expected tool names (commands that are NOT hidden)
	expectedTools := map[string]bool{
		"project":    true,
		"projects":   true,
		"delproject": true,
		"shortcut":   true,
		"task":       true,
		"tasks":      true,
		"done":       true,
		"undone":     true,
		"deltask":    true,
		"due":        true,
		"duration":   true,
		"today":      true,
		"tomorrow":   true,
		"week":       true,
	}

	// Commands that should NOT be generated (hidden)
	hiddenTools := map[string]bool{
		"quit": true,
		"exit": true,
		"help": true,
		"echo": true,
		"chat": true,
	}

	// Check that expected tools are present
	foundTools := make(map[string]bool)
	for _, tool := range tools {
		foundTools[tool.Name] = true

		// Verify hidden tools are not present
		if hiddenTools[tool.Name] {
			t.Errorf("Hidden tool %q should not be in generated tools", tool.Name)
		}
	}

	// Verify all expected tools are present
	for name := range expectedTools {
		if !foundTools[name] {
			t.Errorf("Expected tool %q not found in generated tools", name)
		}
	}

	// Verify tool count
	if len(tools) != len(expectedTools) {
		t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
	}

	// Print tools for verification
	t.Logf("Generated %d tools:", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s: %s", tool.Name, tool.Description)
		if tool.Parameters != nil {
			for name, schema := range tool.Parameters.Properties {
				t.Logf("      param: %s (%s)", name, schema.Description)
			}
		}
	}
}

func TestToolParameterDefinitions(t *testing.T) {
	tools := GenerateToolDefinitions()

	// Create a map for easy lookup
	toolMap := make(map[string]*struct {
		params   []string
		required []string
	})

	for _, tool := range tools {
		entry := &struct {
			params   []string
			required []string
		}{}
		if tool.Parameters != nil {
			for name := range tool.Parameters.Properties {
				entry.params = append(entry.params, name)
			}
			entry.required = tool.Parameters.Required
		}
		toolMap[tool.Name] = entry
	}

	// Test specific commands have correct parameters
	testCases := []struct {
		name           string
		expectedParams []string
	}{
		{"project", []string{"name"}},
		{"projects", nil}, // no params
		{"delproject", []string{"project_id"}},
		{"task", []string{"project_id", "task_name"}},
		{"tasks", []string{"project_id"}},
		{"done", []string{"task_id"}},
		{"undone", []string{"task_id"}},
		{"deltask", []string{"task_id"}},
		{"due", []string{"task_id", "date"}},
		{"duration", []string{"task_id", "duration"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			entry, exists := toolMap[tc.name]
			if !exists {
				t.Fatalf("Tool %q not found", tc.name)
			}

			if len(tc.expectedParams) == 0 {
				if len(entry.params) != 0 {
					t.Errorf("Tool %q: expected no params, got %v", tc.name, entry.params)
				}
				return
			}

			if len(entry.params) != len(tc.expectedParams) {
				t.Errorf("Tool %q: expected %d params, got %d", tc.name, len(tc.expectedParams), len(entry.params))
			}

			// Check each expected param exists
			paramSet := make(map[string]bool)
			for _, p := range entry.params {
				paramSet[p] = true
			}
			for _, expected := range tc.expectedParams {
				if !paramSet[expected] {
					t.Errorf("Tool %q: missing expected param %q", tc.name, expected)
				}
			}
		})
	}
}
