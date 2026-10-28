package web

import (
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"rsc.io/qr"
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

// signIn pairs a new device on the passphrase; a refusal has its status and message for the user.
func (s *Server) signIn(r *http.Request, phrase, name string) (token string, status int, msg string) {
	if s.limiter.blocked("login:"+clientIP(r)) || s.globalLimit.blocked("login") {
		return "", http.StatusTooManyRequests, "Too many attempts. Wait a few minutes."
	}
	token, err := s.auth.Login(phrase, name)
	switch {
	case err == nil:
		return token, 0, ""
	case errors.Is(err, ErrBadPassphrase):
		s.limiter.fail("login:" + clientIP(r))
		s.globalLimit.fail("login")
		return "", http.StatusForbidden, "Wrong passphrase"
	case errors.Is(err, ErrNoPassphrase):
		return "", http.StatusForbidden, "No passphrase is set. Pair with a code instead."
	}
	s.log.Printf("login: %v", err)
	return "", http.StatusForbidden, "Sign-in failed"
}

// recheck asks for the passphrase again when one is set, under the sign-in limits; a refusal has
// its status and message for the user.
func (s *Server) recheck(r *http.Request, phrase string) (status int, msg string, err error) {
	if !s.auth.HasPassphrase() {
		return 0, "", nil
	}
	if s.limiter.blocked("login:"+clientIP(r)) || s.globalLimit.blocked("login") {
		return http.StatusTooManyRequests, "Too many attempts. Wait a few minutes.", nil
	}
	if err := s.auth.CheckPassphrase(phrase); err != nil {
		if !errors.Is(err, ErrBadPassphrase) {
			return 0, "", err
		}
		s.limiter.fail("login:" + clientIP(r))
		s.globalLimit.fail("login")
		return http.StatusForbidden, "Wrong passphrase", nil
	}
	return 0, "", nil
}

// pairLink mints a pairing link on behalf of the requesting device, and its QR code as a PNG data URL.
func (s *Server) pairLink(r *http.Request) (link, png string, err error) {
	token, err := s.auth.NewPairingBy(deviceFrom(r).ID)
	if err != nil {
		return "", "", err
	}
	link = s.linkBase(r) + "/pair/" + token
	code, err := qr.Encode(link, qr.M)
	if err != nil {
		return "", "", err
	}
	code.Scale = 8
	return link, "data:image/png;base64," + base64.StdEncoding.EncodeToString(code.PNG()), nil
}

// linkBase is the address a new device opens: the one serve remembers, else the one this request reached.
func (s *Server) linkBase(r *http.Request) string {
	if base := s.auth.BaseURL(); base != "" {
		return base
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}
