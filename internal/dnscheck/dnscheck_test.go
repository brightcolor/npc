package dnscheck

import "testing"

func TestHasIntersection(t *testing.T) {
	if !HasIntersection([]string{"203.0.113.10"}, []string{"203.0.113.10"}) {
		t.Fatal("expected matching IPv4 address")
	}
	if !HasIntersection([]string{"2001:db8::1"}, []string{"2001:db8:0:0:0:0:0:1"}) {
		t.Fatal("expected normalized IPv6 address to match")
	}
	if HasIntersection([]string{"203.0.113.10"}, []string{"203.0.113.11"}) {
		t.Fatal("did not expect different IPs to match")
	}
}
