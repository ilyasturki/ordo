package web

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string][]time.Time
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{max: max, window: window, hits: make(map[string][]time.Time)}
}

func (l *limiter) recent(key string, now time.Time) []time.Time {
	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if now.Sub(t) < l.window {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.hits, key)
		return nil
	}
	l.hits[key] = kept
	return kept
}

func (l *limiter) blocked(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(key, time.Now())) >= l.max
}

// fail counts a failed attempt against key, dropping every key whose window has emptied.
func (l *limiter) fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for k := range l.hits {
		l.recent(k, now)
	}
	l.hits[key] = append(l.recent(key, now), now)
}

// clientIP trusts X-Forwarded-For only from a proxy on this machine.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if host == "" || host == "@" || (ip != nil && ip.IsLoopback()) {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			parts := strings.Split(fwd, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	return host
}

func deviceName(r *http.Request) string {
	if name := strings.TrimSpace(r.FormValue("name")); name != "" {
		return truncate(name, 60)
	}
	ua := r.UserAgent()
	browser := "Browser"
	for _, b := range [][2]string{{"Firefox/", "Firefox"}, {"Edg/", "Edge"}, {"Chrome/", "Chrome"}, {"Safari/", "Safari"}} {
		if strings.Contains(ua, b[0]) {
			browser = b[1]
			break
		}
	}
	for _, os := range []string{"Android", "iPhone", "iPad", "Linux", "Mac OS X", "Windows"} {
		if strings.Contains(ua, os) {
			return browser + " on " + strings.TrimSuffix(os, " OS X")
		}
	}
	return browser
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

type authPage struct {
	Token   string
	Name    string
	Error   string
	HasPass bool
}

func (s *Server) pairPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "pair", page{Title: "Pair this device", Body: "auth",
		Data: authPage{Token: r.PathValue("token"), Name: deviceName(r)}})
}

func (s *Server) pair(w http.ResponseWriter, r *http.Request) {
	if s.limiter.blocked("pair:" + clientIP(r)) {
		s.render(w, r, http.StatusTooManyRequests, "pair", page{Title: "Pair this device", Body: "auth",
			Data: authPage{Error: "Too many attempts. Wait a few minutes."}})
		return
	}
	cookie, err := s.auth.Pair(r.FormValue("token"), deviceName(r))
	if err != nil {
		s.limiter.fail("pair:" + clientIP(r))
		msg := "Pairing failed"
		if errors.Is(err, ErrBadPairing) {
			msg = "This pairing link is used or expired. Run ordo serve pair for a new one."
		} else {
			s.log.Printf("pair: %v", err)
		}
		s.render(w, r, http.StatusForbidden, "pair", page{Title: "Pair this device", Body: "auth",
			Data: authPage{Error: msg, HasPass: s.auth.HasPassphrase()}})
		return
	}
	s.setCookie(w, r, deviceCookie, cookie, cookieAge)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "login", page{Title: "Sign in", Body: "auth",
		Data: authPage{Name: deviceName(r), HasPass: s.auth.HasPassphrase()}})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	view := authPage{Name: deviceName(r), HasPass: s.auth.HasPassphrase()}
	if s.limiter.blocked("login:"+clientIP(r)) || s.globalLimit.blocked("login") {
		view.Error = "Too many attempts. Wait a few minutes."
		s.render(w, r, http.StatusTooManyRequests, "login", page{Title: "Sign in", Body: "auth", Data: view})
		return
	}
	cookie, err := s.auth.Login(r.FormValue("passphrase"), deviceName(r))
	if err != nil {
		if errors.Is(err, ErrBadPassphrase) {
			s.limiter.fail("login:" + clientIP(r))
			s.globalLimit.fail("login")
		}
		switch {
		case errors.Is(err, ErrBadPassphrase):
			view.Error = "Wrong passphrase"
		case errors.Is(err, ErrNoPassphrase):
			view.Error = "No passphrase is set. Pair with ordo serve pair."
		default:
			s.log.Printf("login: %v", err)
			view.Error = "Sign-in failed"
		}
		s.render(w, r, http.StatusForbidden, "login", page{Title: "Sign in", Body: "auth", Data: view})
		return
	}
	s.setCookie(w, r, deviceCookie, cookie, cookieAge)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if _, err := s.auth.Revoke(deviceFrom(r).ID); err != nil {
		s.fail(w, r, err)
		return
	}
	s.setCookie(w, r, deviceCookie, "", -time.Second)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) devices(w http.ResponseWriter, r *http.Request) {
	devs, err := s.auth.Devices()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "devices", page{Title: "Devices", Body: "devices", Data: devs})
}

func (s *Server) revoke(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := s.auth.Revoke(id); err != nil {
		s.fail(w, r, err)
		return
	}
	if id == deviceFrom(r).ID {
		s.setCookie(w, r, deviceCookie, "", -time.Second)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/devices", http.StatusSeeOther)
}
