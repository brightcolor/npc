package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/importer"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/system"
)

func webUICertSet(r *http.Request) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	cfg, site, err := webUILoadSite(r.FormValue("hostname"))
	if err != nil {
		return err
	}
	site.CertificatePath = r.FormValue("cert_path")
	site.CertificateKeyPath = r.FormValue("key_path")
	if site.CertificatePath == "" || site.CertificateKeyPath == "" {
		return validationError{fmt.Errorf("certificate path and key path are required")}
	}
	site.SSL = true
	site.ACME = formBool(r, "acme")
	if formBool(r, "manual") {
		site.ACME = false
	}
	site.ACMEMethod = normalizeACME(r.FormValue("acme_method"))
	site.DNSProvider = r.FormValue("dns_provider")
	site.RedirectHTTPS = formBool(r, "redirect_https")
	site.HTTP2 = formBool(r, "http2")
	site.UpdatedAt = time.Now().UTC()
	return rewriteSiteAfterCertChange(cfg, site, true, !formBool(r, "no_reload"))
}

func webUICertDelete(r *http.Request) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	cfg, site, err := webUILoadSite(r.FormValue("hostname"))
	if err != nil {
		return err
	}
	certPath, keyPath := site.CertificatePath, site.CertificateKeyPath
	if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath, certPath, keyPath); err != nil {
		return err
	}
	if site.ACME && !formBool(r, "keep_acme") {
		if err := removeAcmeCert(site); err != nil {
			return err
		}
	}
	if formBool(r, "remove_files") {
		_ = os.Remove(certPath)
		_ = os.Remove(keyPath)
	}
	site.SSL = false
	site.ACME = false
	site.ACMEMethod = ""
	site.DNSProvider = ""
	site.CertificatePath = ""
	site.CertificateKeyPath = ""
	site.RedirectHTTPS = false
	site.HTTP2 = false
	site.UpdatedAt = time.Now().UTC()
	return rewriteSiteAfterCertChange(cfg, site, false, !formBool(r, "no_reload"))
}

func webUIImport(r *http.Request) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	files, _ := filepath.Glob(filepath.Join(paths.NginxSitesAvailable, "*.conf"))
	if one := r.FormValue("path"); one != "" {
		files = []string{one}
	}
	force := formBool(r, "force")
	imported := 0
	for _, file := range files {
		candidate := importer.ParseFile(file)
		if candidate.Error != "" || candidate.Site == nil {
			continue
		}
		if _, exists := cfg.Sites[candidate.Site.Hostname]; exists && !force {
			continue
		}
		cfg.Sites[candidate.Site.Hostname] = candidate.Site
		imported++
	}
	if imported == 0 {
		return validationError{fmt.Errorf("no configs were imported")}
	}
	return config.Save("", cfg)
}
