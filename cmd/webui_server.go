package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/brightcolor/npc/internal/config"
)

type webuiSite struct {
	Hostname string
	Backend  string
	State    string
	SSL      string
	Profile  string
	Group    string
	Tags     string
	Config   string
}

type webuiPage struct {
	Version   string
	Status    map[string]any
	Sites     []webuiSite
	Command   string
	RunResult *webuiRunResult
	Catalog   []webuiCommandExample
}

type webuiRunResult struct {
	OK      bool
	Command string
	Output  string
	Error   string
}

type webuiCommandExample struct {
	Title   string
	Command string
	Risky   bool
}

func webUIHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", webUIIndex)
	mux.HandleFunc("/run", webUIRun)
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = webUITemplate.Execute(w, page)
}

func webUIRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	command := strings.TrimSpace(r.FormValue("command"))
	result := runWebUICommand(command, r.FormValue("confirm") == "yes")
	page, err := loadWebUIPage()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	page.Command = command
	page.RunResult = &result
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = webUITemplate.Execute(w, page)
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
	return webuiPage{Version: app.build.Version, Status: statusData(cfg.SortedSites()), Sites: sites, Catalog: webUICommandCatalog()}, nil
}

func webUISiteRow(site *config.Site) webuiSite {
	return webuiSite{
		Hostname: site.Hostname,
		Backend:  site.BackendURL(),
		State:    onOff(siteEnabled(site)),
		SSL:      yesNoPlain(site.SSL),
		Profile:  defaultString(site.Profile, "-"),
		Group:    defaultString(site.Group, "-"),
		Tags:     defaultString(joinTags(site.Tags), "-"),
		Config:   site.ConfigPath,
	}
}

func writeWebJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func runWebUICommand(command string, confirmed bool) webuiRunResult {
	args, err := splitCommandLine(command)
	if err != nil {
		return webuiRunResult{Command: command, Error: err.Error()}
	}
	if len(args) == 0 {
		return webuiRunResult{Command: command, Error: "command is required"}
	}
	if !allowedWebUICommand(args) {
		return webuiRunResult{Command: command, Error: "command is not allowed from the web UI"}
	}
	if webUIWriteCommand(args) && !confirmed {
		return webuiRunResult{Command: command, Error: "write actions require the confirmation checkbox"}
	}
	exe, err := os.Executable()
	if err != nil {
		return webuiRunResult{Command: command, Error: err.Error()}
	}
	runArgs := append([]string{"--no-upgrade"}, args...)
	ctx := exec.Command(exe, runArgs...)
	var out bytes.Buffer
	ctx.Stdout = &out
	ctx.Stderr = &out
	err = ctx.Run()
	result := webuiRunResult{OK: err == nil, Command: "npc " + strings.Join(args, " "), Output: strings.TrimSpace(out.String())}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func allowedWebUICommand(args []string) bool {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "help") {
		return true
	}
	if isQuickCreateArgs(args) {
		return true
	}
	if strings.HasPrefix(args[0], "-") {
		return false
	}
	if args[0] == "completion" {
		return false
	}
	if args[0] == "webui" {
		return len(args) > 1 && (args[1] == "unit" || args[1] == "install-service" || args[1] == "uninstall-service")
	}
	return isKnownCommand(args[0])
}

func webUIWriteCommand(args []string) bool {
	if isQuickCreateArgs(args) {
		return true
	}
	write := map[string]bool{
		"acme": true, "archive": true, "backup": true, "certs": true, "create": true, "delete": true,
		"disable": true, "edit": true, "enable": true, "install-nginx": true, "maintenance": true,
		"migrate": true, "reload": true, "repair": true, "restart": true, "rollback": true, "set": true,
		"unarchive": true, "uninstall": true, "upgrade": true, "webui": true,
	}
	return write[args[0]]
}

func splitCommandLine(value string) ([]string, error) {
	var args []string
	var b strings.Builder
	quote := rune(0)
	escaped := false
	for _, r := range value {
		switch {
		case escaped:
			b.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				b.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t' || r == '\r' || r == '\n':
			if b.Len() > 0 {
				args = append(args, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if b.Len() > 0 {
		args = append(args, b.String())
	}
	return args, nil
}

func webUICommandCatalog() []webuiCommandExample {
	return []webuiCommandExample{
		{"List sites", "list --wide", false}, {"Search", "search app", false}, {"System status", "status", false},
		{"Doctor", "doctor", false}, {"Docker", "docker", false}, {"Logs", "logs", false},
		{"Show site", "show app.example.com", false}, {"Inspect site", "inspect app.example.com", false},
		{"Create proxy", "create --hostname app.example.com --backend-host 127.0.0.1 --backend-port 3000 --backend-scheme http --non-interactive", true},
		{"Quick create", "app.example.com 3000", true}, {"Edit backend", "edit app.example.com --backend-port 8081", true},
		{"Set metadata", "set app.example.com --group production --tags docker,api", true}, {"Archive", "archive app.example.com", true},
		{"Unarchive", "unarchive app.example.com", true}, {"Enable site", "enable app.example.com", true},
		{"Disable site", "disable app.example.com", true}, {"Delete site", "delete app.example.com --force", true},
		{"Test nginx", "test", false}, {"Reload nginx", "reload", true}, {"Restart nginx", "restart", true},
		{"Install Nginx", "install-nginx --force", true}, {"Certificates", "certs", false},
		{"Renew cert", "certs renew app.example.com", true}, {"Backup", "backup", true},
		{"Backup list", "backup list", false}, {"Restore", "backup restore 20260101T120000Z", true},
		{"Maintenance on", "maintenance enable app.example.com", true}, {"Maintenance off", "maintenance disable app.example.com", true},
		{"Repair", "repair app.example.com", true}, {"Diff", "diff app.example.com", false},
		{"Rollback", "rollback app.example.com", true}, {"Firewall", "firewall suggest", false},
		{"Migrate", "migrate", true}, {"Monitor", "monitor", false}, {"Import", "import", false},
		{"Export", "export", false}, {"Upgrade", "upgrade", true}, {"Web UI unit", "webui unit --listen 127.0.0.1:8088", false},
	}
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
