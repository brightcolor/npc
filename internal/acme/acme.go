package acme

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brightcolor/npc/internal/fetch"
	"github.com/brightcolor/npc/internal/secrets"
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
	installerBytes, err := fetch.Bytes("https://get.acme.sh")
	if err != nil {
		return err
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
	if _, err := out.Write(installerBytes); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	args := []string{installer}
	if email != "" {
		args = append(args, "email="+email)
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
	seen := map[string]bool{}
	var paths []string
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		paths = append(paths, candidate)
	}
	sort.Strings(paths)
	return paths
}

func IssueHTTP(hostname, email string) error {
	cmd := CommandPath()
	if cmd == "" {
		return fmt.Errorf("acme.sh was not found")
	}
	args := []string{"--issue", "--server", "letsencrypt", "-d", hostname, "-w", "/var/www/html"}
	if email != "" {
		args = append(args, "--accountemail", email)
	}
	res, err := system.Run(cmd, args...)
	if err != nil {
		return fmt.Errorf("acme.sh issue failed: %s%s", res.Output, DiagnoseOutput(res.Output))
	}
	return nil
}

func IssueDNS(hostname, provider, email string) error {
	cmd := CommandPath()
	if cmd == "" {
		return fmt.Errorf("acme.sh was not found")
	}
	env, err := secrets.ReadEnv(provider)
	if err != nil {
		return err
	}
	args := []string{"--issue", "--server", "letsencrypt", "-d", hostname, "--dns", dnsFlag(provider)}
	if provider == "cloudflare" {
		args = append(args, "--dnssleep", "30")
	}
	if email != "" {
		args = append(args, "--accountemail", email)
	}
	res, err := system.RunEnv(env, cmd, args...)
	if err != nil {
		return fmt.Errorf("acme.sh DNS issue failed: %s%s", res.Output, DiagnoseOutput(res.Output))
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
		return fmt.Errorf("acme.sh install-cert failed: %s%s", res.Output, DiagnoseOutput(res.Output))
	}
	return nil
}

func DiagnoseOutput(output string) string {
	text := strings.ToLower(output)
	var hints []string
	if strings.Contains(text, "invalid response") || strings.Contains(text, "404") || strings.Contains(text, "timeout") {
		hints = append(hints, "verify that DNS points to this server and that port 80 is reachable from the internet")
	}
	if strings.Contains(text, "connection refused") || strings.Contains(text, "timeout") {
		hints = append(hints, "check firewall rules, cloud security groups, and whether Nginx is listening on port 80")
	}
	if strings.Contains(text, "rate limit") || strings.Contains(text, "too many") {
		hints = append(hints, "ACME rate limit may be active; wait before retrying or use the staging CA for tests")
	}
	if strings.Contains(text, "unauthorized") {
		hints = append(hints, "the ACME challenge was not accepted; check the challenge webroot and public HTTP access")
	}
	if strings.Contains(text, "cloudflare") {
		hints = append(hints, "if Cloudflare is enabled, avoid Flexible SSL and ensure HTTP-01 traffic reaches the origin")
	}
	if len(hints) == 0 {
		return ""
	}
	return "\nSuggested checks:\n- " + strings.Join(hints, "\n- ")
}

func IssueCommand(hostname, method, provider, email string) []string {
	switch method {
	case "http":
		return []string{"acme.sh", "--issue", "--server", "letsencrypt", "-d", hostname, "-w", "/var/www/html", "--accountemail", email}
	case "dns":
		return []string{"acme.sh", "--issue", "--server", "letsencrypt", "-d", hostname, "--dns", dnsFlag(provider), "--accountemail", email}
	case "standalone":
		return []string{"acme.sh", "--issue", "--server", "letsencrypt", "-d", hostname, "--standalone", "--accountemail", email}
	case "tls-alpn":
		return []string{"acme.sh", "--issue", "--server", "letsencrypt", "-d", hostname, "--alpn", "--accountemail", email}
	default:
		return []string{"acme.sh", "--issue", "--server", "letsencrypt", "-d", hostname, "--accountemail", email}
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
