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
