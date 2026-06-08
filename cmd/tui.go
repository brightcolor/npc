package cmd

import (
	"fmt"
	"strings"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/system"
	"github.com/brightcolor/npc/internal/updater"
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
	ui.showUpdateOnOpen()
	for {
		ui.header()
		ui.dashboard()
		options := tuiOptions()
		upgradeIndex := -1
		if app.update != nil && app.update.UpdateAvailable {
			upgradeIndex = len(options)
			options = append(options, menuOption{Title: "Upgrade npc", Desc: fmt.Sprintf("Install %s over current %s", app.update.LatestVersion, app.update.CurrentVersion)})
		}
		options = append(options, menuOption{Title: "Quit", Desc: "Leave the terminal UI"})
		choice := ui.actionMenu("Choose an action", options)
		if choice == upgradeIndex {
			if err := runUpgradeVersion(""); err != nil {
				return err
			}
			ui.pause()
			continue
		}
		if err := runTUIChoice(ui, cmd, args, choice); err != nil {
			return err
		}
	}
}

func tuiOptions() []menuOption {
	return []menuOption{
		{Title: "Expose a Docker container", Desc: "Scan running containers and publish one through Nginx"},
		{Title: "Create a custom reverse proxy", Desc: "Enter hostname, backend, TLS, logs, and proxy options manually"},
		{Title: "Configure Cloudflare DNS-01", Desc: "Save Cloudflare API settings for automatic ACME DNS validation"},
		{Title: "List managed sites", Desc: "Show sites tracked in /etc/npc/config.yaml"},
		{Title: "Edit a managed site", Desc: "Change backend, metadata, profile, WebSocket, body size, and logging"},
		{Title: "Delete a managed site", Desc: "Disable a site and optionally remove config, metadata, and certificates"},
		{Title: "Show system status", Desc: "Print Nginx, paths, and managed-site counters"},
	}
}

func runTUIChoice(ui terminalUI, cmd *cobra.Command, args []string, choice int) error {
	switch choice {
	case 0:
		return ui.exposeDocker()
	case 1:
		o := createOptions{}
		if err := promptCreate(&o); err != nil {
			return err
		}
		return ui.previewAndRun(o)
	case 2:
		if err := ui.configureCloudflare(); err != nil {
			return err
		}
	case 3:
		if err := listCommand().RunE(cmd, args); err != nil {
			return err
		}
	case 4:
		if err := ui.editManagedSite(); err != nil {
			return err
		}
	case 5:
		if err := ui.deleteManagedSite(); err != nil {
			return err
		}
	case 6:
		return statusCommand().RunE(cmd, args)
	default:
		return nil
	}
	ui.pause()
	return nil
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

func (ui terminalUI) showUpdateOnOpen() {
	if app.noUpgrade {
		return
	}
	info, err := updater.Check(app.build.RepoOwner, app.build.RepoName, app.build.Version)
	if err != nil {
		if app.verbose {
			fmt.Println(warn("Update check failed: ") + err.Error())
			ui.pause()
		}
		return
	}
	app.update = info
	if !info.UpdateAvailable {
		return
	}
	ui.header()
	fmt.Println(panel("Update Available", "Current: "+info.CurrentVersion, "Latest:  "+info.LatestVersion, "Release: "+info.URL))
	if strings.TrimSpace(info.Changelog) != "" {
		fmt.Println(section("Changelog"))
		fmt.Println(strings.TrimSpace(info.Changelog))
	}
	fmt.Println(dim("Use the Upgrade npc menu entry to install this release, or start with --no-upgrade to skip checks."))
	ui.pause()
}
