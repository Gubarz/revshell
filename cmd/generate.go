package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Short:   "Generate a reverse shell with interactive TUI",
	GroupID: "utility",
	Run: func(cmd *cobra.Command, args []string) {
		params, ok := runTUI()
		if !ok {
			return
		}
		command := getCommand(params)
		encoded := setEncoding(params.Encoding, command)
		fmt.Println(encoded)
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
