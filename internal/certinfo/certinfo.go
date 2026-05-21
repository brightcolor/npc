package certinfo

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/brightcolor/npc/internal/system"
)

type Info struct {
	Path       string    `json:"path"`
	Exists     bool      `json:"exists"`
	Subject    string    `json:"subject,omitempty"`
	Issuer     string    `json:"issuer,omitempty"`
	NotBefore  time.Time `json:"not_before,omitempty"`
	NotAfter   time.Time `json:"not_after,omitempty"`
	DaysLeft   int       `json:"days_left,omitempty"`
	DNSNames   []string  `json:"dns_names,omitempty"`
	AutoRenew  bool      `json:"auto_renew"`
	ParseError string    `json:"parse_error,omitempty"`
}

func Read(path string, autoRenew bool) Info {
	info := Info{Path: path, AutoRenew: autoRenew}
	data, err := os.ReadFile(path)
	if err != nil {
		info.ParseError = err.Error()
		return info
	}
	info.Exists = true
	if len(data) == 0 {
		info.ParseError = "empty certificate file"
		return info
	}
	if !system.Exists("openssl") {
		info.ParseError = "openssl was not found"
		return info
	}
	res, err := system.Run("openssl", "x509", "-in", path, "-noout", "-subject", "-issuer", "-dates", "-ext", "subjectAltName")
	if err != nil {
		info.ParseError = res.Output
		return info
	}
	parseOpenSSLOutput(&info, res.Output)
	return info
}

func parseOpenSSLOutput(info *Info, output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "subject="):
			info.Subject = strings.TrimSpace(strings.TrimPrefix(line, "subject="))
		case strings.HasPrefix(line, "issuer="):
			info.Issuer = strings.TrimSpace(strings.TrimPrefix(line, "issuer="))
		case strings.HasPrefix(line, "notBefore="):
			info.NotBefore = parseOpenSSLTime(strings.TrimPrefix(line, "notBefore="))
		case strings.HasPrefix(line, "notAfter="):
			info.NotAfter = parseOpenSSLTime(strings.TrimPrefix(line, "notAfter="))
			if !info.NotAfter.IsZero() {
				info.DaysLeft = int(time.Until(info.NotAfter).Hours() / 24)
			}
		case strings.Contains(line, "DNS:"):
			info.DNSNames = parseDNSNames(line)
		}
	}
}

func parseOpenSSLTime(value string) time.Time {
	t, _ := time.Parse("Jan 2 15:04:05 2006 MST", strings.TrimSpace(value))
	return t
}

func parseDNSNames(line string) []string {
	parts := strings.Split(line, ",")
	names := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(part), "DNS:"))
		if part != "" && !strings.Contains(part, "X509v3") {
			names = append(names, part)
		}
	}
	return names
}

func Summary(info Info) string {
	if !info.Exists {
		return "missing"
	}
	if info.ParseError != "" {
		return "invalid: " + info.ParseError
	}
	return fmt.Sprintf("%d days", info.DaysLeft)
}
