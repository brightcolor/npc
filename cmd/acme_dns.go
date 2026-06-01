package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/brightcolor/npc/internal/acme"
	"github.com/brightcolor/npc/internal/secrets"
	"github.com/brightcolor/npc/internal/system"
	"github.com/spf13/cobra"
)

func acmeCommand() *cobra.Command {
	root := &cobra.Command{Use: "acme", Short: "Manage acme.sh helper configuration"}
	root.AddCommand(dnsSetupCommand(), cloudflareSetupCommand())
	return root
}

func dnsSetupCommand() *cobra.Command {
	var printOnly bool
	cmd := &cobra.Command{Use: "dns-setup <provider>", Args: cobra.ExactArgs(1), Short: "Create a DNS-01 provider env template", RunE: func(cmd *cobra.Command, args []string) error {
		provider := strings.ToLower(args[0])
		template, err := dnsProviderTemplate(provider)
		if err != nil {
			return validationError{err}
		}
		if printOnly {
			fmt.Print(template)
			return nil
		}
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		path := secrets.EnvPath(provider)
		if fileExists(path) {
			return validationError{fmt.Errorf("%s already exists; edit it manually or move it aside first", path)}
		}
		if err := secrets.WriteProviderEnv(provider, []byte(template)); err != nil {
			return err
		}
		_ = os.Chown(path, 0, 0)
		fmt.Println("Created", path)
		fmt.Println("Edit the placeholder values, keep mode 0600, then use --dns-provider", provider)
		return nil
	}}
	cmd.Flags().BoolVar(&printOnly, "print-template", false, "print template without writing")
	return cmd
}

func dnsProviderTemplate(provider string) (string, error) {
	switch provider {
	case "cloudflare":
		return "CF_Token=\nCF_Account_ID=\n", nil
	case "hetzner":
		return "HETZNER_Token=\n", nil
	case "netcup":
		return "NC_Apikey=\nNC_Apipw=\nNC_CID=\n", nil
	case "ionos":
		return "IONOS_PREFIX=\nIONOS_SECRET=\n", nil
	case "route53":
		return "AWS_ACCESS_KEY_ID=\nAWS_SECRET_ACCESS_KEY=\n", nil
	case "digitalocean":
		return "DO_API_KEY=\n", nil
	case "duckdns":
		return "DuckDNS_Token=\n", nil
	case "custom":
		return "# Add the environment variables required by your acme.sh DNS hook.\n", nil
	default:
		return "", fmt.Errorf("unsupported DNS provider %q; supported: %s", provider, strings.Join(acme.DNSProviders, ", "))
	}
}

func cloudflareSetupCommand() *cobra.Command {
	var token, accountID, zoneID string
	cmd := &cobra.Command{Use: "cloudflare-setup", Short: "Write Cloudflare DNS-01 credentials", RunE: func(cmd *cobra.Command, args []string) error {
		if token == "" || accountID == "" {
			return validationError{fmt.Errorf("--token and --account-id are required")}
		}
		if err := system.RequireRoot(); err != nil {
			return permissionError{err}
		}
		content := cloudflareEnv(token, accountID, zoneID)
		if err := secrets.WriteProviderEnv("cloudflare", []byte(content)); err != nil {
			return err
		}
		fmt.Println("Saved Cloudflare DNS settings to", secrets.EnvPath("cloudflare"))
		return nil
	}}
	cmd.Flags().StringVar(&token, "token", "", "Cloudflare API token with Zone DNS Edit permissions")
	cmd.Flags().StringVar(&accountID, "account-id", "", "Cloudflare account ID")
	cmd.Flags().StringVar(&zoneID, "zone-id", "", "optional Cloudflare zone ID")
	return cmd
}

func cloudflareEnv(token, accountID, zoneID string) string {
	var b strings.Builder
	b.WriteString("CF_Token=" + token + "\n")
	b.WriteString("CF_Account_ID=" + accountID + "\n")
	if zoneID != "" {
		b.WriteString("CF_Zone_ID=" + zoneID + "\n")
	}
	return b.String()
}
