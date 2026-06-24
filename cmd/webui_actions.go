package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/brightcolor/npc/internal/backup"
	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/paths"
	"github.com/brightcolor/npc/internal/renderer"
	"github.com/brightcolor/npc/internal/revision"
	"github.com/brightcolor/npc/internal/system"
	"github.com/brightcolor/npc/internal/validate"
)

type webuiNotice struct {
	Kind    string
	Title   string
	Message string
}

func webUIActions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	notice := runWebUIAction(r)
	page, err := loadWebUIPage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page.Notice = &notice
	renderWebUIPage(w, page)
}

func runWebUIAction(r *http.Request) webuiNotice {
	action := r.FormValue("action")
	if requiresWebUIConfirm(action) && r.FormValue("confirm") != "yes" {
		return webUINotice("danger", "Confirmation required", "Enable the confirmation checkbox before applying this change.")
	}
	var err error
	switch action {
	case "create":
		err = webUICreate(r)
	case "edit":
		err = webUIEdit(r)
	case "enable":
		err = webUIEnableDisable(r, true)
	case "disable":
		err = webUIEnableDisable(r, false)
	case "delete":
		err = webUIDelete(r)
	case "cert-set":
		err = webUICertSet(r)
	case "cert-delete":
		err = webUICertDelete(r)
	case "import":
		err = webUIImport(r)
	default:
		err = fmt.Errorf("unknown action %q", action)
	}
	if err != nil {
		return webUINotice("danger", "Action failed", err.Error())
	}
	return webUINotice("success", "Action completed", "The requested change was applied.")
}

func requiresWebUIConfirm(action string) bool {
	return action != ""
}

func webUINotice(kind, title, message string) webuiNotice {
	return webuiNotice{Kind: kind, Title: title, Message: message}
}

func webUICreate(r *http.Request) error {
	port, err := strconv.Atoi(r.FormValue("backend_port"))
	if err != nil {
		return validationError{fmt.Errorf("backend port must be a number")}
	}
	return executeCreate(createOptions{
		hostname:          r.FormValue("hostname"),
		backendHost:       defaultString(r.FormValue("backend_host"), "127.0.0.1"),
		backendPort:       port,
		backendScheme:     defaultString(r.FormValue("backend_scheme"), "http"),
		profile:           defaultString(r.FormValue("profile"), "generic"),
		clientMaxBodySize: defaultString(r.FormValue("client_max_body_size"), "100M"),
		ssl:               formBool(r, "ssl"),
		acme:              formBool(r, "acme"),
		acmeMethod:        r.FormValue("acme_method"),
		dnsProvider:       r.FormValue("dns_provider"),
		certPath:          r.FormValue("cert_path"),
		keyPath:           r.FormValue("key_path"),
		redirectHTTPS:     formBool(r, "redirect_https"),
		websocket:         formBool(r, "websocket"),
		http2:             formBool(r, "http2"),
		securityHeaders:   r.FormValue("security_headers"),
		accessLog:         formBool(r, "access_log"),
		errorLog:          formBool(r, "error_log"),
		nonInteractive:    true,
		force:             formBool(r, "force"),
		noReload:          formBool(r, "no_reload"),
	})
}

func webUIEdit(r *http.Request) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	cfg, site, err := webUILoadSite(r.FormValue("hostname"))
	if err != nil {
		return err
	}
	if host := r.FormValue("backend_host"); host != "" {
		if err := validate.BackendHost(host); err != nil {
			return validationError{err}
		}
		site.BackendHost = host
	}
	if portValue := r.FormValue("backend_port"); portValue != "" {
		port, err := strconv.Atoi(portValue)
		if err != nil {
			return validationError{fmt.Errorf("backend port must be a number")}
		}
		if err := validate.Port(port); err != nil {
			return validationError{err}
		}
		site.BackendPort = port
	}
	if scheme := r.FormValue("backend_scheme"); scheme != "" {
		if err := validate.BackendScheme(scheme); err != nil {
			return validationError{err}
		}
		site.BackendScheme = scheme
	}
	site.Profile = defaultString(r.FormValue("profile"), site.Profile)
	site.ClientMaxBodySize = defaultString(r.FormValue("client_max_body_size"), site.ClientMaxBodySize)
	site.WebSocket = formBool(r, "websocket")
	site.RedirectHTTPS = formBool(r, "redirect_https")
	site.HTTP2 = formBool(r, "http2")
	site.UpdatedAt = time.Now().UTC()
	return webUIRewriteSite(cfg, site, true, !formBool(r, "no_reload"))
}

func webUIEnableDisable(r *http.Request, enable bool) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	_, site, err := webUILoadSite(r.FormValue("hostname"))
	if err != nil {
		return err
	}
	if enable {
		err = nginx.Enable(site.ConfigPath, site.EnabledPath)
	} else {
		err = nginx.Disable(site.EnabledPath)
	}
	if err != nil {
		return err
	}
	out, err := nginx.Reload()
	if err != nil {
		return nginxTestError{fmt.Errorf("%s", out)}
	}
	return nil
}

func webUIDelete(r *http.Request) error {
	if err := system.RequireRoot(); err != nil {
		return permissionError{err}
	}
	cfg, site, err := webUILoadSite(r.FormValue("hostname"))
	if err != nil {
		return err
	}
	if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath, site.CertificatePath, site.CertificateKeyPath); err != nil {
		return err
	}
	_ = nginx.Disable(site.EnabledPath)
	if formBool(r, "remove_config") {
		_ = os.Remove(site.ConfigPath)
	}
	if formBool(r, "remove_certs") {
		_ = os.Remove(site.CertificatePath)
		_ = os.Remove(site.CertificateKeyPath)
	}
	if formBool(r, "remove_metadata") {
		delete(cfg.Sites, site.Hostname)
		return config.Save("", cfg)
	}
	return nil
}

func webUILoadSite(hostname string) (*config.Config, *config.Site, error) {
	cfg, err := loadManagedConfig()
	if err != nil {
		return nil, nil, err
	}
	site, ok := cfg.FindSite(hostname)
	if !ok {
		return nil, nil, validationError{fmt.Errorf("site %s is not managed by npc", hostname)}
	}
	return cfg, site, nil
}

func webUIRewriteSite(cfg *config.Config, site *config.Site, doBackup, doReload bool) error {
	content, err := renderer.RenderSite(site)
	if err != nil {
		return err
	}
	if doBackup {
		if _, err := backup.Create(paths.ConfigFile, site.ConfigPath, site.EnabledPath); err != nil {
			return err
		}
	}
	if err := nginx.WriteSite(site.ConfigPath, content); err != nil {
		return err
	}
	if _, err := revision.Save(site, content); err != nil {
		return err
	}
	out, err := nginx.Test()
	if err != nil {
		return nginxTestError{fmt.Errorf("nginx -t failed, reload skipped: %s", out)}
	}
	site.LastNginxTest = time.Now().UTC().Format(time.RFC3339)
	if doReload {
		if _, err := nginx.Reload(); err != nil {
			return err
		}
		site.LastReload = time.Now().UTC().Format(time.RFC3339)
	}
	cfg.Sites[site.Hostname] = site
	return config.Save("", cfg)
}

func formBool(r *http.Request, name string) bool {
	return r.FormValue(name) == "on" || r.FormValue(name) == "yes" || r.FormValue(name) == "true"
}
