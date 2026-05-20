package acme

import (
	"strings"
	"testing"
)

func TestDiagnoseOutput(t *testing.T) {
	hints := DiagnoseOutput("Invalid response from http://example.com/.well-known/acme-challenge token timeout")
	if !strings.Contains(hints, "DNS points to this server") {
		t.Fatalf("expected DNS hint, got %q", hints)
	}
}
