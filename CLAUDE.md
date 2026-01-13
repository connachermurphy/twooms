# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Twooms is a chat-based task manager that runs in the terminal. It's a Go CLI application with a command-based architecture.

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

## Architecture

The application uses a **command registry pattern** where commands are automatically registered via `init()` functions in individual command files.

### Command System

- **Registry**: The `commands/` package contains the command registration system
- **Command Files**: Each command lives in its own file (e.g., `commands/quit.go`, `commands/echo.go`)
- **Auto-registration**: Commands use `init()` functions to call `Register(&Command{...})` with their name, description, and handler
- **Handler Return**: Command handlers return `bool` - `true` signals the application should quit, `false` continues the REPL loop

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
            return false // or true to quit
        },
    })
}
```

The command will be automatically registered when the application starts.

### Main Loop

The `main.go` file contains a REPL (Read-Eval-Print Loop) that:
1. Reads user input
2. Checks if input starts with "/" to identify commands
3. Delegates command handling to the commands package
4. Exits when a command handler returns `true`

### Storage Architecture

The application uses a **storage interface pattern** to support swappable backends (currently JSON, designed for future bbolt migration).

#### Current Structure

- **`storage/store.go`**: Defines the `Store` interface with all storage operations
- **`storage/types.go`**: Core data structures (`Project`, `Task`)
- **`storage/json.go`**: JSON file implementation (currently active)
- **Storage location**: `~/.twooms.json`

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
