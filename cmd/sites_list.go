package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func listCommand() *cobra.Command {
	var q siteQuery
	var wide bool
	cmd := &cobra.Command{Use: "list", Short: "List npc-managed sites", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		sites := q.apply(cfg.SortedSites())
		if app.jsonOut {
			return writeJSON(sites)
		}
		if len(cfg.Sites) == 0 {
			fmt.Println("No npc-managed sites found.")
			fmt.Println("Create one with `sudo npc create` or open the UI with `npc`.")
			return nil
		}
		if len(sites) == 0 {
			fmt.Println("No sites matched the selected filters.")
			return nil
		}
		return printSiteList(sites, wide)
	}}
	bindSiteQueryFlags(cmd, &q)
	cmd.Flags().BoolVar(&wide, "wide", false, "show alias, group, tags, profile, and certificate path")
	return cmd
}

func printSiteList(sites []*config.Site, wide bool) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	if wide {
		fmt.Fprintln(w, "HOSTNAME\tALIAS\tGROUP\tTAGS\tENABLED\tSSL\tPROFILE\tBACKEND\tCERT")
	} else {
		fmt.Fprintln(w, "HOSTNAME\tSTATE\tSSL\tCERT\tBACKEND")
	}
	for _, site := range sites {
		state := "off"
		if siteEnabled(site) {
			state = "on"
		}
		cert := "-"
		if site.CertificatePath != "" {
			cert = fmt.Sprintf("%dd", certDays(site))
		}
		if wide {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%v\t%s\t%s\t%s\n",
				site.Hostname, site.Alias, site.Group, strings.Join(site.Tags, ","), state, site.SSL, site.Profile, site.BackendURL(), site.CertificatePath)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%v\t%s\t%s\n", site.Hostname, state, site.SSL, cert, site.BackendURL())
		}
	}
	return w.Flush()
}

func bindSiteQueryFlags(cmd *cobra.Command, q *siteQuery) {
	cmd.Flags().BoolVar(&q.enabled, "enabled", false, "show only enabled sites")
	cmd.Flags().BoolVar(&q.disabled, "disabled", false, "show only disabled sites")
	cmd.Flags().BoolVar(&q.sslOnly, "ssl", false, "show only HTTPS sites")
	cmd.Flags().BoolVar(&q.noSSL, "no-ssl", false, "show only HTTP-only sites")
	cmd.Flags().BoolVar(&q.archived, "archived", false, "show only archived sites")
	cmd.Flags().BoolVar(&q.includeArchived, "all", false, "include archived sites")
	cmd.Flags().StringVar(&q.profile, "profile", "", "filter by profile")
	cmd.Flags().StringVar(&q.domain, "domain", "", "filter by hostname suffix")
	cmd.Flags().StringVar(&q.backend, "backend", "", "filter by backend URL fragment")
	cmd.Flags().StringVar(&q.group, "group", "", "filter by group")
	cmd.Flags().StringVar(&q.tag, "tag", "", "filter by tag")
	cmd.Flags().StringVar(&q.sortBy, "sort", "hostname", "sort by hostname, backend, profile, updated, enabled, or cert-expiry")
}

func searchCommand() *cobra.Command {
	return &cobra.Command{Use: "search <query>", Args: cobra.ExactArgs(1), Short: "Search managed sites", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		sites := siteQuery{search: args[0], includeArchived: true}.apply(cfg.SortedSites())
		if app.jsonOut {
			return writeJSON(sites)
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "HOSTNAME\tALIAS\tGROUP\tTAGS\tSTATE\tBACKEND")
		for _, site := range sites {
			state := "off"
			if siteEnabled(site) {
				state = "on"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", site.Hostname, site.Alias, site.Group, strings.Join(site.Tags, ","), state, site.BackendURL())
		}
		return w.Flush()
	}}
}

func statusCommand() *cobra.Command {
	var q siteQuery
	cmd := &cobra.Command{Use: "status", Short: "Show global status", RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		sites := q.apply(cfg.SortedSites())
		status := statusData(sites)
		if app.jsonOut {
			return writeJSON(status)
		}
		for _, key := range []string{"nginx_installed", "nginx_version", "nginx_active", "active_sites", "disabled_sites", "matched_sites", "config_file", "sites_available", "sites_enabled"} {
			fmt.Printf("%-18s %v\n", key+":", status[key])
		}
		return nil
	}}
	bindSiteQueryFlags(cmd, &q)
	return cmd
}

func statusData(sites []*config.Site) map[string]any {
	enabled := 0
	for _, site := range sites {
		if siteEnabled(site) {
			enabled++
		}
	}
	return map[string]any{
		"nginx_installed": system.Exists("nginx"),
		"nginx_version":   nginx.Version(),
		"nginx_active":    nginx.ServiceActive(),
		"active_sites":    enabled,
		"disabled_sites":  len(sites) - enabled,
		"matched_sites":   len(sites),
		"config_file":     paths.ConfigFile,
		"sites_available": paths.NginxSitesAvailable,
		"sites_enabled":   paths.NginxSitesEnabled,
	}
}
