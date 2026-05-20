package renderer

import (
	"bytes"
	"embed"
	"text/template"

	"github.com/brightcolor/npc/internal/config"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

type View struct {
	*config.Site
	HasHTTP        bool
	ACMEWebroot    bool
	BackendURL     string
	ConnectTimeout string
	SendTimeout    string
	ReadTimeout    string
	ProxyBuffering string
}

func RenderSite(site *config.Site) (string, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/site.conf.tmpl")
	if err != nil {
		return "", err
	}
	view := View{
		Site:           site,
		HasHTTP:        true,
		ACMEWebroot:    site.ACME && site.ACMEMethod == "http",
		BackendURL:     site.BackendURL(),
		ConnectTimeout: "60s",
		SendTimeout:    "60s",
		ReadTimeout:    timeoutFor(site.Profile),
		ProxyBuffering: bufferingFor(site.Profile),
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, view); err != nil {
		return "", err
	}
	return out.String(), nil
}

func timeoutFor(profile string) string {
	switch profile {
	case "upload", "nextcloud", "wordpress":
		return "300s"
	case "streaming", "websocket", "media":
		return "3600s"
	case "api", "static":
		return "30s"
	default:
		return "60s"
	}
}

func bufferingFor(profile string) string {
	switch profile {
	case "streaming", "websocket", "media":
		return "off"
	default:
		return "on"
	}
}
