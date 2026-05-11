package config

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/brightcolor/npc/internal/paths"
	"gopkg.in/yaml.v3"
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
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
