package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Welcome to CLI Chat! Type /help for available commands.")

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
			if handleCommand(input) {
				break // quit signal
			}
		} else {
			fmt.Printf("You said: %s\n", input)
		}
	}
}

func handleCommand(input string) bool {
	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/hello":
		fmt.Println("Hello, world!")
	case "/help":
		printHelp()
	case "/quit", "/exit":
		fmt.Println("Goodbye!")
		return true
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", cmd)
	}

	return false
}

func printHelp() {
	fmt.Println(`Available commands:
  /hello  - Say hello
  /help   - Show this help message
  /quit   - Exit the chat`)
}
