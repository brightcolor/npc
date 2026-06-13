package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brightcolor/npc/internal/config"
)

func TestMergeManagedNginxConfigFilesDiscoversHeaderConfigs(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "app.example.com.conf")
	data := `# Managed by npc
# Do not edit manually unless you know what you are doing.
# Hostname: app.example.com
server {
    listen 80;
    server_name app.example.com;
    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}
`
	if err := os.WriteFile(conf, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.New()
	if err := mergeManagedNginxConfigFiles(cfg, []string{conf}); err != nil {
		t.Fatal(err)
	}
	site, ok := cfg.Sites["app.example.com"]
	if !ok {
		t.Fatal("expected npc-managed config to be discovered")
	}
	if site.Profile != "discovered" {
		t.Fatalf("expected discovered profile, got %q", site.Profile)
	}
	if site.BackendURL() != "http://127.0.0.1:3000" {
		t.Fatalf("unexpected backend URL: %s", site.BackendURL())
	}
	if site.ConfigPath != conf {
		t.Fatalf("unexpected config path: %s", site.ConfigPath)
	}
}
