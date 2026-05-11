package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func listCommand() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List npc-managed sites", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		if app.jsonOut {
			return writeJSON(cfg.SortedSites())
		}
		if len(cfg.Sites) == 0 {
			fmt.Println("No npc-managed sites found.")
			fmt.Println("Create one with `sudo npc create` or open the UI with `npc`.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "HOSTNAME\tENABLED\tSSL\tBACKEND\tCERT")
		for _, site := range cfg.SortedSites() {
			enabled := "no"
			if _, err := os.Lstat(site.EnabledPath); err == nil {
				enabled = "yes"
			}
			fmt.Fprintf(w, "%s\t%s\t%v\t%s\t%s\n", site.Hostname, enabled, site.SSL, site.BackendURL(), site.CertificatePath)
		}
		return w.Flush()
	}}
}

func statusCommand() *cobra.Command {
	return &cobra.Command{Use: "status", Short: "Show global status", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		enabled := 0
		for _, site := range cfg.Sites {
			if _, err := os.Lstat(site.EnabledPath); err == nil {
				enabled++
			}
		}
		status := map[string]any{
			"nginx_installed": system.Exists("nginx"),
			"nginx_version":   nginx.Version(),
			"nginx_active":    nginx.ServiceActive(),
			"active_sites":    enabled,
			"disabled_sites":  len(cfg.Sites) - enabled,
			"config_file":     paths.ConfigFile,
			"sites_available": paths.NginxSitesAvailable,
			"sites_enabled":   paths.NginxSitesEnabled,
		}
		if app.jsonOut {
			return writeJSON(status)
		}
		fmt.Printf("%-18s %v\n", "nginx_installed:", status["nginx_installed"])
		fmt.Printf("%-18s %v\n", "nginx_version:", status["nginx_version"])
		fmt.Printf("%-18s %v\n", "nginx_active:", status["nginx_active"])
		fmt.Printf("%-18s %v\n", "active_sites:", status["active_sites"])
		fmt.Printf("%-18s %v\n", "disabled_sites:", status["disabled_sites"])
		fmt.Printf("%-18s %v\n", "config_file:", status["config_file"])
		fmt.Printf("%-18s %v\n", "sites_available:", status["sites_available"])
		fmt.Printf("%-18s %v\n", "sites_enabled:", status["sites_enabled"])
		return nil
	}}
}

func showCommand() *cobra.Command {
	return &cobra.Command{Use: "show <hostname>", Args: cobra.ExactArgs(1), Short: "Show site details", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		site, ok := cfg.Sites[args[0]]
		if !ok {
			return validationError{fmt.Errorf("site %s is not managed by npc", args[0])}
		}
		if app.jsonOut {
			return writeJSON(site)
		}
		fmt.Printf("Hostname: %s\nBackend: %s\nWebSocket: %v\nSSL: %v\nACME: %s\nConfig: %s\nCertificate: %s\nLast reload: %s\nLast nginx -t: %s\n",
			site.Hostname, site.BackendURL(), site.WebSocket, site.SSL, site.ACMEMethod, site.ConfigPath,
			site.CertificatePath, site.LastReload, site.LastNginxTest)
		return nil
	}}
}

func enableCommand() *cobra.Command {
	return &cobra.Command{Use: "enable <hostname>", Args: cobra.ExactArgs(1), Short: "Enable a site", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		site, err := loadSite(args[0])
		if err != nil {
			return err
		}
		if err := nginx.Enable(site.ConfigPath, site.EnabledPath); err != nil {
			return err
		}
		if out, err := nginx.Reload(); err != nil {
			return nginxTestError{fmt.Errorf("%s", out)}
		}
		fmt.Println("Enabled", site.Hostname)
		return nil
	}}
}

func disableCommand() *cobra.Command {
	return &cobra.Command{Use: "disable <hostname>", Args: cobra.ExactArgs(1), Short: "Disable a site", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		site, err := loadSite(args[0])
		if err != nil {
			return err
		}
		if err := nginx.Disable(site.EnabledPath); err != nil {
			return err
		}
		if out, err := nginx.Reload(); err != nil {
			return nginxTestError{fmt.Errorf("%s", out)}
		}
		fmt.Println("Disabled", site.Hostname)
		return nil
	}}
}

func deleteCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{Use: "delete <hostname>", Args: cobra.ExactArgs(1), Short: "Delete or disable a site", RunE: func(cmd *cobra.Command, args []string) error {
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
		if !force {
			return validationError{fmt.Errorf("delete is destructive; rerun with --force after taking a backup")}
		}
		_ = nginx.Disable(site.EnabledPath)
		_ = os.Remove(site.ConfigPath)
		delete(cfg.Sites, site.Hostname)
		if err := config.Save("", cfg); err != nil {
			return err
		}
		fmt.Println("Deleted", site.Hostname)
		return nil
	}}
	cmd.Flags().BoolVar(&force, "force", false, "confirm destructive deletion")
	return cmd
}

func loadSite(hostname string) (*config.Site, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, err
	}
	site, ok := cfg.Sites[hostname]
	if !ok {
		return nil, validationError{fmt.Errorf("site %s is not managed by npc", hostname)}
	}
	return site, nil
}
