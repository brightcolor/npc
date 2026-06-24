package cmd

import (
	"strings"
	"testing"
)

func TestValidateListenAddress(t *testing.T) {
	valid := []string{"127.0.0.1:8088", "0.0.0.0:8088", "localhost:9000", "[::1]:8088"}
	for _, value := range valid {
		if err := validateListenAddress(value); err != nil {
			t.Fatalf("%s should be valid: %v", value, err)
		}
	}
	invalid := []string{"127.0.0.1", ":8088", "127.0.0.1:99999", "bad host:8088"}
	for _, value := range invalid {
		if err := validateListenAddress(value); err == nil {
			t.Fatalf("%s should be invalid", value)
		}
	}
}

func TestSplitCommandLine(t *testing.T) {
	args, err := splitCommandLine(`create --hostname "app.example.com" --backend-port 3000`)
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 5 || args[2] != "app.example.com" {
		t.Fatalf("unexpected args: %#v", args)
	}
	if _, err := splitCommandLine(`show "app.example.com`); err == nil {
		t.Fatal("expected unterminated quote error")
	}
}

func TestRenderWebUIUnit(t *testing.T) {
	unit := renderWebUIUnit("127.0.0.1:8088")
	for _, want := range []string{"npc web interface", "ExecStart=/usr/local/bin/npc webui --listen 127.0.0.1:8088 --no-upgrade", "NoNewPrivileges=true"} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit missing %q:\n%s", want, unit)
		}
	}
}

func TestWebUIAllowsCertAndImportCommands(t *testing.T) {
	allowed := [][]string{
		{"import", "--yes"},
		{"certs", "set", "app.example.com", "--manual"},
		{"certs", "delete", "app.example.com", "--force"},
	}
	for _, args := range allowed {
		if !allowedWebUICommand(args) {
			t.Fatalf("expected allowed command: %#v", args)
		}
	}
	if !webUIWriteCommand([]string{"certs", "delete", "app.example.com", "--force"}) {
		t.Fatal("certs delete should require write confirmation")
	}
}
