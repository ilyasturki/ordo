package app

import (
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

const historyLimit = 100

type historyEntry struct {
	project   domain.Project
	clipboard ClipboardState
	cursor    data.Cursor
}

type HistoryState struct {
	undo []historyEntry
	redo []historyEntry
	// pendingRedo stashes the redo stack as it stood just before the last
	// recordHistory(). If discardLastHistory() runs (the recorded mutation was
	// cancelled or a no-op), the redo stack is restored from here so the
	// cancelled action does not silently destroy redoable history.
	pendingRedo []historyEntry
}

func NewHistoryState() HistoryState {
	return HistoryState{}
}

func (h *HistoryState) CanUndo() bool { return len(h.undo) > 0 }
func (h *HistoryState) CanRedo() bool { return len(h.redo) > 0 }

func (h *HistoryState) UndoDepth() int { return len(h.undo) }
func (h *HistoryState) RedoDepth() int { return len(h.redo) }

func (h *HistoryState) Reset() {
	h.undo = nil
	h.redo = nil
	h.pendingRedo = nil
}

func cloneProject(p domain.Project) domain.Project {
	out := p
	out.Categories = make([]domain.Category, len(p.Categories))
	for i, c := range p.Categories {
		nc := c
		nc.Tasks = make([]domain.Task, len(c.Tasks))
		copy(nc.Tasks, c.Tasks)
		out.Categories[i] = nc
	}
	return out
}

func cloneClipboard(c ClipboardState) ClipboardState {
	out := c
	if len(c.Tasks) > 0 {
		out.Tasks = make([]domain.Task, len(c.Tasks))
		copy(out.Tasks, c.Tasks)
	}
	if len(c.TaskIDs) > 0 {
		out.TaskIDs = make([]string, len(c.TaskIDs))
		copy(out.TaskIDs, c.TaskIDs)
	}
	if len(c.Categories) > 0 {
		out.Categories = make([]domain.Category, len(c.Categories))
		for i, cat := range c.Categories {
			nc := cat
			if len(cat.Tasks) > 0 {
				nc.Tasks = make([]domain.Task, len(cat.Tasks))
				copy(nc.Tasks, cat.Tasks)
			}
			out.Categories[i] = nc
		}
	}
	if len(c.CategoryIDs) > 0 {
		out.CategoryIDs = make([]string, len(c.CategoryIDs))
		copy(out.CategoryIDs, c.CategoryIDs)
	}
	return out
}

func (m *model) snapshotEntry() historyEntry {
	cursor, _ := m.currentCursor()
	return historyEntry{
		project:   cloneProject(m.project),
		clipboard: cloneClipboard(m.ui.Clipboard),
		cursor:    cursor,
	}
}

// recordHistory captures the current state on the undo stack and clears redo.
// Call BEFORE any project mutation. The cleared redo stack is stashed in
// pendingRedo so that discardLastHistory() can restore it if the mutation
// turns out to be cancelled.
func (m *model) recordHistory() {
	entry := m.snapshotEntry()
	if len(m.ui.History.undo) >= historyLimit {
		drop := len(m.ui.History.undo) - historyLimit + 1
		// Copy into a fresh slice so the dropped entries (each holding a
		// cloned project) are not pinned in the original backing array.
		trimmed := make([]historyEntry, historyLimit-1, historyLimit)
		copy(trimmed, m.ui.History.undo[drop:])
		m.ui.History.undo = trimmed
	}
	m.ui.History.undo = append(m.ui.History.undo, entry)
	m.ui.History.pendingRedo = m.ui.History.redo
	m.ui.History.redo = nil
}

// discardLastHistory removes the most-recently recorded undo entry AND
// restores the redo stack that recordHistory wiped. Used when a recorded
// mutation was cancelled or turned out to be a no-op — without the redo
// restore the cancel would silently destroy redoable history.
func (m *model) discardLastHistory() {
	if n := len(m.ui.History.undo); n > 0 {
		m.ui.History.undo = m.ui.History.undo[:n-1]
	}
	m.ui.History.redo = m.ui.History.pendingRedo
	m.ui.History.pendingRedo = nil
}

func (m *model) undo() {
	if !m.ui.History.CanUndo() {
		m.ui.Screen.StatusMsg = "Nothing to undo"
		return
	}
	current := m.snapshotEntry()
	n := len(m.ui.History.undo) - 1
	entry := m.ui.History.undo[n]
	m.ui.History.undo = m.ui.History.undo[:n]
	m.ui.History.redo = append(m.ui.History.redo, current)
	m.ui.History.pendingRedo = nil
	m.applyHistoryEntry(entry)
	// storeTaskUpdate may have set "Save failed: …"; keep that user-visible
	// instead of overwriting it with the optimistic "Undid change" message.
	if m.ui.Screen.StatusMsg == "" {
		m.ui.Screen.StatusMsg = "Undid change"
	}
}

func (m *model) redo() {
	if !m.ui.History.CanRedo() {
		m.ui.Screen.StatusMsg = "Nothing to redo"
		return
	}
	current := m.snapshotEntry()
	n := len(m.ui.History.redo) - 1
	entry := m.ui.History.redo[n]
	m.ui.History.redo = m.ui.History.redo[:n]
	m.ui.History.undo = append(m.ui.History.undo, current)
	m.ui.History.pendingRedo = nil
	m.applyHistoryEntry(entry)
	if m.ui.Screen.StatusMsg == "" {
		m.ui.Screen.StatusMsg = "Redid change"
	}
}

func (m *model) applyHistoryEntry(e historyEntry) {
	m.project = cloneProject(e.project)
	m.ui.Clipboard = cloneClipboard(e.clipboard)
	m.ui.Filter.releaseHeld()
	m.rebuildPositions()
	if !e.cursor.IsZero() && !m.restoreCursor(e.cursor) {
		m.ui.Selection.SetSelected(findFirstTaskIndex(m.ui.Selection.Positions()))
	}
	m.ensureVisible()
	m.storeTaskUpdate()
}
