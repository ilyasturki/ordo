package app

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/data"
	"github.com/ilyasturki/phasionary/internal/domain"
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
		return "Project changed on disk; press r to reload (unsaved edits are dropped)"
	}
	return "Save failed: " + err.Error()
}

func conflictNotice(names []string) string {
	if len(names) == 1 {
		return "Conflict: " + names[0]
	}
	return fmt.Sprintf("Conflict: %s (+%d more)", names[0], len(names)-1)
}

// Shows msg, or the open project's conflict copies the status line has not named yet.
func (m *model) announceConflicts(msg string) {
	if m.deps.Store == nil {
		m.ui.Screen.StatusMsg = msg
		return
	}
	names := m.deps.Store.ConflictFiles(m.project.ID)
	var seen []string
	if m.ui.Screen.AnnouncedProject == m.project.ID {
		seen = m.ui.Screen.AnnouncedConflicts
	}
	var fresh []string
	for _, name := range names {
		if !slices.Contains(seen, name) {
			fresh = append(fresh, name)
		}
	}
	m.ui.Screen.AnnouncedProject = m.project.ID
	m.ui.Screen.AnnouncedConflicts = names
	if len(fresh) > 0 {
		msg = conflictNotice(fresh)
	}
	m.ui.Screen.StatusMsg = msg
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
	m.announceConflicts("Reloaded from disk")
}

// restoreSelection moves the cursor onto the row identified by (kind, catID,
// taskID), reporting whether it landed. A row addressed by task ID that no
// longer exists degrades to its category header rather than failing outright, so
// deleting a task leaves the cursor nearby instead of at the top of the list.
func (m *model) restoreSelection(kind selection.FocusKind, catID, taskID string) bool {
	switch kind {
	case selection.FocusProject:
		return m.selectItem(selection.FocusProject, "")
	case selection.FocusCategory:
		return catID != "" && m.selectItem(selection.FocusCategory, catID)
	}
	return (taskID != "" && m.selectItem(kind, taskID)) ||
		(catID != "" && m.selectItem(selection.FocusCategory, catID))
}

// taskRowVisible is the filter's verdict alone; shownRows adds nesting and folds on top.
func taskRowVisible(filter *FilterState, task domain.Task, categoryID, parentCategoryID string) bool {
	if task.IsSeparator() {
		return filter == nil || !filter.HasActiveFilter()
	}
	return filter == nil || filter.TaskVisible(task, categoryID, parentCategoryID)
}

func shownRows(cat domain.Category, filter *FilterState, fold *FoldState) []bool {
	shown := make([]bool, len(cat.Tasks))
	for i, task := range cat.Tasks {
		shown[i] = taskRowVisible(filter, task, cat.ID, cat.ParentID)
	}
	for i := 0; i < len(cat.Tasks); {
		end := domain.BlockEnd(cat.Tasks, i)
		folded := fold != nil && fold.IsFolded(cat.Tasks[i].ID)
		for j := i + 1; j < end; j++ {
			shown[i] = shown[i] || shown[j]
			shown[j] = shown[j] && !folded
		}
		i = end
	}
	return shown
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
		for tIndex, task := range category.Tasks {
			if !shown[tIndex] {
				continue
			}
			if task.IsSeparator() {
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
	}
	return positions
}

func (m *model) rebuildPositions() {
	prev, had := m.selectedPosition()
	m.ui.Fold.SetShipped(m.project.Categories)
	positions := rebuildPositions(m.project.Categories, &m.ui.Filter, &m.ui.Fold, m.ui.Screen.ExpandDescriptions, m.ui.Screen.ExpandedTaskID)
	m.ui.Selection.SetPositions(positions)
	if had {
		m.selectItem(prev.Kind, prev.ID)
	}
	// Structure changed (filter, fold, add/delete, reload, undo/redo): the
	// memoized layout is stale.
	m.invalidateLayout()
}
