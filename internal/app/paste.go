package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

// pasteFromClipboard pastes whatever the system clipboard holds, at full
// fidelity when that is the app's own copy.
func (m *model) pasteFromClipboard() tea.Cmd {
	if !m.ui.Clipboard.empty() && m.clipboardVouchedFor() {
		m.pasteInternal()
		return nil
	}
	return readClipboardLines()
}

func (m *model) resolvePaste(msg clipboardLinesMsg) {
	if !m.ui.Modes.IsNormal() {
		return
	}
	if !m.ui.Clipboard.empty() {
		// An unreadable clipboard (OSC 52 only) can't contradict the copy.
		if m.clipboardVouchedFor() || msg.err != nil || strings.TrimSpace(msg.text) == "" ||
			sameClipboardText(msg.text, m.ui.Clipboard.Text) {
			m.pasteInternal()
			return
		}
		m.ui.Clipboard = ClipboardState{}
	}
	m.pasteClipboardLines(msg)
}

// clipboardVouchedFor reports whether the copy wins without reading the
// clipboard: nothing of it reached the clipboard, or a write is still on its way.
func (m *model) clipboardVouchedFor() bool {
	return !m.ui.Clipboard.Written || m.ui.ClipboardWrites > 0
}

func sameClipboardText(a, b string) bool {
	normalize := func(s string) string {
		return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	}
	return normalize(a) == normalize(b)
}

func (m *model) pasteInternal() bool {
	switch {
	case len(m.ui.Clipboard.Tasks) > 0:
		m.pasteTasks()
	case len(m.ui.Clipboard.Categories) > 0:
		m.pasteCategories()
	default:
		return false
	}
	return true
}

// pasteDestination is where task-level rows land: below the cursor's row, or at
// the top of the focused category.
func (m *model) pasteDestination() (catIndex, index int, ok bool) {
	pos, found := m.selectedPosition()
	if !found || len(m.project.Categories) == 0 {
		return 0, 0, false
	}
	switch pos.Kind {
	case selection.FocusProject:
		return 0, 0, true
	case selection.FocusTask, selection.FocusDescription, selection.FocusSeparator:
		return pos.CategoryIndex, pos.TaskIndex + 1, true
	}
	return pos.CategoryIndex, 0, true
}

func (m *model) pasteTasks() {
	catIndex, index, ok := m.pasteDestination()
	if !ok {
		m.ui.Screen.StatusMsg = "No category to paste into"
		return
	}
	clip := m.ui.Clipboard
	var rows []domain.Task
	if clip.IsCut {
		m.recordHistory()
		rows = m.moveRows(clip.TaskIDs, catIndex, index)
		if len(rows) == 0 {
			m.discardLastHistory()
			m.ui.Clipboard = ClipboardState{}
			m.ui.Screen.StatusMsg = "Nothing to move: the cut rows are gone"
			return
		}
	} else {
		// Every ID is minted before the first insert, so a failure pastes nothing.
		rows = make([]domain.Task, len(clip.Tasks))
		for i, src := range clip.Tasks {
			id, err := domain.NewID()
			if err != nil {
				m.ui.Screen.StatusMsg = "Failed to create task ID"
				return
			}
			rows[i] = src
			rows[i].ID = id
			rows[i].UpdatedAt = domain.NowTimestamp()
		}
		m.recordHistory()
		cat := &m.project.Categories[catIndex]
		for i, t := range rows {
			cat.InsertTask(index+i, t)
		}
	}

	if clip.IsCut {
		// The move used the cut up; what x copied stays, as a copy of the moved rows.
		ids := make([]string, len(rows))
		for i, t := range rows {
			ids[i] = t.ID
		}
		m.ui.Clipboard = ClipboardState{Tasks: rows, TaskIDs: ids, Text: clip.Text, Written: clip.Written}
	}
	m.rebuildPositions()
	m.selectTaskLevelRow(rows[0].ID, false)
	m.ensureVisible()
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = pastedRowsStatus(rows, clip.IsCut)
}

// moveRows lifts the live rows these IDs name and reinserts them, in order, at
// index in the category at catIndex, index counted before the lift. A row that
// no longer exists is skipped. Every cut's paste, one row or a range, moves
// through here, so a moved row keeps its ID and every edit made since the cut.
func (m *model) moveRows(ids []string, catIndex, index int) []domain.Task {
	lifted := make(map[string]bool, len(ids))
	for _, id := range ids {
		lifted[id] = true
	}
	dst := &m.project.Categories[catIndex]
	for _, t := range dst.Tasks[:min(index, len(dst.Tasks))] {
		if lifted[t.ID] {
			index--
		}
	}
	var moved []domain.Task
	for _, id := range ids {
		ci, ti := m.findTaskByID(id)
		if ci < 0 {
			continue
		}
		moved = append(moved, m.project.Categories[ci].Tasks[ti])
		_ = m.project.Categories[ci].RemoveTask(ti)
	}
	index = max(0, min(index, len(dst.Tasks)))
	for i, t := range moved {
		dst.InsertTask(index+i, t)
	}
	m.project.UpdatedAt = domain.NowTimestamp()
	return moved
}

func pastedRowsStatus(rows []domain.Task, moved bool) string {
	if len(rows) == 1 {
		if moved {
			return "Moved!"
		}
		return "Pasted!"
	}
	tasks := 0
	for _, t := range rows {
		if !t.IsSeparator() {
			tasks++
		}
	}
	verb := "Pasted "
	if moved {
		verb = "Moved "
	}
	return verb + rowCountLabel(tasks, len(rows)-tasks)
}

// pasteCategories lands the categories in the focused one's place, pushing it
// down.
func (m *model) pasteCategories() {
	pos, ok := m.selectedPosition()
	if !ok {
		return
	}
	index := 0
	if pos.Kind != selection.FocusProject {
		index = pos.CategoryIndex
	}
	clip := m.ui.Clipboard
	var cats []domain.Category
	if clip.IsCut {
		m.recordHistory()
		cats = m.moveCategories(clip.CategoryIDs, index)
		if len(cats) == 0 {
			m.discardLastHistory()
			m.ui.Clipboard = ClipboardState{}
			m.ui.Screen.StatusMsg = "Nothing to move: the cut categories are gone"
			return
		}
	} else {
		cats = make([]domain.Category, len(clip.Categories))
		for i, src := range clip.Categories {
			cat, err := copyCategory(src)
			if err != nil {
				m.ui.Screen.StatusMsg = "Failed to create category ID"
				return
			}
			cats[i] = cat
		}
		m.recordHistory()
		for i, c := range cats {
			m.project.InsertCategory(index+i, c)
		}
	}

	if clip.IsCut {
		ids := make([]string, len(cats))
		for i, c := range cats {
			ids[i] = c.ID
		}
		m.ui.Clipboard = cloneClipboard(ClipboardState{Categories: cats, CategoryIDs: ids, Text: clip.Text, Written: clip.Written})
	}
	m.rebuildPositions()
	firstID := cats[0].ID
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusCategory && m.project.Categories[p.CategoryIndex].ID == firstID
	})
	m.ensureVisible()
	m.storeTaskUpdate()
	verb := "Pasted"
	if clip.IsCut {
		verb = "Moved"
	}
	m.ui.Screen.StatusMsg = fmt.Sprintf("%s %d categor%s", verb, len(cats), plural(len(cats), "y", "ies"))
}

// moveCategories is moveRows for whole categories.
func (m *model) moveCategories(ids []string, index int) []domain.Category {
	lifted := make(map[string]bool, len(ids))
	for _, id := range ids {
		lifted[id] = true
	}
	for _, c := range m.project.Categories[:min(index, len(m.project.Categories))] {
		if lifted[c.ID] {
			index--
		}
	}
	live := make(map[string]domain.Category, len(ids))
	kept := m.project.Categories[:0]
	for _, c := range m.project.Categories {
		if lifted[c.ID] {
			live[c.ID] = c
			continue
		}
		kept = append(kept, c)
	}
	m.project.Categories = kept
	var moved []domain.Category
	for _, id := range ids {
		if c, ok := live[id]; ok {
			moved = append(moved, c)
		}
	}
	index = max(0, min(index, len(m.project.Categories)))
	for i, c := range moved {
		m.project.InsertCategory(index+i, c)
	}
	return moved
}

// copyCategory gives the category and each of its tasks a fresh ID, so the
// project never holds two of one.
func copyCategory(src domain.Category) (domain.Category, error) {
	id, err := domain.NewID()
	if err != nil {
		return domain.Category{}, err
	}
	out := src
	out.ID = id
	out.UpdatedAt = domain.NowTimestamp()
	out.Tasks = make([]domain.Task, len(src.Tasks))
	for i, t := range src.Tasks {
		tid, err := domain.NewID()
		if err != nil {
			return domain.Category{}, err
		}
		t.ID = tid
		out.Tasks[i] = t
	}
	return out, nil
}
