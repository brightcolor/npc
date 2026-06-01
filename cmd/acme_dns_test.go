package cmd

import (
	"strings"
	"testing"
)

func TestCloudflareEnv(t *testing.T) {
	out := cloudflareEnv("token", "account", "zone")
	for _, want := range []string{"CF_Token=token", "CF_Account_ID=account", "CF_Zone_ID=zone"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
}
