package cmd

import (
	"fmt"

	"github.com/brightcolor/npc/internal/secrets"
	"github.com/brightcolor/npc/internal/system"
)

func (ui terminalUI) configureCloudflare() error {
	fmt.Println(section("Cloudflare DNS-01"))
	fmt.Println(dim("npc stores Cloudflare settings in /etc/npc/secrets/cloudflare.env with mode 0600."))
	fmt.Println(dim("Use an API token with Zone DNS Edit permissions. The token is not printed back."))
	fmt.Println()
	token := ui.askSecret("Cloudflare API token")
	accountID := ui.askRequired("Cloudflare account ID")
	zoneID := ui.askDefault("Cloudflare zone ID, optional", "")
	if token == "" {
		return validationError{fmt.Errorf("Cloudflare API token is required")}
	}
	fmt.Println()
	fmt.Println(panel("Cloudflare Review",
		"Secret file: "+secrets.EnvPath("cloudflare"),
		"Token:       hidden",
		"Account ID:  "+accountID,
		"Zone ID:     "+defaultString(zoneID, "-"),
	))
	if !ui.confirm("Save Cloudflare DNS settings now?", true) {
		fmt.Println("No changes were made.")
		return nil
	}
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	if err := secrets.WriteProviderEnv("cloudflare", []byte(cloudflareEnv(token, accountID, zoneID))); err != nil {
		return err
	}
	fmt.Println(ok("Saved " + secrets.EnvPath("cloudflare")))
	return nil
}
