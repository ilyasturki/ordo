package app

import (
	"errors"
	"strings"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func (m *model) storeTaskUpdate() {
	// Content changed: the memoized layout no longer matches the project.
	m.invalidateLayout()
	if m.deps.Saver != nil {
		// Snapshot + hand off to the background writer; the fsync happens off the
		// event loop so the keystroke returns immediately. Errors surface later
		// via the Results subscription (see listenSaveErrors).
		m.deps.Saver.Enqueue(m.project)
		m.checkSaves()
		return
	}
	// No async saver (tests, or a store wired without one): save synchronously
	// so persistence still happens.
	if m.deps.Store == nil {
		return
	}
	if err := m.deps.Store.SaveProjectLocked(m.project); err != nil {
		m.ui.Screen.StatusMsg = saveFailedMessage(err)
	}
	m.checkSaves()
}

func saveFailedMessage(err error) string {
	if errors.Is(err, data.ErrStaleProject) {
		return "Changed on disk: merging your edits"
	}
	return "Save failed: " + err.Error()
}

// invalidateLayout drops the memoized layout so the next render/scroll rebuilds
// it. Call after any change that alters what rows exist or how tall they are.
func (m *model) invalidateLayout() {
	m.ui.layout.layout = nil
}

func (m *model) reloadProject() {
	if m.deps.Store == nil {
		return
	}
	m.ui.Filter.releaseHeld()
	m.ui.Filter.endPeek()
	if status := m.takeOutside(); status != "" {
		m.ui.Screen.StatusMsg = strings.Replace(status, "Changed on disk: reloaded", "Reloaded from disk", 1)
	}
}

// showProject opens a copy from outside, with the cursor on its row while that row exists.
func (m *model) showProject(project domain.Project) {
	prev, hadSel := m.currentCursor()
	m.project = project
	m.checkSaves()
	// A view that landed before this copy lost category IDs in memory: the file still has them.
	m.readView()
	m.rebuildPositions()

	if hadSel && !m.restoreCursor(prev) {
		idx := findFirstTaskIndex(m.ui.Selection.Positions())
		m.ui.Selection.SetSelected(idx)
	}

	m.ensureVisible()
}

// restoreSelection moves the cursor onto the row identified by (kind, catID,
// taskID), reporting whether it landed. A row addressed by task ID that no
// longer exists degrades to its category header rather than failing outright, so
// deleting a task leaves the cursor nearby instead of at the top of the list.
func (m *model) restoreSelection(kind selection.FocusKind, catID, taskID string) bool {
	switch kind {
	case selection.FocusProject:
		return m.selectItem(selection.FocusProject, "")
	case selection.FocusCategory, selection.FocusHidden:
		return catID != "" && m.selectItem(kind, catID)
	}
	return (taskID != "" && m.selectItem(kind, taskID)) ||
		(catID != "" && m.selectItem(selection.FocusCategory, catID))
}

// taskRowVisible is the filter's verdict alone; shownRows adds sections, nesting, peeks and folds on top.
func taskRowVisible(filter *FilterState, task domain.Task, categoryID, parentCategoryID string) bool {
	if filter == nil || filter.holds(task.ID) {
		return true
	}
	if !task.IsTask() {
		return filter.LineMatches(task, categoryID, parentCategoryID)
	}
	return filter.TaskVisible(task, categoryID, parentCategoryID)
}

// showSections shows a line whose section, from a separator to the next or the
// category's end, has a task the filter shows.
func showSections(cat domain.Category, shown []bool) {
	for start := 0; start < len(cat.Tasks); {
		end := start + 1
		for end < len(cat.Tasks) && !cat.Tasks[end].IsSeparator() {
			end++
		}
		visible := false
		for i := start; i < end; i++ {
			visible = visible || (cat.Tasks[i].IsTask() && shown[i])
		}
		for i := start; i < end; i++ {
			shown[i] = shown[i] || (visible && !cat.Tasks[i].IsTask())
		}
		start = end
	}
}

// filterRows is what the filter shows of the category, held rows included, before any peek or fold.
func filterRows(cat domain.Category, filter *FilterState) []bool {
	shown := make([]bool, len(cat.Tasks))
	for i, task := range cat.Tasks {
		shown[i] = taskRowVisible(filter, task, cat.ID, cat.ParentID)
	}
	showSections(cat, shown)
	for i := 0; i < len(cat.Tasks); {
		end := domain.BlockEnd(cat.Tasks, i)
		for j := i + 1; j < end; j++ {
			shown[i] = shown[i] || shown[j]
		}
		i = end
	}
	return shown
}

// hiddenSections maps each separator the filter hides to the end of the rows its line stands for.
func hiddenSections(cat domain.Category, filter *FilterState, shown []bool) map[int]int {
	if filter == nil || !filter.HasActiveFilter() || !filter.CategoryVisible(cat.ID, cat.ParentID) {
		return nil
	}
	var sections map[int]int
	for i, task := range cat.Tasks {
		if task.IsSeparator() && !shown[i] {
			if sections == nil {
				sections = make(map[int]int)
			}
			sections[i] = operations.HiddenSectionEnd(cat.Tasks, shown, i)
		}
	}
	return sections
}

// sectionRows marks the separators of sections and the rows their lines stand for.
func sectionRows(sections map[int]int, n int) []bool {
	in := make([]bool, n)
	for sep, end := range sections {
		for i := sep; i < end; i++ {
			in[i] = true
		}
	}
	return in
}

func shownRows(cat domain.Category, filter *FilterState, fold *FoldState) []bool {
	shown := filterRows(cat, filter)
	if filter != nil && filter.peeks(cat.ID) {
		peekRows(cat, filter, shown)
	}
	if fold != nil {
		for i := 0; i < len(cat.Tasks); {
			end := domain.BlockEnd(cat.Tasks, i)
			if fold.IsFolded(cat.Tasks[i].ID) {
				clear(shown[i+1 : end])
			}
			i = end
		}
	}
	return shown
}

// peekRows shows what the peek reveals: one section's rows, or the category's others.
func peekRows(cat domain.Category, filter *FilterState, shown []bool) {
	sections := hiddenSections(cat, filter, shown)
	for sep, end := range sections {
		if filter.peeksSection(cat.Tasks[sep].ID) {
			for i := sep + 1; i < end; i++ {
				shown[i] = true
			}
			return
		}
	}
	if !filter.peeksRows(cat.ID) {
		return
	}
	in := sectionRows(sections, len(shown))
	for i := range shown {
		shown[i] = shown[i] || !in[i]
	}
}

// hiddenByFilter counts the tasks the filter hides outside a section's line, and the same during a peek.
func hiddenByFilter(cat domain.Category, filter *FilterState) int {
	if filter == nil || !filter.HasActiveFilter() {
		return 0
	}
	shown := filterRows(cat, filter)
	return hiddenOutside(cat, shown, hiddenSections(cat, filter, shown))
}

func hiddenOutside(cat domain.Category, shown []bool, sections map[int]int) int {
	in := sectionRows(sections, len(shown))
	n := 0
	for i, task := range cat.Tasks {
		if !shown[i] && task.IsTask() && !in[i] {
			n++
		}
	}
	return n
}

// sectionHidden counts the rows the line of the separator at sep stands for.
func sectionHidden(cat domain.Category, filter *FilterState, sep int) int {
	shown := filterRows(cat, filter)
	return operations.HiddenSectionEnd(cat.Tasks, shown, sep) - sep - 1
}

// expandedTaskID's description row shows even while descriptions are hidden.
func rebuildPositions(categories []domain.Category, filter *FilterState, fold *FoldState, expandDescriptions bool, expandedTaskID string) []selection.Position {
	positions := make([]selection.Position, 0)
	positions = append(positions, selection.Position{
		Kind:          selection.FocusProject,
		CategoryIndex: -1,
		TaskIndex:     -1,
	})
	foldedParent := ""
	for cIndex, category := range categories {
		if category.ParentID == "" {
			foldedParent = ""
		} else if category.ParentID == foldedParent {
			continue
		}
		positions = append(positions, selection.Position{
			Kind:          selection.FocusCategory,
			CategoryIndex: cIndex,
			TaskIndex:     -1,
			ID:            category.ID,
		})
		if fold != nil && fold.IsFolded(category.ID) {
			if category.ParentID == "" {
				foldedParent = category.ID
			}
			continue
		}
		shown := shownRows(category, filter, fold)
		var sections map[int]int
		hidden := 0
		if filter != nil && filter.HasActiveFilter() {
			base := filterRows(category, filter)
			sections = hiddenSections(category, filter, base)
			hidden = hiddenOutside(category, base, sections)
		}
		for tIndex, task := range category.Tasks {
			if _, ok := sections[tIndex]; ok {
				positions = append(positions, selection.Position{
					Kind:          selection.FocusSection,
					CategoryIndex: cIndex,
					TaskIndex:     tIndex,
					ID:            task.ID,
				})
				continue
			}
			if !shown[tIndex] {
				continue
			}
			if !task.IsTask() {
				positions = append(positions, selection.Position{
					Kind:          selection.FocusSeparator,
					CategoryIndex: cIndex,
					TaskIndex:     tIndex,
					ID:            task.ID,
				})
				continue
			}
			positions = append(positions, selection.Position{
				Kind:          selection.FocusTask,
				CategoryIndex: cIndex,
				TaskIndex:     tIndex,
				ID:            task.ID,
			})
			if (expandDescriptions || task.ID == expandedTaskID) && task.Description != "" {
				positions = append(positions, selection.Position{
					Kind:          selection.FocusDescription,
					CategoryIndex: cIndex,
					TaskIndex:     tIndex,
					ID:            task.ID,
				})
			}
		}
		if hidden > 0 {
			positions = append(positions, selection.Position{
				Kind:          selection.FocusHidden,
				CategoryIndex: cIndex,
				TaskIndex:     -1,
				ID:            category.ID,
			})
		}
	}
	return positions
}

func (m *model) rebuildPositions() {
	m.ui.Filter.pruneCategories(m.project.Categories)
	prev, had := m.selectedPosition()
	positions := rebuildPositions(m.project.Categories, &m.ui.Filter, &m.ui.Fold, m.ui.Screen.ExpandDescriptions, m.ui.Screen.ExpandedTaskID)
	m.ui.Selection.SetPositions(positions)
	if had {
		m.selectItem(prev.Kind, prev.ID)
	}
	// Structure changed (filter, fold, add/delete, reload, undo/redo): the
	// memoized layout is stale.
	m.invalidateLayout()
}
