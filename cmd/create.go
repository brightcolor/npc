package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/system"
	"github.com/brightcolor/npc/internal/validate"
	"github.com/spf13/cobra"
)

type createOptions struct {
	hostname, backendHost, backendScheme, profile   string
	clientMaxBodySize, certPath, keyPath            string
	acmeMethod, dnsProvider, email, securityHeaders string
	backendPort                                     int
	ssl, acme, redirectHTTPS, websocket, http2      bool
	dryRun, force, noReload, noBackup               bool
	nonInteractive, accessLog, errorLog, assumeYes  bool
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
	cmd.Flags().BoolVar(&o.ssl, "ssl", false, "enable HTTPS")
	cmd.Flags().BoolVar(&o.acme, "acme", false, "use acme.sh")
	cmd.Flags().StringVar(&o.acmeMethod, "acme-method", "", "acme method: dns, http, standalone, tls-alpn")
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
		if err := promptCreate(&o); err != nil {
			return err
		}
	} else if missing := missingCreateFields(o); len(missing) > 0 {
		return validationError{fmt.Errorf("missing required fields: %s", strings.Join(missing, ", "))}
	}
	return executeCreate(o)
}

func executeCreate(o createOptions) error {
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
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	if _, exists := cfg.Sites[site.Hostname]; exists && !o.force {
		return validationError{fmt.Errorf("site %s already exists; use edit or --force", site.Hostname)}
	}
	if fileExists(site.ConfigPath) && !nginx.Managed(site.ConfigPath) && !o.force {
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

func ensureRuntimeDependencies(o createOptions) error {
	if !system.Exists("nginx") {
		if o.nonInteractive && !o.force && !o.assumeYes {
			return validationError{fmt.Errorf("nginx is not installed; rerun interactively or use --force to install it")}
		}
		install := o.force || o.assumeYes
		if !install {
			install = promptConfirm("Nginx is not installed. Install it now with apt?", true)
		}
		if !install {
			return validationError{fmt.Errorf("nginx is required before creating a site")}
		}
		fmt.Println("Installing Nginx...")
		if err := nginx.InstallApt(true); err != nil {
			return fmt.Errorf("nginx installation failed: %w", err)
		}
	}
	if o.acme && !acme.Installed() {
		if o.nonInteractive && !o.force && !o.assumeYes {
			return validationError{fmt.Errorf("acme.sh is not installed; rerun interactively or use --force to install it")}
		}
		install := o.force || o.assumeYes
		if !install {
			install = promptConfirm("acme.sh is not installed. Install it now?", true)
		}
		if !install {
			return validationError{fmt.Errorf("acme.sh is required when --acme is enabled")}
		}
		fmt.Println("Installing acme.sh...")
		if err := acme.Install(o.email); err != nil {
			return fmt.Errorf("acme.sh installation failed: %w", err)
		}
	}
	return nil
}

func prepareHTTP01Certificate(site *config.Site) error {
	fmt.Println("Preparing HTTP-01 challenge config...")
	if err := os.MkdirAll("/var/www/html", 0o755); err != nil {
		return err
	}
	challenge := *site
	challenge.SSL = false
	challenge.RedirectHTTPS = false
	challenge.ACME = true
	challenge.ACMEMethod = "http"
	content, err := renderer.RenderSite(&challenge)
	if err != nil {
		return err
	}
	if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
		return err
	}
	if err := nginx.Enable(site.ConfigPath, site.EnabledPath); err != nil {
		return err
	}
	if err := nginx.EnsureServiceRunning(); err != nil {
		return err
	}
	if out, err := nginx.Reload(); err != nil {
		return nginxTestError{fmt.Errorf("nginx challenge config failed, certificate was not requested: %s", out)}
	}
	fmt.Println("Requesting certificate with acme.sh HTTP-01...")
	if err := acme.IssueHTTP(site.Hostname, site.ACMEEmail); err != nil {
		return err
	}
	fmt.Println("Installing certificate into /etc/npc/certs...")
	if err := acme.InstallCert(site.Hostname, site.CertificatePath, site.CertificateKeyPath); err != nil {
		return err
	}
	return nil
}

func missingCreateFields(o createOptions) []string {
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
	configPath, enabledPath := nginx.SitePaths(o.hostname)
	now := time.Now().UTC()
	site := &config.Site{
		Hostname: o.hostname, BackendScheme: o.backendScheme, BackendHost: o.backendHost,
		BackendPort: o.backendPort, Profile: o.profile, WebSocket: o.websocket, HTTP2: o.http2,
		ClientMaxBodySize: o.clientMaxBodySize, SSL: o.ssl, ACME: o.acme, ACMEMethod: normalizeACME(o.acmeMethod),
		DNSProvider: o.dnsProvider, ACMEEmail: o.email, RedirectHTTPS: o.redirectHTTPS, SecurityHeaders: o.securityHeaders,
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

func promptCreate(o *createOptions) error {
	r := bufio.NewReader(os.Stdin)
	ask := func(label, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", label, def)
		} else {
			fmt.Printf("%s: ", label)
		}
		text, _ := r.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			return def
		}
		return text
	}
	o.hostname = ask("Hostname", o.hostname)
	o.backendHost = ask("Backend host", defaultString(o.backendHost, "127.0.0.1"))
	port, err := strconv.Atoi(ask("Backend port", defaultString(strconv.Itoa(o.backendPort), "3000")))
	if err != nil {
		return err
	}
	o.backendPort = port
	o.backendScheme = ask("Backend scheme (http/https)", defaultString(o.backendScheme, "http"))
	o.profile = ask("Profile (generic/websocket/upload/streaming/docker/security-basic)", defaultString(o.profile, "generic"))
	o.websocket = yes(ask("WebSocket support? (y/n)", boolDefault(o.websocket)))
	o.ssl = yes(ask("Enable SSL/TLS? (y/n)", boolDefault(o.ssl)))
	if o.ssl {
		o.redirectHTTPS = yes(ask("Redirect HTTP to HTTPS? (y/n)", boolDefault(true)))
		o.http2 = yes(ask("Enable HTTP/2? (y/n)", boolDefault(true)))
		o.acme = yes(ask("Use acme.sh? (y/n)", boolDefault(o.acme)))
		if o.acme {
			o.acmeMethod = ask("ACME method (http/dns/standalone/tls-alpn)", defaultString(o.acmeMethod, "http"))
			o.email = ask("ACME email", o.email)
			if o.acmeMethod == "dns" {
				o.dnsProvider = ask("DNS provider", o.dnsProvider)
			}
		} else {
			o.certPath = ask("Fullchain path", o.certPath)
			o.keyPath = ask("Private key path", o.keyPath)
		}
	}
	o.clientMaxBodySize = ask("client_max_body_size", defaultString(o.clientMaxBodySize, "100M"))
	return nil
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

func defaultString(v, def string) string {
	if v == "" || v == "0" {
		return def
	}
	return v
}

func boolDefault(v bool) string {
	if v {
		return "y"
	}
	return "n"
}

func yes(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "y" || v == "yes" || v == "ja" || v == "true"
}

func promptConfirm(message string, def bool) bool {
	suffix := " [Y/n]: "
	if !def {
		suffix = " [y/N]: "
	}
	fmt.Print(message + suffix)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return def
	}
	return text == "y" || text == "yes" || text == "ja" || text == "true"
}
