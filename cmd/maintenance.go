package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func maintenanceCommand() *cobra.Command {
	root := &cobra.Command{Use: "maintenance", Short: "Manage maintenance mode"}
	root.AddCommand(maintenanceSetCommand("enable", true), maintenanceSetCommand("disable", false))
	root.AddCommand(&cobra.Command{Use: "edit <hostname>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		pagePath := path.Join(paths.MaintenanceDir, args[0], "index.html")
		fmt.Println("Maintenance page:", pagePath)
		return nil
	}})
	return root
}

func maintenanceSetCommand(name string, enabled bool) *cobra.Command {
	return &cobra.Command{Use: name + " <hostname>", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		site, ok := cfg.Sites[args[0]]
		if !ok {
			return validationError{fmt.Errorf("site %s is not managed by npc", args[0])}
		}
		site.MaintenanceEnabled = enabled
		if enabled {
			dir := path.Join(paths.MaintenanceDir, site.Hostname)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}
			page := path.Join(dir, "index.html")
			if _, err := os.Stat(page); os.IsNotExist(err) {
				_ = os.WriteFile(page, []byte("<!doctype html><title>Maintenance</title><h1>Maintenance</h1><p>This site is temporarily unavailable.</p>\n"), 0o644)
			}
		}
		content, err := renderer.RenderSite(site)
		if err != nil {
			return err
		}
		if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
			return err
		}
		if out, err := nginx.Reload(); err != nil {
			return nginxTestError{fmt.Errorf("%s", out)}
		}
		if err := config.Save("", cfg); err != nil {
			return err
		}
		fmt.Println("Maintenance", name+"d", "for", site.Hostname)
		return nil
	}}
}

func checkCommand() *cobra.Command {
	return &cobra.Command{Use: "check <hostname>", Args: cobra.ExactArgs(1), Short: "Run site health checks", RunE: func(cmd *cobra.Command, args []string) error {
		site, err := loadSite(args[0])
		if err != nil {
			return err
		}
		report := map[string]any{
			"hostname":         site.Hostname,
			"backend":          site.BackendURL(),
			"config_exists":    fileExists(site.ConfigPath),
			"enabled":          fileExists(site.EnabledPath),
			"nginx_test_ok":    false,
			"certificate":      site.CertificatePath,
			"redirect_https":   site.RedirectHTTPS,
			"websocket_config": site.WebSocket,
		}
		if _, err := nginx.Test(); err == nil {
			report["nginx_test_ok"] = true
		}
		if app.jsonOut {
			return writeJSON(report)
		}
		for k, v := range report {
			fmt.Printf("%-18s %v\n", k+":", v)
		}
		return nil
	}}
}
