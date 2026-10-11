package app

import (
	"fmt"
	"slices"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/domain"
	"github.com/ilyasturki/phasionary/internal/ui"
)

// SearchState backs the `/` text search: an incremental, vim/less-style
// find-and-jump over category names, task titles, and task descriptions. The
// search highlights every match and moves the cursor to one; n/N step between
// matches. It complements (does not replace) the `f` filter, which hides rows.
type SearchState struct {
	input          textinput.Model
	query          string    // committed query, "" when no search is in effect: drives the persistent highlight + n/N
	originSelected int       // cursor index when the search began (Esc restores it)
	originScroll   int       // scroll offset when the search began
	originFold     FoldState // fold state when the search began (Esc restores it)
}

// searchQuery is the text to highlight: the live input while typing, the
// committed query after Enter, or empty when no search is in effect.
func (m model) searchQuery() string {
	if m.ui.Modes.IsSearch() {
		return m.ui.Search.input.Value()
	}
	return m.ui.Search.query
}

func (m model) searchMatchStyle(current bool) lipgloss.Style {
	if current {
		return ui.SearchCurrentMatchStyle
	}
	return ui.SearchMatchStyle
}

// startSearch enters search mode, snapshotting the view so Esc can restore it.
func (m *model) startSearch() tea.Cmd {
	if m.ui.Selection.IsEmpty() {
		return nil
	}
	ti := newTextInput()
	cmd := ti.Focus()
	m.ui.Search = SearchState{
		input:          ti,
		originSelected: m.selected(),
		originScroll:   m.ui.Screen.TopRow,
		originFold:     m.ui.Fold.Clone(),
	}
	m.ui.Modes.ToSearch()
	m.ensureVisible()
	return cmd
}

func (m model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.cancelSearch()
		return m, nil
	case "enter":
		m.commitSearch()
		return m, nil
	}
	var cmd tea.Cmd
	m.ui.Search.input, cmd = m.ui.Search.input.Update(msg)
	sanitizeInput(&m.ui.Search.input)
	m.previewSearch()
	return m, cmd
}

// restoreSearchOrigin returns the cursor, scroll, and fold state to the snapshot
// taken when the search began. Restored folds are in-memory only until a caller
// persists them with saveFoldState.
func (m *model) restoreSearchOrigin() {
	m.restoreFolds(m.ui.Search.originFold)
	m.ui.Selection.SetSelected(m.ui.Search.originSelected)
	m.ui.Screen.TopRow = m.ui.Search.originScroll
}

// previewSearch is the incremental step: reset to the pre-search view, then jump
// the cursor to the first match for the current input, revealing a folded
// category only for that match. Fold changes here are in-memory only (never
// persisted), so they unwind cleanly on Esc.
func (m *model) previewSearch() {
	m.restoreSearchOrigin()

	q := m.ui.Search.input.Value()
	if q == "" {
		m.ensureVisible()
		return
	}
	rows := m.searchRows()
	from := m.currentLogicalIndex(rows)
	if idx := nextMatchingRow(rows, from, 1, true, m.rowMatcher(q)); idx >= 0 {
		m.landOnRow(rows[idx])
		return
	}
	m.ensureVisible()
}

// commitSearch confirms the query (Enter). On a hit it keeps the cursor on the
// match, persists any revealed fold, and leaves the query in effect so the
// highlight stays and n/N work. On an empty query or miss it restores the
// pre-search view.
func (m *model) commitSearch() {
	q := m.ui.Search.input.Value()
	m.ui.Modes.ToNormal()
	m.restoreSearchOrigin()

	if q != "" {
		rows := m.searchRows()
		from := m.currentLogicalIndex(rows)
		if idx := nextMatchingRow(rows, from, 1, true, m.rowMatcher(q)); idx >= 0 {
			m.ui.Search.query = q
			m.landOnRow(rows[idx])
			m.saveFoldState()
			return
		}
		m.ui.Screen.StatusMsg = fmt.Sprintf("No matches for %q", q)
	}

	m.clearSearch()
	m.ensureVisible()
}

// cancelSearch (Esc) abandons the search and restores the pre-search cursor,
// scroll, and fold state.
func (m *model) cancelSearch() {
	m.ui.Modes.ToNormal()
	m.restoreSearchOrigin()
	m.clearSearch()
	m.ensureVisible()
}

// clearSearch drops the committed query, removing the persistent highlight and
// disabling n/N. In normal mode this is the vim `:nohlsearch` gesture (Esc); the
// cancel and miss paths reuse it to reset.
func (m *model) clearSearch() {
	m.ui.Search.query = ""
}

func (m *model) searchNext() { m.moveToMatch(1) }
func (m *model) searchPrev() { m.moveToMatch(-1) }

// moveToMatch jumps to the next (dir=+1) or previous (dir=-1) match relative to
// the cursor, wrapping around, revealing a folded category if needed.
func (m *model) moveToMatch(dir int) {
	if m.ui.Search.query == "" {
		return
	}
	rows := m.searchRows()
	from := m.currentLogicalIndex(rows)
	idx := nextMatchingRow(rows, from, dir, false, m.rowMatcher(m.ui.Search.query))
	if idx < 0 {
		return
	}
	m.landOnRow(rows[idx])
	m.saveFoldState()
}

// restoreFolds replaces the fold state and rebuilds positions. It does not
// persist — callers that want the change saved call saveFoldState explicitly.
func (m *model) restoreFolds(fold FoldState) {
	m.ui.Fold = fold.Clone()
	m.rebuildPositions()
}

// searchRows lists every searchable row in display order, ignoring fold state
// (so folded matches stay reachable) but honoring the active task filter (so
// filtered-out tasks stay hidden). It reuses the shared row builder with folds
// disabled, dropping the leading project row (never a search target) so the
// search row model stays in lockstep with the main list.
func (m model) searchRows() []selection.Position {
	positions := rebuildPositions(m.project.Categories, &m.ui.Filter, nil, m.ui.Screen.ExpandDescriptions, "")
	return positions[1:] // positions[0] is always the FocusProject row
}

func (m model) rowMatcher(query string) func(selection.Position) bool {
	return func(r selection.Position) bool { return m.rowMatches(r, query) }
}

// rowMatches reports whether a logical row matches the query. A task row matches
// its title, plus its description while descriptions are collapsed (so collapsed
// description hits still surface, on the parent task). An expanded description
// row owns its own description match.
func (m model) rowMatches(r selection.Position, query string) bool {
	if r.CategoryIndex < 0 || r.CategoryIndex >= len(m.project.Categories) {
		return false
	}
	cat := m.project.Categories[r.CategoryIndex]
	switch r.Kind {
	case selection.FocusCategory:
		return ui.Contains(cat.Name, query)
	case selection.FocusSeparator:
		if r.TaskIndex < 0 || r.TaskIndex >= len(cat.Tasks) {
			return false
		}
		return ui.Contains(cat.Tasks[r.TaskIndex].Title, query)
	case selection.FocusTask:
		if r.TaskIndex < 0 || r.TaskIndex >= len(cat.Tasks) {
			return false
		}
		task := cat.Tasks[r.TaskIndex]
		if ui.Contains(task.Title, query) {
			return true
		}
		return !m.ui.Screen.ExpandDescriptions && ui.Contains(task.Description, query)
	case selection.FocusDescription:
		if r.TaskIndex < 0 || r.TaskIndex >= len(cat.Tasks) {
			return false
		}
		return ui.Contains(cat.Tasks[r.TaskIndex].Description, query)
	}
	return false
}

// currentLogicalIndex maps the current cursor position to its index in rows, or
// -1 when the cursor isn't on a searchable row (e.g. the project line).
func (m model) currentLogicalIndex(rows []selection.Position) int {
	pos, ok := m.selectedPosition()
	if !ok {
		return -1
	}
	if i := slices.Index(rows, pos); i >= 0 {
		return i
	}
	// An expanded task's description row exists only in the main list.
	if pos.Kind == selection.FocusDescription {
		pos.Kind = selection.FocusTask
		return slices.Index(rows, pos)
	}
	return -1
}

// nextMatchingRow scans rows from `from` in direction dir (+1/-1), wrapping
// once, and returns the index of the first match. includeFrom controls whether
// `from` itself is a candidate. A negative `from` starts at the first (dir>0) or
// last (dir<0) row. Returns -1 when nothing matches.
func nextMatchingRow(rows []selection.Position, from, dir int, includeFrom bool, match func(selection.Position) bool) int {
	n := len(rows)
	if n == 0 {
		return -1
	}
	var start int
	switch {
	case from < 0:
		if dir > 0 {
			start = 0
		} else {
			start = n - 1
		}
		includeFrom = true
	case includeFrom:
		start = from
	default:
		start = ((from+dir)%n + n) % n
	}
	for i := 0; i < n; i++ {
		idx := ((start+dir*i)%n + n) % n
		if match(rows[idx]) {
			return idx
		}
	}
	return -1
}

// landOnRow moves the cursor onto the given logical row, unfolding whatever
// hides it. A collapsed description row resolves to its parent task row.
func (m *model) landOnRow(r selection.Position) {
	if r.CategoryIndex < 0 || r.CategoryIndex >= len(m.project.Categories) {
		return
	}
	unfolded := false
	for _, id := range m.hidingFolds(r) {
		if m.ui.Fold.IsFolded(id) {
			m.ui.Fold.Toggle(id)
			unfolded = true
		}
	}
	if unfolded {
		m.rebuildPositions()
	}
	m.selectItem(r.Kind, r.ID)
	m.ensureVisible()
}

func (m *model) hidingFolds(r selection.Position) []string {
	cat := m.project.Categories[r.CategoryIndex]
	var ids []string
	if r.Kind != selection.FocusCategory && r.TaskIndex >= 0 && r.TaskIndex < len(cat.Tasks) {
		ids = append(ids, cat.ID)
		if parent := domain.ParentIndex(cat.Tasks, r.TaskIndex); parent >= 0 {
			ids = append(ids, cat.Tasks[parent].ID)
		}
	}
	if cat.ParentID != "" {
		ids = append(ids, cat.ParentID)
	}
	return ids
}

// matchCount is the number of rows the query matches, shown in the search bar.
func (m model) matchCount(query string) int {
	if query == "" {
		return 0
	}
	match := m.rowMatcher(query)
	count := 0
	for _, r := range m.searchRows() {
		if match(r) {
			count++
		}
	}
	return count
}

// renderSearchBar draws the bottom prompt while typing.
func (m model) renderSearchBar() string {
	value := m.ui.Search.input.Value()
	return m.filterPromptRow(m.ui.Search.input, value, m.matchCount(value), m.ui.Screen.Width)
}
