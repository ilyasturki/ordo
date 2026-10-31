package web

import (
	"context"
	"io"
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

	globalLimit *limiter
}

func New(store *data.Store, images *attachments.Store, auth *Auth, logger *log.Logger) *Server {
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
	}
}

func (s *Server) Handler() http.Handler {
	root := http.NewServeMux()
	root.Handle("/api/", s.api())
	root.HandleFunc("/", notAPI)
	return secureHeaders(root)
}

const (
	aboutAPI  = "This is ordo's sync API. Pair a device with `ordo serve pair`, then scan its code or paste its link in the Ordo app.\n"
	aboutPair = "This is an ordo pairing link. Scan its code or paste this address in the Ordo app.\n"
)

func notAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	about := aboutAPI
	if strings.HasPrefix(r.URL.Path, "/pair/") {
		about = aboutPair
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = io.WriteString(w, about)
}

func secureHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hd := w.Header()
		hd.Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		hd.Set("X-Content-Type-Options", "nosniff")
		hd.Set("Referrer-Policy", "no-referrer")
		hd.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
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
