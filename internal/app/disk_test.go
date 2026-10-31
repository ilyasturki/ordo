package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func tick(t *testing.T, m *model) {
	t.Helper()
	next, cmd := m.Update(diskTickMsg{})
	*m = next.(model)
	require.NotNil(t, cmd, "the tick arms the next one")
}

const changedMessage = "Changed on disk: reloaded"

func TestOutsideChangeReloadsAnIdleCopyOnTheSameRow(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, task.ID)

	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "nothing changed")

	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, task.ID, "Web title") })
	tick(t, m)

	got, _ := findTask(m.project, task.ID)
	assert.Equal(t, "Web title", got.Title)
	assert.Equal(t, "task:"+task.ID, cursorItem(t, m))
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
	assert.Contains(t, m.renderView(), "Web title")

	tick(t, m)
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg, "the tick leaves the status line alone")
	hit(m, "j")
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "the change is told once")
}

func TestOutsideChangeCursorFallsToTheCategoryWhenItsRowGoes(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, task.ID)
	ci, ti := m.findTaskByID(task.ID)
	cat := p.Categories[ci].ID

	onWeb(t, web, p.ID, func(p *domain.Project) { require.NoError(t, p.Categories[ci].RemoveTask(ti)) })
	tick(t, m)

	assert.Equal(t, "category:"+cat, cursorItem(t, m))
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
}

func TestOwnSavesNeverCountAsAnOutsideChange(t *testing.T) {
	cases := map[string]func(t *testing.T, m *model) *data.Saver{
		"synchronous": func(*testing.T, *model) *data.Saver { return nil },
		"saver":       func(t *testing.T, m *model) *data.Saver { return withSaver(t, m) },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			m, _, p := sharedModel(t)
			saver := setup(t, m)
			m.selectItem(selection.FocusTask, tasksIn(p, 1)[0].ID)

			for range 3 {
				hit(m, "l")
				tick(t, m)
			}
			if saver != nil {
				saver.Drain()
			}
			tick(t, m)

			assert.Empty(t, m.ui.Screen.StatusMsg)
			assert.Empty(t, m.ui.Save.mark())
			hit(m, "u")
			assert.NotEqual(t, "Nothing to undo", m.ui.Screen.StatusMsg, "no reload wiped the history")
			if saver != nil {
				require.NoError(t, saver.Close())
			}
		})
	}
}

func TestOutsideChangeKeepsTheRefusedEditsAndTellsOnce(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	mine, theirs := picked[0], picked[1]
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, theirs.ID, "Web title") })
	m.selectItem(selection.FocusTask, mine.ID)
	hit(m, "l")
	raised, _ := findTask(m.project, mine.ID)
	require.Equal(t, "[changed on disk: r drops 1 edit]", m.ui.Save.mark())

	hit(m, "j", "k")
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "the refusal already told it")
	got, _ := findTask(m.project, mine.ID)
	assert.Equal(t, raised.Priority, got.Priority, "no reload drops the edit")
	assert.Equal(t, "[changed on disk: r drops 1 edit]", m.ui.Save.mark())

	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, theirs.ID, "Web title again") })
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "a second version changes nothing for the edit")
	got, _ = findTask(m.project, theirs.ID)
	assert.Equal(t, theirs.Title, got.Title, "still no reload")

	hit(m, "r")
	assert.Equal(t, "Reloaded from disk, dropping 1 edit", m.ui.Screen.StatusMsg)
	got, _ = findTask(m.project, theirs.ID)
	assert.Equal(t, "Web title again", got.Title)
}

func TestOutsideChangeTellsAtOnceOfAnEditThatFailedToSave(t *testing.T) {
	m, _, p := sharedModel(t)
	store := m.deps.Store.(*data.Store)
	path := filepath.Join(store.Dir, p.ID+".json")
	lock := path + ".lock"
	require.NoError(t, os.RemoveAll(lock))
	require.NoError(t, os.Mkdir(lock, 0o755), "a directory no flock can open")
	m.selectItem(selection.FocusTask, tasksIn(p, 1)[0].ID)
	hit(m, "l")
	require.Equal(t, "[not saved: 1 edit]", m.ui.Save.mark())
	raised := cloneProject(m.project)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))
	require.NoError(t, os.WriteFile(path, content, 0o644))
	tick(t, m)

	assert.Equal(t, "[changed on disk: r drops 1 edit]", m.ui.Save.mark())
	assert.Equal(t, "Not saved: the project changed on disk", m.ui.Screen.StatusMsg)
	assert.Equal(t, raised, m.project, "the edit stays until r")
	assert.False(t, quits(hit(m, "q")), "q asks")
	assert.True(t, m.ui.Modes.IsConfirmLeave())
}

func TestOutsideChangeWaitsForADialogToClose(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, task.ID)
	hit(m, "?")
	require.True(t, m.ui.Modes.IsHelp())

	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, task.ID, "Web title") })
	tick(t, m)
	assert.True(t, m.ui.Modes.IsHelp())
	got, _ := findTask(m.project, task.ID)
	assert.Equal(t, task.Title, got.Title, "not while it's open")

	hit(m, "esc")
	got, _ = findTask(m.project, task.ID)
	assert.Equal(t, "Web title", got.Title, "as it closes, no tick needed")
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
	assert.Equal(t, "task:"+task.ID, cursorItem(t, m))
}

func TestOutsideChangeWaitsForThePicker(t *testing.T) {
	m, _, alpha, _ := newPickerTestModel(t, "")
	web := data.NewStore(m.deps.Store.(*data.Store).Dir)
	m.rebuildPositions()
	m.ui.Modes.ToProjectPicker()

	onWeb(t, web, alpha.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	tick(t, m)
	assert.True(t, m.ui.Modes.IsProjectPicker())
	assert.Equal(t, alpha.Name, m.project.Name)

	hit(m, "esc")
	assert.Equal(t, "Elsewhere", m.project.Name)
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
}

func TestOutsideChangeLeavesTheEditFieldAlone(t *testing.T) {
	open := func(t *testing.T) (*model, domain.Task) {
		m, web, p := sharedModel(t)
		task := tasksIn(p, 1)[0]
		m.selectItem(selection.FocusTask, task.ID)
		hit(m, "enter")
		require.True(t, m.ui.Modes.IsEdit())
		hit(m, "!")
		typed := m.ui.Edit.input.Value()

		onWeb(t, web, p.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
		tick(t, m)

		require.True(t, m.ui.Modes.IsEdit())
		require.Equal(t, typed, m.ui.Edit.input.Value())
		require.Equal(t, p.Name, m.project.Name)
		return m, task
	}

	t.Run("cancelled, it reloads", func(t *testing.T) {
		m, task := open(t)
		hit(m, "esc")
		assert.True(t, m.ui.Modes.IsNormal())
		assert.Equal(t, "Elsewhere", m.project.Name)
		assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
		assert.Equal(t, "task:"+task.ID, cursorItem(t, m))
	})
	t.Run("committed, the edit is refused and kept", func(t *testing.T) {
		m, task := open(t)
		hit(m, "enter")
		assert.Equal(t, "[changed on disk: r drops 1 edit]", m.ui.Save.mark())
		got, _ := findTask(m.project, task.ID)
		assert.Equal(t, task.Title+"!", got.Title)
	})
}

func TestOutsideChangeKeepsThePeekAndTheCursorOnItsRow(t *testing.T) {
	m := newStoredModel(t, sampleProject())
	web := data.NewStore(m.deps.Store.(*data.Store).Dir)
	withStubStateManager(m)
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.saveFilter()
	m.rebuildPositions()
	cat := m.project.Categories[0].ID
	selectRow(t, m, selection.FocusHidden, cat)
	m, _ = pressKeys(m, "enter")
	peeked := cursorItem(t, m)

	onWeb(t, web, m.project.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	tick(t, m)

	assert.Equal(t, "Elsewhere", m.project.Name)
	assert.True(t, m.ui.Filter.peeks(cat))
	assert.Equal(t, peeked, cursorItem(t, m))
}

func TestOutsideChangeKeepsARowHeldForTheFilter(t *testing.T) {
	m := newStoredModel(t, sampleProject())
	web := data.NewStore(m.deps.Store.(*data.Store).Dir)
	withStubStateManager(m)
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.saveFilter()
	m.rebuildPositions()
	first := m.project.Categories[0].Tasks[0].ID
	selectRow(t, m, selection.FocusTask, first)
	m, _ = pressKeys(m, "space")
	require.True(t, m.ui.Filter.holds(first))

	onWeb(t, web, m.project.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	tick(t, m)

	assert.Equal(t, "Elsewhere", m.project.Name)
	assert.Equal(t, "task:"+first, cursorItem(t, m))
}

func TestOutsideDeleteIsToldOnceAndAReturnReloads(t *testing.T) {
	m, _, p := sharedModel(t)
	path := filepath.Join(m.deps.Store.(*data.Store).Dir, p.ID+".json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	require.NoError(t, os.Remove(path))
	tick(t, m)
	assert.Equal(t, "Deleted on disk: edits here won't be saved", m.ui.Screen.StatusMsg)
	assert.Equal(t, p.Name, m.project.Name, "the copy stays on screen")

	hit(m, "j")
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "told once")

	require.NoError(t, os.WriteFile(path, content, 0o644))
	tick(t, m)
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
}

func TestOutsideGarbageIsToldOnceAndKeepsTheCopy(t *testing.T) {
	m, web, p := sharedModel(t)
	path := filepath.Join(m.deps.Store.(*data.Store).Dir, p.ID+".json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o644))
	tick(t, m)
	assert.Contains(t, m.ui.Screen.StatusMsg, "Reload failed: ")
	assert.Equal(t, p.Name, m.project.Name)

	hit(m, "j")
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "told once")

	require.NoError(t, os.WriteFile(path, content, 0o644))
	onWeb(t, web, p.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	tick(t, m)
	assert.Equal(t, "Elsewhere", m.project.Name, "a readable version reloads")
}

func TestConflictCopyIsNoneOfOrdosBusiness(t *testing.T) {
	m, _, alpha, _ := newPickerTestModel(t, "")
	store := m.deps.Store.(*data.Store)
	copyName := alpha.ID + ".sync-conflict-20260921-203208-ICE5FOT.json"
	require.NoError(t, os.WriteFile(filepath.Join(store.Dir, copyName), []byte("{}"), 0o644))

	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "a copy appearing changes nothing")

	m.reloadProject()
	assert.Equal(t, "Reloaded from disk", m.ui.Screen.StatusMsg)

	m.ui.Picker.selected = 0
	m.ui.Modes.ToProjectPicker()
	m.selectProject()
	assert.Equal(t, alpha.ID, m.project.ID)
	assert.NotContains(t, m.ui.Screen.StatusMsg, "Conflict")

	projects, err := store.ListProjects()
	require.NoError(t, err)
	assert.Len(t, projects, 2, "nor is it a project")
}
