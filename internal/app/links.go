package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/domain"
)

type openURLResultMsg struct {
	label string
	err   error
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

func openURL(link components.Link) tea.Cmd {
	return func() tea.Msg {
		opener := "xdg-open"
		if runtime.GOOS == "darwin" {
			opener = "open"
		}
		c := exec.Command(opener, link.Target)
		err := c.Start()
		if err == nil {
			go func() { _ = c.Wait() }()
		}
		return openURLResultMsg{label: link.Label, err: err}
	}
}

func (m *model) openLinksForSelected() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok || (pos.Kind != selection.FocusTask && pos.Kind != selection.FocusDescription) {
		return nil
	}
	links := m.taskLinks(m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex])
	switch len(links) {
	case 0:
		return nil
	case 1:
		return m.openLink(links[0])
	default:
		m.ui.URLPicker = components.NewURLPickerState(links)
		m.ui.Modes.ToURLPicker()
		return nil
	}
}

func (m *model) taskLinks(task domain.Task) []components.Link {
	var links []components.Link
	for _, u := range extractURLs(task.Title + "\n" + task.Description) {
		links = append(links, components.Link{Label: u, Target: u})
	}
	for _, img := range m.taskImages(task) {
		links = append(links, components.Link{Label: img.ref.Label(), Target: img.path})
	}
	return links
}

func (m *model) openLink(link components.Link) tea.Cmd {
	if filepath.IsAbs(link.Target) {
		if _, err := os.Stat(link.Target); err != nil {
			m.ui.Screen.StatusMsg = missingImageStatus(link, m.deps.Attachments)
			return nil
		}
		// The label hides the path, and xdg-open runs a .desktop file it is handed.
		if !attachments.IsImageFile(link.Target) {
			m.ui.Screen.StatusMsg = link.Target + " is not an image"
			return nil
		}
	}
	return openURL(link)
}

func missingImageStatus(link components.Link, store *attachments.Store) string {
	name := strings.TrimPrefix(link.Label, attachments.Glyph+" ")
	if store != nil && filepath.Dir(link.Target) == store.Dir() {
		return name + " is not synced yet"
	}
	return link.Target + " is missing"
}

func (m *model) handleOpenURLResult(msg openURLResultMsg) {
	if msg.err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Failed to open %s: %v", msg.label, msg.err)
		return
	}
	m.ui.Screen.StatusMsg = "Opened " + msg.label
}

type taskImage struct {
	ref  attachments.Ref
	path string
}

func (m *model) taskImages(task domain.Task) []taskImage {
	store := m.deps.Attachments
	if store == nil {
		return nil
	}
	var images []taskImage
	seen := map[string]bool{}
	for _, ref := range attachments.Refs(task.Title + "\n" + task.Description) {
		path, ok := store.Path(ref.Target)
		if !ok || seen[path] {
			continue
		}
		seen[path] = true
		images = append(images, taskImage{ref: ref, path: path})
	}
	return images
}

func (m model) imageMissing(target string) bool {
	return m.deps.Attachments != nil && m.deps.Attachments.Missing(target)
}
