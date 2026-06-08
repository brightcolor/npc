package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func MarshalSite(s *Site) []byte {
	var b strings.Builder
	writeSite(&b, s, "")
	return []byte(b.String())
}

func ParseSite(data []byte) (*Site, error) {
	site := &Site{}
	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := splitYAMLLine(strings.TrimSpace(line))
		if ok {
			setSiteField(site, key, val)
		}
	}
	return site, nil
}

func marshalConfig(cfg *Config) []byte {
	if cfg.Version == 0 {
		cfg.Version = 2
	}
	var b strings.Builder
	fmt.Fprintf(&b, "version: %d\nsites:\n", cfg.Version)
	for _, site := range cfg.SortedSites() {
		fmt.Fprintf(&b, "  %s:\n", quote(site.Hostname))
		writeSite(&b, site, "    ")
	}
	return []byte(b.String())
}

func writeSite(b *strings.Builder, s *Site, indent string) {
	writeString(b, indent, "hostname", s.Hostname)
	writeString(b, indent, "backend_scheme", s.BackendScheme)
	writeString(b, indent, "backend_host", s.BackendHost)
	writeInt(b, indent, "backend_port", s.BackendPort)
	writeString(b, indent, "profile", s.Profile)
	writeString(b, indent, "alias", s.Alias)
	writeString(b, indent, "group", s.Group)
	writeString(b, indent, "tags", strings.Join(s.Tags, ","))
	writeBool(b, indent, "archived", s.Archived)
	writeBool(b, indent, "websocket", s.WebSocket)
	writeBool(b, indent, "http2", s.HTTP2)
	writeBool(b, indent, "http3_prepared", s.HTTP3Prepared)
	writeString(b, indent, "client_max_body_size", s.ClientMaxBodySize)
	writeBool(b, indent, "ssl_enabled", s.SSL)
	writeBool(b, indent, "acme_enabled", s.ACME)
	writeString(b, indent, "acme_method", s.ACMEMethod)
	writeString(b, indent, "acme_ca", s.ACMECA)
	writeString(b, indent, "dns_provider", s.DNSProvider)
	writeString(b, indent, "acme_email", s.ACMEEmail)
	writeBool(b, indent, "redirect_https", s.RedirectHTTPS)
	writeBool(b, indent, "hsts_enabled", s.HSTSEnabled)
	writeBool(b, indent, "basic_auth_enabled", s.BasicAuthEnabled)
	writeBool(b, indent, "ip_allowlist_enabled", s.IPAllowlistEnabled)
	writeString(b, indent, "security_headers", s.SecurityHeaders)
	writeString(b, indent, "compression", s.Compression)
	writeBool(b, indent, "rate_limit_enabled", s.RateLimitEnabled)
	writeBool(b, indent, "maintenance_enabled", s.MaintenanceEnabled)
	writeString(b, indent, "config_path", s.ConfigPath)
	writeString(b, indent, "enabled_path", s.EnabledPath)
	writeString(b, indent, "access_log", s.AccessLog)
	writeString(b, indent, "error_log", s.ErrorLog)
	writeString(b, indent, "certificate_path", s.CertificatePath)
	writeString(b, indent, "certificate_key_path", s.CertificateKeyPath)
	writeString(b, indent, "last_reload", s.LastReload)
	writeString(b, indent, "last_successful_nginx_test", s.LastNginxTest)
	if !s.CreatedAt.IsZero() {
		writeString(b, indent, "created_at", s.CreatedAt.Format(time.RFC3339))
	}
	if !s.UpdatedAt.IsZero() {
		writeString(b, indent, "updated_at", s.UpdatedAt.Format(time.RFC3339))
	}
	writeString(b, indent, "managed_by", s.ManagedBy)
}

func parseConfig(data []byte, cfg *Config) error {
	cfg.Sites = map[string]*Site{}
	var current *Site
	for _, raw := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		line := strings.TrimSpace(raw)
		if indent == 0 {
			key, val, ok := splitYAMLLine(line)
			if ok && key == "version" {
				cfg.Version, _ = strconv.Atoi(val)
			}
			continue
		}
		if indent == 2 && strings.HasSuffix(line, ":") {
			name := unquote(strings.TrimSuffix(line, ":"))
			current = &Site{Hostname: name}
			cfg.Sites[name] = current
			continue
		}
		if current != nil && indent >= 4 {
			key, val, ok := splitYAMLLine(line)
			if ok {
				setSiteField(current, key, val)
			}
		}
	}
	return nil
}

func splitYAMLLine(line string) (string, string, bool) {
	key, val, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	return strings.TrimSpace(key), unquote(strings.TrimSpace(val)), true
}

func setSiteField(s *Site, key, val string) {
	switch key {
	case "hostname":
		s.Hostname = val
	case "backend_scheme":
		s.BackendScheme = val
	case "backend_host":
		s.BackendHost = val
	case "backend_port":
		s.BackendPort, _ = strconv.Atoi(val)
	case "profile":
		s.Profile = val
	case "alias":
		s.Alias = val
	case "group":
		s.Group = val
	case "tags":
		s.Tags = parseCSV(val)
	case "archived":
		s.Archived = parseBool(val)
	case "websocket":
		s.WebSocket = parseBool(val)
	case "http2":
		s.HTTP2 = parseBool(val)
	case "ssl_enabled":
		s.SSL = parseBool(val)
	case "acme_enabled":
		s.ACME = parseBool(val)
	case "acme_method":
		s.ACMEMethod = val
	case "acme_ca":
		s.ACMECA = val
	case "dns_provider":
		s.DNSProvider = val
	case "redirect_https":
		s.RedirectHTTPS = parseBool(val)
	case "security_headers":
		s.SecurityHeaders = val
	case "maintenance_enabled":
		s.MaintenanceEnabled = parseBool(val)
	case "config_path":
		s.ConfigPath = val
	case "enabled_path":
		s.EnabledPath = val
	case "certificate_path":
		s.CertificatePath = val
	case "certificate_key_path":
		s.CertificateKeyPath = val
	case "created_at":
		s.CreatedAt = parseTime(val)
	case "updated_at":
		s.UpdatedAt = parseTime(val)
	case "managed_by":
		s.ManagedBy = val
	default:
		setExtraSiteField(s, key, val)
	}
}

func setExtraSiteField(s *Site, key, val string) {
	switch key {
	case "http3_prepared":
		s.HTTP3Prepared = parseBool(val)
	case "client_max_body_size":
		s.ClientMaxBodySize = val
	case "acme_email":
		s.ACMEEmail = val
	case "hsts_enabled":
		s.HSTSEnabled = parseBool(val)
	case "basic_auth_enabled":
		s.BasicAuthEnabled = parseBool(val)
	case "ip_allowlist_enabled":
		s.IPAllowlistEnabled = parseBool(val)
	case "compression":
		s.Compression = val
	case "rate_limit_enabled":
		s.RateLimitEnabled = parseBool(val)
	case "access_log":
		s.AccessLog = val
	case "error_log":
		s.ErrorLog = val
	case "last_reload":
		s.LastReload = val
	case "last_successful_nginx_test":
		s.LastNginxTest = val
	}
}

func writeString(b *strings.Builder, indent, key, val string) {
	if val != "" {
		fmt.Fprintf(b, "%s%s: %s\n", indent, key, quote(val))
	}
}

func writeInt(b *strings.Builder, indent, key string, val int) {
	if val != 0 {
		fmt.Fprintf(b, "%s%s: %d\n", indent, key, val)
	}
}

func writeBool(b *strings.Builder, indent, key string, val bool) {
	if val {
		fmt.Fprintf(b, "%s%s: true\n", indent, key)
	}
}

func quote(val string) string {
	if val == "" {
		return `""`
	}
	return strconv.Quote(val)
}

func unquote(val string) string {
	if parsed, err := strconv.Unquote(val); err == nil {
		return parsed
	}
	return val
}

func parseBool(val string) bool {
	return val == "true" || val == "yes" || val == "1"
}

func parseTime(val string) time.Time {
	t, _ := time.Parse(time.RFC3339, val)
	return t
}

func parseCSV(val string) []string {
	var values []string
	for _, item := range strings.Split(val, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			values = append(values, item)
		}
	}
	return values
}
