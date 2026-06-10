package cmd

import (
	"fmt"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/charmbracelet/lipgloss"
)

var (
	tuiAccent = lipgloss.Color("63")
	tuiMuted  = lipgloss.Color("241")
	tuiOK     = lipgloss.Color("42")
	tuiWarn   = lipgloss.Color("214")
	tuiBad    = lipgloss.Color("203")
)

func (m bubbleModel) View() string {
	var b strings.Builder
	b.WriteString(m.headerView())
	b.WriteString("\n")
	if m.actions {
		b.WriteString(m.actionView())
	} else {
		b.WriteString(m.mainView())
	}
	b.WriteString("\n")
	b.WriteString(m.helpView())
	return b.String()
}

func (m bubbleModel) headerView() string {
	title := lipgloss.NewStyle().Foreground(tuiAccent).Bold(true).Render("npc")
	sub := lipgloss.NewStyle().Foreground(tuiMuted).Render("Nginx Proxy Configurator")
	update := ""
	if app.update != nil && app.update.UpdateAvailable {
		update = lipgloss.NewStyle().Foreground(tuiWarn).Render(" update " + app.update.CurrentVersion + " -> " + app.update.LatestVersion)
	}
	return title + "  " + sub + "  " + update + "\n" + lipgloss.NewStyle().Foreground(tuiMuted).Render(m.summary())
}

func (m bubbleModel) mainView() string {
	leftWidth := max(55, m.width/2)
	rightWidth := max(36, m.width-leftWidth-4)
	left := lipgloss.NewStyle().Width(leftWidth).Render(m.tabsView() + "\n" + m.tableView(leftWidth))
	right := lipgloss.NewStyle().Width(rightWidth).Render(m.detailView(rightWidth))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
}

func (m bubbleModel) tabsView() string {
	names := []string{"Sites", "Problems", "All"}
	parts := make([]string, len(names))
	for i, name := range names {
		style := lipgloss.NewStyle().Padding(0, 1).Foreground(tuiMuted)
		if m.tab == i {
			style = style.Foreground(tuiAccent).Bold(true).Underline(true)
		}
		parts[i] = style.Render(name)
	}
	search := lipgloss.NewStyle().Foreground(tuiMuted).Render(" / " + m.filter)
	if m.search {
		search = lipgloss.NewStyle().Foreground(tuiWarn).Render(" search: " + m.filter)
	}
	return strings.Join(parts, "") + search
}

func (m bubbleModel) tableView(width int) string {
	sites := m.visibleSites()
	if len(sites) == 0 {
		return lipgloss.NewStyle().Foreground(tuiMuted).Render("No matching sites.")
	}
	rows := []string{lipgloss.NewStyle().Foreground(tuiMuted).Render("HOSTNAME                        ST SSL CERT GROUP      BACKEND")}
	limit := max(5, m.height-10)
	start := 0
	if m.cursor >= limit {
		start = m.cursor - limit + 1
	}
	for i := start; i < len(sites) && i < start+limit; i++ {
		rows = append(rows, m.siteRow(sites[i], i == m.cursor, width))
	}
	return strings.Join(rows, "\n")
}

func (m bubbleModel) siteRow(site *config.Site, selected bool, width int) string {
	state := "off"
	if siteEnabled(site) {
		state = "on "
	}
	ssl := "no "
	if site.SSL {
		ssl = "yes"
	}
	cert := "-"
	if site.CertificatePath != "" {
		cert = fmt.Sprintf("%dd", certDays(site))
	}
	line := fmt.Sprintf("%s %s %s %4s %-10s %s",
		truncateCell(site.Hostname, 30), state, ssl, cert, truncateCell(site.Group, 10), truncateCell(site.BackendURL(), max(12, width-58)))
	style := lipgloss.NewStyle()
	if selected {
		style = style.Foreground(tuiAccent).Bold(true)
	} else if siteProblemLabels(site) != "" {
		style = style.Foreground(tuiWarn)
	}
	return style.Render(line)
}

func (m bubbleModel) detailView(width int) string {
	sites := m.visibleSites()
	if len(sites) == 0 || m.cursor >= len(sites) {
		return box(width, "Details", "No site selected")
	}
	site := sites[m.cursor]
	lines := []string{
		"Host:    " + site.Hostname,
		"Alias:   " + emptyDash(site.Alias),
		"Group:   " + emptyDash(site.Group),
		"Tags:    " + emptyDash(strings.Join(site.Tags, ",")),
		"State:   " + stateText(site),
		"Backend: " + site.BackendURL(),
		"Profile: " + emptyDash(site.Profile),
		"SSL:     " + fmt.Sprintf("%v", site.SSL),
		"ACME:    " + emptyDash(site.ACMEMethod),
		"Issues:  " + emptyDash(siteProblemLabels(site)),
	}
	if app.update != nil && app.update.UpdateAvailable {
		lines = append(lines, "", "Update:", app.update.CurrentVersion+" -> "+app.update.LatestVersion)
	}
	if changelog := compactChangelog(); changelog != "" && m.tab == 2 {
		lines = append(lines, "", "Changelog:", changelog)
	}
	return box(width, "Details", lines...)
}

func (m bubbleModel) actionView() string {
	return box(max(50, m.width/2), "Actions",
		"s  show/list output",
		"e  status summary",
		"c  create reverse proxy",
		"d  expose Docker container",
		"f  configure Cloudflare DNS-01",
		"esc close actions")
}

func (m bubbleModel) helpView() string {
	if m.search {
		return lipgloss.NewStyle().Foreground(tuiMuted).Render("type to filter · enter/esc stop search · backspace delete")
	}
	return lipgloss.NewStyle().Foreground(tuiMuted).Render("↑/↓ move · tab switch · / search · enter actions · c create · d docker · f cloudflare · r reload · q quit")
}

func box(width int, title string, lines ...string) string {
	style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(tuiAccent).Padding(0, 1).Width(width)
	body := lipgloss.NewStyle().Bold(true).Foreground(tuiAccent).Render(title) + "\n"
	body += strings.Join(lines, "\n")
	return style.Render(body)
}

func stateText(site *config.Site) string {
	if site.Archived {
		return "archived"
	}
	if siteEnabled(site) {
		return "enabled"
	}
	return "disabled"
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
