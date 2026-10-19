package web

import (
	"encoding/base64"
	"errors"
	"html/template"
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

// pair answers the pair page, or the login page's pairing form when it sends from=login.
func (s *Server) pair(w http.ResponseWriter, r *http.Request) {
	failed := func(status int, msg string) {
		if r.FormValue("from") == "login" {
			s.render(w, r, status, "login", page{Title: "Sign in", Body: "auth",
				Data: authPage{Name: deviceName(r), HasPass: s.auth.HasPassphrase(), Error: msg}})
			return
		}
		s.render(w, r, status, "pair", page{Title: "Pair this device", Body: "auth",
			Data: authPage{Error: msg, HasPass: s.auth.HasPassphrase()}})
	}
	if s.limiter.blocked("pair:" + clientIP(r)) {
		failed(http.StatusTooManyRequests, "Too many attempts. Wait a few minutes.")
		return
	}
	cookie, err := s.auth.Pair(r.FormValue("token"), deviceName(r))
	if err != nil {
		s.limiter.fail("pair:" + clientIP(r))
		msg := "Pairing failed"
		if errors.Is(err, ErrBadPairing) {
			msg = "This pairing code is used or expired. Get a new one from Devices on a signed-in device, or run ordo serve pair."
		} else {
			s.log.Printf("pair: %v", err)
		}
		failed(http.StatusForbidden, msg)
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
			view.Error = "No passphrase is set. Pair with a code instead."
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

type deviceView struct {
	Device
	From string
	Also string
}

type devicesPage struct {
	Devices []deviceView
	Also    string
	HasPass bool
}

func (s *Server) devices(w http.ResponseWriter, r *http.Request) {
	devs, err := s.auth.Devices()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	names := make(map[string]string, len(devs))
	for _, d := range devs {
		names[d.ID] = d.Name
	}
	view := devicesPage{HasPass: s.auth.HasPassphrase()}
	for _, d := range devs {
		var also []string
		for _, c := range lineage(devs, d.ID)[1:] {
			also = append(also, c.Name)
		}
		v := deviceView{Device: d, From: names[d.PairedBy], Also: strings.Join(also, ", ")}
		if d.ID == deviceFrom(r).ID {
			view.Also = v.Also
		}
		view.Devices = append(view.Devices, v)
	}
	s.render(w, r, http.StatusOK, "devices", page{Title: "Devices", Body: "devices", Data: view})
}

type pairCodePage struct {
	HasPass bool
	Error   string
	Link    string
	QR      template.URL
	Minutes int
}

func (s *Server) pairCodePage(w http.ResponseWriter, r *http.Request) {
	s.render(w, r, http.StatusOK, "devices_pair", page{Title: "Pair a new device", Body: "devices",
		Data: pairCodePage{HasPass: s.auth.HasPassphrase()}})
}

// pairCode mints a pairing link for another device, once the passphrase is entered again when one is set.
func (s *Server) pairCode(w http.ResponseWriter, r *http.Request) {
	view := pairCodePage{HasPass: s.auth.HasPassphrase(), Minutes: int(PairTTL.Minutes())}
	refuse := func(status int, msg string) {
		view.Error = msg
		s.render(w, r, status, "devices_pair", page{Title: "Pair a new device", Body: "devices", Data: view})
	}
	if view.HasPass {
		if s.limiter.blocked("login:"+clientIP(r)) || s.globalLimit.blocked("login") {
			refuse(http.StatusTooManyRequests, "Too many attempts. Wait a few minutes.")
			return
		}
		if err := s.auth.CheckPassphrase(r.FormValue("passphrase")); err != nil {
			if !errors.Is(err, ErrBadPassphrase) {
				s.fail(w, r, err)
				return
			}
			s.limiter.fail("login:" + clientIP(r))
			s.globalLimit.fail("login")
			refuse(http.StatusForbidden, "Wrong passphrase")
			return
		}
	}
	token, err := s.auth.NewPairingBy(deviceFrom(r).ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	view.Link = s.linkBase(r) + "/pair/" + token
	code, err := qr.Encode(view.Link, qr.M)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	code.Scale = 8
	view.QR = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(code.PNG()))
	s.render(w, r, http.StatusOK, "devices_pair", page{Title: "Pair a new device", Body: "devices", Data: view})
}

// linkBase is the address a new device opens: the one serve remembers, else the one this browser uses.
func (s *Server) linkBase(r *http.Request) string {
	if base := s.auth.BaseURL(); base != "" {
		return base
	}
	if origin := r.Header.Get("Origin"); origin != "" && origin != "null" {
		return origin
	}
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
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
