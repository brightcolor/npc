package cmd

import (
	"fmt"

	"github.com/brightcolor/npc/internal/docker"
	"github.com/brightcolor/npc/internal/renderer"
)

func (ui terminalUI) exposeDocker() error {
	if !docker.Installed() {
		return validationError{fmt.Errorf("docker was not found")}
	}
	fmt.Println(section("Docker Discovery"))
	fmt.Println(dim("Scanning running containers. Docker containers and Compose files will not be modified."))
	containers, err := docker.RunningContainers()
	if err != nil {
		return err
	}
	if len(containers) == 0 {
		fmt.Println(emptyState("No running Docker containers", "Start a container or use the custom reverse proxy flow."))
		return nil
	}
	labels := make([]string, 0, len(containers))
	for _, c := range containers {
		ports := "no ports"
		if c.PortsRaw != "" {
			ports = c.PortsRaw
		}
		labels = append(labels, fmt.Sprintf("%-28s %s %s", c.Name, cyan("container"), dim(c.Image+" | "+ports)))
	}
	container := containers[ui.menu("Select a container to expose", labels)]
	if len(container.Ports) == 0 {
		return validationError{fmt.Errorf("container %s has no exposed or published ports", container.Name)}
	}
	portLabels := make([]string, 0, len(container.Ports))
	for _, port := range container.Ports {
		portLabels = append(portLabels, port.Label())
	}
	port := container.Ports[ui.menu("Select the backend port", portLabels)]
	o := createOptions{
		hostname:          ui.askRequired("Public hostname, for example app.example.com"),
		backendHost:       port.BackendHost(container.Name),
		backendPort:       port.BackendPort(),
		backendScheme:     "http",
		profile:           "docker",
		clientMaxBodySize: "100M",
		nonInteractive:    true,
	}
	o.backendScheme = ui.askDefault("Backend scheme", o.backendScheme)
	o.websocket = ui.confirm("Enable WebSocket headers?", false)
	o.ssl = ui.confirm("Enable HTTPS?", false)
	if o.ssl {
		ui.fillTLSOptions(&o)
	}
	o.accessLog = ui.confirm("Enable per-site access log?", true)
	o.errorLog = ui.confirm("Enable per-site error log?", true)
	if !port.Published {
		fmt.Println(warn("Note:") + " the selected port is not published on the host.")
	}
	return ui.previewAndRun(o)
}

func (ui terminalUI) fillTLSOptions(o *createOptions) {
	o.redirectHTTPS = ui.confirm("Redirect HTTP to HTTPS?", true)
	o.http2 = ui.confirm("Enable HTTP/2?", true)
	o.acme = ui.confirm("Use acme.sh?", true)
	if !o.acme {
		o.certPath = ui.askRequired("Fullchain path")
		o.keyPath = ui.askRequired("Private key path")
		return
	}
	if cloudflareDNSReady() && ui.confirm("Use saved Cloudflare DNS-01 credentials?", true) {
		o.acmeMethod = "dns"
		o.dnsProvider = "cloudflare"
	} else {
		o.acmeMethod = ui.askDefault("ACME method (http/dns/standalone/tls-alpn)", "http")
	}
	o.email = ui.askDefault("ACME account email, optional", "")
	if o.acmeMethod == "dns" {
		o.dnsProvider = ui.askDefault("DNS provider", defaultString(o.dnsProvider, "cloudflare"))
	}
}

func (ui terminalUI) previewAndRun(o createOptions) error {
	applyEnvironmentDefaults(&o)
	site, err := buildSite(o)
	if err != nil {
		return validationError{err}
	}
	content, err := renderer.RenderSite(site)
	if err != nil {
		return err
	}
	fmt.Println(panel("Reverse Proxy Review",
		"Hostname: "+site.Hostname,
		"Backend:  "+site.BackendURL(),
		"Profile:  "+site.Profile,
		"Config:   "+site.ConfigPath,
		"Enabled:  "+site.EnabledPath,
		"SSL:      "+yesNo(site.SSL),
		"Logs:     "+yesNo(site.AccessLog != "" || site.ErrorLog != ""),
	))
	if ui.confirm("Show rendered Nginx config?", false) {
		fmt.Println(content)
	}
	if !ui.confirm("Create this reverse proxy now?", true) {
		fmt.Println("No changes were made.")
		return nil
	}
	if err := executeCreate(o); err != nil {
		return fmt.Errorf("could not create reverse proxy: %w", err)
	}
	return nil
}
