package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/docker"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/system"
)

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
	fmt.Println(accent("+------------------------------------------------------------------------------+"))
	fmt.Println(accent("|") + " " + bold("npc") + "  " + cyan("Nginx Proxy Configurator") + strings.Repeat(" ", 44) + accent("|"))
	fmt.Println(accent("|") + " " + dim("Reverse proxies, TLS automation, Docker discovery, safe reloads") + strings.Repeat(" ", 9) + accent("|"))
	fmt.Println(accent("+------------------------------------------------------------------------------+"))
	fmt.Printf("  %s  %s  %s  %s\n", pill("SAFE"), pill("TLS"), pill("DOCKER"), dim("Backups before writes. nginx -t before reloads."))
	fmt.Println()
}

func (ui terminalUI) dashboard() {
	cfg, _ := config.Load("")
	active, problems := 0, 0
	for _, site := range cfg.Sites {
		if _, err := os.Lstat(site.EnabledPath); err == nil {
			active++
		}
		if siteProblemLabels(site) != "" {
			problems++
		}
	}
	fmt.Println(section("Control Plane"))
	fmt.Println(panel("Runtime",
		"Nginx:        "+badge(system.Exists("nginx"))+"    Service: "+badge(nginx.ServiceActive()),
		"Docker:       "+badge(docker.Installed())+"    Sites:   "+fmt.Sprintf("%d active / %d total / %d issues", active, len(cfg.Sites), problems),
		"Config:       "+ok("tracked")+"    Guard:   "+ok("nginx -t before reload"),
		"Cloudflare:   "+badge(cloudflareDNSReady())+"    ACME:    "+ok("Let's Encrypt default"),
	))
	if app.update != nil && app.update.UpdateAvailable {
		fmt.Println(panel("Update", "Current: "+warn(app.update.CurrentVersion), "Latest:  "+ok(app.update.LatestVersion)))
	} else {
		fmt.Println(dim("  Version " + app.build.Version + " | latest checked"))
	}
	if len(cfg.Sites) == 0 {
		fmt.Println()
		fmt.Println(emptyState("No managed sites yet", "Expose a Docker container or create a custom reverse proxy to get started."))
	}
	fmt.Println()
}

func (ui terminalUI) actionMenu(title string, options []menuOption) int {
	for {
		fmt.Println(section(title))
		for i, option := range options {
			number := fmt.Sprintf("%02d", i+1)
			fmt.Printf("  %s %s\n", accent("["+number+"]"), bold(option.Title))
			fmt.Printf("       %s\n", dim(option.Desc))
		}
		fmt.Println()
		fmt.Print(dim("Select an option: "))
		text, _ := ui.reader.ReadString('\n')
		value, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && value >= 1 && value <= len(options) {
			fmt.Println()
			return value - 1
		}
		fmt.Println(warn("Invalid selection."))
	}
}

func (ui terminalUI) menu(title string, options []string) int {
	for {
		fmt.Println(section(title))
		for i, option := range options {
			fmt.Printf("  %s %s\n", accent(fmt.Sprintf("[%02d]", i+1)), option)
		}
		fmt.Print(dim("Select an option: "))
		text, _ := ui.reader.ReadString('\n')
		value, err := strconv.Atoi(strings.TrimSpace(text))
		if err == nil && value >= 1 && value <= len(options) {
			fmt.Println()
			return value - 1
		}
		fmt.Println(warn("Invalid selection."))
	}
}

func (ui terminalUI) askRequired(label string) string {
	for {
		value := ui.askDefault(label, "")
		if value != "" {
			return value
		}
		fmt.Println(warn("Required value."))
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

func (ui terminalUI) askSecret(label string) string {
	fmt.Print(label + ": ")
	disableEcho()
	text, _ := ui.reader.ReadString('\n')
	enableEcho()
	fmt.Println()
	return strings.TrimSpace(text)
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
}

func disableEcho() { _ = exec.Command("stty", "-echo").Run() }
func enableEcho()  { _ = exec.Command("stty", "echo").Run() }
