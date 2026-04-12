package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	ctype     string
	cmethod   string
	cip       string
	cport     string
	cshell    string
	cencoding string
)

var customCmd = &cobra.Command{
	Use:   "custom <type> [method] [ip] [port] [shell] [encoding]",
	Short: "Generate a custom shell (scriptable)",
	Long: `Generate a custom reverse shell payload.
Only the <type> is strictly required. Any omitted optional arguments 
will gracefully fall back to configuration file settings or sensible defaults
(e.g., your local IP address).
You can mix flags and positional arguments. Positional arguments take precedence.`,
	Example: `  revshell custom bash i 10.10.14.20 4444
  revshell custom powershell base64
  revshell custom bash -p 4444
  revshell custom -t python -i 10.10.10.10 -p 9001`,
	GroupID: "utility",
	Args:    cobra.MaximumNArgs(6),
	Run: func(cmd *cobra.Command, args []string) {
		params := CommandParams{}

		shellType := ctype
		method := cmethod
		ip := cip
		port := cport
		shell := cshell
		encoding := cencoding

		if len(args) > 0 {
			shellType = args[0]
		}
		if len(args) > 1 {
			method = args[1]
		}
		if len(args) > 2 {
			ip = args[2]
		}
		if len(args) > 3 {
			port = args[3]
		}
		if len(args) > 4 {
			shell = args[4]
		}
		if len(args) > 5 {
			encoding = args[5]
		}

		if shellType == "" {
			fmt.Println("Error: shell type is required (e.g. 'revshell custom bash i')")
			return
		}
		if method == "" {
			methods := getMethod(shellType)
			if len(methods) > 0 {
				method = methods[0]
			}
		}
		if ip == "" {
			ips := getIP()
			if len(ips) > 0 {
				ip = ips[0]
			}
		}
		config := readConfigFromFile()
		if port == "" {
			if config.Port != "" {
				port = config.Port
			} else {
				port = DefaultPort
			}
		}
		if shell == "" {
			if config.Shell != "" {
				shell = config.Shell
			} else {
				shell = DefaultShell
			}
		}
		if encoding == "" {
			encoding = "none"
		}

		params.Name = shellType
		params.Method = method
		params.IPAddress = ip
		params.Port = port
		params.Shell = shell
		params.Encoding = encoding

		command := getCommand(params)
		encoded := setEncoding(params.Encoding, command)
		fmt.Println(encoded)
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		case 0:
			// Complete shell types
			return getType(), cobra.ShellCompDirectiveNoFileComp
		case 1:
			// Complete methods
			return getMethod(args[0]), cobra.ShellCompDirectiveNoFileComp
		case 2:
			// Complete IP
			return getIP(), cobra.ShellCompDirectiveNoFileComp
		case 3:
			// Complete common ports
			// We can also grab the port from config if they have one set
			config := readConfigFromFile()
			ports := []string{"9001", "4444", "8080", "1337"}
			if config.Port != "" {
				// Put config port first if it exists
				ports = append([]string{config.Port}, ports...)
			}
			return ports, cobra.ShellCompDirectiveNoFileComp
		case 4:
			// Complete shell
			shells := append([]string(nil), list["shells"]...)
			return shells, cobra.ShellCompDirectiveNoFileComp
		case 5:
			// Complete encoding
			enc := append([]string(nil), list["encodings"]...)
			return enc, cobra.ShellCompDirectiveNoFileComp
		default:
			// No completions for extra arguments
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func init() {
	rootCmd.AddCommand(customCmd)
	customCmd.Flags().StringP("type", "t", "", "Shell type (bash/python/powershell/etc)")
	customCmd.Flags().StringVarP(&cmethod, "method", "m", "", "Method to use")
	customCmd.Flags().StringVarP(&cip, "ip", "i", "", "IP Address to connect back to")
	customCmd.Flags().StringVarP(&cport, "port", "p", "", "Port to connect back to")
	customCmd.Flags().StringVarP(&cshell, "shell", "s", "", "Shell to invoke (bash, /bin/sh, etc.)")
	customCmd.Flags().StringVarP(&cencoding, "encoding", "e", "", "Encoding for the payload (base64, encode, none)")

	// We can register flag completion matching too
	customCmd.RegisterFlagCompletionFunc("type", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return getType(), cobra.ShellCompDirectiveNoFileComp
	})
	customCmd.RegisterFlagCompletionFunc("shell", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return list["shells"], cobra.ShellCompDirectiveNoFileComp
	})
	customCmd.RegisterFlagCompletionFunc("encoding", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return list["encodings"], cobra.ShellCompDirectiveNoFileComp
	})
}
