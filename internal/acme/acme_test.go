package acme

import "testing"

func TestIssueCommandDNS(t *testing.T) {
	cmd := IssueCommand("app.example.com", "dns", "cloudflare", "admin@example.com", "")
	joined := ""
	for _, part := range cmd {
		joined += part + " "
	}
	if joined == "" || cmd[0] != "acme.sh" || cmd[6] != "--dns" || cmd[7] != "dns_cf" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}

func TestIssueCommandUsesLetsEncrypt(t *testing.T) {
	cmd := IssueCommand("app.example.com", "dns", "cloudflare", "", "")
	if cmd[2] != "--server" || cmd[3] != "letsencrypt" {
		t.Fatalf("expected letsencrypt server in command: %#v", cmd)
	}
}

func TestIssueCommandAllowsManualCA(t *testing.T) {
	cmd := IssueCommand("app.example.com", "http", "", "", "buypass")
	if cmd[3] != "buypass" {
		t.Fatalf("expected manual CA in command: %#v", cmd)
	}
}

func TestValidateCARejectsUnsupportedCA(t *testing.T) {
	if err := ValidateCA("zerossl"); err == nil {
		t.Fatal("expected unsupported CA to be rejected")
	}
}
