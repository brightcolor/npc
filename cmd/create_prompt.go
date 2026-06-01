package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/brightcolor/npc/internal/acme"
)

func promptCreate(o *createOptions) error {
	r := bufio.NewReader(os.Stdin)
	ask := func(label, def string) string {
		if def != "" {
			fmt.Printf("%s [%s]: ", label, def)
		} else {
			fmt.Printf("%s: ", label)
		}
		text, _ := r.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			return def
		}
		return text
	}
	o.hostname = ask("Hostname", o.hostname)
	o.backendHost = ask("Backend host", defaultString(o.backendHost, "127.0.0.1"))
	port, err := strconv.Atoi(ask("Backend port", defaultString(strconv.Itoa(o.backendPort), "3000")))
	if err != nil {
		return err
	}
	o.backendPort = port
	o.backendScheme = ask("Backend scheme (http/https)", defaultString(o.backendScheme, "http"))
	o.profile = ask("Profile (generic/websocket/upload/streaming/docker/security-basic)", defaultString(o.profile, "generic"))
	o.websocket = yes(ask("WebSocket support? (y/n)", boolDefault(o.websocket)))
	o.ssl = yes(ask("Enable SSL/TLS? (y/n)", boolDefault(o.ssl)))
	if o.ssl {
		promptTLSOptions(o, ask)
	}
	o.clientMaxBodySize = ask("client_max_body_size", defaultString(o.clientMaxBodySize, "100M"))
	return nil
}

func promptTLSOptions(o *createOptions, ask func(string, string) string) {
	o.redirectHTTPS = yes(ask("Redirect HTTP to HTTPS? (y/n)", boolDefault(true)))
	o.http2 = yes(ask("Enable HTTP/2? (y/n)", boolDefault(true)))
	o.acme = yes(ask("Use acme.sh? (y/n)", boolDefault(o.acme)))
	if !o.acme {
		o.certPath = ask("Fullchain path", o.certPath)
		o.keyPath = ask("Private key path", o.keyPath)
		return
	}
	o.acmeCA = ask("ACME CA (letsencrypt/zerossl/buypass)", defaultString(o.acmeCA, acme.DefaultCA))
	if cloudflareDNSReady() && o.acmeMethod == "" {
		o.acmeMethod = "dns"
		o.dnsProvider = "cloudflare"
		fmt.Println("Cloudflare DNS credentials found; DNS-01 is the default ACME method.")
	}
	o.acmeMethod = ask("ACME method (http/dns/standalone/tls-alpn)", defaultString(o.acmeMethod, "http"))
	o.email = ask("ACME email, optional", o.email)
	if o.acmeMethod == "dns" {
		o.dnsProvider = ask("DNS provider", defaultString(o.dnsProvider, "cloudflare"))
	}
}

func defaultString(v, def string) string {
	if v == "" || v == "0" {
		return def
	}
	return v
}

func boolDefault(v bool) string {
	if v {
		return "y"
	}
	return "n"
}

func yes(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "y" || v == "yes" || v == "ja" || v == "true"
}

func promptConfirm(message string, def bool) bool {
	suffix := " [Y/n]: "
	if !def {
		suffix = " [y/N]: "
	}
	fmt.Print(message + suffix)
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return def
	}
	return text == "y" || text == "yes" || text == "ja" || text == "true"
}
