package cmd

import (
	"fmt"
	"strings"
)

func panel(title string, lines ...string) string {
	width := 78
	var b strings.Builder
	b.WriteString(accent("+"))
	b.WriteString(accent(strings.Repeat("-", width-2)))
	b.WriteString(accent("+\n"))
	b.WriteString(accent("| ") + bold(title) + strings.Repeat(" ", width-3-len(title)) + accent("|\n"))
	b.WriteString(accent("|") + strings.Repeat(" ", width-2) + accent("|\n"))
	for _, line := range lines {
		if len(line) > width-4 {
			line = line[:width-7] + "..."
		}
		b.WriteString(accent("| ") + line + strings.Repeat(" ", width-3-len(line)) + accent("|\n"))
	}
	b.WriteString(accent("+"))
	b.WriteString(accent(strings.Repeat("-", width-2)))
	b.WriteString(accent("+"))
	return b.String()
}

func section(title string) string { return "\n" + accent("== ") + bold(title) + accent(" ==") }

func yesNo(v bool) string {
	if v {
		return ok("yes")
	}
	return fail("no")
}

func badge(v bool) string {
	if v {
		return ok("yes")
	}
	return fail("no")
}

func emptyState(title, body string) string { return panel(title, body) }

func ok(s string) string     { return "\033[32m" + s + "\033[0m" }
func fail(s string) string   { return "\033[31m" + s + "\033[0m" }
func bold(s string) string   { return "\033[1m" + s + "\033[0m" }
func cyan(s string) string   { return "\033[36m" + s + "\033[0m" }
func dim(s string) string    { return "\033[2m" + s + "\033[0m" }
func warn(s string) string   { return "\033[33m" + s + "\033[0m" }
func accent(s string) string { return "\033[35m" + s + "\033[0m" }

func pill(s string) string {
	return fmt.Sprintf("%s%s%s", accent("["), ok(s), accent("]"))
}
