package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/docker"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func tuiCommand() *cobra.Command {
	return &cobra.Command{Use: "tui", Short: "Open the interactive terminal UI", RunE: runTUI}
}

func runTUI(cmd *cobra.Command, args []string) error {
	ui := newTerminalUI()
	ui.header()
	for {
		ui.dashboard()
		choice := ui.menu("What do you want to do?", []string{
			"Expose a Docker container",
			"Create a custom reverse proxy",
			"List managed sites",
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
			if err := listCommand().RunE(cmd, args); err != nil {
				return err
			}
			ui.pause()
		case 3:
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
	fmt.Println(cyan("npc") + " " + bold("Nginx Proxy Configurator"))
	fmt.Println(dim("Secure reverse proxies from your terminal."))
	fmt.Println(dim("Backups before writes. nginx -t before reloads."))
	fmt.Println()
}

func (ui terminalUI) dashboard() {
	cfg, _ := config.Load("")
	active := 0
	for _, site := range cfg.Sites {
		if _, err := os.Lstat(site.EnabledPath); err == nil {
			active++
		}
	}
	fmt.Println(panel("System",
		fmt.Sprintf("Nginx:  %s", yesNo(system.Exists("nginx"))),
		fmt.Sprintf("Active: %s", yesNo(nginx.ServiceActive())),
		fmt.Sprintf("Docker: %s", yesNo(docker.Installed())),
		fmt.Sprintf("Sites:  %d active / %d total", active, len(cfg.Sites)),
	))
	if len(cfg.Sites) == 0 {
		fmt.Println(dim("No npc-managed sites yet. Use Docker expose or custom reverse proxy to create one."))
	} else {
		fmt.Println(bold("Managed sites"))
		for _, site := range cfg.SortedSites() {
			enabled := "disabled"
			if _, err := os.Lstat(site.EnabledPath); err == nil {
				enabled = "enabled"
			}
			fmt.Printf("  %s  %s  %s\n", cyan(site.Hostname), dim(enabled), site.BackendURL())
		}
	}
	fmt.Println()
}

func (ui terminalUI) exposeDocker() error {
	if !docker.Installed() {
		return validationError{fmt.Errorf("docker was not found")}
	}
	fmt.Println(bold("Docker discovery"))
	fmt.Println(dim("Scanning running containers. Docker containers and Compose files will not be modified."))
	fmt.Println()
	containers, err := docker.RunningContainers()
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		fmt.Println(warn("No running Docker containers found."))
		return nil
	}
	labels := make([]string, 0, len(containers))
	for _, c := range containers {
		ports := "no ports"
		if c.PortsRaw != "" {
			ports = c.PortsRaw
		}
		labels = append(labels, fmt.Sprintf("%s  %s", c.Name, dim(c.Image+" | "+ports)))
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
		fmt.Println(warn("Note:") + " the selected port is not published on the host. Nginx must be able to resolve and reach the container name.")
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
	fmt.Println(panel("Planned site",
		"Hostname: "+site.Hostname,
		"Backend:  "+site.BackendURL(),
		"Config:   "+site.ConfigPath,
		"Enabled:  "+site.EnabledPath,
		"SSL:      "+yesNo(site.SSL),
	))
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
		fmt.Println(bold(title))
		for i, option := range options {
			fmt.Printf("  %s) %s\n", cyan(strconv.Itoa(i+1)), option)
		}
		fmt.Print(dim("Select: "))
		text, _ := ui.reader.ReadString('\n')
		value, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && value >= 1 && value <= len(options) {
			fmt.Println()
			return value - 1
		}
		fmt.Println(warn("Please enter a valid number."))
	}
}

func (ui terminalUI) askRequired(label string) string {
	for {
		value := ui.askDefault(label, "")
		if strings.TrimSpace(value) != "" {
			return value
		}
		fmt.Println(warn("This value is required."))
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

func (ui terminalUI) pause() {
	fmt.Print(dim("Press Enter to continue..."))
	_, _ = ui.reader.ReadString('\n')
	fmt.Println()
}

func panel(title string, lines ...string) string {
	width := len(title) + 4
	for _, line := range lines {
		if len(line)+4 > width {
			width = len(line) + 4
		}
	}
	var b strings.Builder
	separator := "+" + strings.Repeat("-", width-2) + "+"
	b.WriteString(cyan(separator) + "\n")
	b.WriteString(cyan("| ") + bold(title) + strings.Repeat(" ", width-len(title)-3) + cyan("|") + "\n")
	b.WriteString(cyan(separator) + "\n")
	for _, line := range lines {
		b.WriteString(cyan("| ") + line + strings.Repeat(" ", width-len(line)-3) + cyan("|") + "\n")
	}
	b.WriteString(cyan(separator))
	return b.String()
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func bold(s string) string { return "\033[1m" + s + "\033[0m" }
func cyan(s string) string { return "\033[36m" + s + "\033[0m" }
func dim(s string) string  { return "\033[2m" + s + "\033[0m" }
func warn(s string) string { return "\033[33m" + s + "\033[0m" }
