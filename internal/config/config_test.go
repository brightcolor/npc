package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestConfigReadWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := New()
	cfg.Sites["app.example.com"] = &Site{
		Hostname: "app.example.com", BackendScheme: "http", BackendHost: "127.0.0.1",
		BackendPort: 3000, ACMECA: "letsencrypt", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), ManagedBy: "npc",
	}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Sites["app.example.com"].BackendURL() != "http://127.0.0.1:3000" {
		t.Fatalf("unexpected backend URL: %s", loaded.Sites["app.example.com"].BackendURL())
	}
	if loaded.Sites["app.example.com"].ACMECA != "letsencrypt" {
		t.Fatalf("unexpected ACME CA: %s", loaded.Sites["app.example.com"].ACMECA)
	}
}
