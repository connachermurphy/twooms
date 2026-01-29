package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"
	"github.com/joho/godotenv"

	"twooms/commands"
	"twooms/llm"
	"twooms/storage"
)

func main() {
	// Load .env file if present (errors ignored - file is optional)
	godotenv.Load()

	// Initialize storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	// Also try loading from ~/.twooms.env
	godotenv.Load(filepath.Join(homeDir, ".twooms.env"))

	dbPath := filepath.Join(homeDir, ".twooms.json")
	store, err := storage.NewJSONStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Set store for commands to use
	commands.SetStore(store)

	// Initialize LLM client (optional)
	ctx := context.Background()
	llmClient, err := llm.NewGeminiClient(ctx)
	if err != nil {
		if err == llm.ErrMissingAPIKey {
			fmt.Fprintf(os.Stderr, "Warning: %v (LLM features disabled)\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error initializing LLM client: %v\n", err)
			os.Exit(1)
		}
	} else {
		commands.SetLLMClient(llmClient)
		defer llmClient.Close()
	}

	// Start REPL with readline support
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing readline: %v\n", err)
		os.Exit(1)
	}
	defer rl.Close()

	fmt.Println("Welcome to Twooms! Type /help for available commands.")

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			break
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Default to /chat if no slash command specified
		if !strings.HasPrefix(input, "/") {
			input = "/chat " + input
		}

		quit, err := commands.Execute(input)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
		if quit {
			break
		}
	}
}
