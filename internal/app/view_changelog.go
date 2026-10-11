package app

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/ilyasturki/ordo"
	"github.com/ilyasturki/ordo/internal/ui"
	"github.com/ilyasturki/ordo/internal/version"
)

// The title, the blank under it, the two scroll indicators, the blank above
// the hints, and the hints.
const changelogOwnRows = 6

var markdownLink = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

func (m model) changelogView() string {
	lines := m.changelogLines()
	height := m.changelogViewportHeight()
	start := min(m.ui.Changelog.ScrollOffset, m.changelogMaxScroll())
	end := min(start+height, len(lines))

	out := []string{
		ui.DialogTitleStyle.Render("Changelog") + ui.MutedStyle.Render("  running "+version.Version),
		"",
	}
	if start > 0 {
		out = append(out, ui.MutedStyle.Render(scrollMoreAbove))
	}
	out = append(out, lines[start:end]...)
	if end < len(lines) {
		out = append(out, ui.MutedStyle.Render(scrollMoreBelow))
	}
	out = append(out, "", ui.RenderHints([]ui.Hint{{Key: "esc/q", Label: "close"}}))
	return m.dialogStyle().Render(strings.Join(out, "\n"))
}

func (m model) changelogLines() []string {
	return renderChangelog(ordo.Changelog, m.dialogWidth())
}

func (m model) changelogViewportHeight() int {
	return ui.DialogBodyHeight(m.ui.Screen.Height, changelogOwnRows, 5)
}

func (m model) changelogMaxScroll() int {
	return max(len(m.changelogLines())-m.changelogViewportHeight(), 0)
}

// renderChangelog lays CHANGELOG.md out for the dialog: the preamble before the
// first release and releases with no notes are dropped.
func renderChangelog(src string, width int) []string {
	var out []string
	for _, release := range strings.Split(src, "\n## ")[1:] {
		heading, body, _ := strings.Cut(release, "\n")
		if strings.TrimSpace(body) == "" {
			continue
		}
		if len(out) > 0 {
			out = append(out, "", "")
		}
		name, date, _ := strings.Cut(heading, " - ")
		title := ui.DialogTitleStyle.Render(strings.Trim(name, "[]"))
		if date != "" {
			title += ui.MutedStyle.Render("  " + date)
		}
		out = append(out, title)
		out = append(out, renderChangelogBody(body, width)...)
	}
	return out
}

func renderChangelogBody(body string, width int) []string {
	var out []string
	gap, underSection := true, false
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			gap = !underSection
			continue
		}
		if gap {
			out = append(out, "")
			gap = false
		}
		line = markdownLink.ReplaceAllString(line, "$1")
		var section string
		section, underSection = strings.CutPrefix(line, "### ")
		if underSection {
			out = append(out, ui.HeaderStyle.Render(section))
		} else if item, ok := strings.CutPrefix(line, "- "); ok {
			out = append(out, wrapChangelogLine(item, "• ", "  ", width)...)
		} else {
			out = append(out, wrapChangelogLine(line, "", "", width)...)
		}
	}
	return out
}

// wrapChangelogLine wraps before styling, so no escape sequence straddles a row
// break; a code span split by the wrap is restyled on its next row.
func wrapChangelogLine(text, first, rest string, width int) []string {
	rows := strings.Split(ansi.Wrap(text, max(width-len(rest), 1), ""), "\n")
	inCode := false
	for i, row := range rows {
		prefix := rest
		if i == 0 {
			prefix = first
		}
		var styled string
		styled, inCode = styleCodeSpans(row, inCode)
		rows[i] = prefix + styled
	}
	return rows
}

func styleCodeSpans(row string, inCode bool) (string, bool) {
	var b strings.Builder
	for i, part := range strings.Split(row, "`") {
		if i > 0 {
			inCode = !inCode
		}
		if inCode && part != "" {
			b.WriteString(ui.ShortcutKeyStyle.Render(part))
		} else {
			b.WriteString(part)
		}
	}
	return b.String(), inCode
}
