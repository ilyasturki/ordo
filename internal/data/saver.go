package data

import "github.com/ilyasturki/phasionary/internal/domain"

// saveJob is an immutable snapshot queued for the background writer.
type saveJob struct {
	id   string
	data []byte
}

// Saver persists projects asynchronously so the UI never blocks on fsync. A
// single worker goroutine serializes writes in order; rapid edits coalesce
// because only the most recent pending snapshot is kept (intermediate states
// never reach disk). Callers Enqueue on their own goroutine — Enqueue marshals
// the snapshot there, so the bytes handed to the worker are immutable and free
// of data races against further edits. Close drains the final write, so the
// last edit is always durably persisted before exit.
type Saver struct {
	store   *Store
	jobs    chan saveJob // capacity 1, latest-wins
	results chan error   // capacity 1, best-effort error surface
	done    chan struct{}
}

// NewSaver starts the background writer. Call Close to stop it and flush.
func NewSaver(store *Store) *Saver {
	s := &Saver{
		store:   store,
		jobs:    make(chan saveJob, 1),
		results: make(chan error, 1),
		done:    make(chan struct{}),
	}
	go s.run()
	return s
}

func (s *Saver) run() {
	defer close(s.done)
	defer close(s.results)
	for job := range s.jobs {
		if err := s.store.WriteProjectLocked(job.id, job.data); err != nil {
			// Surface the error best-effort. If one is already queued unread,
			// keep it rather than block the writer.
			select {
			case s.results <- err:
			default:
			}
		}
	}
}

// Enqueue marshals a snapshot on the caller's goroutine, then schedules the
// write. If a write is already queued but not yet started it is replaced — only
// the newest project state needs to reach disk. Must not be called after Close.
func (s *Saver) Enqueue(project domain.Project) {
	data, err := s.store.marshalProject(project)
	if err != nil {
		select {
		case s.results <- err:
		default:
		}
		return
	}
	job := saveJob{id: project.ID, data: data}
	for {
		select {
		case s.jobs <- job:
			return
		case <-s.jobs:
			// Queue full with a stale snapshot: drop it and retry with the newer
			// one, so only the latest project state reaches disk. A concurrent
			// pull by the worker just means the stale write happens and is
			// immediately superseded by this newer one.
		}
	}
}

// Results delivers errors from background writes (nil channel semantics never
// occur; the channel closes when the worker stops). Read it as a subscription.
func (s *Saver) Results() <-chan error { return s.results }

// Close stops accepting work and blocks until the worker has drained the last
// queued write, guaranteeing the final edit is on disk before the caller exits.
func (s *Saver) Close() {
	close(s.jobs)
	<-s.done
}
