package importer

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
)

var (
	serverNamePattern = regexp.MustCompile(`(?m)^\s*server_name\s+([^;]+);`)
	proxyPassPattern  = regexp.MustCompile(`(?m)^\s*proxy_pass\s+([^;]+);`)
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
	_, enabledPath := nginx.SitePaths(hostname)
	now := time.Now().UTC()
	candidate.Site = &config.Site{
		Hostname: hostname, BackendScheme: u.Scheme, BackendHost: u.Hostname(), BackendPort: port,
		Profile: "imported", ClientMaxBodySize: "100M", ConfigPath: path, EnabledPath: enabledPath,
		CreatedAt: now, UpdatedAt: now, ManagedBy: "npc",
	}
	return candidate
}
