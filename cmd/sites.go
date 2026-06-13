package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func showCommand() *cobra.Command {
	return &cobra.Command{Use: "show <hostname>", Args: cobra.ExactArgs(1), Short: "Show site details", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		site, ok := cfg.FindSite(args[0])
		if !ok {
			return validationError{fmt.Errorf("site %s is not managed by npc", args[0])}
		}
		if app.jsonOut {
			return writeJSON(site)
		}
		fmt.Printf("Hostname: %s\nAlias: %s\nGroup: %s\nTags: %s\nArchived: %v\nBackend: %s\nWebSocket: %v\nSSL: %v\nACME: %s\nACME CA: %s\nConfig: %s\nCertificate: %s\nLast reload: %s\nLast nginx -t: %s\n",
			site.Hostname, site.Alias, site.Group, strings.Join(site.Tags, ","), site.Archived, site.BackendURL(), site.WebSocket, site.SSL, site.ACMEMethod, defaultString(site.ACMECA, "letsencrypt"), site.ConfigPath,
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
	var q siteQuery
	var yes bool
	cmd := &cobra.Command{Use: "disable <hostname>", Short: "Disable a site", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		if len(args) == 0 {
			return disableBulk(q, yes)
		}
		if len(args) != 1 {
			return validationError{fmt.Errorf("expected one hostname or filtered bulk flags")}
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
	bindSiteQueryFlags(cmd, &q)
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm filtered bulk disable")
	return cmd
}

func disableBulk(q siteQuery, yes bool) error {
	if !yes || (q.tag == "" && q.group == "") {
		return validationError{fmt.Errorf("bulk disable requires --tag or --group plus --yes")}
	}
	cfg, err := loadManagedConfig()
	if err != nil {
		return err
	}
	sites := q.apply(cfg.SortedSites())
	for _, site := range sites {
		if err := nginx.Disable(site.EnabledPath); err != nil {
			return err
		}
		fmt.Println("Disabled", site.Hostname)
	}
	if out, err := nginx.Reload(); err != nil {
		return nginxTestError{fmt.Errorf("%s", out)}
	}
	return nil
}

func deleteCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{Use: "delete <hostname>", Args: cobra.ExactArgs(1), Short: "Delete or disable a site", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		site, ok := cfg.FindSite(args[0])
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
	cfg, err := loadManagedConfig()
	if err != nil {
		return nil, err
	}
	site, ok := cfg.FindSite(hostname)
	if !ok {
		return nil, validationError{fmt.Errorf("site %s is not managed by npc", hostname)}
	}
	return site, nil
}
