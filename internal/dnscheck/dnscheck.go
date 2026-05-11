package dnscheck

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Result struct {
	Hostname  string   `json:"hostname"`
	ServerIPs []string `json:"server_ips"`
	DNSIPs    []string `json:"dns_ips"`
	Match     bool     `json:"match"`
}

func VerifyHostnamePointsHere(ctx context.Context, hostname string) (*Result, error) {
	dnsIPs, err := Resolve(ctx, hostname)
	if err != nil {
		return nil, err
	}
	serverIPs, err := PublicIPs(ctx)
	if err != nil {
		return nil, err
	}
	result := &Result{Hostname: hostname, ServerIPs: serverIPs, DNSIPs: dnsIPs}
	result.Match = HasIntersection(serverIPs, dnsIPs)
	if !result.Match {
		return result, fmt.Errorf("DNS for %s does not point to this server; resolved DNS: %s; server public IPs: %s", hostname, strings.Join(dnsIPs, ", "), strings.Join(serverIPs, ", "))
	}
	return result, nil
}

func Resolve(ctx context.Context, hostname string) ([]string, error) {
	records, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %s: %w", hostname, err)
	}
	var ips []string
	for _, record := range records {
		if record.IP == nil || record.IP.IsLoopback() || record.IP.IsUnspecified() {
			continue
		}
		ips = appendUnique(ips, record.IP.String())
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("DNS lookup for %s returned no usable A/AAAA records", hostname)
	}
	return ips, nil
}

func PublicIPs(ctx context.Context) ([]string, error) {
	endpoints := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
	}
	var ips []string
	var lastErr error
	client := &http.Client{Timeout: 5 * time.Second}
	for _, endpoint := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 128))
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			lastErr = fmt.Errorf("%s returned HTTP %d", endpoint, resp.StatusCode)
			continue
		}
		ip := net.ParseIP(strings.TrimSpace(string(body)))
		if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
			lastErr = fmt.Errorf("%s returned invalid public IP %q", endpoint, strings.TrimSpace(string(body)))
			continue
		}
		ips = appendUnique(ips, ip.String())
	}
	if len(ips) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("could not determine this server's public IP: %w", lastErr)
		}
		return nil, fmt.Errorf("could not determine this server's public IP")
	}
	return ips, nil
}

func HasIntersection(a, b []string) bool {
	seen := map[string]bool{}
	for _, value := range a {
		if ip := net.ParseIP(strings.TrimSpace(value)); ip != nil {
			seen[ip.String()] = true
		}
	}
	for _, value := range b {
		if ip := net.ParseIP(strings.TrimSpace(value)); ip != nil && seen[ip.String()] {
			return true
		}
	}
	return false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
