package revision

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/brightcolor/npc/internal/config"
	"gopkg.in/yaml.v3"
)

func TestLoadSiteFromRevision(t *testing.T) {
	site := &config.Site{Hostname: "app.example.com", BackendScheme: "http", BackendHost: "127.0.0.1", BackendPort: 3000}
	data, err := yaml.Marshal(site)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "site.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSite(Revision{Metadata: path})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Hostname != site.Hostname {
		t.Fatalf("hostname = %q", loaded.Hostname)
	}
}
