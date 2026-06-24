package cmd

import (
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/brightcolor/npc/internal/config"
)

type webuiSite struct {
	Hostname           string
	Backend            string
	BackendHost        string
	BackendPort        int
	BackendScheme      string
	State              string
	SSL                string
	Profile            string
	Group              string
	Tags               string
	Config             string
	ClientMaxBodySize  string
	WebSocket          bool
	RedirectHTTPS      bool
	HTTP2              bool
	CertificatePath    string
	CertificateKeyPath string
	ACME               bool
	ACMEMethod         string
	DNSProvider        string
}

type webuiPage struct {
	Version string
	Status  map[string]any
	Sites   []webuiSite
	Notice  *webuiNotice
}

func webUIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webUIIndex)
	mux.HandleFunc("/actions", webUIActions)
	mux.HandleFunc("/api/status", webUIStatusAPI)
	mux.HandleFunc("/api/sites", webUISitesAPI)
	return mux
}

func webUIIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	page, err := loadWebUIPage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	renderWebUIPage(w, page)
}

func webUIStatusAPI(w http.ResponseWriter, r *http.Request) {
	page, err := loadWebUIPage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeWebJSON(w, page.Status)
}

func webUISitesAPI(w http.ResponseWriter, r *http.Request) {
	cfg, err := loadManagedConfig()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeWebJSON(w, cfg.SortedSites())
}

func loadWebUIPage() (webuiPage, error) {
	cfg, err := loadManagedConfig()
	if err != nil {
		return webuiPage{}, err
	}
	sites := make([]webuiSite, 0, len(cfg.Sites))
	for _, site := range cfg.SortedSites() {
		sites = append(sites, webUISiteRow(site))
	}
	return webuiPage{Version: app.build.Version, Status: statusData(cfg.SortedSites()), Sites: sites}, nil
}

func webUISiteRow(site *config.Site) webuiSite {
	return webuiSite{
		Hostname:           site.Hostname,
		Backend:            site.BackendURL(),
		BackendHost:        site.BackendHost,
		BackendPort:        site.BackendPort,
		BackendScheme:      site.BackendScheme,
		State:              onOff(siteEnabled(site)),
		SSL:                yesNoPlain(site.SSL),
		Profile:            defaultString(site.Profile, "-"),
		Group:              defaultString(site.Group, "-"),
		Tags:               defaultString(joinTags(site.Tags), "-"),
		Config:             site.ConfigPath,
		ClientMaxBodySize:  site.ClientMaxBodySize,
		WebSocket:          site.WebSocket,
		RedirectHTTPS:      site.RedirectHTTPS,
		HTTP2:              site.HTTP2,
		CertificatePath:    site.CertificatePath,
		CertificateKeyPath: site.CertificateKeyPath,
		ACME:               site.ACME,
		ACMEMethod:         site.ACMEMethod,
		DNSProvider:        site.DNSProvider,
	}
}

func writeWebJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func renderWebUIPage(w http.ResponseWriter, page webuiPage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = webUITemplate.Execute(w, page)
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}

func yesNoPlain(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func joinTags(tags []string) string {
	out := ""
	for _, tag := range tags {
		if out != "" {
			out += ", "
		}
		out += tag
	}
	return out
}

var webUITemplate = template.Must(template.New("webui").Parse(webUIHTML))
