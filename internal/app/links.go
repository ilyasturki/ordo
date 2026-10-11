package app

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/phasionary/internal/app/components"
	"github.com/ilyasturki/phasionary/internal/app/selection"
)

type openURLResultMsg struct {
	url string
	err error
}

// urlPattern excludes whitespace and common markdown/HTML delimiters
// (<, >, ", ', backtick) so URLs embedded in surrounding markup parse cleanly.
var urlPattern = regexp.MustCompile(`https?://[^\s<>"'\x60]+`)

const urlTrailingTrim = ".,;:!?)]}>"

func extractURLs(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, m := range matches {
		u := strings.TrimRight(m, urlTrailingTrim)
		if u == "" {
			continue
		}
		if _, dup := seen[u]; dup {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		opener := "xdg-open"
		if runtime.GOOS == "darwin" {
			opener = "open"
		}
		c := exec.Command(opener, url)
		err := c.Start()
		if err == nil {
			go func() { _ = c.Wait() }()
		}
		return openURLResultMsg{url: url, err: err}
	}
}

func (m *model) openLinksForSelected() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		return nil
	}
	task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
	urls := extractURLs(task.Title)
	switch len(urls) {
	case 0:
		return nil
	case 1:
		return openURL(urls[0])
	default:
		m.ui.URLPicker = components.NewURLPickerState(urls)
		m.ui.Modes.ToURLPicker()
		return nil
	}
}

func (m *model) handleOpenURLResult(msg openURLResultMsg) {
	if msg.err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Failed to open URL: %v", msg.err)
		return
	}
	m.ui.Screen.StatusMsg = "Opened " + msg.url
}
