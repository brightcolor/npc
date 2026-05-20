package certinfo

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
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
	block, _ := pem.Decode(data)
	if block == nil {
		info.ParseError = "no PEM certificate block found"
		return info
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		info.ParseError = err.Error()
		return info
	}
	info.Subject = cert.Subject.String()
	info.Issuer = cert.Issuer.String()
	info.NotBefore = cert.NotBefore
	info.NotAfter = cert.NotAfter
	info.DaysLeft = int(time.Until(cert.NotAfter).Hours() / 24)
	info.DNSNames = cert.DNSNames
	return info
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
