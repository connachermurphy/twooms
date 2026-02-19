# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Twooms is a chat-based task manager that runs in the terminal. It's a Go CLI application with a command-based architecture.

## Public-Facing Content

When writing GitHub issues, PRs, commit messages, and any other public-facing content, always remove information specific to the user's own projects (project names, task names, personal data). Keep examples generic.

## Building and Running

Build the project:
```bash
go build
```

Run the compiled binary:
```bash
./twooms
```

Run directly without building:
```bash
go run main.go
```

## Git Workflow

**Always work on a feature branch**, never commit directly to `main`. Follow this workflow:

1. Create a branch for the issue: `git checkout -b <issue-number>-<short-description>`
2. Make changes and commit
3. Push and create a PR: `git push -u origin <branch-name>` then `gh pr create`
4. After merge, switch back to main and clean up: `git checkout main && git branch -d <branch-name> && git pull`

## Architecture

The application uses a **command registry pattern** where commands are automatically registered via `init()` functions in individual command files.

### Command System

- **Registry**: The `commands/` package contains the command registration system
- **Command Files**: Each command lives in its own file:
  - `commands/commands.go` - Registry, `Execute()`, `SetStore()`/`GetStore()`, `SetLLMClient()`/`GetLLMClient()`
  - `commands/help.go` - `/help` command
  - `commands/quit.go` - `/quit` and `/exit` commands
  - `commands/echo.go` - `/echo` command
  - `commands/project.go` - `/project`, `/projects`, `/delproject` commands
  - `commands/task.go` - `/task`, `/tasks`, `/done`, `/undone`, `/deltask`, `/due`, `/duration` commands
  - `commands/chat.go` - `/chat` command (LLM-powered assistant)
- **Auto-registration**: Commands use `init()` functions to call `Register(&Command{...})` with their name, description, and handler
- **Handler Return**: Command handlers return `bool` - `true` signals the application should quit, `false` continues the REPL loop
- **Execute Return**: The `Execute(input)` function returns `(bool, error)` - the bool from the handler, plus any execution errors

### Adding New Commands

To add a new command, create a new file in `commands/` following this pattern:
```go
package commands

import "fmt"

func init() {
    Register(&Command{
        Name:        "/yourcommand",
        Description: "Description here",
        Handler: func(args []string) bool {
            // Implementation
            // Use GetStore() to access storage operations
            return false // or true to quit
        },
    })
}
```

The command will be automatically registered when the application starts.

**Command aliasing**: You can register multiple commands in a single file to create aliases (see `quit.go` which registers both `/quit` and `/exit`).

**Accessing storage**: Use `GetStore()` to access the storage interface for database operations:
```go
// Example: creating a project
project, err := GetStore().CreateProject(name)
if err != nil {
    fmt.Printf("Error: %v\n", err)
    return false
}
```

### Available Commands

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/quit`, `/exit` | Exit Twooms |
| `/echo <message>` | Echo your message |
| `/project <name>` | Create a new project |
| `/projects` | List all projects |
| `/delproject <project-id>` | Delete a project and its tasks |
| `/task <project-id> <name>` | Add a task to a project |
| `/tasks <project-id>` | List tasks in a project |
| `/done <task-id>` | Mark a task as done |
| `/undone <task-id>` | Mark a task as not done |
| `/deltask <task-id>` | Delete a task |
| `/due <task-id> <YYYY-MM-DD\|none>` | Set or clear a task's due date |
| `/duration <task-id> <duration>` | Set a task's duration (15m, 30m, 1h, 2h, 4h) |
| `/chat <message>` | Chat with the AI assistant |

### Main Loop

The `main.go` file contains a REPL (Read-Eval-Print Loop) that:
1. Reads user input
2. Checks if input starts with "/" to identify commands
3. Delegates command handling to the commands package
4. Exits when a command handler returns `true`

### LLM Integration

The application integrates with Google's Gemini API to provide an AI-powered chat assistant.

#### Architecture

- **`llm/client.go`**: Defines the `Client` interface and error types
- **`llm/gemini.go`**: Gemini API implementation with tool calling support
- **`llm/types.go`**: Response and configuration types

#### Configuration

Set the `GEMINI_API_KEY` environment variable to enable LLM features:
```bash
export GEMINI_API_KEY=your-api-key
```

#### Tool Calling

The `/chat` command uses Gemini's function calling to execute Twooms commands. When you ask the assistant to perform tasks (e.g., "create a project called Work"), it:
1. Interprets your natural language request
2. Calls the appropriate command(s) via tool definitions
3. Returns a human-friendly response

Tool definitions are auto-generated from registered commands using `GenerateToolDefinitions()` in `commands/commands.go`. Commands with `Hidden: true` are excluded from tool generation.

### Storage Architecture

The application uses a **storage interface pattern** to support swappable backends (currently JSON, designed for future bbolt migration).

#### Current Structure

- **`storage/store.go`**: Defines the `Store` interface with all storage operations
- **`storage/types.go`**: Core data structures (`Project`, `Task`, `Duration`)
- **`storage/json.go`**: JSON file implementation (currently active)
- **Storage location**: `~/.twooms.json`

#### Task Fields

The `Task` struct includes:
- `ID`, `ProjectID`, `Name`, `Done`, `CreatedAt` - core fields
- `DueDate` - optional due date (`*time.Time`)
- `Duration` - estimated time to complete (valid values: `15m`, `30m`, `1h`, `2h`, `4h`)

#### Migrating to bbolt

To add bbolt support while keeping JSON as fallback:

**Step 1: Implement bbolt backend**

Create `storage/bbolt.go`:
```go
package storage

import "go.etcd.io/bbolt"

type BBoltStore struct {
    db *bolt.DB
}

func NewBBoltStore(filename string) (*BBoltStore, error) {
    db, err := bolt.Open(filename, 0600, nil)
    if err != nil {
        return nil, err
    }

    // Initialize buckets
    err = db.Update(func(tx *bolt.Tx) error {
        if _, err := tx.CreateBucketIfNotExists([]byte("projects")); err != nil {
            return err
        }
        if _, err := tx.CreateBucketIfNotExists([]byte("tasks")); err != nil {
            return err
        }
        return nil
    })

    if err != nil {
        return nil, err
    }

    return &BBoltStore{db: db}, nil
}

// Implement all Store interface methods
// Use json.Marshal/Unmarshal to serialize Project/Task structs
// Store with ID as key, serialized struct as value
```

Key implementation details:
- Use two top-level buckets: `"projects"` and `"tasks"`
- Serialize structs with `json.Marshal()` before storing
- Use ID fields as keys (e.g., `[]byte("proj-1")`)
- For `ListTasks(projectID)`, iterate all tasks and filter by `ProjectID` field
- Implement ID generation using a separate `"meta"` bucket with counters

**Step 2: Switch in main.go**

Replace the storage initialization:
```go
// OLD (JSON):
dbPath := filepath.Join(homeDir, ".twooms.json")
store, err := storage.NewJSONStore(dbPath)

// NEW (bbolt):
dbPath := filepath.Join(homeDir, ".twooms.db")
store, err := storage.NewBBoltStore(dbPath)
```

That's it! All commands continue working unchanged because they use the `Store` interface.

**Migration Tools**: If users have existing JSON data, create a migration command that:
1. Opens both stores
2. Reads all projects/tasks from JSON
3. Writes them to bbolt
4. Renames `.twooms.json` to `.twooms.json.backup`
