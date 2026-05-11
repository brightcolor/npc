package acme

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brightcolor/npc/internal/system"
)

var DNSProviders = []string{"cloudflare", "hetzner", "netcup", "ionos", "route53", "digitalocean", "duckdns", "custom"}

func Installed() bool {
	return CommandPath() != ""
}

func CommandPath() string {
	if system.Exists("acme.sh") {
		return "acme.sh"
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		"/root/.acme.sh/acme.sh",
		filepath.Join(os.Getenv("HOME"), ".acme.sh", "acme.sh"),
		filepath.Join(home, ".acme.sh", "acme.sh"),
		"/usr/local/bin/acme.sh",
		"/usr/bin/acme.sh",
	}
	if matches, err := filepath.Glob("/home/*/.acme.sh/acme.sh"); err == nil {
		candidates = append(candidates, matches...)
	}
	for _, candidate := range candidates {
		if candidate == "" || strings.Contains(candidate, "\x00") {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
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
		args = append(args, "-m", email)
	}
	res, err := system.Run("sh", args...)
	if err != nil {
		return fmt.Errorf("acme.sh installer failed: %s", res.Output)
	}
	if CommandPath() == "" {
		return fmt.Errorf("acme.sh installer completed but acme.sh was not found; searched: %s; installer output: %s", strings.Join(searchPaths(), ", "), res.Output)
	}
	return nil
}

func searchPaths() []string {
	home, _ := os.UserHomeDir()
	paths := []string{
		"/root/.acme.sh/acme.sh",
		filepath.Join(os.Getenv("HOME"), ".acme.sh", "acme.sh"),
		filepath.Join(home, ".acme.sh", "acme.sh"),
		"/usr/local/bin/acme.sh",
		"/usr/bin/acme.sh",
	}
	if matches, err := filepath.Glob("/home/*/.acme.sh/acme.sh"); err == nil {
		paths = append(paths, matches...)
	}
	sort.Strings(paths)
	return paths
}

func IssueHTTP(hostname, email string) error {
	cmd := CommandPath()
	if cmd == "" {
		return fmt.Errorf("acme.sh was not found")
	}
	args := []string{"--issue", "-d", hostname, "-w", "/var/www/html"}
	if email != "" {
		args = append(args, "--accountemail", email)
	}
	res, err := system.Run(cmd, args...)
	if err != nil {
		return fmt.Errorf("acme.sh issue failed: %s", res.Output)
	}
	return nil
}

func InstallCert(hostname, fullchainPath, keyPath string) error {
	cmd := CommandPath()
	if cmd == "" {
		return fmt.Errorf("acme.sh was not found")
	}
	if err := os.MkdirAll(filepath.Dir(fullchainPath), 0o700); err != nil {
		return err
	}
	args := []string{
		"--install-cert", "-d", hostname,
		"--key-file", keyPath,
		"--fullchain-file", fullchainPath,
		"--reloadcmd", "systemctl reload nginx",
	}
	res, err := system.Run(cmd, args...)
	if err != nil {
		return fmt.Errorf("acme.sh install-cert failed: %s", res.Output)
	}
	return nil
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
