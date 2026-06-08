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
		BackendPort: 3000, Alias: "app", Group: "prod", Tags: []string{"docker", "prod"},
		ACMECA: "letsencrypt", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(), ManagedBy: "npc",
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
	if loaded.Sites["app.example.com"].Alias != "app" || len(loaded.Sites["app.example.com"].Tags) != 2 {
		t.Fatalf("metadata was not preserved: %#v", loaded.Sites["app.example.com"])
	}
	if site, ok := loaded.FindSite("app"); !ok || site.Hostname != "app.example.com" {
		t.Fatalf("alias lookup failed")
	}
}
