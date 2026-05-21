package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/revision"
	"github.com/brightcolor/npc/internal/system"
	"github.com/brightcolor/npc/internal/textdiff"
	"github.com/spf13/cobra"
)

func diffCommand() *cobra.Command {
	var revID string
	cmd := &cobra.Command{Use: "diff <hostname>", Args: cobra.ExactArgs(1), Short: "Compare live, rendered, and revision configs", RunE: func(cmd *cobra.Command, args []string) error {
		site, err := loadSite(args[0])
		if err != nil {
			return err
		}
		rendered, err := renderer.RenderSite(site)
		if err != nil {
			return err
		}
		live, _ := os.ReadFile(site.ConfigPath)
		if app.jsonOut {
			return writeJSON(map[string]any{"hostname": site.Hostname, "config_path": site.ConfigPath, "live_matches_rendered": string(live) == rendered})
		}
		fmt.Print(textdiff.Unified("live:"+site.ConfigPath, string(live), "rendered:"+site.Hostname, rendered))
		rev, err := selectedRevision(site.Hostname, revID)
		if err == nil && rev.Config != "" {
			data, _ := os.ReadFile(rev.Config)
			fmt.Println()
			fmt.Print(textdiff.Unified("revision:"+rev.ID, string(data), "rendered:"+site.Hostname, rendered))
		}
		return nil
	}}
	cmd.Flags().StringVar(&revID, "revision", "", "revision id to compare; defaults to latest")
	return cmd
}

func rollbackCommand() *cobra.Command {
	var revID string
	var dryRun bool
	cmd := &cobra.Command{Use: "rollback <hostname>", Args: cobra.ExactArgs(1), Short: "Restore a previous config revision safely", RunE: func(cmd *cobra.Command, args []string) error {
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
		rev, err := selectedRevision(site.Hostname, revID)
		if err != nil {
			return validationError{fmt.Errorf("revision not found for %s", site.Hostname)}
		}
		data, err := os.ReadFile(rev.Config)
		if err != nil {
			return err
		}
		if dryRun {
			fmt.Printf("Would restore revision %s to %s\n\n%s", rev.ID, site.ConfigPath, string(data))
			return nil
		}
		if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath); err != nil {
			return err
		}
		restoredSite, err := revision.LoadSite(*rev)
		if err != nil {
			return err
		}
		restoredSite.ConfigPath = site.ConfigPath
		restoredSite.EnabledPath = site.EnabledPath
		if err := nginx.WriteSite(site.ConfigPath, string(data)); err != nil {
			return err
		}
		out, err := nginx.Test()
		if err != nil {
			return nginxTestError{fmt.Errorf("nginx -t failed after rollback, reload skipped: %s", out)}
		}
		restoredSite.LastNginxTest = time.Now().UTC().Format(time.RFC3339)
		if _, err := nginx.Reload(); err != nil {
			return err
		}
		restoredSite.LastReload = time.Now().UTC().Format(time.RFC3339)
		restoredSite.UpdatedAt = time.Now().UTC()
		cfg.Sites[site.Hostname] = restoredSite
		return config.Save("", cfg)
	}}
	cmd.Flags().StringVar(&revID, "revision", "", "revision id to restore; defaults to latest")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be restored")
	return cmd
}

func selectedRevision(hostname, id string) (*revision.Revision, error) {
	if id != "" {
		return revision.Find(hostname, id)
	}
	rev, err := revision.Latest(hostname)
	if rev == nil && err == nil {
		err = errors.New("no revisions found")
	}
	return rev, err
}
