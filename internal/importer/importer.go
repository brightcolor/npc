package importer

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
)

var (
	serverNamePattern      = regexp.MustCompile(`(?m)^\s*server_name\s+([^;]+);`)
	proxyPassPattern       = regexp.MustCompile(`(?m)^\s*proxy_pass\s+([^;]+);`)
	sslCertPattern         = regexp.MustCompile(`(?m)^\s*ssl_certificate\s+([^;]+);`)
	sslKeyPattern          = regexp.MustCompile(`(?m)^\s*ssl_certificate_key\s+([^;]+);`)
	accessLogPattern       = regexp.MustCompile(`(?m)^\s*access_log\s+([^;\s]+)`)
	errorLogPattern        = regexp.MustCompile(`(?m)^\s*error_log\s+([^;\s]+)`)
	clientMaxBodyPattern   = regexp.MustCompile(`(?m)^\s*client_max_body_size\s+([^;]+);`)
	redirectHTTPSPattern   = regexp.MustCompile(`(?m)return\s+30[128]\s+https://`)
	webSocketHeaderPattern = regexp.MustCompile(`(?m)proxy_set_header\s+Upgrade\s+\$http_upgrade`)
	http2ListenPattern     = regexp.MustCompile(`(?m)listen\s+443\s+ssl\s+http2`)
)

type Candidate struct {
	Path    string       `json:"path"`
	Managed bool         `json:"managed"`
	Site    *config.Site `json:"site,omitempty"`
	Error   string       `json:"error,omitempty"`
}

func ParseFile(path string) Candidate {
	candidate := Candidate{Path: path, Managed: nginx.Managed(path)}
	data, err := os.ReadFile(path)
	if err != nil {
		candidate.Error = err.Error()
		return candidate
	}
	text := string(data)
	hostMatch := serverNamePattern.FindStringSubmatch(text)
	proxyMatch := proxyPassPattern.FindStringSubmatch(text)
	if len(hostMatch) < 2 || len(proxyMatch) < 2 {
		candidate.Error = "server_name or proxy_pass not found"
		return candidate
	}
	hostname := strings.Fields(hostMatch[1])[0]
	target := strings.TrimSpace(proxyMatch[1])
	u, err := url.Parse(target)
	if err != nil || u.Scheme == "" || u.Hostname() == "" {
		candidate.Error = fmt.Sprintf("unsupported proxy_pass %q", target)
		return candidate
	}
	port := 80
	if u.Scheme == "https" {
		port = 443
	}
	if u.Port() != "" {
		if parsed, err := strconv.Atoi(u.Port()); err == nil {
			port = parsed
		}
	}
	enabledPath := filepath.Join(paths.NginxSitesEnabled, filepath.Base(path))
	now := time.Now().UTC()
	candidate.Site = &config.Site{
		Hostname: hostname, BackendScheme: u.Scheme, BackendHost: u.Hostname(), BackendPort: port,
		Profile: "imported", ClientMaxBodySize: firstMatch(clientMaxBodyPattern, text, "100M"),
		ConfigPath: path, EnabledPath: enabledPath, WebSocket: webSocketHeaderPattern.MatchString(text),
		RedirectHTTPS: redirectHTTPSPattern.MatchString(text), HTTP2: http2ListenPattern.MatchString(text),
		AccessLog: firstMatch(accessLogPattern, text, ""), ErrorLog: firstMatch(errorLogPattern, text, ""),
		CreatedAt: now, UpdatedAt: now, ManagedBy: "npc",
	}
	candidate.Site.CertificatePath = firstMatch(sslCertPattern, text, "")
	candidate.Site.CertificateKeyPath = firstMatch(sslKeyPattern, text, "")
	candidate.Site.SSL = candidate.Site.CertificatePath != "" && candidate.Site.CertificateKeyPath != ""
	if strings.Contains(candidate.Site.CertificatePath, ".acme.sh") || strings.Contains(candidate.Site.CertificateKeyPath, ".acme.sh") {
		candidate.Site.ACME = true
	}
	return candidate
}

func firstMatch(pattern *regexp.Regexp, text, fallback string) string {
	match := pattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return fallback
	}
	return strings.Trim(strings.TrimSpace(match[1]), `"'`)
}
