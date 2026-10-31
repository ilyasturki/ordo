package app

import (
	"errors"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/data"
)

const diskPollInterval = time.Second

type diskTickMsg struct{}

func diskTick() tea.Cmd {
	return tea.Tick(diskPollInterval, func(time.Time) tea.Msg { return diskTickMsg{} })
}

// noticeOutsideChange runs after every event, the idle tick included. Edits the
// copy holds keep the file's change from applying, so it only tells of them; a
// clean copy reloads, once normal mode leaves no dialog or input to pull rows from.
func (m *model) noticeOutsideChange() {
	if m.deps.Store == nil {
		return
	}
	if n, _ := m.deps.Store.Unsaved(m.project.ID); n == 0 && !m.ui.Modes.IsNormal() {
		return
	}
	if !m.deps.Store.OutsideChange(m.project.ID) {
		return
	}
	told := m.ui.Save.Err
	m.checkSaves()
	if m.ui.Save.Err != nil {
		if !errors.Is(told, m.ui.Save.Err) {
			m.ui.Screen.StatusMsg = saveFailedMessage(m.ui.Save.Err)
		}
		return
	}
	project, err := m.deps.Store.LoadProjectByID(m.project.ID)
	switch {
	case errors.Is(err, data.ErrProjectNotFound):
		m.ui.Screen.StatusMsg = "Deleted on disk: edits here won't be saved"
	case err != nil:
		m.ui.Screen.StatusMsg = "Reload failed: " + err.Error()
	default:
		m.takeReloaded(project)
		m.ui.Screen.StatusMsg = "Changed on disk: reloaded"
	}
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
