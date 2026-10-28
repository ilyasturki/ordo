package web

import (
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/data"
)

const (
	deviceCookie = "ordo_device"
	cookieAge    = 400 * 24 * time.Hour
)

type Server struct {
	store   *data.Store
	images  *attachments.Store
	auth    *Auth
	log     *log.Logger
	hub     *hub
	limiter *limiter
	csrf    *http.CrossOriginProtection
	app     *app

	globalLimit *limiter
}

func New(store *data.Store, images *attachments.Store, auth *Auth, logger *log.Logger) (*Server, error) {
	a, err := loadApp(bundle)
	if err != nil {
		return nil, err
	}
	return &Server{
		store:   store,
		images:  images,
		auth:    auth,
		log:     logger,
		hub:     newHub(),
		limiter: newLimiter(5, 15*time.Minute),
		// Across every client, so a botnet spreading guesses still meets a ceiling.
		globalLimit: newLimiter(50, 15*time.Minute),
		csrf:        http.NewCrossOriginProtection(),
		app:         a,
	}, nil
}

func (s *Server) Handler() http.Handler {
	root := http.NewServeMux()
	root.Handle("/api/", s.api())
	root.Handle("/", s.app)
	return secureHeaders(s.app.scripts, root)
}

func secureHeaders(scripts string, h http.Handler) http.Handler {
	csp := "default-src 'self'; img-src 'self' blob: data:; style-src 'self' 'unsafe-inline'; " +
		"script-src 'self' 'wasm-unsafe-eval'" + scripts + "; connect-src 'self'; object-src 'none'; " +
		"base-uri 'none'; form-action 'self'; frame-ancestors 'none'"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hd := w.Header()
		hd.Set("Content-Security-Policy", csp)
		hd.Set("X-Content-Type-Options", "nosniff")
		hd.Set("Referrer-Policy", "no-referrer")
		hd.Set("Permissions-Policy", "camera=(self), geolocation=(), microphone=()")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			hd.Set("Strict-Transport-Security", "max-age=31536000")
		}
		h.ServeHTTP(w, r)
	})
}

type ctxKey struct{}

func deviceFrom(r *http.Request) Device {
	d, _ := r.Context().Value(ctxKey{}).(Device)
	return d
}

func withDevice(r *http.Request, dev Device) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxKey{}, dev))
}

func secureRequest(r *http.Request) bool {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	return host == "localhost" || strings.HasSuffix(host, ".localhost")
}

func (s *Server) setCookie(w http.ResponseWriter, r *http.Request, name, value string, age time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(age.Seconds()),
		HttpOnly: true,
		Secure:   secureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Server) image(w http.ResponseWriter, r *http.Request) {
	target := attachments.DirName + "/" + r.PathValue("name")
	if !attachments.Internal(target) {
		http.NotFound(w, r)
		return
	}
	p, ok := s.images.Path(target)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	http.ServeFile(w, r, p)
}
