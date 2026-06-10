package cmd

import (
	"fmt"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/brightcolor/npc/internal/docker"
	"github.com/brightcolor/npc/internal/nginx"
	"github.com/brightcolor/npc/internal/system"
	tea "github.com/charmbracelet/bubbletea"
)

type bubbleModel struct {
	sites   []*config.Site
	filter  string
	cursor  int
	tab     int
	width   int
	height  int
	search  bool
	actions bool
	action  string
	err     error
}

func newBubbleModel() (bubbleModel, error) {
	cfg, err := config.Load("")
	if err != nil {
		return bubbleModel{}, err
	}
	return bubbleModel{sites: cfg.SortedSites(), width: 100, height: 30}, nil
}

func (m bubbleModel) Init() tea.Cmd { return nil }

func (m bubbleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m bubbleModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.search {
		return m.handleSearchKey(msg)
	}
	if m.actions {
		return m.handleActionKey(msg)
	}
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "tab":
		m.tab = (m.tab + 1) % 3
		m.cursor = 0
	case "/":
		m.search = true
	case "enter":
		m.actions = true
	case "c":
		m.action = "create"
		return m, tea.Quit
	case "d":
		m.action = "docker"
		return m, tea.Quit
	case "f":
		m.action = "cloudflare"
		return m, tea.Quit
	case "u":
		if app.update != nil && app.update.UpdateAvailable {
			m.action = "upgrade"
			return m, tea.Quit
		}
	case "r":
		next, err := newBubbleModel()
		m.sites, m.err = next.sites, err
	}
	return m, nil
}

func (m bubbleModel) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.search = false
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
		}
	}
	m.cursor = 0
	return m, nil
}

func (m bubbleModel) handleActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.actions = false
	case "s":
		m.action = "list"
		return m, tea.Quit
	case "e":
		m.action = "status"
		return m, tea.Quit
	case "c":
		m.action = "create"
		return m, tea.Quit
	case "d":
		m.action = "docker"
		return m, tea.Quit
	case "f":
		m.action = "cloudflare"
		return m, tea.Quit
	}
	return m, nil
}

func (m *bubbleModel) move(delta int) {
	items := m.visibleSites()
	if len(items) == 0 {
		m.cursor = 0
		return
	}
	m.cursor = (m.cursor + delta + len(items)) % len(items)
}

func (m bubbleModel) visibleSites() []*config.Site {
	q := siteQuery{search: m.filter}
	if m.tab == 1 {
		q.search = ""
		return problemSites(m.sites)
	}
	if m.tab == 2 {
		q.includeArchived = true
	}
	return q.apply(m.sites)
}

func problemSites(sites []*config.Site) []*config.Site {
	out := []*config.Site{}
	for _, site := range sites {
		if siteProblemLabels(site) != "" {
			out = append(out, site)
		}
	}
	return out
}

func (m bubbleModel) summary() string {
	active, problems := 0, 0
	for _, site := range m.sites {
		if siteEnabled(site) {
			active++
		}
		if siteProblemLabels(site) != "" {
			problems++
		}
	}
	return fmt.Sprintf("Nginx %s  Docker %s  Sites %d/%d active  Issues %d",
		yesBadge(system.Exists("nginx") && nginx.ServiceActive()), yesBadge(docker.Installed()), active, len(m.sites), problems)
}

func yesBadge(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func truncateCell(value string, width int) string {
	value = strings.ReplaceAll(value, "\t", " ")
	if len(value) <= width {
		return value + strings.Repeat(" ", width-len(value))
	}
	if width <= 1 {
		return value[:width]
	}
	return value[:width-1] + "…"
}
