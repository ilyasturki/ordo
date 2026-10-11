package app

import (
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/attachments"
)

// uuidPattern matches both the app's own NewID format and standard UUIDs.
var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// numberPattern matches runs of digits. It is applied only after URLs and
// UUIDs are scrubbed out, so it never reports the digit groups inside them.
var numberPattern = regexp.MustCompile(`\d+`)

const yankPreviewMax = 50

// yankItemSet accumulates yank candidates in insertion order, deduping by value
// and labelling each with its preview. It replaces the per-function dedup
// closures the entity scan and the picker menu would otherwise each repeat.
type yankItemSet struct {
	items []components.YankItem
	seen  map[string]struct{}
}

func newYankItemSet() *yankItemSet {
	return &yankItemSet{seen: make(map[string]struct{})}
}

// add appends val (trimmed) unless it is empty or already present.
func (s *yankItemSet) add(val string) {
	val = strings.TrimSpace(val)
	if val == "" {
		return
	}
	if _, dup := s.seen[val]; dup {
		return
	}
	s.seen[val] = struct{}{}
	s.items = append(s.items, components.YankItem{Label: yankPreview(val), Value: val})
}

// extractEntities scans the given texts for copy-worthy substrings: UUIDs,
// URLs, and bare numbers, in that order. Values are deduped across kinds.
func extractEntities(texts ...string) []components.YankItem {
	joined := strings.Join(texts, "\n")
	set := newYankItemSet()

	for _, u := range uuidPattern.FindAllString(joined, -1) {
		set.add(u)
	}
	for _, u := range extractURLs(joined) {
		set.add(u)
	}
	// Blank out URLs/UUIDs first so their internal digits aren't reported.
	scrubbed := uuidPattern.ReplaceAllString(joined, " ")
	scrubbed = urlPattern.ReplaceAllString(scrubbed, " ")
	for _, n := range numberPattern.FindAllString(scrubbed, -1) {
		set.add(n)
	}
	return set.items
}

// buildYankItems assembles the menu for the focused item: its full text first,
// the description (for tasks), then any entities detected within that text.
func (m *model) buildYankItems(pos selection.Position) []components.YankItem {
	set := newYankItemSet()
	addEntities := func(texts ...string) {
		for _, e := range extractEntities(texts...) {
			set.add(e.Value)
		}
	}

	switch pos.Kind {
	case selection.FocusProject:
		set.add(m.project.Name)
		addEntities(m.project.Name)
	case selection.FocusCategory:
		cat := m.project.Categories[pos.CategoryIndex]
		set.add(cat.Name)
		addEntities(cat.Name)
	case selection.FocusTask, selection.FocusDescription:
		task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
		set.add(task.Title)
		if task.Description != "" {
			set.add(task.Description)
		}
		for _, img := range m.taskImages(task) {
			set.add(img.path)
		}
		// A hash's digits are no number worth copying.
		addEntities(attachments.Strip(task.Title), attachments.Strip(task.Description))
	}
	return set.items
}

// yankPartForSelected opens the yank picker for the focused item. With a single
// candidate it copies straight away, mirroring how gx opens a lone URL.
func (m *model) yankPartForSelected() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok {
		return nil
	}
	items := m.buildYankItems(pos)
	switch len(items) {
	case 0:
		return nil
	case 1:
		return m.copyYankItem(items[0])
	default:
		m.ui.YankPicker = components.NewYankPickerState(items)
		m.ui.Modes.ToYankPicker()
		return nil
	}
}

func (m *model) copyYankItem(it components.YankItem) tea.Cmd {
	return m.copyToClipboard(it.Value, it.Label)
}

func yankPreview(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > yankPreviewMax {
		return string(r[:yankPreviewMax-1]) + "…"
	}
	return s
}
