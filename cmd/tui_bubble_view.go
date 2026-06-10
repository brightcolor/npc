package cmd

import (
	"fmt"
	"strings"

	"github.com/brightcolor/npc/internal/config"
	"github.com/charmbracelet/lipgloss"
)

var (
	tuiFg     = lipgloss.Color("252")
	tuiDim    = lipgloss.Color("245")
	tuiAccent = lipgloss.Color("39")
	tuiWarn   = lipgloss.Color("220")
	tuiBad    = lipgloss.Color("203")
)

func (m bubbleModel) View() string {
	if m.actions {
		return m.commandPalette()
	}
	return strings.Join([]string{
		m.topBar(),
		m.filterBar(),
		m.table(),
		m.footer(),
	}, "\n")
}

func (m bubbleModel) topBar() string {
	title := style(tuiAccent, true).Render("npc")
	scope := []string{"sites", "problems", "all"}[m.tab]
	update := ""
	if app.update != nil && app.update.UpdateAvailable {
		update = style(tuiWarn, false).Render("  update " + app.update.CurrentVersion + " -> " + app.update.LatestVersion)
	}
	return fmt.Sprintf("%s  %s  %s%s", title, style(tuiDim, false).Render(m.summary()), style(tuiAccent, false).Render("view:"+scope), update)
}

func (m bubbleModel) filterBar() string {
	prompt := "filter"
	if m.search {
		prompt = "filter*"
	}
	value := m.filter
	if value == "" {
		value = "-"
	}
	return style(tuiDim, false).Render(prompt+": ") + value
}

func (m bubbleModel) table() string {
	sites := m.visibleSites()
	if len(sites) == 0 {
		return "\n" + style(tuiDim, false).Render("No matching sites.")
	}
	width := max(88, m.width)
	rows := []string{style(tuiDim, false).Render(header(width))}
	limit := max(8, m.height-6)
	start := 0
	if m.cursor >= limit {
		start = m.cursor - limit + 1
	}
	for i := start; i < len(sites) && i < start+limit; i++ {
		rows = append(rows, m.row(sites[i], i == m.cursor, width))
	}
	return strings.Join(rows, "\n")
}

func header(width int) string {
	return fmt.Sprintf("%s %s %s %s %s %s %s",
		truncateCell("HOSTNAME", 32), "ST", "SSL", truncateCell("CERT", 6),
		truncateCell("GROUP", 12), truncateCell("BACKEND", max(18, width-78)), "ISSUES")
}

func (m bubbleModel) row(site *config.Site, selected bool, width int) string {
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
	issues := siteProblemLabels(site)
	if issues == "" {
		issues = "-"
	}
	line := fmt.Sprintf("%s %s %s %s %s %s %s",
		truncateCell(site.Hostname, 32), state, ssl, truncateCell(cert, 6),
		truncateCell(emptyDash(site.Group), 12),
		truncateCell(site.BackendURL(), max(18, width-78)),
		truncateCell(issues, 18))
	rowStyle := lipgloss.NewStyle().Foreground(tuiFg)
	if siteProblemLabels(site) != "" {
		rowStyle = rowStyle.Foreground(tuiWarn)
	}
	if selected {
		rowStyle = rowStyle.Foreground(lipgloss.Color("230")).Background(tuiAccent).Bold(true)
	}
	return rowStyle.Render(line)
}

func (m bubbleModel) footer() string {
	help := "up/down move  tab view  / filter  enter actions  c create  d docker  f cloudflare  r reload  q quit"
	if m.search {
		help = "type filter  backspace delete  enter/esc apply"
	}
	return "\n" + style(tuiDim, false).Render(help)
}

func (m bubbleModel) commandPalette() string {
	lines := []string{
		style(tuiAccent, true).Render("npc actions"),
		"",
		"s  print list output",
		"e  print status summary",
		"c  create reverse proxy",
		"d  expose Docker container",
		"f  configure Cloudflare DNS-01",
		"",
		style(tuiDim, false).Render("esc close  q quit"),
	}
	return strings.Join(lines, "\n")
}

func style(color lipgloss.Color, bold bool) lipgloss.Style {
	s := lipgloss.NewStyle().Foreground(color)
	if bold {
		s = s.Bold(true)
	}
	return s
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
