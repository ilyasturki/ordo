package app

import (
	"phasionary/internal/app/selection"
	"phasionary/internal/data"
	"phasionary/internal/domain"
)

// cursorKindNames maps focus kinds to the strings stored in state.json. The
// names are part of the on-disk format: renaming one silently drops every saved
// cursor of that kind, which degrades to "start on the first task".
var cursorKindNames = map[selection.FocusKind]string{
	selection.FocusProject:     "project",
	selection.FocusCategory:    "category",
	selection.FocusTask:        "task",
	selection.FocusSeparator:   "separator",
	selection.FocusDescription: "description",
}

func cursorKindFromName(name string) (selection.FocusKind, bool) {
	for kind, n := range cursorKindNames {
		if n == name {
			return kind, true
		}
	}
	return 0, false
}

// currentCursor describes the focused row by stable IDs so it can be resolved
// again against a project that changed in the meantime. Reports false when
// nothing is selected, which is the case for an empty project.
func (m model) currentCursor() (data.Cursor, bool) {
	pos, ok := m.selectedPosition()
	if !ok {
		return data.Cursor{}, false
	}
	name, ok := cursorKindNames[pos.Kind]
	if !ok {
		return data.Cursor{}, false
	}

	cursor := data.Cursor{Kind: name}
	if pos.Kind == selection.FocusProject {
		return cursor, true
	}
	if pos.CategoryIndex < 0 || pos.CategoryIndex >= len(m.project.Categories) {
		return data.Cursor{}, false
	}

	cat := m.project.Categories[pos.CategoryIndex]
	cursor.CategoryID = cat.ID
	// Separators and descriptions share the task addressing: both hang off an
	// entry in the category's Tasks slice. An out-of-range index leaves TaskID
	// empty, which resolves back to the category header rather than nothing.
	if pos.Kind != selection.FocusCategory && pos.TaskIndex >= 0 && pos.TaskIndex < len(cat.Tasks) {
		cursor.TaskID = cat.Tasks[pos.TaskIndex].ID
	}
	return cursor, true
}

// saveCursorState writes the focused row for the current project straight to
// disk. Leaving a project has to do this rather than queue it: switching A → B →
// A inside one save interval would otherwise restore A's stale row, since
// applyStoredCursor reads back what is stored, not what is queued.
func (m *model) saveCursorState() {
	if m.deps.StateManager == nil || m.project.ID == "" {
		return
	}
	// A project with nothing selected stores the zero cursor, clearing any entry
	// left over from when it still had rows.
	cursor, _ := m.currentCursor()
	_ = m.deps.StateManager.SetCursor(m.project.ID, cursor)
	m.forgetPendingCursor(m.project.ID)
	m.ui.Cursor = cursorTracker{projectID: m.project.ID, cursor: cursor}
}

// trackCursor hands the focused row to the background saver whenever it moves.
// Recording continuously is what makes the cursor survive a force quit: a
// SIGKILL, a crash, or a terminal that goes away never reaches the save on exit,
// and until this existed such a session simply forgot where it was.
func (m *model) trackCursor() {
	if m.deps.CursorSaver == nil || m.project.ID == "" {
		return
	}
	cursor, _ := m.currentCursor()
	if m.ui.Cursor.projectID == m.project.ID && m.ui.Cursor.cursor == cursor {
		return
	}
	m.ui.Cursor = cursorTracker{projectID: m.project.ID, cursor: cursor}
	m.deps.CursorSaver.Set(m.project.ID, cursor)
}

// forgetPendingCursor discards a queued cursor for a project whose stored entry
// has just been settled some other way, so the background writer cannot land on
// top of it.
func (m *model) forgetPendingCursor(projectID string) {
	if m.deps.CursorSaver != nil {
		m.deps.CursorSaver.Drop(projectID)
	}
}

// restoreCursor moves the selection onto a previously saved row, reporting
// whether it landed.
func (m *model) restoreCursor(cursor data.Cursor) bool {
	if cursor.IsZero() {
		return false
	}
	kind, ok := cursorKindFromName(cursor.Kind)
	if !ok {
		return false
	}
	return m.restoreSelection(kind, cursor.CategoryID, cursor.TaskID)
}

// applyStoredCursor puts the cursor where this project was last left, falling
// back to the first task when nothing is stored or the remembered row is
// unreachable — deleted elsewhere, or hidden behind a fold or filter. Call after
// rebuildPositions, since it selects over the current position list.
func (m *model) applyStoredCursor() {
	if m.deps.StateManager != nil && m.restoreCursor(m.deps.StateManager.GetCursor(m.project.ID)) {
		return
	}
	m.ui.Selection.SetSelected(findFirstTaskIndex(m.ui.Selection.Positions()))
}

type itemRef struct {
	kind selection.FocusKind
	id   string
}

// selectItem falls back to the nearest shown ancestor when a fold or the filter hides the item.
func (m *model) selectItem(kind selection.FocusKind, id string) bool {
	for _, ref := range m.itemAnchors(kind, id) {
		if m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			return p.Kind == ref.kind && p.ID == ref.id
		}) {
			return true
		}
	}
	return false
}

func (m *model) itemAnchors(kind selection.FocusKind, id string) []itemRef {
	refs := []itemRef{{kind, id}}
	switch kind {
	case selection.FocusProject:
		return refs
	case selection.FocusCategory:
		if ci, err := m.project.FindCategoryByID(id); err == nil {
			if parent := domain.ParentIndex(m.project.Categories, ci); parent >= 0 {
				refs = append(refs, itemRef{selection.FocusCategory, m.project.Categories[parent].ID})
			}
		}
		return refs
	case selection.FocusDescription:
		refs = append(refs, itemRef{selection.FocusTask, id})
	}
	ci, ti := m.findTaskByID(id)
	if ci < 0 {
		return refs
	}
	cat := m.project.Categories[ci]
	if parent := domain.ParentIndex(cat.Tasks, ti); parent >= 0 {
		refs = append(refs, itemRef{selection.FocusTask, cat.Tasks[parent].ID})
	}
	return append(refs, m.itemAnchors(selection.FocusCategory, cat.ID)...)
}
