package web

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ilyasturki/ordo/internal/domain"
)

type hub struct {
	mu   sync.Mutex
	subs map[string]map[chan string]struct{}
}

func newHub() *hub { return &hub{subs: make(map[string]map[chan string]struct{})} }

func (h *hub) subscribe(pid string) (chan string, func()) {
	ch := make(chan string, 1)
	h.mu.Lock()
	if h.subs[pid] == nil {
		h.subs[pid] = make(map[chan string]struct{})
	}
	h.subs[pid][ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs[pid], ch)
		h.mu.Unlock()
	}
}

func (h *hub) publish(pid, stamp string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[pid] {
		select {
		case <-ch:
		default:
		}
		ch <- stamp
	}
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	if domain.ValidateID(pid) != nil {
		http.NotFound(w, r)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, done := s.hub.subscribe(pid)
	defer done()
	fmt.Fprint(w, "retry: 3000\n\n")
	if v := s.version(pid); v != "" {
		fmt.Fprintf(w, "event: change\ndata: %s\n\n", v)
	}
	flusher.Flush()
	beat := time.NewTicker(25 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case stamp := <-ch:
			fmt.Fprintf(w, "event: change\ndata: %s\n\n", stamp)
		case <-beat.C:
			fmt.Fprint(w, ": beat\n\n")
		}
		flusher.Flush()
	}
}

// Watch publishes each project file's new stamp as Syncthing, the TUI or a request rewrites it.
func (s *Server) Watch(ctx context.Context) error {
	dir, err := filepath.EvalSymlinks(s.store.Dir)
	if err != nil {
		return err
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()
	if err := w.Add(dir); err != nil {
		return err
	}
	pending := make(map[string]bool)
	tick := time.NewTimer(time.Hour)
	tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-w.Events:
			if !ok {
				return nil
			}
			id, isJSON := strings.CutSuffix(filepath.Base(ev.Name), ".json")
			if !isJSON || domain.ValidateID(id) != nil || ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			pending[id] = true
			tick.Reset(150 * time.Millisecond)
		case err, ok := <-w.Errors:
			if !ok {
				return nil
			}
			s.log.Printf("watch: %v", err)
		case <-tick.C:
			for id := range pending {
				if v := s.version(id); v != "" {
					s.hub.publish(id, v)
				}
			}
			clear(pending)
		}
	}
}

type undoEntry struct {
	project string
	before  domain.Project
	after   domain.Project
	at      time.Time
}

type undoStash struct {
	mu      sync.Mutex
	entries map[string]undoEntry
}

const undoTTL = 10 * time.Minute

func newUndoStash() *undoStash { return &undoStash{entries: make(map[string]undoEntry)} }

func (u *undoStash) put(pid string, before, after domain.Project) string {
	u.mu.Lock()
	defer u.mu.Unlock()
	now := time.Now()
	for k, e := range u.entries {
		if now.Sub(e.at) > undoTTL {
			delete(u.entries, k)
		}
	}
	key := rand.Text()[:16]
	u.entries[key] = undoEntry{project: pid, before: before, after: cloneProject(after), at: now}
	return key
}

func (u *undoStash) take(key string) (undoEntry, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	e, ok := u.entries[key]
	delete(u.entries, key)
	return e, ok && time.Since(e.at) <= undoTTL
}
