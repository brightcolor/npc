package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/paths"
)

type Config struct {
	Version int              `yaml:"version" json:"version"`
	Sites   map[string]*Site `yaml:"sites" json:"sites"`
}

type Site struct {
	Hostname           string    `yaml:"hostname" json:"hostname"`
	BackendScheme      string    `yaml:"backend_scheme" json:"backend_scheme"`
	BackendHost        string    `yaml:"backend_host" json:"backend_host"`
	BackendPort        int       `yaml:"backend_port" json:"backend_port"`
	Profile            string    `yaml:"profile,omitempty" json:"profile,omitempty"`
	WebSocket          bool      `yaml:"websocket" json:"websocket"`
	HTTP2              bool      `yaml:"http2" json:"http2"`
	HTTP3Prepared      bool      `yaml:"http3_prepared,omitempty" json:"http3_prepared,omitempty"`
	ClientMaxBodySize  string    `yaml:"client_max_body_size" json:"client_max_body_size"`
	SSL                bool      `yaml:"ssl_enabled" json:"ssl_enabled"`
	ACME               bool      `yaml:"acme_enabled" json:"acme_enabled"`
	ACMEMethod         string    `yaml:"acme_method,omitempty" json:"acme_method,omitempty"`
	DNSProvider        string    `yaml:"dns_provider,omitempty" json:"dns_provider,omitempty"`
	ACMEEmail          string    `yaml:"acme_email,omitempty" json:"acme_email,omitempty"`
	RedirectHTTPS      bool      `yaml:"redirect_https" json:"redirect_https"`
	HSTSEnabled        bool      `yaml:"hsts_enabled" json:"hsts_enabled"`
	BasicAuthEnabled   bool      `yaml:"basic_auth_enabled" json:"basic_auth_enabled"`
	IPAllowlistEnabled bool      `yaml:"ip_allowlist_enabled" json:"ip_allowlist_enabled"`
	SecurityHeaders    string    `yaml:"security_headers,omitempty" json:"security_headers,omitempty"`
	Compression        string    `yaml:"compression,omitempty" json:"compression,omitempty"`
	RateLimitEnabled   bool      `yaml:"rate_limit_enabled,omitempty" json:"rate_limit_enabled,omitempty"`
	MaintenanceEnabled bool      `yaml:"maintenance_enabled,omitempty" json:"maintenance_enabled,omitempty"`
	ConfigPath         string    `yaml:"config_path" json:"config_path"`
	EnabledPath        string    `yaml:"enabled_path" json:"enabled_path"`
	AccessLog          string    `yaml:"access_log,omitempty" json:"access_log,omitempty"`
	ErrorLog           string    `yaml:"error_log,omitempty" json:"error_log,omitempty"`
	CertificatePath    string    `yaml:"certificate_path,omitempty" json:"certificate_path,omitempty"`
	CertificateKeyPath string    `yaml:"certificate_key_path,omitempty" json:"certificate_key_path,omitempty"`
	LastReload         string    `yaml:"last_reload,omitempty" json:"last_reload,omitempty"`
	LastNginxTest      string    `yaml:"last_successful_nginx_test,omitempty" json:"last_successful_nginx_test,omitempty"`
	CreatedAt          time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt          time.Time `yaml:"updated_at" json:"updated_at"`
	ManagedBy          string    `yaml:"managed_by" json:"managed_by"`
}

func New() *Config {
	return &Config{Version: 1, Sites: map[string]*Site{}}
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = paths.ConfigFile
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := parseConfig(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Sites == nil {
		cfg.Sites = map[string]*Site{}
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	if path == "" {
		path = paths.ConfigFile
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data := marshalConfig(cfg)
	return os.WriteFile(path, data, 0o600)
}

func MarshalSite(s *Site) []byte {
	var b strings.Builder
	writeSite(&b, s, "")
	return []byte(b.String())
}

func ParseSite(data []byte) (*Site, error) {
	site := &Site{}
	for _, line := range strings.Split(string(data), "\n") {
		key, val, ok := splitYAMLLine(strings.TrimSpace(line))
		if !ok {
			continue
		}
		setSiteField(site, key, val)
	}
	return site, nil
}

func (c *Config) SortedSites() []*Site {
	sites := make([]*Site, 0, len(c.Sites))
	for _, site := range c.Sites {
		sites = append(sites, site)
	}
	sort.Slice(sites, func(i, j int) bool { return sites[i].Hostname < sites[j].Hostname })
	return sites
}

func (s *Site) BackendURL() string {
	return s.BackendScheme + "://" + s.BackendHost + ":" + itoa(s.BackendPort)
}

func marshalConfig(cfg *Config) []byte {
	if cfg.Version == 0 {
		cfg.Version = 1
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
	writeBool(b, indent, "websocket", s.WebSocket)
	writeBool(b, indent, "http2", s.HTTP2)
	writeBool(b, indent, "http3_prepared", s.HTTP3Prepared)
	writeString(b, indent, "client_max_body_size", s.ClientMaxBodySize)
	writeBool(b, indent, "ssl_enabled", s.SSL)
	writeBool(b, indent, "acme_enabled", s.ACME)
	writeString(b, indent, "acme_method", s.ACMEMethod)
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
	case "websocket":
		s.WebSocket = parseBool(val)
	case "http2":
		s.HTTP2 = parseBool(val)
	case "http3_prepared":
		s.HTTP3Prepared = parseBool(val)
	case "client_max_body_size":
		s.ClientMaxBodySize = val
	case "ssl_enabled":
		s.SSL = parseBool(val)
	case "acme_enabled":
		s.ACME = parseBool(val)
	case "acme_method":
		s.ACMEMethod = val
	case "dns_provider":
		s.DNSProvider = val
	case "acme_email":
		s.ACMEEmail = val
	case "redirect_https":
		s.RedirectHTTPS = parseBool(val)
	case "hsts_enabled":
		s.HSTSEnabled = parseBool(val)
	case "basic_auth_enabled":
		s.BasicAuthEnabled = parseBool(val)
	case "ip_allowlist_enabled":
		s.IPAllowlistEnabled = parseBool(val)
	case "security_headers":
		s.SecurityHeaders = val
	case "compression":
		s.Compression = val
	case "rate_limit_enabled":
		s.RateLimitEnabled = parseBool(val)
	case "maintenance_enabled":
		s.MaintenanceEnabled = parseBool(val)
	case "config_path":
		s.ConfigPath = val
	case "enabled_path":
		s.EnabledPath = val
	case "access_log":
		s.AccessLog = val
	case "error_log":
		s.ErrorLog = val
	case "certificate_path":
		s.CertificatePath = val
	case "certificate_key_path":
		s.CertificateKeyPath = val
	case "last_reload":
		s.LastReload = val
	case "last_successful_nginx_test":
		s.LastNginxTest = val
	case "created_at":
		s.CreatedAt = parseTime(val)
	case "updated_at":
		s.UpdatedAt = parseTime(val)
	case "managed_by":
		s.ManagedBy = val
	}
}

func writeString(b *strings.Builder, indent, key, val string) {
	if val == "" {
		return
	}
	fmt.Fprintf(b, "%s%s: %s\n", indent, key, quote(val))
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
	if val == "" {
		return ""
	}
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

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
