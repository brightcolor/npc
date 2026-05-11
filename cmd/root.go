package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"

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
	build   BuildInfo
	jsonOut bool
	verbose bool
}

var app appState

func Execute(info BuildInfo) {
	app.build = info
	root := newRootCommand()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(exitCode(err))
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "npc",
		Short:        "Nginx Proxy Configurator",
		Long:         "npc installs, configures, manages, tests, and updates Nginx reverse proxy sites.",
		SilenceUsage: true,
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
			return runTUI(cmd, args)
		},
	}
	cmd.PersistentFlags().BoolVar(&app.jsonOut, "json", false, "print machine-readable JSON where supported")
	cmd.PersistentFlags().BoolVar(&app.verbose, "verbose", false, "print technical details")
	cmd.Flags().Bool("install", false, "install current binary to /usr/local/bin/npc")
	cmd.Flags().Bool("upgrade", false, "upgrade npc from GitHub Releases")
	cmd.Flags().Bool("version", false, "show version")
	cmd.AddCommand(createCommand(), listCommand(), statusCommand(), showCommand(), editCommand())
	cmd.AddCommand(enableCommand(), disableCommand(), deleteCommand(), testCommand(), reloadCommand(), restartCommand())
	cmd.AddCommand(installNginxCommand(), backupCommand(), restoreCommand(), certsCommand(), doctorCommand(), logsCommand())
	cmd.AddCommand(upgradeCommand(), uninstallCommand(), maintenanceCommand(), checkCommand(), exportCommand(), importCommand(), dockerCommand(), tuiCommand())
	return cmd
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
	default:
		return 1
	}
}

type validationError struct{ error }
type nginxTestError struct{ error }
type permissionError struct{ error }
