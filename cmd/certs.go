package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/certinfo"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func certsCommand() *cobra.Command {
	root := &cobra.Command{Use: "certs", Short: "List and renew certificates", RunE: listCerts}
	root.AddCommand(renewCertCommand())
	root.AddCommand(issueCertCommand())
	root.AddCommand(setCertCommand(), deleteCertCommand())
	root.AddCommand(&cobra.Command{Use: "renew-all", Short: "Run acme.sh renewal for all certificates", RunE: renewAllCerts})
	return root
}

func setCertCommand() *cobra.Command {
	var certPath, keyPath, method, ca, provider string
	var acmeManaged, manual, noReload, noBackup bool
	cmd := &cobra.Command{Use: "set <hostname>", Args: cobra.ExactArgs(1), Short: "Update certificate metadata and rewrite the Nginx site", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		site, ok := cfg.FindSite(args[0])
		if !ok {
			return validationError{fmt.Errorf("site %s is not managed by npc", args[0])}
		}
		if certPath != "" {
			site.CertificatePath = certPath
		}
		if keyPath != "" {
			site.CertificateKeyPath = keyPath
		}
		if site.CertificatePath == "" || site.CertificateKeyPath == "" {
			return validationError{fmt.Errorf("certificate path and key path are required")}
		}
		site.SSL = true
		if acmeManaged {
			site.ACME = true
		}
		if manual {
			site.ACME = false
			site.ACMEMethod = ""
			site.ACMECA = ""
			site.DNSProvider = ""
		}
		if method != "" {
			site.ACMEMethod = normalizeACME(method)
		}
		if ca != "" {
			site.ACMECA = ca
		}
		if provider != "" {
			site.DNSProvider = provider
		}
		site.UpdatedAt = time.Now().UTC()
		return rewriteSiteAfterCertChange(cfg, site, !noBackup, !noReload)
	}}
	cmd.Flags().StringVar(&certPath, "cert-path", "", "fullchain certificate path")
	cmd.Flags().StringVar(&keyPath, "key-path", "", "private key path")
	cmd.Flags().BoolVar(&acmeManaged, "acme", false, "mark this certificate as acme.sh managed")
	cmd.Flags().BoolVar(&manual, "manual", false, "mark this certificate as manually managed")
	cmd.Flags().StringVar(&method, "acme-method", "", "acme method metadata")
	cmd.Flags().StringVar(&ca, "acme-ca", "", "acme CA metadata")
	cmd.Flags().StringVar(&provider, "dns-provider", "", "DNS provider metadata")
	cmd.Flags().BoolVar(&noReload, "no-reload", false, "skip nginx reload")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "skip backup")
	return cmd
}

func deleteCertCommand() *cobra.Command {
	var force, deleteFiles, keepAcme, noReload, noBackup bool
	cmd := &cobra.Command{Use: "delete <hostname>", Args: cobra.ExactArgs(1), Short: "Remove certificate metadata, acme.sh registration, and optionally files", RunE: func(cmd *cobra.Command, args []string) error {
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		if !force {
			return validationError{fmt.Errorf("certificate deletion is destructive; rerun with --force")}
		}
		cfg, err := loadManagedConfig()
		if err != nil {
			return err
		}
		site, ok := cfg.FindSite(args[0])
		if !ok {
			return validationError{fmt.Errorf("site %s is not managed by npc", args[0])}
		}
		certPath, keyPath := site.CertificatePath, site.CertificateKeyPath
		if !noBackup {
			if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath, certPath, keyPath); err != nil {
				return err
			}
		}
		if site.ACME && !keepAcme {
			if err := removeAcmeCert(site); err != nil {
				return err
			}
		}
		if deleteFiles {
			_ = os.Remove(certPath)
			_ = os.Remove(keyPath)
		}
		site.SSL = false
		site.ACME = false
		site.ACMEMethod = ""
		site.ACMECA = ""
		site.DNSProvider = ""
		site.CertificatePath = ""
		site.CertificateKeyPath = ""
		site.RedirectHTTPS = false
		site.HTTP2 = false
		site.UpdatedAt = time.Now().UTC()
		return rewriteSiteAfterCertChange(cfg, site, false, !noReload)
	}}
	cmd.Flags().BoolVar(&force, "force", false, "confirm certificate deletion")
	cmd.Flags().BoolVar(&deleteFiles, "files", false, "delete certificate and key files from disk")
	cmd.Flags().BoolVar(&keepAcme, "keep-acme", false, "do not remove the cert from acme.sh")
	cmd.Flags().BoolVar(&noReload, "no-reload", false, "skip nginx reload")
	cmd.Flags().BoolVar(&noBackup, "no-backup", false, "skip backup")
	return cmd
}

func renewCertCommand() *cobra.Command {
	var expiring bool
	var days int
	cmd := &cobra.Command{Use: "renew [hostname]", Short: "Renew one certificate or expiring managed certificates", RunE: func(cmd *cobra.Command, args []string) error {
		if expiring {
			if len(args) != 0 {
				return validationError{fmt.Errorf("--expiring does not accept a hostname")}
			}
			return renewExpiringCerts(days)
		}
		if len(args) != 1 {
			return validationError{fmt.Errorf("expected hostname or --expiring")}
		}
		return renewCert(cmd, args)
	}}
	cmd.Flags().BoolVar(&expiring, "expiring", false, "renew managed ACME certificates expiring soon")
	cmd.Flags().IntVar(&days, "days", 30, "expiry threshold for --expiring")
	return cmd
}

func listCerts(cmd *cobra.Command, args []string) error {
	cfg, err := loadManagedConfig()
	if err != nil {
		return err
	}
	if app.jsonOut {
		rows := map[string]certinfo.Info{}
		for _, site := range cfg.SortedSites() {
			if site.CertificatePath != "" {
				rows[site.Hostname] = certinfo.Read(site.CertificatePath, site.ACME)
			}
		}
		return writeJSON(rows)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "HOSTNAME\tSSL\tACME\tMETHOD\tEXPIRES\tISSUER\tCERTIFICATE")
	for _, site := range cfg.SortedSites() {
		info := certinfo.Info{}
		if site.CertificatePath != "" {
			info = certinfo.Read(site.CertificatePath, site.ACME)
		}
		issuer := info.Issuer
		if issuer == "" {
			issuer = "-"
		}
		fmt.Fprintf(w, "%s\t%v\t%v\t%s\t%s\t%s\t%s\n", site.Hostname, site.SSL, site.ACME, site.ACMEMethod, certinfo.Summary(info), issuer, site.CertificatePath)
	}
	return w.Flush()
}

func renewCert(cmd *cobra.Command, args []string) error {
	site, err := loadSite(args[0])
	if err != nil {
		return err
	}
	if !site.ACME {
		return validationError{fmt.Errorf("%s is not configured for acme.sh", site.Hostname)}
	}
	return renewOneSite(site)
}

func renewAllCerts(cmd *cobra.Command, args []string) error {
	if !acme.Installed() {
		return validationError{fmt.Errorf("acme.sh was not found")}
	}
	res, err := system.Run(acme.CommandPath(), "--cron")
	fmt.Println(res.Output)
	if err != nil {
		return fmt.Errorf("acme.sh renew-all failed: %w%s", err, acme.DiagnoseOutput(res.Output))
	}
	return err
}

func renewExpiringCerts(days int) error {
	cfg, err := loadManagedConfig()
	if err != nil {
		return err
	}
	renewed := 0
	for _, site := range cfg.SortedSites() {
		if !site.ACME {
			continue
		}
		info := certinfo.Read(site.CertificatePath, site.ACME)
		if !info.Exists || info.DaysLeft <= days {
			if err := renewOneSite(site); err != nil {
				return err
			}
			renewed++
		}
	}
	fmt.Printf("Renewed %d expiring certificate(s)\n", renewed)
	return nil
}

func renewOneSite(site *config.Site) error {
	if !acme.Installed() {
		return validationError{fmt.Errorf("acme.sh was not found")}
	}
	res, err := system.Run(acme.CommandPath(), "--renew", "-d", site.Hostname)
	fmt.Println(res.Output)
	if err != nil {
		return fmt.Errorf("acme.sh renew failed for %s: %w%s", site.Hostname, err, acme.DiagnoseOutput(res.Output))
	}
	return nil
}

func removeAcmeCert(site *config.Site) error {
	if !acme.Installed() {
		return validationError{fmt.Errorf("acme.sh was not found")}
	}
	args := []string{"--remove", "-d", site.Hostname}
	if strings.Contains(site.CertificatePath, "_ecc") || strings.Contains(site.CertificateKeyPath, "_ecc") {
		args = append(args, "--ecc")
	}
	res, err := system.Run(acme.CommandPath(), args...)
	fmt.Println(res.Output)
	if err != nil {
		return fmt.Errorf("acme.sh remove failed for %s: %w%s", site.Hostname, err, acme.DiagnoseOutput(res.Output))
	}
	return nil
}

func rewriteSiteAfterCertChange(cfg *config.Config, site *config.Site, doBackup, doReload bool) error {
	content, err := renderer.RenderSite(site)
	if err != nil {
		return err
	}
	if doBackup {
		if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath, site.CertificatePath, site.CertificateKeyPath); err != nil {
			return err
		}
	}
	if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
		return err
	}
	out, err := nginx.Test()
	if err != nil {
		return nginxTestError{fmt.Errorf("nginx -t failed, reload skipped: %s", out)}
	}
	site.LastNginxTest = time.Now().UTC().Format(time.RFC3339)
	if doReload {
		if _, err := nginx.Reload(); err != nil {
			return err
		}
		site.LastReload = time.Now().UTC().Format(time.RFC3339)
	}
	cfg.Sites[site.Hostname] = site
	if err := config.Save("", cfg); err != nil {
		return err
	}
	fmt.Println("Updated certificate settings for", site.Hostname)
	return nil
}
