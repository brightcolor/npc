package acme

import (
	"fmt"

	"github.com/brightcolor/npc/internal/system"
)

var DNSProviders = []string{"cloudflare", "hetzner", "netcup", "ionos", "route53", "digitalocean", "duckdns", "custom"}

func Installed() bool {
	return system.Exists("acme.sh")
}

func IssueCommand(hostname, method, provider, email string) []string {
	switch method {
	case "http":
		return []string{"acme.sh", "--issue", "-d", hostname, "-w", "/var/www/html", "--accountemail", email}
	case "dns":
		return []string{"acme.sh", "--issue", "-d", hostname, "--dns", dnsFlag(provider), "--accountemail", email}
	case "standalone":
		return []string{"acme.sh", "--issue", "-d", hostname, "--standalone", "--accountemail", email}
	case "tls-alpn":
		return []string{"acme.sh", "--issue", "-d", hostname, "--alpn", "--accountemail", email}
	default:
		return []string{"acme.sh", "--issue", "-d", hostname, "--accountemail", email}
	}
}

func dnsFlag(provider string) string {
	switch provider {
	case "cloudflare":
		return "dns_cf"
	case "hetzner":
		return "dns_hetzner"
	case "digitalocean":
		return "dns_dgon"
	case "route53":
		return "dns_aws"
	default:
		return fmt.Sprintf("dns_%s", provider)
	}
}
