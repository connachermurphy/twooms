package commands

import fmt

func init() {
	Register(&Command{
		Name:        /echo,
		Description: Echo your message,
		Handler: func(args []string) bool {
			fmt.Println(strings.Join(args,  ))
			return false
		},
	})
}
