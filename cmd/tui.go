package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/docker"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/spf13/cobra"
)

func tuiCommand() *cobra.Command {
	return &cobra.Command{Use: "tui", Short: "Open the interactive terminal UI", RunE: runTUI}
}

func runTUI(cmd *cobra.Command, args []string) error {
	ui := newTerminalUI()
	ui.header()
	for {
		choice := ui.menu("What do you want to do?", []string{
			"Expose a Docker container",
			"Create a custom reverse proxy",
			"Show status",
			"Quit",
		})
		switch choice {
		case 0:
			if err := ui.exposeDocker(); err != nil {
				return err
			}
		case 1:
			o := createOptions{}
			if err := promptCreate(&o); err != nil {
				return err
			}
			if err := ui.previewAndRun(o); err != nil {
				return err
			}
		case 2:
			return statusCommand().RunE(cmd, args)
		default:
			return nil
		}
	}
}

type terminalUI struct {
	reader *bufio.Reader
}

func newTerminalUI() terminalUI {
	return terminalUI{reader: bufio.NewReader(os.Stdin)}
}

func (ui terminalUI) header() {
	fmt.Println("\033[1;36m")
	fmt.Println("  _   _ ____   ____")
	fmt.Println(" | \\ | |  _ \\ / ___|")
	fmt.Println(" |  \\| | |_) | |")
	fmt.Println(" | |\\  |  __/| |___")
	fmt.Println(" |_| \\_|_|    \\____|")
	fmt.Println("\033[0m\033[1mNginx Proxy Configurator\033[0m")
	fmt.Println("Secure reverse proxies from your terminal.")
	fmt.Println()
}

func (ui terminalUI) exposeDocker() error {
	if !docker.Installed() {
		return validationError{fmt.Errorf("docker was not found")}
	}
	containers, err := docker.RunningContainers()
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		fmt.Println("No running Docker containers found.")
		return nil
	}
	labels := make([]string, 0, len(containers))
	for _, c := range containers {
		ports := "no ports"
		if c.PortsRaw != "" {
			ports = c.PortsRaw
		}
		labels = append(labels, fmt.Sprintf("%s  \033[2m%s | %s\033[0m", c.Name, c.Image, ports))
	}
	container := containers[ui.menu("Select a container to expose", labels)]
	if len(container.Ports) == 0 {
		return validationError{fmt.Errorf("container %s has no exposed or published ports", container.Name)}
	}
	portLabels := make([]string, 0, len(container.Ports))
	for _, port := range container.Ports {
		portLabels = append(portLabels, port.Label())
	}
	port := container.Ports[ui.menu("Select the backend port", portLabels)]
	o := createOptions{
		hostname:          ui.askRequired("Public hostname, for example app.example.com"),
		backendHost:       port.BackendHost(container.Name),
		backendPort:       port.BackendPort(),
		backendScheme:     "http",
		profile:           "docker",
		clientMaxBodySize: "100M",
		nonInteractive:    true,
	}
	o.backendScheme = ui.askDefault("Backend scheme", o.backendScheme)
	o.websocket = ui.confirm("Enable WebSocket headers?", false)
	o.ssl = ui.confirm("Enable HTTPS?", false)
	if o.ssl {
		o.redirectHTTPS = ui.confirm("Redirect HTTP to HTTPS?", true)
		o.http2 = ui.confirm("Enable HTTP/2?", true)
		o.acme = ui.confirm("Use acme.sh?", true)
		if o.acme {
			o.acmeMethod = ui.askDefault("ACME method (http/dns/standalone/tls-alpn)", "http")
			o.email = ui.askRequired("ACME account email")
			if o.acmeMethod == "dns" {
				o.dnsProvider = ui.askRequired("DNS provider")
			}
		} else {
			o.certPath = ui.askRequired("Fullchain path")
			o.keyPath = ui.askRequired("Private key path")
		}
	}
	o.accessLog = ui.confirm("Enable per-site access log?", true)
	o.errorLog = ui.confirm("Enable per-site error log?", true)
	if !port.Published {
		fmt.Println("\033[33mNote:\033[0m the selected port is not published on the host. Nginx must be able to resolve and reach the container name.")
	}
	return ui.previewAndRun(o)
}

func (ui terminalUI) previewAndRun(o createOptions) error {
	site, err := buildSite(o)
	if err != nil {
		return validationError{err}
	}
	content, err := renderer.RenderSite(site)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("\033[1mPlanned site\033[0m")
	fmt.Printf("  Hostname: %s\n", site.Hostname)
	fmt.Printf("  Backend:  %s\n", site.BackendURL())
	fmt.Printf("  Config:   %s\n", site.ConfigPath)
	fmt.Printf("  SSL:      %v\n", site.SSL)
	fmt.Println()
	if ui.confirm("Show rendered Nginx config?", false) {
		fmt.Println()
		fmt.Println(content)
	}
	if !ui.confirm("Create this reverse proxy now?", true) {
		fmt.Println("No changes were made.")
		return nil
	}
	return executeCreate(o)
}

func (ui terminalUI) menu(title string, options []string) int {
	for {
		fmt.Println("\033[1m" + title + "\033[0m")
		for i, option := range options {
			fmt.Printf("  \033[36m%d\033[0m) %s\n", i+1, option)
		}
		fmt.Print("Select: ")
		text, _ := ui.reader.ReadString('\n')
		value, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && value >= 1 && value <= len(options) {
			fmt.Println()
			return value - 1
		}
		fmt.Println("Please enter a valid number.")
	}
}

func (ui terminalUI) askRequired(label string) string {
	for {
		value := ui.askDefault(label, "")
		if strings.TrimSpace(value) != "" {
			return value
		}
		fmt.Println("This value is required.")
	}
}

func (ui terminalUI) askDefault(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	text, _ := ui.reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return def
	}
	return text
}

func (ui terminalUI) confirm(label string, def bool) bool {
	suffix := " [Y/n]: "
	if !def {
		suffix = " [y/N]: "
	}
	fmt.Print(label + suffix)
	text, _ := ui.reader.ReadString('\n')
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return def
	}
	return text == "y" || text == "yes" || text == "ja" || text == "true"
}
