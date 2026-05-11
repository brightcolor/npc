package validate

import "testing"

func TestHostname(t *testing.T) {
	valid := []string{"app.example.com", "a-b.example.co.uk", "*.example.com"}
	for _, host := range valid {
		if err := Hostname(host, true); err != nil {
			t.Fatalf("expected %s to be valid: %v", host, err)
		}
	}
	invalid := []string{"", "http://app.example.com", "app.example.com/path", "localhost", "-bad.example.com"}
	for _, host := range invalid {
		if err := Hostname(host, true); err == nil {
			t.Fatalf("expected %s to be invalid", host)
		}
	}
}

func TestPort(t *testing.T) {
	for _, port := range []int{1, 80, 65535} {
		if err := Port(port); err != nil {
			t.Fatalf("expected %d to be valid", port)
		}
	}
	for _, port := range []int{0, -1, 65536} {
		if err := Port(port); err == nil {
			t.Fatalf("expected %d to be invalid", port)
		}
	}
}

func TestCIDRorIP(t *testing.T) {
	for _, value := range []string{"192.0.2.1", "2001:db8::1", "10.0.0.0/8"} {
		if err := CIDRorIP(value); err != nil {
			t.Fatalf("expected %s to be valid: %v", value, err)
		}
	}
	if err := CIDRorIP("bad"); err == nil {
		t.Fatal("expected invalid CIDR/IP")
	}
}
