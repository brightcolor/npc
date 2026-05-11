package cmd

import "testing"

func TestBuildSiteFromFlags(t *testing.T) {
	site, err := buildSite(createOptions{
		hostname: "app.example.com", backendHost: "127.0.0.1", backendPort: 3000,
		backendScheme: "http", clientMaxBodySize: "100M", websocket: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if site.BackendURL() != "http://127.0.0.1:3000" {
		t.Fatalf("unexpected backend URL: %s", site.BackendURL())
	}
	if !site.WebSocket {
		t.Fatal("expected websocket to be enabled")
	}
}

func TestBuildSiteRequiresCertificateForManualSSL(t *testing.T) {
	_, err := buildSite(createOptions{
		hostname: "app.example.com", backendHost: "127.0.0.1", backendPort: 3000,
		backendScheme: "http", clientMaxBodySize: "100M", ssl: true,
	})
	if err == nil {
		t.Fatal("expected missing certificate paths to fail")
	}
}
