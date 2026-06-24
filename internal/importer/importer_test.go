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

func TestParseFileDetectsSSLLogsAndWebSocket(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.conf")
	content := `
server {
    listen 80;
    server_name app.example.com;
    return 301 https://$host$request_uri;
}
server {
    listen 443 ssl http2;
    server_name app.example.com;
    ssl_certificate /root/.acme.sh/app.example.com_ecc/fullchain.cer;
    ssl_certificate_key /root/.acme.sh/app.example.com_ecc/app.example.com.key;
    access_log /var/log/nginx/app.access.log;
    error_log /var/log/nginx/app.error.log;
    client_max_body_size 512M;
    location / {
        proxy_pass https://10.0.0.5:8443;
        proxy_set_header Upgrade $http_upgrade;
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
	site := candidate.Site
	if !site.SSL || !site.ACME || !site.HTTP2 || !site.WebSocket || !site.RedirectHTTPS {
		t.Fatalf("expected ssl/acme/http2/websocket/redirect metadata: %#v", site)
	}
	if site.BackendURL() != "https://10.0.0.5:8443" {
		t.Fatalf("backend = %q", site.BackendURL())
	}
	if site.ClientMaxBodySize != "512M" {
		t.Fatalf("body size = %q", site.ClientMaxBodySize)
	}
	if site.CertificatePath == "" || site.CertificateKeyPath == "" || site.AccessLog == "" || site.ErrorLog == "" {
		t.Fatalf("expected parsed paths: %#v", site)
	}
}
