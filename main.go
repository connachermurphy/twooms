package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"twooms/commands"
	"twooms/storage"
)

func main() {
	// Initialize storage
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	dbPath := filepath.Join(homeDir, ".twooms.json")
	store, err := storage.NewJSONStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	// Set store for commands to use
	commands.SetStore(store)

	// Start REPL
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Welcome to Twooms! Type /help for available commands.")

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.HasPrefix(input, "/") {
			quit, err := commands.Execute(input)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			if quit {
				break
			}
		} else {
			fmt.Printf("You said: %s\n", input)
		}
	}
}
