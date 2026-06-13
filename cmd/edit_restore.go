package cmd

import (
	"fmt"
	"time"

	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/revision"
	"github.com/brightcolor/npc/internal/system"
	"github.com/brightcolor/npc/internal/validate"
	"github.com/spf13/cobra"
)

func editCommand() *cobra.Command {
	var o createOptions
	cmd := &cobra.Command{Use: "edit <hostname>", Args: cobra.ExactArgs(1), Short: "Edit an existing site", RunE: func(cmd *cobra.Command, args []string) error {
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
		if o.backendHost != "" {
			if err := validate.BackendHost(o.backendHost); err != nil {
				return validationError{err}
			}
			site.BackendHost = o.backendHost
		}
		if o.backendPort != 0 {
			if err := validate.Port(o.backendPort); err != nil {
				return validationError{err}
			}
			site.BackendPort = o.backendPort
		}
		if o.backendScheme != "" {
			if err := validate.BackendScheme(o.backendScheme); err != nil {
				return validationError{err}
			}
			site.BackendScheme = o.backendScheme
		}
		site.WebSocket = o.websocket || site.WebSocket
		if o.clientMaxBodySize != "" {
			site.ClientMaxBodySize = o.clientMaxBodySize
		}
		site.UpdatedAt = time.Now().UTC()
		content, err := renderer.RenderSite(site)
		if err != nil {
			return err
		}
		if o.dryRun {
			return printCreatePlan(site, content)
		}
		if !o.noBackup {
			if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath); err != nil {
				return err
			}
		}
		if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
			return err
		}
		if _, err := revision.Save(site, content); err != nil {
			return err
		}
		out, err := nginx.Test()
		if err != nil {
			return nginxTestError{fmt.Errorf("nginx -t failed, reload skipped: %s", out)}
		}
		site.LastNginxTest = time.Now().UTC().Format(time.RFC3339)
		if !o.noReload {
			if _, err := nginx.Reload(); err != nil {
				return err
			}
			site.LastReload = time.Now().UTC().Format(time.RFC3339)
		}
		return config.Save("", cfg)
	}}
	cmd.Flags().StringVar(&o.backendHost, "backend-host", "", "new backend host")
	cmd.Flags().IntVar(&o.backendPort, "backend-port", 0, "new backend port")
	cmd.Flags().StringVar(&o.backendScheme, "backend-scheme", "", "new backend scheme")
	cmd.Flags().BoolVar(&o.websocket, "websocket", false, "enable WebSocket")
	cmd.Flags().StringVar(&o.clientMaxBodySize, "client-max-body-size", "", "new client_max_body_size")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "show planned config")
	cmd.Flags().BoolVar(&o.noReload, "no-reload", false, "skip reload")
	cmd.Flags().BoolVar(&o.noBackup, "no-backup", false, "skip backup")
	return cmd
}

func restoreCommand() *cobra.Command {
	return &cobra.Command{Use: "restore", Short: "List available backups", RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Use `npc backup list` and `sudo npc backup restore <id-or-path>`.")
		backups, err := backup.List()
		if err != nil {
			return err
		}
		for _, item := range backups {
			fmt.Println(item)
		}
		return nil
	}}
}
