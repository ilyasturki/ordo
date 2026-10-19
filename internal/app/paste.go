package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/clipboard"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func (m *model) pasteFromClipboard() tea.Cmd {
	if !m.ui.Clipboard.empty() && m.clipboardVouchedFor() {
		m.pasteInternal()
		return nil
	}
	// Only another app can have replaced our last write, and reaching one costs the terminal its focus.
	if w := m.ui.LastWrite; w.fresh && m.ui.FocusReported {
		return m.resolvePaste(clipboardLinesMsg{text: w.text})
	}
	return readClipboardLines()
}

func (m *model) resolvePaste(msg clipboardLinesMsg) tea.Cmd {
	if !m.ui.Modes.IsNormal() {
		return nil
	}
	// Bytes other than text are another app's copy, never ordo's own.
	notText := msg.image != nil || errors.Is(msg.err, clipboard.ErrNotText) || errors.Is(msg.err, clipboard.ErrTooLarge)
	retiresCopy := false
	if !m.ui.Clipboard.empty() {
		// An unreadable clipboard can't contradict the copy.
		if m.clipboardVouchedFor() || !notText && (msg.err != nil || strings.TrimSpace(msg.text) == "" ||
			sameClipboardText(msg.text, m.ui.Clipboard.Text)) {
			m.pasteInternal()
			return nil
		}
		retiresCopy = true
	}
	switch {
	case msg.image != nil:
		return m.attachImages([]pendingImage{clipboardImage(msg.image)}, retiresCopy)
	case msg.err != nil:
		m.ui.Screen.StatusMsg = clipboardReadFailure(msg.err)
		return nil
	case allImageFiles(msg.paths):
		return m.attachImages(imageFiles(msg.paths), retiresCopy)
	}
	return m.pasteTextOrImages(msg.text, retiresCopy)
}

func (m *model) clipboardVouchedFor() bool {
	return !m.ui.Clipboard.Written || m.ui.ClipboardWrites > 0
}

func sameClipboardText(a, b string) bool {
	normalize := func(s string) string {
		return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	}
	return normalize(a) == normalize(b)
}

func (m *model) pasteInternal() {
	switch {
	case len(m.ui.Clipboard.Tasks) > 0:
		m.pasteTasks()
	case len(m.ui.Clipboard.Categories) > 0:
		m.pasteCategories()
	}
}

func (m *model) pasteDestination() (catIndex, index int, parentID string, ok bool) {
	pos, found := m.selectedPosition()
	if !found || len(m.project.Categories) == 0 {
		return 0, 0, "", false
	}
	switch pos.Kind {
	case selection.FocusProject:
		return 0, 0, "", true
	case selection.FocusTask, selection.FocusDescription, selection.FocusSeparator:
		tasks := m.project.Categories[pos.CategoryIndex].Tasks
		return pos.CategoryIndex, domain.BlockEnd(tasks, pos.TaskIndex), tasks[pos.TaskIndex].ParentID, true
	}
	return pos.CategoryIndex, 0, "", true
}

func (m *model) pasteTasks() {
	catIndex, index, parentID, ok := m.pasteDestination()
	if !ok {
		m.ui.Screen.StatusMsg = "No category to paste into"
		return
	}
	clip := m.ui.Clipboard
	var rows []domain.Task
	if clip.IsCut {
		m.recordHistory()
		rows = m.moveRows(clip.TaskIDs, catIndex, index, parentID)
		if len(rows) == 0 {
			m.discardLastHistory()
			m.ui.Clipboard = ClipboardState{}
			m.ui.Screen.StatusMsg = "Nothing to move: the cut rows are gone"
			return
		}
	} else {
		// Every ID is minted before the first insert, so a failure pastes nothing.
		rows = make([]domain.Task, len(clip.Tasks))
		fresh := make(map[string]string, len(clip.Tasks))
		for i, src := range clip.Tasks {
			id, err := domain.NewID()
			if err != nil {
				m.ui.Screen.StatusMsg = "Failed to create task ID"
				return
			}
			fresh[src.ID] = id
			rows[i] = src
			rows[i].ID = id
			rows[i].UpdatedAt = domain.NowTimestamp()
		}
		for i := range rows {
			if id, ok := fresh[rows[i].ParentID]; ok {
				rows[i].ParentID = id
			}
		}
		m.recordHistory()
		cat := &m.project.Categories[catIndex]
		index = operations.SettleTasks(cat.Tasks, rows, index, parentID)
		cat.Tasks = slices.Insert(cat.Tasks, index, rows...)
		cat.UpdatedAt = domain.NowTimestamp()
	}

	if clip.IsCut {
		// The move used the cut up; what x copied stays, as a copy of the moved rows.
		ids := make([]string, len(rows))
		for i, t := range rows {
			ids[i] = t.ID
		}
		m.ui.Clipboard = ClipboardState{Tasks: rows, TaskIDs: ids, Text: clip.Text, Written: clip.Written, Moved: true}
	}
	m.rebuildPositions()
	m.selectTaskLevelRow(rows[0].ID)
	m.ensureVisible()
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = pastedRowsStatus(rows, clip.IsCut)
}

func (m *model) moveRows(ids []string, catIndex, index int, parentID string) []domain.Task {
	rows, err := operations.MoveRows(&m.project, ids, m.project.Categories[catIndex].ID, index, parentID)
	if err != nil {
		return nil
	}
	return rows
}

func pastedRowsStatus(rows []domain.Task, moved bool) string {
	if len(rows) == 1 {
		if moved {
			return "Moved!"
		}
		return "Pasted!"
	}
	tasks := countTasks(rows)
	verb := "Pasted "
	if moved {
		verb = "Moved "
	}
	return verb + rowCountLabel(tasks, len(rows)-tasks)
}

func (m *model) categoryDestination() (index int, parentID string, ok bool) {
	pos, found := m.selectedPosition()
	if !found {
		return 0, "", false
	}
	if pos.Kind == selection.FocusProject || pos.CategoryIndex < 0 {
		return 0, "", true
	}
	return pos.CategoryIndex, m.project.Categories[pos.CategoryIndex].ParentID, true
}

func settleCategories(cats, rows []domain.Category, index int, parentID string) int {
	if !domain.SettleParents(rows, parentID) && parentID != "" {
		for i, c := range cats {
			if c.ID == parentID {
				index = i
				break
			}
		}
	}
	moving := make(map[string]bool, len(rows))
	for _, r := range rows {
		moving[r.ID] = true
	}
	var taken []domain.Category
	for _, c := range cats {
		if !moving[c.ID] {
			taken = append(taken, c)
		}
	}
	for i := range rows {
		rows[i].Name = domain.UniqueCategoryName(taken, rows[i].ParentID, rows[i].Name)
		taken = append(taken, rows[i])
	}
	return index
}

func (m *model) pasteCategories() {
	index, parentID, ok := m.categoryDestination()
	if !ok {
		return
	}
	clip := m.ui.Clipboard
	var cats []domain.Category
	if clip.IsCut {
		m.recordHistory()
		cats = m.moveCategories(clip.CategoryIDs, index, parentID)
		if len(cats) == 0 {
			m.discardLastHistory()
			m.ui.Clipboard = ClipboardState{}
			m.ui.Screen.StatusMsg = "Nothing to move: the cut categories are gone"
			return
		}
	} else {
		cats = make([]domain.Category, len(clip.Categories))
		fresh := make(map[string]string, len(clip.Categories))
		for i, src := range clip.Categories {
			cat, err := copyCategory(src)
			if err != nil {
				m.ui.Screen.StatusMsg = "Failed to create category ID"
				return
			}
			fresh[src.ID] = cat.ID
			cats[i] = cat
		}
		for i := range cats {
			if id, ok := fresh[cats[i].ParentID]; ok {
				cats[i].ParentID = id
			}
		}
		m.recordHistory()
		index = settleCategories(m.project.Categories, cats, index, parentID)
		m.project.Categories = slices.Insert(m.project.Categories, index, cats...)
		m.project.UpdatedAt = domain.NowTimestamp()
	}

	if clip.IsCut {
		ids := make([]string, len(cats))
		for i, c := range cats {
			ids[i] = c.ID
		}
		m.ui.Clipboard = cloneClipboard(ClipboardState{Categories: cats, CategoryIDs: ids, Text: clip.Text, Written: clip.Written, Moved: true})
	}
	m.rebuildPositions()
	m.selectItem(selection.FocusCategory, cats[0].ID)
	m.ensureVisible()
	m.storeTaskUpdate()
	verb := "Pasted"
	if clip.IsCut {
		verb = "Moved"
	}
	m.ui.Screen.StatusMsg = fmt.Sprintf("%s %d categor%s", verb, len(cats), plural(len(cats), "y", "ies"))
}

// moveCategories counts index before the lift and skips an ID no longer in the project.
func (m *model) moveCategories(ids []string, index int, parentID string) []domain.Category {
	cats := m.project.Categories
	picked := make(map[string]bool, len(ids))
	for _, id := range ids {
		picked[id] = true
	}
	moved := domain.PickBlocks(cats, picked)
	if len(moved) == 0 {
		return nil
	}
	lifted := make(map[string]bool, len(moved))
	for _, c := range moved {
		lifted[c.ID] = true
	}
	if lifted[parentID] {
		parentID = ""
	}
	index = settleCategories(cats, moved, index, parentID)
	for _, c := range cats[:min(index, len(cats))] {
		if lifted[c.ID] {
			index--
		}
	}
	var kept []domain.Category
	for _, c := range cats {
		if !lifted[c.ID] {
			kept = append(kept, c)
		}
	}
	index = max(0, min(index, len(kept)))
	m.project.Categories = slices.Insert(kept, index, moved...)
	m.project.UpdatedAt = domain.NowTimestamp()
	return moved
}

// Fresh IDs throughout: the project must never hold two of one.
func copyCategory(src domain.Category) (domain.Category, error) {
	id, err := domain.NewID()
	if err != nil {
		return domain.Category{}, err
	}
	out := src
	out.ID = id
	out.UpdatedAt = domain.NowTimestamp()
	out.Tasks = make([]domain.Task, len(src.Tasks))
	fresh := make(map[string]string, len(src.Tasks))
	for i, t := range src.Tasks {
		tid, err := domain.NewID()
		if err != nil {
			return domain.Category{}, err
		}
		fresh[t.ID] = tid
		t.ID = tid
		out.Tasks[i] = t
	}
	for i := range out.Tasks {
		if parent := out.Tasks[i].ParentID; parent != "" {
			out.Tasks[i].ParentID = fresh[parent]
		}
	}
	return out, nil
}
