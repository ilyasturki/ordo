package web

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/ilyasturki/ordo/internal/domain"
)

// hub tells each listener which project file changed; a slow listener only keeps the latest.
type hub struct {
	mu   sync.Mutex
	subs map[chan string]struct{}
}

func newHub() *hub { return &hub{subs: make(map[chan string]struct{})} }

func (h *hub) subscribe() (chan string, func()) {
	ch := make(chan string, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

func (h *hub) publish(pid string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case <-ch:
		default:
		}
		ch <- pid
	}
}

// events streams the ID of each project file that changes, so an open app syncs at once.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		apiError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	ch, done := s.hub.subscribe()
	defer done()
	fmt.Fprint(w, "retry: 3000\n\n")
	flusher.Flush()
	beat := time.NewTicker(25 * time.Second)
	defer beat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case pid := <-ch:
			fmt.Fprintf(w, "event: change\ndata: %s\n\n", pid)
		case <-beat.C:
			fmt.Fprint(w, ": beat\n\n")
		}
		flusher.Flush()
	}
}

// Watch publishes each project file that Syncthing, the TUI or a request writes or deletes.
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
			if !isJSON || domain.ValidateID(id) != nil || ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
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
				s.hub.publish(id)
			}
			clear(pending)
		}
	}
}
