package cmd

import (
	"fmt"
	"strings"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/system"
	"github.com/brightcolor/npc/internal/updater"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func tuiCommand() *cobra.Command {
	return &cobra.Command{Use: "tui", Short: "Open the interactive terminal UI", RunE: runTUI}
}

func runTUI(cmd *cobra.Command, args []string) error {
	ui := newTerminalUI()
	if err := ui.ensureStartupDependencies(); err != nil {
		return err
	}
	if !app.noUpgrade {
		app.update, _ = updater.Check(app.build.RepoOwner, app.build.RepoName, app.build.Version)
	}
	model, err := newBubbleModel()
	if err != nil {
		return err
	}
	final, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}
	if m, ok := final.(bubbleModel); ok {
		return runBubbleAction(ui, cmd, args, m.action)
	}
	return nil
}

func runBubbleAction(ui terminalUI, cmd *cobra.Command, args []string, action string) error {
	switch action {
	case "create":
		o := createOptions{}
		if err := promptCreate(&o); err != nil {
			return err
		}
		return ui.previewAndRun(o)
	case "docker":
		return ui.exposeDocker()
	case "cloudflare":
		return ui.configureCloudflare()
	case "list":
		return listCommand().RunE(cmd, args)
	case "status":
		return statusCommand().RunE(cmd, args)
	case "upgrade":
		return runUpgradeVersion("")
	default:
		return nil
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
	if missingNginx {
		if err := ui.offerNginxInstall(); err != nil {
			return err
		}
	}
	if missingACME {
		if err := ui.offerACMEInstall(); err != nil {
			return err
		}
	}
	ui.pause()
	return nil
}

func (ui terminalUI) offerNginxInstall() error {
	fmt.Println(warn("Nginx is not installed."))
	if !ui.confirm("Install Nginx now with apt?", true) {
		fmt.Println(dim("Skipping Nginx installation. Write actions will ask again before continuing."))
		return nil
	}
	if err := system.RequireRoot(); err != nil {
		return permissionError{fmt.Errorf("installing Nginx requires root; rerun with sudo npc")}
	}
	fmt.Println(dim("Running apt update and apt install nginx..."))
	if err := nginx.InstallApt(true); err != nil {
		return fmt.Errorf("nginx installation failed: %w", err)
	}
	fmt.Println(ok("Nginx installed."))
	return nil
}

func (ui terminalUI) offerACMEInstall() error {
	fmt.Println(warn("acme.sh is not installed."))
	if !ui.confirm("Install acme.sh now?", true) {
		fmt.Println(dim("Skipping acme.sh installation. ACME certificate flows will ask again before continuing."))
		return nil
	}
	if err := system.RequireRoot(); err != nil {
		return permissionError{fmt.Errorf("installing acme.sh requires root; rerun with sudo npc")}
	}
	email := ui.askDefault("ACME account email, optional", "")
	fmt.Println(dim("Downloading and running the official acme.sh installer..."))
	if err := acme.Install(email); err != nil {
		return fmt.Errorf("acme.sh installation failed: %w", err)
	}
	fmt.Println(ok("acme.sh installed."))
	return nil
}

func compactChangelog() string {
	if app.update == nil || strings.TrimSpace(app.update.Changelog) == "" {
		return ""
	}
	text := strings.TrimSpace(app.update.Changelog)
	if len(text) > 700 {
		return text[:700] + "..."
	}
	return text
}
