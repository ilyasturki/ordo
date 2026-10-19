package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

// sharedModel opens a project in the TUI the way Run does, beside a second
// store standing in for the web app writing to the same directory.
func sharedModel(t *testing.T) (*model, *data.Store, domain.Project) {
	t.Helper()
	dir := t.TempDir()
	tui, web := data.NewStore(dir), data.NewStore(dir)
	p, err := tui.CreateProject("Shared")
	require.NoError(t, err)
	p, err = tui.LoadProjectByID(p.ID)
	require.NoError(t, err)
	m := newTestModel(t, p)
	m.deps.Store = tui
	m.rebuildPositions()
	return m, web, p
}

func withSaver(t *testing.T, m *model) *data.Saver {
	t.Helper()
	saver := data.NewSaver(m.deps.Store.(*data.Store))
	m.deps.Saver = saver
	return saver
}

// tasksIn returns the first task of the first n categories that have one.
func tasksIn(p domain.Project, n int) []domain.Task {
	var out []domain.Task
	for _, c := range p.Categories {
		if len(c.Tasks) > 0 && len(out) < n {
			out = append(out, c.Tasks[0])
		}
	}
	return out
}

func onWeb(t *testing.T, web *data.Store, id string, fn func(*domain.Project)) {
	t.Helper()
	_, err := web.WithProjectLocked(id, func(p *domain.Project) error {
		fn(p)
		return nil
	})
	require.NoError(t, err)
}

func retitle(p *domain.Project, taskID, title string) {
	for ci := range p.Categories {
		for ti := range p.Categories[ci].Tasks {
			if p.Categories[ci].Tasks[ti].ID == taskID {
				p.Categories[ci].Tasks[ti].Title = title
			}
		}
	}
}

func onDisk(t *testing.T, s *data.Store, id string) domain.Project {
	t.Helper()
	p, err := s.LoadProjectByID(id)
	require.NoError(t, err)
	return p
}

func findTask(p domain.Project, id string) (domain.Task, bool) {
	for _, c := range p.Categories {
		for _, task := range c.Tasks {
			if task.ID == id {
				return task, true
			}
		}
	}
	return domain.Task{}, false
}

func hit(m *model, keys ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range keys {
		var next tea.Model
		next, cmd = m.Update(key(k))
		*m = next.(model)
	}
	return cmd
}

func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// Research scenarios a and c: the web edits first, then the TUI.
func TestOutsideEditKeepsTheRefusalInTheHeaderUntilReload(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	mine, theirs := picked[0], picked[1]
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, theirs.ID, "Web title") })

	m.selectItem(selection.FocusTask, mine.ID)
	hit(m, "l")
	assert.Equal(t, "[changed on disk: r drops 1 edit]", m.ui.Save.mark())
	hit(m, "j")
	assert.Empty(t, m.ui.Screen.StatusMsg, "the next key clears the status line")
	assert.Contains(t, m.renderView(), "[changed on disk: r drops 1 edit]", "but not the header mark")
	hit(m, "k", "h", "h")
	assert.Contains(t, m.renderView(), "[changed on disk: r drops 3 edits]")

	assert.False(t, quits(hit(m, "q")), "q asks before dropping them")
	assert.True(t, m.ui.Modes.IsConfirmLeave())
	assert.Contains(t, m.renderView(), "Quit and drop 3 edits?")
	hit(m, "n")
	assert.True(t, m.ui.Modes.IsNormal())

	disk := onDisk(t, web, p.ID)
	got, _ := findTask(disk, mine.ID)
	assert.Equal(t, mine.Priority, got.Priority, "the refused edits never reached the file")
	got, _ = findTask(disk, theirs.ID)
	assert.Equal(t, "Web title", got.Title)

	hit(m, "r")
	assert.Empty(t, m.ui.Save.mark())
	assert.Equal(t, "Reloaded from disk, dropping 3 edits", m.ui.Screen.StatusMsg)
	got, _ = findTask(m.project, theirs.ID)
	assert.Equal(t, "Web title", got.Title)
	assert.True(t, quits(hit(m, "q")), "nothing left to lose")
}

func TestHeaderMarkShowsWhileScrolled(t *testing.T) {
	m, web, p := sharedModel(t)
	onWeb(t, web, p.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	m.ui.Screen.Height = 6
	hit(m, "G", "l")

	view := m.renderView()
	require.Contains(t, view, scrollMoreAbove)
	assert.Contains(t, view, "[changed on disk: r drops 1 edit]")
}

// Research scenario d: the TUI's title editor is open on a task the web deletes.
func TestEditorOpenWhileTheWebDeletesItsTask(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, task.ID)
	hit(m, "enter")
	require.True(t, m.ui.Modes.IsEdit())
	onWeb(t, web, p.ID, func(p *domain.Project) {
		for ci := range p.Categories {
			for ti, other := range p.Categories[ci].Tasks {
				if other.ID == task.ID {
					require.NoError(t, p.Categories[ci].RemoveTask(ti))
					return
				}
			}
		}
	})

	hit(m, "!", "enter")
	assert.Equal(t, "[changed on disk: r drops 1 edit]", m.ui.Save.mark())
	assert.False(t, quits(hit(m, "ctrl+c")), "ctrl+c asks too")
	assert.True(t, m.ui.Modes.IsConfirmLeave())
	assert.True(t, quits(hit(m, "y")))
	_, kept := findTask(onDisk(t, web, p.ID), task.ID)
	assert.False(t, kept, "the refused edit doesn't bring the task back")
}

func TestCancellingTheLeaveConfirmReturnsToTheCoveredMode(t *testing.T) {
	m, web, p := sharedModel(t)
	onWeb(t, web, p.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	m.selectItem(selection.FocusTask, tasksIn(p, 1)[0].ID)
	hit(m, "l", "enter")
	require.True(t, m.ui.Modes.IsEdit())

	hit(m, "ctrl+c")
	require.True(t, m.ui.Modes.IsConfirmLeave())
	hit(m, "esc")
	assert.True(t, m.ui.Modes.IsEdit(), "the title editor is still open")
	assert.True(t, quits(hit(m, "ctrl+c", "ctrl+c")), "a second ctrl+c quits")
}

// Research scenario e: an undo after the web changed the file.
func TestUndoAfterAnOutsideEdit(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	mine, theirs := picked[0], picked[1]
	m.selectItem(selection.FocusTask, mine.ID)
	hit(m, "l")
	require.Empty(t, m.ui.Save.mark(), "saved before the web wrote")
	raised, _ := findTask(m.project, mine.ID)
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, theirs.ID, "Web title") })

	hit(m, "u")
	assert.Equal(t, "[changed on disk: r drops 1 edit]", m.ui.Save.mark(), "the undo shows but never landed")
	disk := onDisk(t, web, p.ID)
	got, _ := findTask(disk, mine.ID)
	assert.Equal(t, raised.Priority, got.Priority)
	got, _ = findTask(disk, theirs.ID)
	assert.Equal(t, "Web title", got.Title)
	assert.False(t, quits(hit(m, "q")))

	hit(m, "esc", "r")
	assert.Empty(t, m.ui.Save.mark())
	got, _ = findTask(m.project, mine.ID)
	assert.Equal(t, raised.Priority, got.Priority)
	hit(m, "u")
	assert.Equal(t, "Nothing to undo", m.ui.Screen.StatusMsg, "replaying history now would revert the web's edit")
}

// Research scenario g: cut, a paste the stale guard refuses, r, then p.
func TestCutSurvivesARefusedPaste(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	cut, theirs := picked[0], picked[1]
	target := p.Categories[len(p.Categories)-1].ID
	m.selectItem(selection.FocusTask, cut.ID)
	m.cutSelectedTask()
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, theirs.ID, "Web title") })

	m.selectItem(selection.FocusCategory, target)
	m.pasteTasks()
	require.NotEmpty(t, m.ui.Save.mark(), "the move was refused")
	m.reloadProject()
	assert.True(t, m.ui.Clipboard.IsCut, "r gives the cut back")
	m.selectItem(selection.FocusCategory, target)
	m.pasteTasks()

	copies := 0
	disk := onDisk(t, web, p.ID)
	for _, c := range disk.Categories {
		for _, task := range c.Tasks {
			if task.Title == cut.Title {
				copies++
				assert.Equal(t, cut.ID, task.ID)
				assert.Equal(t, target, c.ID)
			}
		}
	}
	assert.Equal(t, 1, copies)
}

func TestReloadKeepsTheCopyOfACutThatLanded(t *testing.T) {
	m, _, p := sharedModel(t)
	cut := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, cut.ID)
	m.cutSelectedTask()
	m.selectItem(selection.FocusCategory, p.Categories[len(p.Categories)-1].ID)
	m.pasteTasks()

	m.reloadProject()

	assert.False(t, m.ui.Clipboard.IsCut, "p pastes a copy, as after any move")
}

// The research's snapshot race, through the TUI: r must see an edit still queued.
func TestReloadWaitsForTheQueuedSnapshot(t *testing.T) {
	m, web, p := sharedModel(t)
	saver := withSaver(t, m)
	mine := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, mine.ID)
	hit(m, "l")
	raised, _ := findTask(m.project, mine.ID)

	hit(m, "r")

	got, _ := findTask(m.project, mine.ID)
	assert.Equal(t, raised.Priority, got.Priority)
	assert.Equal(t, "Reloaded from disk", m.ui.Screen.StatusMsg)
	onWeb(t, web, p.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	hit(m, "r", "h")
	saver.Drain()
	m.checkSaves()
	assert.Empty(t, m.ui.Save.mark(), "the reloaded copy saves again")
	assert.NoError(t, saver.Close())
}

func TestQuitAnywayLeavesTheErrorForOrdoToPrint(t *testing.T) {
	m, web, p := sharedModel(t)
	saver := withSaver(t, m)
	onWeb(t, web, p.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	m.selectItem(selection.FocusTask, tasksIn(p, 1)[0].ID)
	hit(m, "l")

	assert.False(t, quits(hit(m, "q")), "q waits for the write, then asks")
	assert.True(t, quits(hit(m, "y")))

	err := saver.Close()
	require.ErrorIs(t, err, data.ErrStaleProject)
	assert.Equal(t, `1 edit to "Shared" not saved: project changed on disk since it was loaded`, err.Error())
}

func TestSwitchingProjectsAsksBeforeDroppingEdits(t *testing.T) {
	m, _, alpha, beta := newPickerTestModel(t, "")
	web := data.NewStore(m.deps.Store.(*data.Store).Dir)
	m.rebuildPositions()
	onWeb(t, web, alpha.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	m.selectItem(selection.FocusTask, tasksIn(alpha, 1)[0].ID)
	hit(m, "l")

	m.ui.Modes.ToProjectPicker()
	m.selectProject()
	assert.True(t, m.ui.Modes.IsConfirmLeave())
	assert.Equal(t, alpha.ID, m.project.ID)
	assert.Contains(t, m.renderView(), "Switch projects and drop 1 edit?")
	hit(m, "esc")
	assert.True(t, m.ui.Modes.IsProjectPicker())

	m.selectProject()
	hit(m, "y")
	assert.Equal(t, beta.ID, m.project.ID)
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Empty(t, m.ui.Save.mark())
}
