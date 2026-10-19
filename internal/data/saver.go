package data

import (
	"errors"
	"sync"

	"github.com/ilyasturki/ordo/internal/domain"
)

// saveJob is an immutable snapshot queued for the background writer.
type saveJob struct {
	id   string
	data []byte
	copy *lineage
	seq  int
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
	pending sync.WaitGroup

	mu    sync.Mutex
	names map[string]string
}

// NewSaver starts the background writer. Call Close to stop it and flush.
func NewSaver(store *Store) *Saver {
	s := &Saver{
		store:   store,
		jobs:    make(chan saveJob, 1),
		results: make(chan error, 1),
		done:    make(chan struct{}),
		names:   make(map[string]string),
	}
	go s.run()
	return s
}

func (s *Saver) run() {
	defer close(s.done)
	defer close(s.results)
	for job := range s.jobs {
		err := s.store.writeSnapshot(job.id, job.data, job.copy)
		if s.store.settle(job.copy, job.seq, err) {
			s.report(err)
		}
		s.pending.Done()
	}
}

// report surfaces an outcome best-effort. If one is already queued unread,
// keep it rather than block the writer: the reader asks the store for the
// current state anyway.
func (s *Saver) report(err error) {
	select {
	case s.results <- err:
	default:
	}
}

// Enqueue marshals a snapshot on the caller's goroutine, then schedules the
// write. If a write is already queued but not yet started it is replaced — only
// the newest project state needs to reach disk. The snapshot belongs to the
// copy of the project loaded last, so a reload after it can't let it through.
// Must not be called after Close, nor concurrently with Drain.
func (s *Saver) Enqueue(project domain.Project) {
	l, seq := s.store.queue(project.ID)
	s.mu.Lock()
	s.names[project.ID] = project.Name
	s.mu.Unlock()
	data, err := s.store.marshalProject(project)
	if err != nil {
		s.store.settle(l, seq, err)
		s.report(err)
		return
	}
	job := saveJob{id: project.ID, data: data, copy: l, seq: seq}
	s.pending.Add(1)
	for {
		select {
		case s.jobs <- job:
			return
		case <-s.jobs:
			// Queue full with a stale snapshot: drop it and retry with the newer
			// one, so only the latest project state reaches disk. A concurrent
			// pull by the worker just means the stale write happens and is
			// immediately superseded by this newer one.
			s.pending.Done()
		}
	}
}

// Results delivers the error of each failed background write, and nil for the
// first one that succeeds after a failure. The channel closes when the worker
// stops. Read it as a subscription.
func (s *Saver) Results() <-chan error { return s.results }

// Drain blocks until every queued write has landed or failed.
func (s *Saver) Drain() { s.pending.Wait() }

// Close stops accepting work and blocks until the worker has drained the last
// queued write, guaranteeing the final edit is on disk before the caller exits.
// It returns an *UnsavedError for each project whose open copy holds edits
// that never reached disk.
func (s *Saver) Close() error {
	close(s.jobs)
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	var errs []error
	for id, name := range s.names {
		if n, err := s.store.Unsaved(id); err != nil {
			errs = append(errs, &UnsavedError{Name: name, Edits: n, Err: err})
		}
	}
	return errors.Join(errs...)
}
