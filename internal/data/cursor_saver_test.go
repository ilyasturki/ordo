package data

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// recordingRepo is a StateRepository that only implements the cursor half; the
// saver never touches the rest.
type recordingRepo struct {
	mu      sync.Mutex
	cursors map[string]Cursor
	writes  int
}

func newRecordingRepo() *recordingRepo {
	return &recordingRepo{cursors: make(map[string]Cursor)}
}

func (r *recordingRepo) SetCursor(projectID string, cursor Cursor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writes++
	if cursor.IsZero() {
		delete(r.cursors, projectID)
		return nil
	}
	r.cursors[projectID] = cursor
	return nil
}

func (r *recordingRepo) snapshot() (map[string]Cursor, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]Cursor, len(r.cursors))
	for k, v := range r.cursors {
		out[k] = v
	}
	return out, r.writes
}

func (r *recordingRepo) GetProjectForDir() string                   { return "" }
func (r *recordingRepo) SetProjectForDir(string) error              { return nil }
func (r *recordingRepo) GetProjectOrder() []string                  { return nil }
func (r *recordingRepo) SetProjectOrder([]string) error             { return nil }
func (r *recordingRepo) GetFoldedCategories(string) []string        { return nil }
func (r *recordingRepo) SetFoldedCategories(string, []string) error { return nil }
func (r *recordingRepo) DeleteFoldedCategories(string) error        { return nil }

func (r *recordingRepo) GetCursor(projectID string) Cursor {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cursors[projectID]
}

func (r *recordingRepo) DeleteCursor(projectID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.cursors, projectID)
	return nil
}

var _ StateRepository = (*recordingRepo)(nil)

func task(id string) Cursor {
	return Cursor{Kind: "task", CategoryID: "c1", TaskID: id}
}

// Close is the exit path: whatever the last interval still holds has to reach
// disk before the process goes away.
func TestCursorSaver_CloseFlushesPending(t *testing.T) {
	repo := newRecordingRepo()
	// A window long enough that the ticker cannot fire on its own during the test.
	s := newCursorSaver(repo, time.Hour)

	s.Set("p1", task("t3"))
	s.Close()

	cursors, _ := repo.snapshot()
	assert.Equal(t, task("t3"), cursors["p1"])
}

// Holding j must not put an fsync on every row: the moves inside one window
// collapse into a single write of the final position.
func TestCursorSaver_CoalescesRapidMoves(t *testing.T) {
	repo := newRecordingRepo()
	s := newCursorSaver(repo, time.Hour)

	for _, id := range []string{"t1", "t2", "t3", "t4"} {
		s.Set("p1", task(id))
	}
	s.Close()

	cursors, writes := repo.snapshot()
	assert.Equal(t, task("t4"), cursors["p1"])
	assert.Equal(t, 1, writes, "a burst of movement should cost one write")
}

// The whole point of the background saver: the row reaches disk while the
// session is still running, so a kill -9 finds it there.
func TestCursorSaver_WritesWithoutClose(t *testing.T) {
	repo := newRecordingRepo()
	s := newCursorSaver(repo, time.Millisecond)
	defer s.Close()

	s.Set("p1", task("t7"))

	assert.Eventually(t, func() bool {
		cursors, _ := repo.snapshot()
		return cursors["p1"] == task("t7")
	}, time.Second, 5*time.Millisecond)
}

// Cursors are per project, so a queued write for one must not stand in for
// another.
func TestCursorSaver_KeepsProjectsSeparate(t *testing.T) {
	repo := newRecordingRepo()
	s := newCursorSaver(repo, time.Hour)

	s.Set("p1", task("a1"))
	s.Set("p2", task("b1"))
	s.Close()

	cursors, _ := repo.snapshot()
	assert.Equal(t, task("a1"), cursors["p1"])
	assert.Equal(t, task("b1"), cursors["p2"])
}

// A zero cursor is a real value — a project left with nothing selected clears
// its entry rather than keeping a row that no longer means anything.
func TestCursorSaver_ZeroCursorClearsEntry(t *testing.T) {
	repo := newRecordingRepo()
	repo.cursors["p1"] = task("t1")
	s := newCursorSaver(repo, time.Hour)

	s.Set("p1", Cursor{})
	s.Close()

	cursors, _ := repo.snapshot()
	assert.NotContains(t, cursors, "p1")
}

// Deleting a project removes its stored cursor; a write still sitting in the
// queue would otherwise put it straight back.
func TestCursorSaver_DropDiscardsQueuedWrite(t *testing.T) {
	repo := newRecordingRepo()
	s := newCursorSaver(repo, time.Hour)

	s.Set("p1", task("t1"))
	s.Set("p2", task("t2"))
	s.Drop("p1")
	s.Close()

	cursors, _ := repo.snapshot()
	assert.NotContains(t, cursors, "p1")
	assert.Equal(t, task("t2"), cursors["p2"], "dropping one project must not affect another")
}
