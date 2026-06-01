package cmd

import "github.com/brightcolor/npc/internal/secrets"

func cloudflareDNSReady() bool {
	_, err := secrets.ReadEnv("cloudflare")
	return err == nil
}

func applyEnvironmentDefaults(o *createOptions) {
	if o.ssl && o.acme && cloudflareDNSReady() {
		if o.acmeMethod == "" {
			o.acmeMethod = "dns"
		}
		if o.acmeMethod == "dns" && o.dnsProvider == "" {
			o.dnsProvider = "cloudflare"
		}
	}
}
