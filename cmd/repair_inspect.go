package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/certinfo"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/dnscheck"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/revision"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func repairCommand() *cobra.Command {
	var dryRun bool
	var noReload bool
	cmd := &cobra.Command{Use: "repair <hostname>", Args: cobra.ExactArgs(1), Short: "Re-render and safely repair a managed site", RunE: func(cmd *cobra.Command, args []string) error {
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
		content, err := renderer.RenderSite(site)
		if err != nil {
			return err
		}
		if dryRun {
			return printCreatePlan(site, content)
		}
		if _, err := revision.Save(site, content); err != nil {
			return err
		}
		if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath); err != nil {
			return err
		}
		if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
			return err
		}
		if err := nginx.Enable(site.ConfigPath, site.EnabledPath); err != nil {
			return err
		}
		out, err := nginx.Test()
		if err != nil {
			return nginxTestError{fmt.Errorf("nginx -t failed, reload skipped: %s", out)}
		}
		site.LastNginxTest = time.Now().UTC().Format(time.RFC3339)
		if !noReload {
			if _, err := nginx.Reload(); err != nil {
				return err
			}
			site.LastReload = time.Now().UTC().Format(time.RFC3339)
		}
		site.UpdatedAt = time.Now().UTC()
		cfg.Sites[site.Hostname] = site
		if err := config.Save("", cfg); err != nil {
			return err
		}
		fmt.Println("Repaired", site.Hostname)
		return nil
	}}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show rendered config without writing")
	cmd.Flags().BoolVar(&noReload, "no-reload", false, "write config without reloading nginx")
	return cmd
}

func inspectCommand() *cobra.Command {
	return &cobra.Command{Use: "inspect <hostname>", Args: cobra.ExactArgs(1), Short: "Inspect a managed site and related runtime state", RunE: func(cmd *cobra.Command, args []string) error {
		site, err := loadSite(args[0])
		if err != nil {
			return err
		}
		enabled := false
		if _, err := os.Lstat(site.EnabledPath); err == nil {
			enabled = true
		}
		cert := certinfo.Info{}
		if site.CertificatePath != "" {
			cert = certinfo.Read(site.CertificatePath, site.ACME)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		var dnsResult any
		if result, err := dnscheck.VerifyHostnamePointsHere(ctx, site.Hostname); err == nil {
			dnsResult = result
		} else {
			dnsResult = err.Error()
		}
		report := map[string]any{
			"site": site, "enabled": enabled, "config_exists": fileExists(site.ConfigPath),
			"enabled_path_exists": fileExists(site.EnabledPath), "nginx_active": nginx.ServiceActive(),
			"certificate": cert, "dns": dnsResult,
		}
		if app.jsonOut {
			return writeJSON(report)
		}
		fmt.Printf("Hostname: %s\nBackend: %s\nEnabled: %v\nConfig: %s\nSymlink: %s\nNginx active: %v\nCert: %s\nDNS: %v\n",
			site.Hostname, site.BackendURL(), enabled, site.ConfigPath, site.EnabledPath, nginx.ServiceActive(), certinfo.Summary(cert), dnsResult)
		return nil
	}}
}
