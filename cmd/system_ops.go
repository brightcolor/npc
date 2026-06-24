package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/config"
	dockerapi "github.com/brightcolor/npc/internal/docker"
	"github.com/brightcolor/npc/internal/importer"
	"github.com/brightcolor/npc/internal/installer"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func runInstall(cmd *cobra.Command, args []string) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	if err := installer.InstallCurrentBinary(); err != nil {
		return err
	}
	fmt.Println("Installed npc to", paths.InstallPath)
	return nil
}

func backupCommand() *cobra.Command {
	var q siteQuery
	root := &cobra.Command{Use: "backup", Short: "Back up npc-managed files", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		files := []string{paths.ConfigFile}
		for _, site := range q.apply(cfg.SortedSites()) {
			files = append(files, site.ConfigPath, site.EnabledPath)
		}
		set, err := backup.Create(files...)
		if err != nil {
			return err
		}
		if app.jsonOut {
			return writeJSON(set)
		}
		fmt.Println("Backup created:", set.Dir)
		return nil
	}}
	bindSiteQueryFlags(root, &q)
	root.AddCommand(&cobra.Command{Use: "list", Short: "List backups", RunE: func(cmd *cobra.Command, args []string) error {
		backups, err := backup.List()
		if err != nil {
			return err
		}
		if app.jsonOut {
			return writeJSON(backups)
		}
		for _, item := range backups {
			fmt.Println(item)
		}
		return nil
	}})
	root.AddCommand(&cobra.Command{Use: "restore <id-or-path>", Args: cobra.ExactArgs(1), Short: "Restore a backup by id or path", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		restored, err := backup.Restore(args[0])
		if err != nil {
			return err
		}
		fmt.Println("Restored files:")
		for _, item := range restored {
			fmt.Println(" ", item)
		}
		return nil
	}})
	return root
}

func uninstallCommand() *cobra.Command {
	var force bool
	cmd := &cobra.Command{Use: "uninstall", Short: "Uninstall npc components", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		if !force {
			return validationError{fmt.Errorf("uninstall is destructive; rerun with --force and remove selected files manually if needed")}
		}
		_ = os.Remove(paths.InstallPath)
		fmt.Println("Removed", paths.InstallPath)
		return nil
	}}
	cmd.Flags().BoolVar(&force, "force", false, "confirm binary removal")
	return cmd
}

func doctorCommand() *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Run diagnostics", RunE: func(cmd *cobra.Command, args []string) error {
		report := map[string]any{
			"root":             system.IsRoot(),
			"systemctl":        system.Exists("systemctl"),
			"nginx":            system.Exists("nginx"),
			"nginx_version":    nginx.Version(),
			"nginx_active":     nginx.ServiceActive(),
			"config_file":      paths.ConfigFile,
			"sites_available":  dirExists(paths.NginxSitesAvailable),
			"sites_enabled":    dirExists(paths.NginxSitesEnabled),
			"acme_sh":          system.Exists("acme.sh"),
			"docker":           system.Exists("docker"),
			"ufw":              system.Exists("ufw"),
			"firewalld":        system.Exists("firewall-cmd"),
			"nft":              system.Exists("nft"),
			"vibecoded_notice": "This project started from a broad generated specification; review configs before production use.",
		}
		if app.jsonOut {
			return writeJSON(report)
		}
		for k, v := range report {
			fmt.Printf("%-20s %v\n", k+":", v)
		}
		if out, err := nginx.Test(); err != nil {
			fmt.Println("nginx -t:", out)
		}
		return nil
	}}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func exportCommand() *cobra.Command {
	return &cobra.Command{Use: "export", Short: "Export npc configuration", RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(paths.ConfigFile)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	}}
}

func importCommand() *cobra.Command {
	var yes bool
	var force bool
	var onePath string
	cmd := &cobra.Command{Use: "import", Short: "Inspect existing Nginx sites for import", RunE: func(cmd *cobra.Command, args []string) error {
		files, _ := filepath.Glob(filepath.Join(paths.NginxSitesAvailable, "*.conf"))
		if onePath != "" {
			files = []string{onePath}
		}
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		imported := 0
		for _, file := range files {
			candidate := importer.ParseFile(file)
			status := "manual"
			if candidate.Managed {
				status = "managed-by-npc"
			}
			if candidate.Error != "" {
				fmt.Printf("%s\t%s\t%s\n", file, status, candidate.Error)
				continue
			}
			fmt.Printf("%s\t%s\t%s -> %s\n", file, status, candidate.Site.Hostname, candidate.Site.BackendURL())
			if yes {
				if _, exists := cfg.Sites[candidate.Site.Hostname]; exists && !force {
					continue
				}
				cfg.Sites[candidate.Site.Hostname] = candidate.Site
				imported++
			}
		}
		if yes {
			if err := config.Save("", cfg); err != nil {
				return err
			}
			fmt.Printf("Imported %d site(s)\n", imported)
			return nil
		}
		fmt.Println("No files were imported. Re-run with --yes after reviewing candidates.")
		return nil
	}}
	cmd.Flags().BoolVar(&yes, "yes", false, "import detected manual sites into npc metadata")
	cmd.Flags().BoolVar(&force, "force", false, "replace existing npc metadata with parsed config metadata")
	cmd.Flags().StringVar(&onePath, "path", "", "import one explicit Nginx config path instead of scanning sites-available")
	return cmd
}

func dockerCommand() *cobra.Command {
	return &cobra.Command{Use: "docker", Short: "Show Docker containers and ports", RunE: func(cmd *cobra.Command, args []string) error {
		if !dockerapi.Installed() {
			return validationError{fmt.Errorf("docker was not found")}
		}
		containers, err := dockerapi.RunningContainers()
		if err != nil {
			return err
		}
		if app.jsonOut {
			return writeJSON(containers)
		}
		for _, container := range containers {
			fmt.Printf("%s\t%s\t%s\t%s\n", container.Name, container.Image, container.PortsRaw, container.Networks)
		}
		return nil
	}}
}
