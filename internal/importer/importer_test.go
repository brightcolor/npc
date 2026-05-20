package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFileDetectsReverseProxy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.conf")
	content := `
server {
    listen 80;
    server_name app.example.com;
    location / {
        proxy_pass http://127.0.0.1:3000;
    }
}
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	candidate := ParseFile(path)
	if candidate.Error != "" {
		t.Fatalf("unexpected parse error: %s", candidate.Error)
	}
	if candidate.Site.Hostname != "app.example.com" {
		t.Fatalf("hostname = %q", candidate.Site.Hostname)
	}
	if candidate.Site.BackendURL() != "http://127.0.0.1:3000" {
		t.Fatalf("backend = %q", candidate.Site.BackendURL())
	}
	if candidate.Site.ConfigPath != path {
		t.Fatalf("config path = %q", candidate.Site.ConfigPath)
	}
}
