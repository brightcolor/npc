package acme

import "testing"

func TestIssueCommandDNS(t *testing.T) {
	cmd := IssueCommand("app.example.com", "dns", "cloudflare", "admin@example.com")
	joined := ""
	for _, part := range cmd {
		joined += part + " "
	}
	if joined == "" || cmd[0] != "acme.sh" || cmd[4] != "--dns" || cmd[5] != "dns_cf" {
		t.Fatalf("unexpected command: %#v", cmd)
	}
}
