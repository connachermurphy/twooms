package commands

import (
	"fmt"
	"strings"
)

func init() {
	Register(&Command{
		Name:        "/echo",
		Description: "Echo your message",
		Hidden:      true,
		Handler: func(args []string) bool {
			fmt.Println(strings.Join(args, " "))
			return false
		},
	})
}
