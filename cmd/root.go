package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const (
	DefaultPort  = "9001"
	DefaultShell = "sh"
	DefaultIP    = "10.10.13.37"
)

var (
	version = "0.2.2"
)

var rootCmd = &cobra.Command{
	Use:   "revshell [command]",
	Short: "Generate reverse shell commands for various languages and methods.",
	Long: `A simplified reverse shell generator.

Available commands:
  generate    Generate a reverse shell with interactive prompts
  list        List available shell types, methods, etc.
  <shelltype> Direct commands for common shells (bash, powershell, etc.)

Examples:
  revshell bash 10.10.14.20 4444          # Quick bash reverse shell
  revshell powershell 10.10.14.20         # Quick powershell, default port
  revshell generate                       # Interactive mode
  revshell list types                     # List available shell types

*** Use at your own risk. Keep it legal. ***
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			runInteractiveMode(cmd, args)
			return
		}
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.SetVersionTemplate(`revshell version {{.Version}}
`)
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.AddGroup(&cobra.Group{
		ID:    "shell",
		Title: "Shell Commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "utility",
		Title: "Utility Commands:",
	})
}

func runInteractiveMode(cmd *cobra.Command, args []string) {
	generateCmd.Run(cmd, args)
}
