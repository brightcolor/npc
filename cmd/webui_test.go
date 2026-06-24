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

func TestRenderWebUIUnit(t *testing.T) {
	unit := renderWebUIUnit("127.0.0.1:8088")
	for _, want := range []string{"npc web interface", "ExecStart=/usr/local/bin/npc webui --listen 127.0.0.1:8088 --no-upgrade", "NoNewPrivileges=true"} {
		if !strings.Contains(unit, want) {
			t.Fatalf("unit missing %q:\n%s", want, unit)
		}
	}
}

func TestWebUITemplateUsesFormsInsteadOfCommandConsole(t *testing.T) {
	for _, want := range []string{`name="action" value="create"`, `name="action" value="edit"`, `name="action" value="cert-set"`, `name="action" value="cert-issue"`, `name="action" value="import"`, `data-host=`, `fillForm(form)`} {
		if !strings.Contains(webUIHTML, want) {
			t.Fatalf("template missing %q", want)
		}
	}
	if strings.Contains(webUIHTML, "npc command") {
		t.Fatal("template should not expose a command console")
	}
}
