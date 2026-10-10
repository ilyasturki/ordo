package app

import (
	"errors"

	"phasionary/internal/app/selection"
	"phasionary/internal/data"
	"phasionary/internal/domain"
)

func (m *model) storeTaskUpdate() {
	// Content changed: the memoized layout no longer matches the project.
	m.invalidateLayout()
	if m.deps.Saver != nil {
		// Snapshot + hand off to the background writer; the fsync happens off the
		// event loop so the keystroke returns immediately. Errors surface later
		// via the Results subscription (see listenSaveErrors).
		m.deps.Saver.Enqueue(m.project)
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
}

func saveFailedMessage(err error) string {
	if errors.Is(err, data.ErrStaleProject) {
		return "Project changed on disk; press R to reload (unsaved edits are dropped)"
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

	prev, hadSel := m.currentCursor()

	project, err := m.deps.Store.LoadProject(m.project.ID)
	if err != nil {
		m.ui.Screen.StatusMsg = "Reload failed: " + err.Error()
		return
	}

	m.project = project
	m.ui.History.Reset()
	m.rebuildPositions()

	if hadSel && !m.restoreCursor(prev) {
		idx := findFirstTaskIndex(m.ui.Selection.Positions())
		m.ui.Selection.SetSelected(idx)
	}

	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Reloaded from disk"
}

// restoreSelection moves the cursor onto the row identified by (kind, catID,
// taskID), reporting whether it landed. A row addressed by task ID that no
// longer exists degrades to its category header rather than failing outright, so
// deleting a task leaves the cursor nearby instead of at the top of the list.
func (m *model) restoreSelection(kind selection.FocusKind, catID, taskID string) bool {
	categories := m.project.Categories

	selectCategory := func() bool {
		if catID == "" {
			return false
		}
		return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			return p.Kind == selection.FocusCategory &&
				p.CategoryIndex >= 0 && p.CategoryIndex < len(categories) &&
				categories[p.CategoryIndex].ID == catID
		})
	}
	selectTaskRow := func(want selection.FocusKind) bool {
		if taskID == "" {
			return false
		}
		return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			if p.Kind != want || p.CategoryIndex < 0 || p.CategoryIndex >= len(categories) {
				return false
			}
			cat := categories[p.CategoryIndex]
			return p.TaskIndex >= 0 && p.TaskIndex < len(cat.Tasks) && cat.Tasks[p.TaskIndex].ID == taskID
		})
	}

	switch kind {
	case selection.FocusProject:
		return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			return p.Kind == selection.FocusProject
		})
	case selection.FocusCategory:
		return selectCategory()
	case selection.FocusTask, selection.FocusSeparator:
		return selectTaskRow(kind) || selectCategory()
	case selection.FocusDescription:
		// A description row only exists while descriptions are expanded, which is
		// a config toggle that may differ from the session that saved it — land on
		// the parent task instead of drifting to the top.
		return selectTaskRow(selection.FocusDescription) ||
			selectTaskRow(selection.FocusTask) ||
			selectCategory()
	}
	return false
}

// taskRowVisible is the single source of truth for whether a row in a
// category's Tasks slice is shown. Separators are hidden whenever any filter is
// active (they carry no status/priority to match against); ordinary tasks defer
// to the filter's per-task rule.
func taskRowVisible(filter *FilterState, task domain.Task, categoryID string) bool {
	if task.IsSeparator() {
		return filter == nil || !filter.HasActiveFilter()
	}
	return filter == nil || filter.TaskVisible(task, categoryID)
}

// expandedTaskID's description row shows even while descriptions are hidden.
func rebuildPositions(categories []domain.Category, filter *FilterState, fold *FoldState, expandDescriptions bool, expandedTaskID string) []selection.Position {
	positions := make([]selection.Position, 0)
	positions = append(positions, selection.Position{
		Kind:          selection.FocusProject,
		CategoryIndex: -1,
		TaskIndex:     -1,
	})
	for cIndex, category := range categories {
		positions = append(positions, selection.Position{
			Kind:          selection.FocusCategory,
			CategoryIndex: cIndex,
			TaskIndex:     -1,
		})
		if fold != nil && fold.IsFolded(category.ID) {
			continue
		}
		for tIndex, task := range category.Tasks {
			if !taskRowVisible(filter, task, category.ID) {
				continue
			}
			if task.IsSeparator() {
				positions = append(positions, selection.Position{
					Kind:          selection.FocusSeparator,
					CategoryIndex: cIndex,
					TaskIndex:     tIndex,
				})
				continue
			}
			positions = append(positions, selection.Position{
				Kind:          selection.FocusTask,
				CategoryIndex: cIndex,
				TaskIndex:     tIndex,
			})
			if (expandDescriptions || task.ID == expandedTaskID) && task.Description != "" {
				positions = append(positions, selection.Position{
					Kind:          selection.FocusDescription,
					CategoryIndex: cIndex,
					TaskIndex:     tIndex,
				})
			}
		}
	}
	return positions
}

func (m *model) rebuildPositions() {
	positions := rebuildPositions(m.project.Categories, &m.ui.Filter, &m.ui.Fold, m.ui.Screen.ExpandDescriptions, m.ui.Screen.ExpandedTaskID)
	m.ui.Selection.SetPositions(positions)
	// Structure changed (filter, fold, add/delete, reload, undo/redo): the
	// memoized layout is stale.
	m.invalidateLayout()
}
