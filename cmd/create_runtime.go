package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/dnscheck"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/system"
)

func ensureRuntimeDependencies(o createOptions) error {
	if !system.Exists("nginx") {
		if o.nonInteractive && !o.force && !o.assumeYes {
			return validationError{fmt.Errorf("nginx is not installed; rerun interactively or use --force to install it")}
		}
		install := o.force || o.assumeYes || promptConfirm("Nginx is not installed. Install it now with apt?", true)
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
		install := o.force || o.assumeYes || promptConfirm("acme.sh is not installed. Install it now?", true)
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
	fmt.Println("Checking DNS for HTTP-01 validation...")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if result, err := dnscheck.VerifyHostnamePointsHere(ctx, site.Hostname); err != nil {
		if result != nil {
			return networkError{fmt.Errorf("%w", err)}
		}
		return networkError{err}
	}
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
	if err := acme.IssueHTTP(site.Hostname, site.ACMEEmail, site.ACMECA); err != nil {
		return err
	}
	return installACMECert(site)
}

func prepareDNS01Certificate(site *config.Site) error {
	fmt.Println("Requesting certificate with acme.sh DNS-01 provider", site.DNSProvider+"...")
	if err := acme.IssueDNS(site.Hostname, site.DNSProvider, site.ACMEEmail, site.ACMECA); err != nil {
		return err
	}
	return installACMECert(site)
}

func installACMECert(site *config.Site) error {
	fmt.Println("Installing certificate into /etc/npc/certs...")
	return acme.InstallCert(site.Hostname, site.CertificatePath, site.CertificateKeyPath)
}
