package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/updater"
	"github.com/spf13/cobra"
)

type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	RepoOwner string `json:"repo_owner"`
	RepoName  string `json:"repo_name"`
}

type appState struct {
	build     BuildInfo
	jsonOut   bool
	verbose   bool
	noUpgrade bool
	update    *updater.ReleaseInfo
}

var app appState

func Execute(info BuildInfo) {
	app.build = info
	args := os.Args[1:]
	noUpgrade := stripNoUpgradeFlag(&args)
	app.noUpgrade = noUpgrade
	if isQuickCreateArgs(args) {
		if !app.noUpgrade {
			printUpgradeNotice()
		}
		if err := runQuickCreate(args[0], args[1]); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(exitCode(err))
		}
		return
	}
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(exitCode(err))
	}
}

func stripNoUpgradeFlag(args *[]string) bool {
	filtered := (*args)[:0]
	found := false
	for _, arg := range *args {
		if arg == "--no-upgrade" {
			found = true
			continue
		}
		filtered = append(filtered, arg)
	}
	*args = filtered
	return found
}

func isQuickCreateArgs(args []string) bool {
	if len(args) != 2 || strings.HasPrefix(args[0], "-") {
		return false
	}
	if isKnownCommand(args[0]) {
		return false
	}
	_, err := strconv.Atoi(args[1])
	return err == nil
}

func isKnownCommand(name string) bool {
	known := map[string]bool{
		"backup": true, "certs": true, "check": true, "completion": true, "create": true,
		"delete": true, "disable": true, "docker": true, "doctor": true, "edit": true,
		"enable": true, "export": true, "help": true, "import": true, "install-nginx": true,
		"list": true, "logs": true, "maintenance": true, "reload": true, "restart": true,
		"restore": true, "show": true, "status": true, "test": true, "tui": true,
		"uninstall": true, "upgrade": true, "repair": true, "inspect": true,
	}
	return known[name]
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "npc [hostname port]",
		Short:        "Nginx Proxy Configurator",
		Long:         "npc installs, configures, manages, tests, and updates Nginx reverse proxy sites.",
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if shouldPrintUpgradeNotice(cmd) {
				printUpgradeNotice()
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if v, _ := cmd.Flags().GetBool("version"); v {
				printVersion()
				return nil
			}
			if install, _ := cmd.Flags().GetBool("install"); install {
				return runInstall(cmd, args)
			}
			if upgrade, _ := cmd.Flags().GetBool("upgrade"); upgrade {
				return runUpgrade(cmd, args)
			}
			if len(args) == 2 {
				return runQuickCreate(args[0], args[1])
			}
			if len(args) != 0 {
				return validationError{fmt.Errorf("expected either no arguments or quick mode: npc <hostname> <port>")}
			}
			return runTUI(cmd, args)
		},
	}
	cmd.PersistentFlags().BoolVar(&app.jsonOut, "json", false, "print machine-readable JSON where supported")
	cmd.PersistentFlags().BoolVar(&app.verbose, "verbose", false, "print technical details")
	cmd.PersistentFlags().BoolVar(&app.noUpgrade, "no-upgrade", app.noUpgrade, "skip automatic update check")
	cmd.Flags().Bool("install", false, "install current binary to /usr/local/bin/npc")
	cmd.Flags().Bool("upgrade", false, "upgrade npc from GitHub Releases")
	cmd.Flags().Bool("version", false, "show version")
	cmd.AddCommand(createCommand(), listCommand(), statusCommand(), showCommand(), editCommand(), repairCommand(), inspectCommand())
	cmd.AddCommand(enableCommand(), disableCommand(), deleteCommand(), testCommand(), reloadCommand(), restartCommand())
	cmd.AddCommand(installNginxCommand(), backupCommand(), restoreCommand(), certsCommand(), doctorCommand(), logsCommand())
	cmd.AddCommand(upgradeCommand(), uninstallCommand(), maintenanceCommand(), checkCommand(), exportCommand(), importCommand(), dockerCommand(), tuiCommand())
	return cmd
}

func shouldPrintUpgradeNotice(cmd *cobra.Command) bool {
	if app.noUpgrade || app.jsonOut {
		return false
	}
	if cmd.CommandPath() == "npc" || cmd.CommandPath() == "npc tui" {
		return false
	}
	if cmd.CommandPath() == "npc upgrade" || cmd.CommandPath() == "npc completion" {
		return false
	}
	if v, _ := cmd.Flags().GetBool("version"); v {
		return false
	}
	if install, _ := cmd.Flags().GetBool("install"); install {
		return false
	}
	if upgrade, _ := cmd.Flags().GetBool("upgrade"); upgrade {
		return false
	}
	return true
}

func printUpgradeNotice() {
	info, err := updater.Check(app.build.RepoOwner, app.build.RepoName, app.build.Version)
	if err != nil {
		if app.verbose {
			fmt.Fprintln(os.Stderr, "Update check failed:", err)
		}
		return
	}
	if info.UpdateAvailable {
		fmt.Fprintf(os.Stderr, "Update available: npc %s -> %s. Run `sudo npc upgrade` or add `--no-upgrade` to skip this check.\n", info.CurrentVersion, info.LatestVersion)
	}
}

func runQuickCreate(hostname, portValue string) error {
	port, err := strconv.Atoi(portValue)
	if err != nil {
		return validationError{fmt.Errorf("port must be a number")}
	}
	return executeCreate(createOptions{
		hostname:          hostname,
		backendHost:       "127.0.0.1",
		backendPort:       port,
		backendScheme:     "http",
		profile:           "websocket",
		clientMaxBodySize: "100M",
		ssl:               true,
		acme:              true,
		acmeMethod:        "http",
		redirectHTTPS:     true,
		websocket:         true,
		http2:             true,
		securityHeaders:   "standard",
		accessLog:         true,
		errorLog:          true,
		nonInteractive:    true,
		assumeYes:         true,
	})
}

func printVersion() {
	info := map[string]string{
		"version":    app.build.Version,
		"commit":     app.build.Commit,
		"date":       app.build.Date,
		"go_version": runtime.Version(),
		"os_arch":    runtime.GOOS + "/" + runtime.GOARCH,
	}
	if app.jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(info)
		return
	}
	fmt.Printf("npc %s\ncommit: %s\nbuilt: %s\ngo: %s\nplatform: %s\n",
		info["version"], info["commit"], info["date"], info["go_version"], info["os_arch"])
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func exitCode(err error) int {
	switch err.(type) {
	case validationError:
		return 2
	case nginxTestError:
		return 3
	case permissionError:
		return 5
	case networkError:
		return 6
	default:
		return 1
	}
}

type validationError struct{ error }
type nginxTestError struct{ error }
type permissionError struct{ error }
type networkError struct{ error }
