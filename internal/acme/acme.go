package acme

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/brightcolor/npc/internal/system"
)

var DNSProviders = []string{"cloudflare", "hetzner", "netcup", "ionos", "route53", "digitalocean", "duckdns", "custom"}

func Installed() bool {
	return system.Exists("acme.sh")
}

func Install(email string) error {
	resp, err := http.Get("https://get.acme.sh")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("failed to download acme.sh installer: HTTP %d", resp.StatusCode)
	}
	dir, err := os.MkdirTemp("", "npc-acme-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	installer := filepath.Join(dir, "acme.sh")
	out, err := os.OpenFile(installer, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	args := []string{installer, "--install"}
	if email != "" {
		args = append(args, "--accountemail", email)
	}
	_, err = system.Run("sh", args...)
	return err
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
