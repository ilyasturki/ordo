package app

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

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
	retiresCopy := false
	if !m.ui.Clipboard.empty() {
		// An unreadable clipboard (OSC 52 only) can't contradict the copy.
		if m.clipboardVouchedFor() || msg.err != nil || strings.TrimSpace(msg.text) == "" ||
			sameClipboardText(msg.text, m.ui.Clipboard.Text) {
			m.pasteInternal()
			return
		}
		retiresCopy = true
	}
	if msg.err != nil {
		m.ui.Screen.StatusMsg = clipboardReadFailure(msg.err)
		return
	}
	m.pasteText(msg.text, retiresCopy)
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

// pasteDestination is where pasted rows land: the category, the index among
// its tasks, and the parent task whose subtasks they join, "" at the top level.
// They go below the focused row at its depth, after a parent's subtasks.
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

// settleTasks sets the parents of rows pasted at index among the subtasks of
// parentID (see domain.SettleParents) and returns where they go: rows that
// can't all be subtasks there land after the parent's block instead.
func settleTasks(tasks, rows []domain.Task, index int, parentID string) int {
	if domain.SettleParents(rows, parentID) || parentID == "" {
		return index
	}
	for i, t := range tasks {
		if t.ID == parentID {
			return domain.BlockEnd(tasks, i)
		}
	}
	return index
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
		index = settleTasks(cat.Tasks, rows, index, parentID)
		cat.Tasks = slices.Insert(cat.Tasks, index, rows...)
		cat.UpdatedAt = domain.NowTimestamp()
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

// withSubtasks adds to ids the subtasks of every parent they name, as they
// stand now, right after their parent.
func (m *model) withSubtasks(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	var out []string
	for _, id := range ids {
		ci, ti := m.findTaskByID(id)
		if ci < 0 || seen[id] {
			continue
		}
		tasks := m.project.Categories[ci].Tasks
		for _, t := range tasks[ti:domain.BlockEnd(tasks, ti)] {
			if !seen[t.ID] {
				seen[t.ID] = true
				out = append(out, t.ID)
			}
		}
	}
	return out
}

// moveRows moves the live rows ids name, each parent with its subtasks, to
// index in the category at catIndex, among the subtasks of parentID when they
// can all be subtasks. index is counted before the lift; an ID no longer in the
// project is skipped.
func (m *model) moveRows(ids []string, catIndex, index int, parentID string) []domain.Task {
	ids = m.withSubtasks(ids)
	var moved []domain.Task
	for _, id := range ids {
		ci, ti := m.findTaskByID(id)
		moved = append(moved, m.project.Categories[ci].Tasks[ti])
	}
	if len(moved) == 0 {
		return nil
	}
	dst := &m.project.Categories[catIndex]
	index = settleTasks(dst.Tasks, moved, index, parentID)
	lifted := make(map[string]bool, len(ids))
	for _, id := range ids {
		lifted[id] = true
	}
	for _, t := range dst.Tasks[:min(index, len(dst.Tasks))] {
		if lifted[t.ID] {
			index--
		}
	}
	for i := range m.project.Categories {
		cat := &m.project.Categories[i]
		kept := cat.Tasks[:0]
		for _, t := range cat.Tasks {
			if !lifted[t.ID] {
				kept = append(kept, t)
			}
		}
		if len(kept) != len(cat.Tasks) {
			cat.Tasks = kept
			cat.UpdatedAt = domain.NowTimestamp()
		}
	}
	index = max(0, min(index, len(dst.Tasks)))
	dst.Tasks = slices.Insert(dst.Tasks, index, moved...)
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

// categoryDestination is where pasted categories land: before the focused
// category, at its depth, so among its parent's sub-categories when it is one.
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

// settleCategories sets the parents of categories pasted at index among the
// sub-categories of parentID (see domain.SettleParents) and returns where they
// go: categories that can't all be sub-categories there land before the parent
// instead. Each takes a free name among its new siblings.
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
		m.ui.Clipboard = cloneClipboard(ClipboardState{Categories: cats, CategoryIDs: ids, Text: clip.Text, Written: clip.Written})
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

// moveCategories moves the live categories ids name, each parent with its
// sub-categories, to index, among the sub-categories of parentID when they can
// all be sub-categories, else before that parent. index is counted before the
// lift; an ID no longer in the project is skipped.
func (m *model) moveCategories(ids []string, index int, parentID string) []domain.Category {
	cats := m.project.Categories
	lifted := make(map[string]bool, len(ids))
	var moved []domain.Category
	for _, id := range ids {
		ci, err := m.project.FindCategoryByID(id)
		if err != nil || lifted[id] {
			continue
		}
		for _, c := range cats[ci:domain.BlockEnd(cats, ci)] {
			if !lifted[c.ID] {
				lifted[c.ID] = true
				moved = append(moved, c)
			}
		}
	}
	if len(moved) == 0 {
		return nil
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
