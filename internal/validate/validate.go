package validate

import (
	"fmt"
	"net"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
)

var fqdnLabel = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func Hostname(host string, allowWildcard bool) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("hostname is required")
	}
	if strings.Contains(host, "://") || strings.Contains(host, "/") {
		return fmt.Errorf("hostname must not include scheme or path")
	}
	if strings.HasPrefix(host, "*.") {
		if !allowWildcard {
			return fmt.Errorf("wildcard hostnames are not enabled for this operation")
		}
		host = strings.TrimPrefix(host, "*.")
	}
	if len(host) > 253 || !strings.Contains(host, ".") {
		return fmt.Errorf("hostname must be a fully qualified domain name")
	}
	for _, label := range strings.Split(host, ".") {
		if !fqdnLabel.MatchString(label) {
			return fmt.Errorf("invalid hostname label %q", label)
		}
	}
	return nil
}

func Port(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func PortString(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("port must be a number")
	}
	return port, Port(port)
}

func BackendScheme(scheme string) error {
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("backend scheme must be http or https")
	}
	return nil
}

func BackendHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("backend host is required")
	}
	if strings.ContainsAny(host, "/:") && net.ParseIP(host) == nil {
		return fmt.Errorf("backend host must not include scheme, port, or path")
	}
	return nil
}

func CIDRorIP(value string) error {
	value = strings.TrimSpace(value)
	if _, err := netip.ParseAddr(value); err == nil {
		return nil
	}
	if _, err := netip.ParsePrefix(value); err == nil {
		return nil
	}
	return fmt.Errorf("%q is not a valid IP or CIDR", value)
}
