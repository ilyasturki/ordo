package app

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/data"
	"github.com/ilyasturki/phasionary/internal/domain"
)

// stubCursorSaver records what the background writer was handed. The throttling
// itself belongs to data.CursorSaver and is tested there; here only the handoff
// matters.
type stubCursorSaver struct {
	ids     []string
	cursors []data.Cursor
	dropped []string
}

func (s *stubCursorSaver) Set(projectID string, cursor data.Cursor) {
	s.ids = append(s.ids, projectID)
	s.cursors = append(s.cursors, cursor)
}

func (s *stubCursorSaver) Drop(projectID string) { s.dropped = append(s.dropped, projectID) }

// last returns the most recent (project, cursor) pair handed to the saver.
func (s *stubCursorSaver) last(t *testing.T) (string, data.Cursor) {
	t.Helper()
	require.NotEmpty(t, s.cursors, "nothing was handed to the cursor saver")
	return s.ids[len(s.ids)-1], s.cursors[len(s.cursors)-1]
}

func withStubCursorSaver(m *model) *stubCursorSaver {
	stub := &stubCursorSaver{}
	m.deps.CursorSaver = stub
	return stub
}

// The cursor has to reach the saver as it moves, not only on the way out: a
// force quit never runs the save on exit.
func TestTrackCursor_RecordsMovesAsTheyHappen(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	saver := withStubCursorSaver(m)
	selectTask(t, m, "t1")

	_, _ = m.Update(tea.KeyPressMsg{Code: 'j'})

	id, cursor := saver.last(t)
	assert.Equal(t, "p1", id)
	assert.Equal(t, data.Cursor{Kind: "task", CategoryID: "c1", TaskID: "t2"}, cursor)
}

// The behaviour this whole path exists for, end to end over the real state
// file: move the cursor, then walk away without quitting — no exit save, no
// Close, exactly what a kill -9 or a closed terminal leaves behind. The next
// session must still find the row.
func TestCursorReachesDiskWithoutQuitting(t *testing.T) {
	stateDir := t.TempDir()
	state := data.NewStateManager(stateDir, "/work/dir")
	require.NoError(t, state.Load())
	saver := data.NewCursorSaver(state)
	t.Cleanup(saver.Close)

	m := newTestModel(t, sampleProject())
	m.deps.StateManager = state
	m.deps.CursorSaver = saver
	selectTask(t, m, "t1")

	_, _ = m.Update(tea.KeyPressMsg{Code: 'j'})

	// A separate manager, as a new process would read it.
	next := data.NewStateManager(stateDir, "/work/dir")
	want := data.Cursor{Kind: "task", CategoryID: "c1", TaskID: "t2"}
	assert.Eventually(t, func() bool {
		return next.GetCursor("p1") == want
	}, 3*time.Second, 20*time.Millisecond, "a moved cursor must reach state.json on its own")
}

// Most events leave the cursor alone. Handing the same row over again would
// wake the writer for nothing.
func TestTrackCursor_SkipsEventsThatLeaveTheCursorPut(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	saver := withStubCursorSaver(m)
	_, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	require.Len(t, saver.cursors, 1)

	_, _ = m.Update(tea.BlurMsg{})
	_, _ = m.Update(tea.FocusMsg{})

	assert.Len(t, saver.cursors, 1, "only a move should reach the saver")
}

// Folding a category pulls the cursor onto its header — a move like any other,
// and one the next session must reopen on.
func TestTrackCursor_RecordsCursorPulledByAFold(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	saver := withStubCursorSaver(m)
	selectTask(t, m, "t2")

	_, _ = m.Update(tea.KeyPressMsg{Code: 'z'})
	_, _ = m.Update(tea.KeyPressMsg{Code: 'c'})

	_, cursor := saver.last(t)
	assert.Equal(t, data.Cursor{Kind: "category", CategoryID: "c1"}, cursor)
}

// The synchronous save is authoritative for the row it writes, so the queued
// copy must go: a slower background write would otherwise land on top of it.
func TestSaveCursorState_DropsTheQueuedWrite(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	saver := withStubCursorSaver(m)
	selectTask(t, m, "t4")

	m.saveCursorState()

	assert.Equal(t, []string{"p1"}, saver.dropped)
}

// A row saved synchronously is already recorded; the next event must not queue
// it a second time.
func TestSaveCursorState_LeavesNothingForTheSaverToRepeat(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	saver := withStubCursorSaver(m)
	selectTask(t, m, "t4")
	m.saveCursorState()

	_, _ = m.Update(tea.BlurMsg{})

	assert.Empty(t, saver.cursors)
}

// Deleting a project deletes its stored cursor. A write still sitting in the
// queue would put it back moments later.
func TestConfirmDeleteProject_DropsTheQueuedCursor(t *testing.T) {
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	keep, err := store.ImportProject(domain.Project{
		Name: "Keep",
		Categories: []domain.Category{{
			ID:    "ck",
			Name:  "Cat",
			Tasks: []domain.Task{{ID: "k1", Title: "One", Status: domain.StatusTodo}},
		}},
	})
	require.NoError(t, err)
	doomed, err := store.ImportProject(domain.Project{Name: "Doomed"})
	require.NoError(t, err)

	m := newTestModel(t, keep)
	stub := withStubStateManager(m)
	saver := withStubCursorSaver(m)
	m.deps.Store = store
	stub.cursors[doomed.ID] = data.Cursor{Kind: "category", CategoryID: "cd"}

	m.ui.Picker = ProjectPickerState{projects: []domain.Project{keep, doomed}, selected: 1}
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteProject, ProjectID: doomed.ID}
	m.confirmDeleteProject()

	assert.NotContains(t, stub.cursors, doomed.ID)
	assert.Contains(t, saver.dropped, doomed.ID)
}

func TestConfirmDeleteProject_FallbackKeepsTheStaleGuard(t *testing.T) {
	dir := t.TempDir()
	seed := data.NewStore(dir)
	keep, err := seed.CreateProject("Keep")
	require.NoError(t, err)
	doomed, err := seed.CreateProject("Doomed")
	require.NoError(t, err)

	store := data.NewStore(dir)
	m := newTestModel(t, doomed)
	withStubStateManager(m)
	withStubCursorSaver(m)
	m.deps.Store = store
	m.ui.Picker = ProjectPickerState{projects: []domain.Project{doomed, keep}, selected: 0}
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteProject, ProjectID: doomed.ID}

	m.confirmDeleteProject()
	require.Equal(t, keep.ID, m.project.ID)

	_, err = seed.WithProjectLocked(keep.ID, func(p *domain.Project) error {
		p.Name = "Renamed elsewhere"
		return nil
	})
	require.NoError(t, err)
	assert.ErrorIs(t, store.SaveProjectLocked(m.project), data.ErrStaleProject)
}
