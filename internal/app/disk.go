package app

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/merge"
)

const diskPollInterval = time.Second

type diskTickMsg struct{}

func diskTick() tea.Cmd {
	return tea.Tick(diskPollInterval, func(time.Time) tea.Msg { return diskTickMsg{} })
}

// noticeOutsideChange runs after every event, the idle tick included. A dialog
// or input holds rows, so the change waits for normal mode.
func (m *model) noticeOutsideChange() {
	if m.deps.Store == nil || !m.ui.Modes.IsNormal() {
		return
	}
	if !m.deps.Store.OutsideChange(m.project.ID) {
		return
	}
	m.checkSaves()
	if status := m.takeOutside(); status != "" {
		m.ui.Screen.StatusMsg = status
	}
}

// takeOutside returns what to tell, nothing when a collision opened the dialog.
func (m *model) takeOutside() string {
	if m.deps.Saver != nil {
		m.deps.Saver.Drain()
	}
	out, err := m.deps.Store.LoadOutside(m.project.ID)
	switch {
	case errors.Is(err, data.ErrProjectNotFound):
		return "Deleted on disk: edits here won't be saved"
	case err != nil:
		return "Reload failed: " + err.Error()
	}
	res := merge.Three(out.Parent, m.project, out.Theirs)
	if len(res.Collisions) > 0 {
		m.openConflict(res, out.Theirs)
		return ""
	}
	if m.takeMerged(res.Project, out.Theirs) {
		return "Changed on disk: merged with your edits"
	}
	return "Changed on disk: reloaded"
}

func (m *model) takeMerged(merged, theirs domain.Project) bool {
	m.ui.History.rebase(m.project, merged)
	m.showProject(merged)
	if merge.Same(merged, theirs) {
		return false
	}
	m.storeTaskUpdate()
	return true
}

// noticeViewChange applies a view file another writer changed, on the same terms as a reload.
func (m *model) noticeViewChange() {
	if m.deps.StateManager == nil || m.project.ID == "" || !m.ui.Modes.IsNormal() {
		return
	}
	if !m.deps.StateManager.ViewChanged(m.project.ID) || !m.readView() {
		return
	}
	m.rebuildPositions()
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Changed on disk: folds and filter applied"
}

// readView takes the folds and filter the view file holds, reporting whether what shows differs.
func (m *model) readView() bool {
	if m.deps.StateManager == nil {
		return false
	}
	fold := loadFoldState(m.deps.StateManager, m.project)
	filter := loadFilterState(m.deps.StateManager, m.project)
	changed := !m.ui.Fold.sameAs(fold)
	m.ui.Fold = fold
	if !m.ui.Filter.sameValues(filter) {
		m.ui.Filter.takeValues(filter)
		changed = true
	}
	return changed
}
