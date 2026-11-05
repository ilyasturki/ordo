package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

// collide leaves the dialog open: the user commits "Typed" over a title the web changed meanwhile.
func collide(t *testing.T, m *model, web *data.Store, p domain.Project, task domain.Task) {
	t.Helper()
	m.selectItem(selection.FocusTask, task.ID)
	hit(m, "enter")
	require.True(t, m.ui.Modes.IsEdit())
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, task.ID, "Web title") })
	m.ui.Edit.input.SetValue("Typed")
	hit(m, "enter")
	if m.deps.Saver != nil {
		m.deps.Saver.Drain()
	}
	tick(t, m)
	require.True(t, m.ui.Modes.IsConflict())
}

// swapFile puts a version on disk the way a sync tool does: by rename, with no ordo writing it.
func swapFile(t *testing.T, m *model, p domain.Project) {
	t.Helper()
	path := filepath.Join(m.deps.Store.(*data.Store).Dir, p.ID+".json")
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path+".swap", raw, 0o644))
	require.NoError(t, os.Rename(path+".swap", path))
}

func titleOnDisk(t *testing.T, web *data.Store, p domain.Project, id string) string {
	t.Helper()
	got, ok := findTask(onDisk(t, web, p.ID), id)
	require.True(t, ok)
	return got.Title
}

func TestCollisionOffersBothValues(t *testing.T) {
	m, web, p := sharedModel(t)
	collide(t, m, web, p, tasksIn(p, 1)[0])

	view := m.renderView()
	assert.Contains(t, view, "Changed on disk: 1 conflict")
	assert.Contains(t, view, "Typed")
	assert.Contains(t, view, "Web title")
}

func TestPickingMineSavesItAndUndoGivesTheirsBack(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	collide(t, m, web, p, task)

	hit(m, "m")
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Typed", titleOnDisk(t, web, p, task.ID))

	hit(m, "u")
	assert.Equal(t, "Web title", titleOnDisk(t, web, p, task.ID))
}

func TestPickingTheirsLeavesTheFileAndUndoGivesMineBack(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	collide(t, m, web, p, task)

	hit(m, "t")
	assert.Equal(t, "Web title", titleOnDisk(t, web, p, task.ID))
	got, _ := findTask(m.project, task.ID)
	assert.Equal(t, "Web title", got.Title)

	hit(m, "u")
	assert.Equal(t, "Typed", titleOnDisk(t, web, p, task.ID))
}

func TestPickingBothKeepsACopyBelow(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	collide(t, m, web, p, task)
	before := len(flatTasks(onDisk(t, web, p.ID)))

	hit(m, "b")

	disk := flatTasks(onDisk(t, web, p.ID))
	assert.Len(t, disk, before+1)
	titles := make([]string, len(disk))
	for i, row := range disk {
		titles[i] = row.Title
	}
	assert.Contains(t, titles, "Web title")
	assert.Contains(t, titles, "Typed")
}

func flatTasks(p domain.Project) []domain.Task {
	var out []domain.Task
	for _, c := range p.Categories {
		out = append(out, c.Tasks...)
	}
	return out
}

func TestACapitalSettlesTheRest(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	m.selectItem(selection.FocusTask, picked[0].ID)
	m.ui.Modes.ToNormal()
	mine := cloneProject(m.project)
	retitle(&mine, picked[0].ID, "Mine A")
	retitle(&mine, picked[1].ID, "Mine B")
	onWeb(t, web, p.ID, func(p *domain.Project) {
		retitle(p, picked[0].ID, "Web A")
		retitle(p, picked[1].ID, "Web B")
	})
	m.project = mine
	tick(t, m)
	require.True(t, m.ui.Modes.IsConflict())
	assert.Contains(t, m.renderView(), "1 of 2 conflicts")

	hit(m, "M")

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Mine A", titleOnDisk(t, web, p, picked[0].ID))
	assert.Equal(t, "Mine B", titleOnDisk(t, web, p, picked[1].ID))
}

func TestOtherKeysLeaveTheDialogOpen(t *testing.T) {
	m, web, p := sharedModel(t)
	collide(t, m, web, p, tasksIn(p, 1)[0])
	hit(m, "esc", "q", "j")
	assert.True(t, m.ui.Modes.IsConflict())
}

func TestBothSidesAreKeptBeforeTheDialogAsks(t *testing.T) {
	m, web, p := sharedModel(t)
	collide(t, m, web, p, tasksIn(p, 1)[0])

	dir := filepath.Join(filepath.Dir(m.deps.Store.(*data.Store).Dir), "recovery")
	mine, err := filepath.Glob(filepath.Join(dir, p.ID+".*.mine.json"))
	require.NoError(t, err)
	theirs, err := filepath.Glob(filepath.Join(dir, p.ID+".*.theirs.json"))
	require.NoError(t, err)
	require.Len(t, mine, 1)
	require.Len(t, theirs, 1)
	raw, err := os.ReadFile(mine[0])
	require.NoError(t, err)
	assert.Contains(t, string(raw), "Typed")
}

func TestCtrlCInTheDialogExitsNamingTheCopy(t *testing.T) {
	m, web, p := sharedModel(t)
	saver := withSaver(t, m)
	task := tasksIn(p, 1)[0]
	collide(t, m, web, p, task)
	assert.Equal(t, "Web title", titleOnDisk(t, web, p, task.ID))

	assert.True(t, quits(hit(m, "ctrl+c")))

	err := saver.Close()
	require.ErrorIs(t, err, data.ErrStaleProject)
	assert.Contains(t, err.Error(), "recovery")
}

func TestQuitWithAChangeWaitingOpensTheDialogInstead(t *testing.T) {
	m, _, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	elsewhere := cloneProject(p)
	m.selectItem(selection.FocusTask, task.ID)
	hit(m, "enter")
	m.ui.Edit.input.SetValue("Typed")
	hit(m, "enter")
	retitle(&elsewhere, task.ID, "Web title")
	swapFile(t, m, elsewhere)

	assert.False(t, quits(hit(m, "q")))
	assert.True(t, m.ui.Modes.IsConflict())
	assert.False(t, m.ui.Modes.IsConfirmLeave())
}

// The edit saved, then a sync tool swapped in a version that never saw it.
func TestAReplacedEditComesBack(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	elsewhere := cloneProject(p)
	m.selectItem(selection.FocusTask, picked[0].ID)
	hit(m, "l")
	raised, _ := findTask(m.project, picked[0].ID)

	retitle(&elsewhere, picked[1].ID, "From the other device")
	swapFile(t, m, elsewhere)
	tick(t, m)

	assert.True(t, m.ui.Modes.IsNormal(), "no dialog: nothing collides")
	assert.Equal(t, "Changed on disk: merged with your edits", m.ui.Screen.StatusMsg)
	disk := onDisk(t, web, p.ID)
	got, _ := findTask(disk, picked[0].ID)
	assert.Equal(t, raised.Priority, got.Priority, "the edit is back in the file")
	got, _ = findTask(disk, picked[1].ID)
	assert.Equal(t, "From the other device", got.Title)
}

// An agent through the CLI, after the user: the same field, and still no conflict.
func TestALocalWriterOverTheSameFieldIsNoConflict(t *testing.T) {
	m, web, p := sharedModel(t)
	task := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, task.ID)
	hit(m, "enter")
	m.ui.Edit.input.SetValue("Typed")
	hit(m, "enter")
	require.Equal(t, "Typed", titleOnDisk(t, web, p, task.ID))

	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, task.ID, "Agent one") })
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, task.ID, "Agent two") })
	tick(t, m)

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
	got, _ := findTask(m.project, task.ID)
	assert.Equal(t, "Agent two", got.Title)
}

// No write log speaks for this one: the file holds another row of the session, so it was built on it.
func TestAForeignWriterBuiltOnMyEditsIsNoConflict(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	m.selectItem(selection.FocusTask, picked[0].ID)
	hit(m, "l")
	m.selectItem(selection.FocusTask, picked[1].ID)
	hit(m, "l")

	built := onDisk(t, web, p.ID)
	retitle(&built, picked[0].ID, "Foreign")
	for ci := range built.Categories {
		for ti := range built.Categories[ci].Tasks {
			if built.Categories[ci].Tasks[ti].ID == picked[0].ID {
				built.Categories[ci].Tasks[ti].Priority = domain.PriorityTrivial
			}
		}
	}
	swapFile(t, m, built)
	tick(t, m)

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, changedMessage, m.ui.Screen.StatusMsg)
	got, _ := findTask(m.project, picked[0].ID)
	assert.Equal(t, domain.PriorityTrivial, got.Priority)
}

func TestUndoSurvivesAnOutsideChangeAndKeepsIt(t *testing.T) {
	m, web, p := sharedModel(t)
	picked := tasksIn(p, 2)
	m.selectItem(selection.FocusTask, picked[0].ID)
	hit(m, "l")
	onWeb(t, web, p.ID, func(p *domain.Project) { retitle(p, picked[1].ID, "Web title") })
	tick(t, m)

	hit(m, "u")

	disk := onDisk(t, web, p.ID)
	got, _ := findTask(disk, picked[0].ID)
	assert.Equal(t, picked[0].Priority, got.Priority)
	got, _ = findTask(disk, picked[1].ID)
	assert.Equal(t, "Web title", got.Title)
}

func TestReloadRetriesAnEditThatFailedToSave(t *testing.T) {
	m, web, p := sharedModel(t)
	store := m.deps.Store.(*data.Store)
	lock := filepath.Join(store.Dir, p.ID+".json.lock")
	require.NoError(t, os.RemoveAll(lock))
	require.NoError(t, os.Mkdir(lock, 0o755), "a directory no flock can open")
	task := tasksIn(p, 1)[0]
	m.selectItem(selection.FocusTask, task.ID)
	hit(m, "l")
	require.Equal(t, "[not saved: 1 edit]", m.ui.Save.mark())
	raised, _ := findTask(m.project, task.ID)

	require.NoError(t, os.Remove(lock))
	hit(m, "r")

	assert.Empty(t, m.ui.Save.mark())
	got, _ := findTask(onDisk(t, web, p.ID), task.ID)
	assert.Equal(t, raised.Priority, got.Priority)
}

// The refusal lands with the next editor already open: ctrl+c commits it, merges, and quits, with nothing to ask.
func TestCtrlCInAnEditorOverARefusedEditMergesAndQuits(t *testing.T) {
	m, web, p := sharedModel(t)
	saver := withSaver(t, m)
	picked := tasksIn(p, 2)
	onWeb(t, web, p.ID, func(p *domain.Project) { p.Name = "Elsewhere" })
	m.selectItem(selection.FocusTask, picked[0].ID)
	hit(m, "l")
	raised, _ := findTask(m.project, picked[0].ID)
	m.selectItem(selection.FocusTask, picked[1].ID)
	hit(m, "enter")
	require.True(t, m.ui.Modes.IsEdit())
	m.ui.Edit.input.SetValue("Typed")
	saver.Drain()

	assert.True(t, quits(hit(m, "ctrl+c")), "nothing collides, so nothing asks")
	assert.False(t, m.ui.Modes.IsConfirmLeave())

	require.NoError(t, saver.Close())
	disk := onDisk(t, web, p.ID)
	assert.Equal(t, "Elsewhere", disk.Name)
	got, _ := findTask(disk, picked[0].ID)
	assert.Equal(t, raised.Priority, got.Priority)
	got, _ = findTask(disk, picked[1].ID)
	assert.Equal(t, "Typed", got.Title)
}
