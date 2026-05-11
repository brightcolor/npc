package renderer

import (
	"strings"
	"testing"
	"time"

	"github.com/brightcolor/npc/internal/config"
)

func baseSite() *config.Site {
	return &config.Site{
		Hostname: "app.example.com", BackendScheme: "http", BackendHost: "127.0.0.1", BackendPort: 3000,
		ClientMaxBodySize: "100M", ConfigPath: "/etc/nginx/sites-available/app.example.com.conf",
		EnabledPath: "/etc/nginx/sites-enabled/app.example.com.conf", CreatedAt: time.Now(), UpdatedAt: time.Now(), ManagedBy: "npc",
	}
}

func TestRenderHTTPOnly(t *testing.T) {
	out, err := RenderSite(baseSite())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Managed by npc", "listen 80;", "proxy_pass http://127.0.0.1:3000;"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderWebSocketAndSSL(t *testing.T) {
	site := baseSite()
	site.SSL = true
	site.HTTP2 = true
	site.WebSocket = true
	site.RedirectHTTPS = true
	site.CertificatePath = "/cert/fullchain.pem"
	site.CertificateKeyPath = "/cert/key.pem"
	out, err := RenderSite(site)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"listen 443 ssl http2;", "return 301 https://$host$request_uri;", "proxy_set_header Upgrade $http_upgrade;"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
