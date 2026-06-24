package cmd

import (
	"fmt"
	"path"
	"time"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

type certIssueOptions struct {
	method, provider, email, ca string
	redirectHTTPS, http2        bool
	noReload, noBackup          bool
}

func issueCertCommand() *cobra.Command {
	o := certIssueOptions{method: "http", ca: acme.DefaultCA, redirectHTTPS: true, http2: true}
	cmd := &cobra.Command{Use: "issue <hostname>", Args: cobra.ExactArgs(1), Short: "Issue a certificate for an existing managed site", RunE: func(cmd *cobra.Command, args []string) error {
		return issueCertificateForSite(args[0], o)
	}}
	bindCertIssueFlags(cmd, &o)
	return cmd
}

func bindCertIssueFlags(cmd *cobra.Command, o *certIssueOptions) {
	cmd.Flags().StringVar(&o.method, "method", o.method, "ACME method: http or dns")
	cmd.Flags().StringVar(&o.provider, "dns-provider", o.provider, "DNS provider for DNS-01, for example cloudflare")
	cmd.Flags().StringVar(&o.email, "email", o.email, "ACME account email, optional")
	cmd.Flags().StringVar(&o.ca, "acme-ca", o.ca, "ACME CA: letsencrypt or buypass")
	cmd.Flags().BoolVar(&o.redirectHTTPS, "redirect-https", o.redirectHTTPS, "enable HTTP to HTTPS redirect after issuing")
	cmd.Flags().BoolVar(&o.http2, "http2", o.http2, "enable HTTP/2 after issuing")
	cmd.Flags().BoolVar(&o.noReload, "no-reload", false, "skip nginx reload after writing config")
	cmd.Flags().BoolVar(&o.noBackup, "no-backup", false, "skip backup before writing config")
}

func issueCertificateForSite(name string, o certIssueOptions) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	if !acme.Installed() {
		return validationError{fmt.Errorf("acme.sh was not found; install it first or run an interactive create flow")}
	}
	cfg, err := loadManagedConfig()
	if err != nil {
		return err
	}
	site, ok := cfg.FindSite(name)
	if !ok {
		return validationError{fmt.Errorf("site %s is not managed by npc", name)}
	}
	o.method = normalizeACME(o.method)
	if o.method == "" {
		o.method = "http"
	}
	if err := prepareSiteForIssue(site, o); err != nil {
		return err
	}
	if o.method == "dns" {
		if site.DNSProvider == "" {
			return validationError{fmt.Errorf("--dns-provider is required for DNS-01")}
		}
		if err := prepareDNS01Certificate(site); err != nil {
			return err
		}
	} else if o.method == "http" {
		if err := prepareHTTP01Certificate(site); err != nil {
			return err
		}
	} else {
		return validationError{fmt.Errorf("unsupported ACME method %q", o.method)}
	}
	site.SSL = true
	site.ACME = true
	site.RedirectHTTPS = o.redirectHTTPS
	site.HTTP2 = o.http2
	site.UpdatedAt = time.Now().UTC()
	return rewriteSiteAfterCertChange(cfg, site, !o.noBackup, !o.noReload)
}

func prepareSiteForIssue(site *config.Site, o certIssueOptions) error {
	o.method = normalizeACME(o.method)
	if o.method == "" {
		o.method = "http"
	}
	if o.ca == "" {
		o.ca = acme.DefaultCA
	}
	site.ACME = true
	site.ACMEMethod = o.method
	site.ACMECA = acme.NormalizeCA(o.ca)
	site.ACMEEmail = o.email
	site.DNSProvider = o.provider
	if site.CertificatePath == "" {
		site.CertificatePath = path.Join(paths.CertsDir, site.Hostname, "fullchain.cer")
	}
	if site.CertificateKeyPath == "" {
		site.CertificateKeyPath = path.Join(paths.CertsDir, site.Hostname, site.Hostname+".key")
	}
	return acme.ValidateCA(site.ACMECA)
}
