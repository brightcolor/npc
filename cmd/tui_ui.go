package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

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
	cfg, _ := loadManagedConfig()
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
	if choice, ok := ui.keyboardActionMenu(title, options); ok {
		return choice
	}
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
	menuOptions := make([]menuOption, 0, len(options))
	for _, option := range options {
		menuOptions = append(menuOptions, menuOption{Title: option})
	}
	if choice, ok := ui.keyboardMenu(title, menuOptions, true); ok {
		return choice
	}
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

func (ui terminalUI) keyboardActionMenu(title string, options []menuOption) (int, bool) {
	return ui.keyboardMenu(title, options, false)
}

func (ui terminalUI) keyboardMenu(title string, options []menuOption, compact bool) (int, bool) {
	if len(options) == 0 || !enableRawInput() {
		return 0, false
	}
	defer restoreInput()
	selected := 0
	lines := 0
	for {
		if lines > 0 {
			fmt.Printf("\033[%dA\033[J", lines)
		}
		lines = renderKeyboardMenu(title, options, selected, compact)
		key, number := readUIKey()
		switch {
		case key == keyCancel:
			fmt.Println()
			return 0, false
		case number > 0 && number <= len(options):
			fmt.Println()
			return number - 1, true
		case key == keyEnter:
			fmt.Println()
			return selected, true
		case key == keyUp:
			selected = (selected - 1 + len(options)) % len(options)
		case key == keyDown:
			selected = (selected + 1) % len(options)
		}
	}
}

func renderKeyboardMenu(title string, options []menuOption, selected int, compact bool) int {
	lines := 0
	fmt.Println(section(title))
	lines += 2
	for i, option := range options {
		cursor := " "
		number := fmt.Sprintf("%02d", i+1)
		titleText := option.Title
		if i == selected {
			cursor = ">"
			titleText = selectedText(titleText)
		}
		fmt.Printf("  %s %s %s\n", accent(cursor), accent("["+number+"]"), titleText)
		lines++
		if !compact && option.Desc != "" {
			desc := option.Desc
			if i == selected {
				desc = cyan(desc)
			} else {
				desc = dim(desc)
			}
			fmt.Printf("        %s\n", desc)
			lines++
		}
	}
	fmt.Println()
	fmt.Println(dim("Use Up/Down and Enter. Press a number to jump, q or Esc for number mode."))
	return lines + 2
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
