package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/acme"
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

type createOptions struct {
	hostname, backendHost, backendScheme, profile           string
	alias, group, tags                                      string
	clientMaxBodySize, certPath, keyPath                    string
	acmeMethod, acmeCA, dnsProvider, email, securityHeaders string
	backendPort                                             int
	ssl, acme, redirectHTTPS, websocket, http2              bool
	dryRun, force, noReload, noBackup                       bool
	nonInteractive, accessLog, errorLog, assumeYes          bool
}

var createOpts createOptions

func createCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "create", Short: "Create a reverse proxy site", RunE: runCreate}
	bindCreateFlags(cmd, &createOpts)
	return cmd
}

func bindCreateFlags(cmd *cobra.Command, o *createOptions) {
	cmd.Flags().StringVar(&o.hostname, "hostname", "", "public hostname")
	cmd.Flags().StringVar(&o.backendHost, "backend-host", "", "backend host")
	cmd.Flags().IntVar(&o.backendPort, "backend-port", 0, "backend port")
	cmd.Flags().StringVar(&o.backendScheme, "backend-scheme", "http", "backend scheme: http or https")
	cmd.Flags().StringVar(&o.profile, "profile", "generic", "proxy profile")
	cmd.Flags().StringVar(&o.alias, "alias", "", "short site alias")
	cmd.Flags().StringVar(&o.group, "group", "", "site group")
	cmd.Flags().StringVar(&o.tags, "tags", "", "comma-separated site tags")
	cmd.Flags().BoolVar(&o.ssl, "ssl", false, "enable HTTPS")
	cmd.Flags().BoolVar(&o.acme, "acme", false, "use acme.sh")
	cmd.Flags().StringVar(&o.acmeMethod, "acme-method", "", "acme method: dns, http, standalone, tls-alpn")
	cmd.Flags().StringVar(&o.acmeCA, "acme-ca", acme.DefaultCA, "ACME CA: letsencrypt or buypass")
	cmd.Flags().StringVar(&o.dnsProvider, "dns-provider", "", "DNS-01 provider")
	cmd.Flags().StringVar(&o.email, "email", "", "ACME account email")
	cmd.Flags().BoolVar(&o.redirectHTTPS, "redirect-https", false, "redirect HTTP to HTTPS")
	cmd.Flags().BoolVar(&o.websocket, "websocket", false, "enable WebSocket headers")
	cmd.Flags().BoolVar(&o.http2, "http2", false, "enable HTTP/2 on HTTPS listener")
	cmd.Flags().StringVar(&o.clientMaxBodySize, "client-max-body-size", "100M", "Nginx client_max_body_size")
	cmd.Flags().StringVar(&o.certPath, "cert-path", "", "existing fullchain path")
	cmd.Flags().StringVar(&o.keyPath, "key-path", "", "existing private key path")
	cmd.Flags().BoolVar(&o.nonInteractive, "non-interactive", false, "fail instead of prompting")
	cmd.Flags().BoolVar(&o.dryRun, "dry-run", false, "show planned changes without writing")
	cmd.Flags().BoolVar(&o.force, "force", false, "overwrite npc-managed site without prompting")
	cmd.Flags().BoolVar(&o.noReload, "no-reload", false, "write config without reloading nginx")
	cmd.Flags().BoolVar(&o.noBackup, "no-backup", false, "skip backup")
	cmd.Flags().StringVar(&o.securityHeaders, "security-headers", "", "security header profile")
	cmd.Flags().BoolVar(&o.accessLog, "access-log", false, "enable site access log")
	cmd.Flags().BoolVar(&o.errorLog, "error-log", false, "enable site error log")
}

func runCreate(cmd *cobra.Command, args []string) error {
	o := createOpts
	if !o.nonInteractive {
		applyEnvironmentDefaults(&o)
		if err := promptCreate(&o); err != nil {
			return err
		}
	} else if missing := missingCreateFields(o); len(missing) > 0 {
		return validationError{fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))}
	}
	return executeCreate(o)
}

func executeCreate(o createOptions) error {
	applyEnvironmentDefaults(&o)
	site, err := buildSite(o)
	if err != nil {
		return validationError{err}
	}
	content, err := renderer.RenderSite(site)
	if err != nil {
		return err
	}
	if o.dryRun {
		return printCreatePlan(site, content)
	}
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	if err := ensureRuntimeDependencies(o); err != nil {
		return err
	}
	if err := nginx.EnsureServiceRunning(); err != nil {
		return err
	}
	cfg, err := loadManagedConfig()
	if err != nil {
		return err
	}
	if _, exists := cfg.Sites[site.Hostname]; exists && !o.force {
		return validationError{fmt.Errorf("site %s already exists; use edit or --force", site.Hostname)}
	}
	if fileExists(site.ConfigPath) && !o.force {
		if nginx.Managed(site.ConfigPath) {
			return validationError{fmt.Errorf("%s exists and is managed by npc; use edit or --force", site.ConfigPath)}
		}
		return validationError{fmt.Errorf("%s exists and is not managed by npc; import it or choose another hostname", site.ConfigPath)}
	}
	if !o.noBackup {
		if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath); err != nil {
			return err
		}
	}
	if site.SSL && site.ACME && site.ACMEMethod == "http" {
		if err := prepareHTTP01Certificate(site); err != nil {
			return err
		}
		content, err = renderer.RenderSite(site)
		if err != nil {
			return err
		}
	}
	if site.SSL && site.ACME && site.ACMEMethod == "dns" {
		if err := prepareDNS01Certificate(site); err != nil {
			return err
		}
		content, err = renderer.RenderSite(site)
		if err != nil {
			return err
		}
	}
	if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
		return err
	}
	if _, err := revision.Save(site, content); err != nil {
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
	if !o.noReload {
		if _, err := nginx.Reload(); err != nil {
			return err
		}
		site.LastReload = time.Now().UTC().Format(time.RFC3339)
	}
	cfg.Sites[site.Hostname] = site
	if err := config.Save("", cfg); err != nil {
		return err
	}
	fmt.Printf("Created %s -> %s\n", site.Hostname, site.BackendURL())
	return nil
}

func missingCreateFields(o createOptions) []string {
	applyEnvironmentDefaults(&o)
	var missing []string
	if o.hostname == "" {
		missing = append(missing, "--hostname")
	}
	if o.backendHost == "" {
		missing = append(missing, "--backend-host")
	}
	if o.backendPort == 0 {
		missing = append(missing, "--backend-port")
	}
	if o.backendScheme == "" {
		missing = append(missing, "--backend-scheme")
	}
	if o.ssl && !o.acme {
		if o.certPath == "" {
			missing = append(missing, "--cert-path")
		}
		if o.keyPath == "" {
			missing = append(missing, "--key-path")
		}
	}
	if o.ssl && o.acme && o.acmeMethod == "dns" && o.dnsProvider == "" {
		missing = append(missing, "--dns-provider")
	}
	return missing
}

func buildSite(o createOptions) (*config.Site, error) {
	applyProfileDefaults(&o)
	if err := validate.Hostname(o.hostname, true); err != nil {
		return nil, err
	}
	if err := validate.BackendHost(o.backendHost); err != nil {
		return nil, err
	}
	if err := validate.Port(o.backendPort); err != nil {
		return nil, err
	}
	if err := validate.BackendScheme(o.backendScheme); err != nil {
		return nil, err
	}
	if o.ssl && !o.acme && (o.certPath == "" || o.keyPath == "") {
		return nil, fmt.Errorf("--cert-path and --key-path are required when --ssl is used without --acme")
	}
	if o.acme {
		o.acmeCA = acme.NormalizeCA(o.acmeCA)
		if err := acme.ValidateCA(o.acmeCA); err != nil {
			return nil, err
		}
	}
	acmeCA := ""
	if o.acme {
		acmeCA = o.acmeCA
	}
	configPath, enabledPath := nginx.SitePaths(o.hostname)
	now := time.Now().UTC()
	site := &config.Site{
		Hostname: o.hostname, Alias: o.alias, Group: o.group, Tags: splitTags(o.tags),
		BackendScheme: o.backendScheme, BackendHost: o.backendHost,
		BackendPort: o.backendPort, Profile: o.profile, WebSocket: o.websocket, HTTP2: o.http2,
		ClientMaxBodySize: o.clientMaxBodySize, SSL: o.ssl, ACME: o.acme, ACMEMethod: normalizeACME(o.acmeMethod),
		ACMECA: acmeCA, DNSProvider: o.dnsProvider, ACMEEmail: o.email, RedirectHTTPS: o.redirectHTTPS, SecurityHeaders: o.securityHeaders,
		ConfigPath: configPath, EnabledPath: enabledPath, CertificatePath: o.certPath, CertificateKeyPath: o.keyPath,
		CreatedAt: now, UpdatedAt: now, ManagedBy: "npc",
	}
	if o.accessLog {
		site.AccessLog = path.Join("/var/log/nginx", o.hostname+".access.log")
	}
	if o.errorLog {
		site.ErrorLog = path.Join("/var/log/nginx", o.hostname+".error.log")
	}
	if site.ACME && site.SSL {
		site.CertificatePath = path.Join(paths.CertsDir, o.hostname, "fullchain.cer")
		site.CertificateKeyPath = path.Join(paths.CertsDir, o.hostname, o.hostname+".key")
	}
	return site, nil
}

func applyProfileDefaults(o *createOptions) {
	if o.profile == "" {
		o.profile = "generic"
	}
	if o.clientMaxBodySize == "" {
		o.clientMaxBodySize = "100M"
	}
	switch o.profile {
	case "websocket", "node", "grafana":
		o.websocket = true
	case "upload", "nextcloud":
		if o.clientMaxBodySize == "100M" {
			o.clientMaxBodySize = "1G"
		}
	case "wordpress":
		if o.clientMaxBodySize == "100M" {
			o.clientMaxBodySize = "256M"
		}
	case "api":
		if o.securityHeaders == "" {
			o.securityHeaders = "standard"
		}
	case "security-basic":
		if o.securityHeaders == "" {
			o.securityHeaders = "standard"
		}
	}
}

func printCreatePlan(site *config.Site, content string) error {
	plan := map[string]any{"site": site, "files": []string{site.ConfigPath, site.EnabledPath}, "nginx_config": content}
	if app.jsonOut {
		return writeJSON(plan)
	}
	fmt.Printf("Dry run for %s\nConfig: %s\nEnabled symlink: %s\nBackend: %s\n\n%s", site.Hostname, site.ConfigPath, site.EnabledPath, site.BackendURL(), content)
	return nil
}

func normalizeACME(method string) string {
	if method == "dns" {
		return "dns"
	}
	if method == "" {
		return "http"
	}
	return method
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
