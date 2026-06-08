package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/revision"
	"github.com/brightcolor/npc/internal/system"
	"github.com/brightcolor/npc/internal/validate"
)

func (ui terminalUI) selectManagedSite(title string) (*config.Config, *config.Site, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, err
	}
	query := ui.askDefault("Search/filter sites, optional", "")
	sites := siteQuery{search: query}.apply(cfg.SortedSites())
	if len(sites) == 0 {
		fmt.Println(emptyState("No matching managed sites", "Clear the search or create/import a site first."))
		return cfg, nil, nil
	}
	labels := make([]string, 0, len(sites))
	for _, site := range sites {
		state := fail("off")
		if siteEnabled(site) {
			state = ok("on")
		}
		problems := siteProblemLabels(site)
		if problems == "" {
			problems = dim("ok")
		} else {
			problems = warn(problems)
		}
		labels = append(labels, fmt.Sprintf("%-30s %-10s %-18s %s", site.Hostname, state, problems, dim(site.BackendURL())))
	}
	return cfg, sites[ui.menu(title, labels)], nil
}

func siteProblemLabels(site *config.Site) string {
	var labels []string
	if !siteEnabled(site) {
		labels = append(labels, "disabled")
	}
	if site.SSL && certDays(site) <= 30 {
		labels = append(labels, "cert")
	}
	if site.Archived {
		labels = append(labels, "archived")
	}
	return strings.Join(labels, ",")
}

func (ui terminalUI) editManagedSite() error {
	cfg, original, err := ui.selectManagedSite("Select a site to edit")
	if err != nil || original == nil {
		return err
	}
	site := *original
	fmt.Println(panel("Current Site",
		"Hostname: "+site.Hostname,
		"Backend:  "+site.BackendURL(),
		"Profile:  "+site.Profile,
		"Config:   "+site.ConfigPath,
	))
	fmt.Println()
	backendHost := ui.askDefault("Backend host", site.BackendHost)
	if err := validate.BackendHost(backendHost); err != nil {
		return validationError{err}
	}
	backendPortText := ui.askDefault("Backend port", strconv.Itoa(site.BackendPort))
	backendPort, err := strconv.Atoi(backendPortText)
	if err != nil {
		return validationError{fmt.Errorf("backend port must be a number")}
	}
	if err := validate.Port(backendPort); err != nil {
		return validationError{err}
	}
	backendScheme := ui.askDefault("Backend scheme", site.BackendScheme)
	if err := validate.BackendScheme(backendScheme); err != nil {
		return validationError{err}
	}
	site.BackendHost = backendHost
	site.BackendPort = backendPort
	site.BackendScheme = backendScheme
	site.Profile = ui.askDefault("Profile", defaultString(site.Profile, "generic"))
	site.Alias = ui.askDefault("Alias, optional", site.Alias)
	site.Group = ui.askDefault("Group, optional", site.Group)
	site.Tags = splitTags(ui.askDefault("Tags, comma-separated optional", strings.Join(site.Tags, ",")))
	site.ClientMaxBodySize = ui.askDefault("client_max_body_size", defaultString(site.ClientMaxBodySize, "100M"))
	site.WebSocket = ui.confirm("Enable WebSocket headers?", site.WebSocket)
	site.SecurityHeaders = ui.askDefault("Security headers profile, empty/standard", site.SecurityHeaders)
	if ui.confirm("Enable per-site access log?", site.AccessLog != "") && site.AccessLog == "" {
		site.AccessLog = "/var/log/nginx/" + site.Hostname + ".access.log"
	}
	if ui.confirm("Enable per-site error log?", site.ErrorLog != "") && site.ErrorLog == "" {
		site.ErrorLog = "/var/log/nginx/" + site.Hostname + ".error.log"
	}
	site.UpdatedAt = time.Now().UTC()
	content, err := renderer.RenderSite(&site)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println(panel("Edit Review",
		"Hostname: "+site.Hostname,
		"Backend:  "+site.BackendURL(),
		"Profile:  "+site.Profile,
		"Config:   "+site.ConfigPath,
	))
	if ui.confirm("Show rendered Nginx config?", false) {
		fmt.Println()
		fmt.Println(content)
	}
	if !ui.confirm("Apply these changes now?", true) {
		fmt.Println("No changes were made.")
		return nil
	}
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath); err != nil {
		return err
	}
	if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
		return err
	}
	if _, err := revision.Save(&site, content); err != nil {
		return err
	}
	out, err := nginx.Test()
	if err != nil {
		return nginxTestError{fmt.Errorf("nginx -t failed, reload skipped: %s", out)}
	}
	site.LastNginxTest = time.Now().UTC().Format(time.RFC3339)
	if _, err := nginx.Reload(); err != nil {
		return err
	}
	site.LastReload = time.Now().UTC().Format(time.RFC3339)
	cfg.Sites[site.Hostname] = &site
	if err := config.Save("", cfg); err != nil {
		return err
	}
	fmt.Println(ok("Updated " + site.Hostname))
	return nil
}

func (ui terminalUI) deleteManagedSite() error {
	cfg, site, err := ui.selectManagedSite("Select a site to delete")
	if err != nil || site == nil {
		return err
	}
	fmt.Println(panel("Delete Review",
		"Hostname: "+site.Hostname,
		"Backend:  "+site.BackendURL(),
		"Config:   "+site.ConfigPath,
		"Enabled:  "+site.EnabledPath,
		"Cert:     "+site.CertificatePath,
	))
	fmt.Println(warn("This can remove files from disk. A backup is recommended before continuing."))
	backupFirst := ui.confirm("Create a backup first?", true)
	removeConfig := ui.confirm("Delete Nginx config file?", true)
	removeMetadata := ui.confirm("Delete npc metadata entry?", true)
	removeCerts := ui.confirm("Delete certificate files?", false)
	if !ui.confirm("Disable the site and apply deletion now?", false) {
		fmt.Println("No changes were made.")
		return nil
	}
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	if backupFirst {
		if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath, site.CertificatePath, site.CertificateKeyPath); err != nil {
			return err
		}
	}
	_ = nginx.Disable(site.EnabledPath)
	if removeConfig {
		_ = os.Remove(site.ConfigPath)
	}
	if removeCerts {
		_ = os.Remove(site.CertificatePath)
		_ = os.Remove(site.CertificateKeyPath)
	}
	if removeMetadata {
		delete(cfg.Sites, site.Hostname)
		if err := config.Save("", cfg); err != nil {
			return err
		}
	}
	out, err := nginx.Test()
	if err != nil {
		return nginxTestError{fmt.Errorf("nginx -t failed, reload skipped: %s", out)}
	}
	if _, err := nginx.Reload(); err != nil {
		return err
	}
	fmt.Println(ok("Deleted/disabled " + site.Hostname))
	return nil
}
