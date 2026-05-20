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
	root.AddCommand(&cobra.Command{Use: "renew <hostname>", Short: "Renew one managed ACME certificate", Args: cobra.ExactArgs(1), RunE: renewCert})
	root.AddCommand(&cobra.Command{Use: "renew-all", Short: "Run acme.sh renewal for all certificates", RunE: renewAllCerts})
	return root
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
	if !acme.Installed() {
		return validationError{fmt.Errorf("acme.sh was not found")}
	}
	res, err := system.Run(acme.CommandPath(), "--renew", "-d", site.Hostname)
	fmt.Println(res.Output)
	if err != nil {
		return fmt.Errorf("acme.sh renew failed: %w%s", err, acme.DiagnoseOutput(res.Output))
	}
	return err
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
