package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/acme"
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
	if err := ui.ensureStartupDependencies(); err != nil {
		return err
	}
	for {
		ui.header()
		ui.dashboard()
		choice := ui.actionMenu("Choose an action", []menuOption{
			{Title: "Expose a Docker container", Desc: "Scan running containers and publish one through Nginx"},
			{Title: "Create a custom reverse proxy", Desc: "Enter hostname, backend, TLS, logs, and proxy options manually"},
			{Title: "List managed sites", Desc: "Show sites tracked in /etc/npc/config.yaml"},
			{Title: "Show system status", Desc: "Print Nginx, paths, and managed-site counters"},
			{Title: "Quit", Desc: "Leave the terminal UI"},
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

func (ui terminalUI) ensureStartupDependencies() error {
	missingNginx := !system.Exists("nginx")
	missingACME := !acme.Installed()
	if !missingNginx && !missingACME {
		return nil
	}
	fmt.Println(section("Startup Checks"))
	fmt.Println(dim("npc checks required tools before you start configuring reverse proxies."))
	fmt.Println()
	if missingNginx {
		fmt.Println(warn("Nginx is not installed."))
		if ui.confirm("Install Nginx now with apt?", true) {
			if err := system.RequireRoot(); err != nil {
				return permissionError{fmt.Errorf("installing Nginx requires root; rerun with sudo npc")}
			}
			fmt.Println(dim("Running apt update and apt install nginx..."))
			if err := nginx.InstallApt(true); err != nil {
				return fmt.Errorf("nginx installation failed: %w", err)
			}
			fmt.Println(ok("Nginx installed."))
		} else {
			fmt.Println(dim("Skipping Nginx installation. Write actions will ask again before continuing."))
		}
		fmt.Println()
	}
	if missingACME {
		fmt.Println(warn("acme.sh is not installed."))
		if ui.confirm("Install acme.sh now?", true) {
			if err := system.RequireRoot(); err != nil {
				return permissionError{fmt.Errorf("installing acme.sh requires root; rerun with sudo npc")}
			}
			email := ui.askDefault("ACME account email, optional", "")
			fmt.Println(dim("Downloading and running the official acme.sh installer..."))
			if err := acme.Install(email); err != nil {
				return fmt.Errorf("acme.sh installation failed: %w", err)
			}
			fmt.Println(ok("acme.sh installed."))
		} else {
			fmt.Println(dim("Skipping acme.sh installation. ACME certificate flows will ask again before continuing."))
		}
		fmt.Println()
	}
	ui.pause()
	return nil
}

type terminalUI struct {
	reader *bufio.Reader
}

type menuOption struct {
	Title string
	Desc  string
}

func newTerminalUI() terminalUI {
	return terminalUI{reader: bufio.NewReader(os.Stdin)}
}

func (ui terminalUI) header() {
	fmt.Print("\033[2J\033[H")
	fmt.Println(cyan("+------------------------------------------------------------------------------+"))
	fmt.Println(cyan("|") + " " + bold("npc") + "  Nginx Proxy Configurator" + strings.Repeat(" ", 45) + cyan("|"))
	fmt.Println(cyan("|") + " " + dim("Reverse proxies, TLS, Docker discovery, and safe Nginx reloads") + strings.Repeat(" ", 12) + cyan("|"))
	fmt.Println(cyan("+------------------------------------------------------------------------------+"))
	fmt.Printf("  %s  %s  %s\n", ok("SAFE DEFAULTS"), cyan("DRY RUN READY"), dim("Backups before writes. nginx -t before reloads."))
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
	fmt.Println(section("Control Plane"))
	fmt.Printf("  %-16s %-14s %-16s %-14s\n", "Nginx", badge(system.Exists("nginx")), "Service", badge(nginx.ServiceActive()))
	fmt.Printf("  %-16s %-14s %-16s %s\n", "Docker", badge(docker.Installed()), "Sites", fmt.Sprintf("%d active / %d total", active, len(cfg.Sites)))
	fmt.Printf("  %-16s %-14s %-16s %s\n", "Config", ok("TRACKED"), "Reload guard", ok("nginx -t"))
	if len(cfg.Sites) == 0 {
		fmt.Println()
		fmt.Println(emptyState("No managed sites yet", "Expose a Docker container or create a custom reverse proxy to get started."))
	} else {
		fmt.Println()
		fmt.Println(section("Managed Sites"))
		for _, site := range cfg.SortedSites() {
			enabled := fail("disabled")
			if _, err := os.Lstat(site.EnabledPath); err == nil {
				enabled = ok("enabled")
			}
			fmt.Printf("  %-34s %-18s %s\n", cyan(site.Hostname), enabled, dim(site.BackendURL()))
		}
	}
	fmt.Println()
}

func (ui terminalUI) exposeDocker() error {
	if !docker.Installed() {
		return validationError{fmt.Errorf("docker was not found")}
	}
	fmt.Println(section("Docker Discovery"))
	fmt.Println(dim("Scanning running containers. Docker containers and Compose files will not be modified."))
	fmt.Println(dim("Published ports become 127.0.0.1:<host-port>; container-only ports are marked before use."))
	fmt.Println()
	containers, err := docker.RunningContainers()
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		fmt.Println(emptyState("No running Docker containers", "Start a container or use the custom reverse proxy flow."))
		return nil
	}
	labels := make([]string, 0, len(containers))
	for _, c := range containers {
		ports := "no ports"
		if c.PortsRaw != "" {
			ports = c.PortsRaw
		}
		labels = append(labels, fmt.Sprintf("%-28s %s %s", c.Name, cyan("container"), dim(c.Image+" | "+ports)))
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
	fmt.Println(panel("Reverse Proxy Review",
		"Hostname: "+site.Hostname,
		"Backend:  "+site.BackendURL(),
		"Profile:  "+site.Profile,
		"Config:   "+site.ConfigPath,
		"Enabled:  "+site.EnabledPath,
		"SSL:      "+yesNo(site.SSL),
		"Logs:     "+yesNo(site.AccessLog != "" || site.ErrorLog != ""),
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
	if err := executeCreate(o); err != nil {
		return fmt.Errorf("could not create reverse proxy: %w", err)
	}
	return nil
}

func (ui terminalUI) actionMenu(title string, options []menuOption) int {
	for {
		fmt.Println(section(title))
		for i, option := range options {
			number := fmt.Sprintf("[%d]", i+1)
			fmt.Printf("  %s %s\n", cyan(number), bold(option.Title))
			fmt.Printf("      %s\n", dim(option.Desc))
		}
		fmt.Println()
		fmt.Print(dim("Select an option: "))
		text, _ := ui.reader.ReadString('\n')
		value, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && value >= 1 && value <= len(options) {
			fmt.Println()
			return value - 1
		}
		fmt.Println(warn("Please enter a valid number."))
	}
}

func (ui terminalUI) menu(title string, options []string) int {
	for {
		fmt.Println(section(title))
		for i, option := range options {
			fmt.Printf("  %s %s\n", cyan(fmt.Sprintf("[%d]", i+1)), option)
		}
		fmt.Println()
		fmt.Print(dim("Select an option: "))
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
		fmt.Printf("%s %s ", label, dim("["+def+"]"))
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

func section(title string) string {
	return bold(title) + "\n" + cyan(strings.Repeat("-", 78))
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func badge(v bool) string {
	if v {
		return ok("[ready]")
	}
	return fail("[missing]")
}

func emptyState(title, body string) string {
	return warn(title) + "\n  " + dim(body)
}

func ok(s string) string   { return "\033[32m" + s + "\033[0m" }
func fail(s string) string { return "\033[31m" + s + "\033[0m" }
func bold(s string) string { return "\033[1m" + s + "\033[0m" }
func cyan(s string) string { return "\033[36m" + s + "\033[0m" }
func dim(s string) string  { return "\033[2m" + s + "\033[0m" }
func warn(s string) string { return "\033[33m" + s + "\033[0m" }
