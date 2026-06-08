package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/certinfo"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func certsCommand() *cobra.Command {
	root := &cobra.Command{Use: "certs", Short: "List and renew certificates", RunE: listCerts}
	root.AddCommand(renewCertCommand())
	root.AddCommand(&cobra.Command{Use: "renew-all", Short: "Run acme.sh renewal for all certificates", RunE: renewAllCerts})
	return root
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
	cfg, err := config.Load("")
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
	cfg, err := config.Load("")
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
