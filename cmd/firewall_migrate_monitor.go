package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/certinfo"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func firewallCommand() *cobra.Command {
	root := &cobra.Command{Use: "firewall", Short: "Show firewall guidance"}
	root.AddCommand(&cobra.Command{Use: "suggest", Short: "Suggest firewall commands without changing rules", RunE: func(cmd *cobra.Command, args []string) error {
		report := firewallSuggestions()
		if app.jsonOut {
			return writeJSON(report)
		}
		fmt.Println("Firewall tools:")
		for k, v := range report["tools"].(map[string]bool) {
			fmt.Printf("  %-10s %v\n", k, v)
		}
		fmt.Println("\nSuggested commands:")
		for _, line := range report["commands"].([]string) {
			fmt.Println("  " + line)
		}
		fmt.Println("\nNotes:")
		for _, line := range report["notes"].([]string) {
			fmt.Println("  - " + line)
		}
		return nil
	}})
	return root
}

func firewallSuggestions() map[string]any {
	tools := map[string]bool{
		"ufw":       system.Exists("ufw"),
		"firewalld": system.Exists("firewall-cmd"),
		"nft":       system.Exists("nft"),
	}
	commands := []string{
		"ufw allow 80/tcp",
		"ufw allow 443/tcp",
		"firewall-cmd --permanent --add-service=http",
		"firewall-cmd --permanent --add-service=https",
		"firewall-cmd --reload",
		"nft list ruleset",
	}
	notes := []string{
		"HTTP-01 needs inbound TCP/80 from the public internet.",
		"Public HTTPS needs inbound TCP/443.",
		"DNS-01 does not need inbound validation ports, but users still need 443 for HTTPS traffic.",
		"npc does not change firewall rules automatically.",
	}
	return map[string]any{"tools": tools, "commands": commands, "notes": notes}
}

func migrateCommand() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{Use: "migrate", Short: "Migrate npc config schema safely", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		report := []string{}
		if cfg.Version < 2 {
			report = append(report, "set config version to 2")
			cfg.Version = 2
		}
		for _, dir := range []string{paths.EtcDir, paths.SecretsDir, paths.CertsDir, paths.BackupsDir, paths.StateDir, paths.MaintenanceDir} {
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				report = append(report, "create "+dir)
			}
		}
		if app.jsonOut {
			return writeJSON(map[string]any{"version": cfg.Version, "planned_changes": report, "dry_run": dryRun})
		}
		if len(report) == 0 {
			fmt.Println("No config migrations needed.")
			return nil
		}
		for _, item := range report {
			fmt.Println(item)
		}
		if dryRun {
			return nil
		}
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		for _, dir := range []string{paths.EtcDir, paths.SecretsDir, paths.CertsDir, paths.BackupsDir, paths.StateDir, paths.MaintenanceDir} {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
		}
		return config.Save("", cfg)
	}}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show migrations without writing")
	return cmd
}

func monitorCommand() *cobra.Command {
	var prometheus bool
	var onlyProblems bool
	cmd := &cobra.Command{Use: "monitor", Aliases: []string{"health"}, Short: "Print health and monitoring output", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load("")
		if err != nil {
			return err
		}
		enabled := 0
		for _, site := range cfg.Sites {
			if fileExists(site.EnabledPath) {
				enabled++
			}
		}
		nginxOK := 0
		if _, err := nginx.Test(); err == nil {
			nginxOK = 1
		}
		data := map[string]any{
			"timestamp":      time.Now().UTC().Format(time.RFC3339),
			"nginx_active":   nginx.ServiceActive(),
			"nginx_test_ok":  nginxOK == 1,
			"sites_total":    len(cfg.Sites),
			"sites_enabled":  enabled,
			"sites_disabled": len(cfg.Sites) - enabled,
			"site_problems":  siteProblems(cfg, onlyProblems),
		}
		if app.jsonOut {
			return writeJSON(data)
		}
		if prometheus {
			fmt.Printf("npc_nginx_active %d\n", boolInt(nginx.ServiceActive()))
			fmt.Printf("npc_nginx_test_ok %d\n", nginxOK)
			fmt.Printf("npc_sites_total %d\n", len(cfg.Sites))
			fmt.Printf("npc_sites_enabled %d\n", enabled)
			fmt.Printf("npc_sites_disabled %d\n", len(cfg.Sites)-enabled)
			return nil
		}
		if onlyProblems {
			return printSiteProblems(data["site_problems"].([]map[string]any))
		}
		for k, v := range data {
			if k == "site_problems" {
				continue
			}
			fmt.Printf("%-18s %v\n", strings.ReplaceAll(k, "_", " ")+":", v)
		}
		return nil
	}}
	cmd.Flags().BoolVar(&prometheus, "prometheus", false, "print Prometheus text format")
	cmd.Flags().BoolVar(&onlyProblems, "only-problems", false, "show only sites with health problems")
	return cmd
}

func siteProblems(cfg *config.Config, onlyProblems bool) []map[string]any {
	var rows []map[string]any
	for _, site := range cfg.SortedSites() {
		var problems []string
		if !siteEnabled(site) && !site.Archived {
			problems = append(problems, "disabled")
		}
		if !fileExists(site.ConfigPath) {
			problems = append(problems, "missing-config")
		}
		if site.SSL {
			info := certinfo.Read(site.CertificatePath, site.ACME)
			if !info.Exists {
				problems = append(problems, "missing-cert")
			} else if info.ParseError != "" {
				problems = append(problems, "invalid-cert")
			} else if info.DaysLeft <= 30 {
				problems = append(problems, fmt.Sprintf("cert-expiring-%dd", info.DaysLeft))
			}
		}
		if onlyProblems && len(problems) == 0 {
			continue
		}
		rows = append(rows, map[string]any{"hostname": site.Hostname, "alias": site.Alias, "group": site.Group, "problems": problems})
	}
	return rows
}

func printSiteProblems(rows []map[string]any) error {
	if len(rows) == 0 {
		fmt.Println("No site problems found.")
		return nil
	}
	for _, row := range rows {
		fmt.Printf("%-34s %s\n", row["hostname"], strings.Join(anyStrings(row["problems"]), ", "))
	}
	return nil
}

func anyStrings(v any) []string {
	items, _ := v.([]string)
	return items
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
