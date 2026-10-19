package web

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/version"
)

//go:embed templates static
var assets embed.FS

const (
	deviceCookie = "ordo_device"
	foldCookie   = "ordo_fold"
	lastCookie   = "ordo_last"
	cookieAge    = 400 * 24 * time.Hour
)

type Server struct {
	store   *data.Store
	images  *attachments.Store
	auth    *Auth
	log     *log.Logger
	pages   map[string]*template.Template
	assetV  string
	hub     *hub
	limiter *limiter
	undo    *undoStash

	globalLimit *limiter
}

func New(store *data.Store, images *attachments.Store, auth *Auth, logger *log.Logger) (*Server, error) {
	s := &Server{
		store:   store,
		images:  images,
		auth:    auth,
		log:     logger,
		hub:     newHub(),
		limiter: newLimiter(5, 15*time.Minute),
		// Across every client, so a botnet spreading guesses still meets a ceiling.
		globalLimit: newLimiter(50, 15*time.Minute),
		undo:        newUndoStash(),
	}
	if err := s.hashAssets(); err != nil {
		return nil, err
	}
	pages, err := parsePages(s.funcs())
	if err != nil {
		return nil, err
	}
	s.pages = pages
	return s, nil
}

func (s *Server) hashAssets() error {
	h := sha256.New()
	err := fs.WalkDir(assets, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := assets.ReadFile(p)
		h.Write(b)
		return err
	})
	s.assetV = hex.EncodeToString(h.Sum(nil))[:12]
	return err
}

func parsePages(funcs template.FuncMap) (map[string]*template.Template, error) {
	names, err := fs.Glob(assets, "templates/page_*.html")
	if err != nil {
		return nil, err
	}
	pages := make(map[string]*template.Template, len(names))
	for _, name := range names {
		t, err := template.New("layout.html").Funcs(funcs).ParseFS(assets, "templates/layout.html", "templates/parts.html", name)
		if err != nil {
			return nil, err
		}
		pages[strings.TrimSuffix(strings.TrimPrefix(path.Base(name), "page_"), ".html")] = t
	}
	return pages, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets, "static")
	files := http.FileServerFS(static)
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheForever(files)))
	mux.HandleFunc("GET /manifest.webmanifest", s.manifest)
	mux.HandleFunc("GET /pair/{token}", s.pairPage)
	mux.HandleFunc("POST /pair", s.pair)
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)

	app := http.NewServeMux()
	app.HandleFunc("GET /{$}", s.home)
	app.HandleFunc("GET /projects", s.projects)
	app.HandleFunc("GET /p/{pid}", s.project)
	app.HandleFunc("GET /p/{pid}/t/{tid}", s.task)
	app.HandleFunc("GET /p/{pid}/events", s.events)
	app.HandleFunc("GET /p/{pid}/img/{name}", s.image)
	app.HandleFunc("POST /p/{pid}/do/{action}", s.do)
	app.HandleFunc("POST /p/{pid}/undo/{key}", s.undoAction)
	app.HandleFunc("POST /fold/{cid}", s.fold)
	app.HandleFunc("GET /devices", s.devices)
	app.HandleFunc("GET /devices/pair", s.pairCodePage)
	app.HandleFunc("POST /devices/pair", s.pairCode)
	app.HandleFunc("POST /devices/{id}/revoke", s.revoke)
	app.HandleFunc("POST /logout", s.logout)
	mux.Handle("/", s.requireDevice(app))

	csrf := http.NewCrossOriginProtection()
	return secureHeaders(csrf.Handler(mux))
}

func cacheForever(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		h.ServeHTTP(w, r)
	})
}

func secureHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hd := w.Header()
		hd.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' blob: data:; style-src 'self'; script-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
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

func (s *Server) requireDevice(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(deviceCookie)
		var (
			dev   Device
			renew bool
			ok    bool
		)
		if err == nil {
			dev, renew, ok = s.auth.Check(c.Value)
		}
		if !ok {
			if r.Method == http.MethodGet && r.Header.Get("HX-Request") == "" {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			w.Header().Set("HX-Redirect", "/login")
			http.Error(w, "sign in again", http.StatusUnauthorized)
			return
		}
		if renew {
			s.setCookie(w, r, deviceCookie, c.Value, cookieAge)
		}
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, dev)))
	})
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
		HttpOnly: name == deviceCookie,
		Secure:   secureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
}

type page struct {
	Title   string
	Body    string
	Device  Device
	Version string
	Data    any
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, name string, p page) {
	t, ok := s.pages[name]
	if !ok {
		panic("web: no page " + name)
	}
	p.Device = deviceFrom(r)
	p.Version = version.Version
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Vary", "HX-Request")
	w.WriteHeader(status)
	if err := t.Execute(w, p); err != nil {
		s.log.Printf("render %s: %v", name, err)
	}
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	msg := "Something went wrong"
	switch {
	case errors.Is(err, data.ErrProjectNotFound):
		status, msg = http.StatusNotFound, "That project is gone"
	case errors.Is(err, domain.ErrTaskNotFound):
		status, msg = http.StatusNotFound, "That task is gone"
	default:
		s.log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	}
	s.render(w, r, status, "error", page{Title: msg, Data: msg})
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(lastCookie); err == nil && domain.ValidateID(c.Value) == nil {
		if _, err := s.store.LoadProjectByID(c.Value); err == nil {
			http.Redirect(w, r, "/p/"+c.Value, http.StatusSeeOther)
			return
		}
	}
	projects, err := s.store.ListProjects()
	if err == nil && len(projects) == 1 {
		http.Redirect(w, r, "/p/"+projects[0].ID, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

type projectSummary struct {
	ID      string
	Name    string
	Initial string
	Counts  countsView
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	projects, err := s.store.ListProjects()
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, http.StatusOK, "projects", page{Title: "Projects", Body: "projects", Data: summaries(projects)})
}

func summaries(projects []domain.Project) []projectSummary {
	list := make([]projectSummary, 0, len(projects))
	for _, p := range projects {
		sum := projectSummary{ID: p.ID, Name: p.Name, Counts: projectCounts(&p)}
		if r := []rune(strings.TrimSpace(p.Name)); len(r) > 0 {
			sum.Initial = strings.ToUpper(string(r[0]))
		}
		list = append(list, sum)
	}
	return list
}

type projectPage struct {
	P        *domain.Project
	Projects []projectSummary
	Cats     []catView
	Stamp    string
	Targets  []catOption
}

type catOption struct {
	ID   string
	Name string
	Sub  bool
}

func categoryOptions(p *domain.Project) []catOption {
	opts := make([]catOption, len(p.Categories))
	for i, c := range p.Categories {
		opts[i] = catOption{ID: c.ID, Name: p.CategoryPath(i), Sub: c.ParentID != ""}
	}
	return opts
}

func foldedSet(r *http.Request) map[string]bool {
	set := make(map[string]bool)
	if c, err := r.Cookie(foldCookie); err == nil {
		for _, id := range strings.Split(c.Value, ".") {
			if id != "" {
				set[id] = true
			}
		}
	}
	return set
}

func (s *Server) project(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.LoadProjectByID(r.PathValue("pid"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.setCookie(w, r, lastCookie, p.ID, cookieAge)
	view := projectPage{
		P:       &p,
		Cats:    buildCategories(&p, foldedSet(r)),
		Stamp:   s.version(p.ID),
		Targets: categoryOptions(&p),
	}
	if all, err := s.store.ListProjects(); err == nil {
		view.Projects = summaries(all)
	}
	s.render(w, r, http.StatusOK, "project", page{Title: p.Name, Body: "project", Data: view})
}

type taskPage struct {
	P          *domain.Project
	Cat        domain.Category
	CatPath    string
	Row        rowView
	Parent     *domain.Task
	Text       string
	Images     []imageView
	ImageLines []string
	Lossless   bool
	Targets    []catOption
	Stamp      string
	Back       string
}

func (s *Server) task(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.LoadProjectByID(r.PathValue("pid"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	ci, ti, err := locate(&p, r.PathValue("tid"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	cat := p.Categories[ci]
	t := cat.Tasks[ti]
	if t.IsSeparator() {
		http.Redirect(w, r, "/p/"+p.ID, http.StatusSeeOther)
		return
	}
	row := newRow(t)
	row.ChildDone, row.ChildTotal = cat.ChildProgress(ti)
	for _, c := range cat.Tasks[ti+1 : domain.BlockEnd(cat.Tasks, ti)] {
		row.Children = append(row.Children, newRow(c))
	}
	view := taskPage{
		P:       &p,
		Cat:     cat,
		CatPath: p.CategoryPath(ci),
		Row:     row,
		Targets: categoryOptions(&p),
		Stamp:   s.version(p.ID),
		Back:    "/p/" + p.ID + "#t-" + t.ID,
	}
	if pi := domain.ParentIndex(cat.Tasks, ti); pi >= 0 {
		parent := cat.Tasks[pi]
		view.Parent = &parent
	}
	view.Text, view.ImageLines, view.Lossless = descriptionFields(t.Description)
	view.Images = imagesOf(strings.Join(view.ImageLines, "\n"))
	s.render(w, r, http.StatusOK, "task", page{Title: t.Title, Body: "task", Data: view})
}

// version fingerprints the project file, so a page tells its own save from another writer's.
func (s *Server) version(pid string) string {
	if domain.ValidateID(pid) != nil {
		return ""
	}
	raw, err := os.ReadFile(filepath.Join(s.store.Dir, pid+".json"))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:8])
}

func locate(p *domain.Project, id string) (int, int, error) {
	for ci := range p.Categories {
		for ti, t := range p.Categories[ci].Tasks {
			if t.ID == id {
				return ci, ti, nil
			}
		}
	}
	return -1, -1, domain.ErrTaskNotFound
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

func (s *Server) fold(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("cid")
	if domain.ValidateID(id) != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	set := foldedSet(r)
	if set[id] {
		delete(set, id)
	} else {
		set[id] = true
	}
	ids := make([]string, 0, len(set))
	for k := range set {
		ids = append(ids, k)
	}
	slices.Sort(ids)
	s.setCookie(w, r, foldCookie, strings.Join(ids, "."), cookieAge)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) manifest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/manifest+json")
	fmt.Fprintf(w, `{
  "name": "Ordo",
  "short_name": "Ordo",
  "start_url": "/",
  "scope": "/",
  "display": "standalone",
  "background_color": "#121318",
  "theme_color": "#121318",
  "icons": [
    {"src": "/static/icon.svg?v=%[1]s", "sizes": "any", "type": "image/svg+xml", "purpose": "any"},
    {"src": "/static/icon-192.png?v=%[1]s", "sizes": "192x192", "type": "image/png", "purpose": "any"},
    {"src": "/static/icon-512.png?v=%[1]s", "sizes": "512x512", "type": "image/png", "purpose": "any"},
    {"src": "/static/icon-maskable.png?v=%[1]s", "sizes": "512x512", "type": "image/png", "purpose": "maskable"}
  ]
}
`, s.assetV)
}

func backTo(r *http.Request, pid string) string {
	back := r.FormValue("back")
	u, err := url.Parse(back)
	if back == "" || err != nil || u.IsAbs() || u.Host != "" || !strings.HasPrefix(u.Path, "/p/"+pid) {
		return "/p/" + pid
	}
	return u.RequestURI()
}

func (s *Server) funcs() template.FuncMap {
	return template.FuncMap{
		"icon":          icon,
		"asset":         func(name string) string { return "/static/" + name + "?v=" + s.assetV },
		"statusLabel":   func(st string) string { return statusLabels[st] },
		"priorityLabel": func(p string) string { return priorityLabels[p] },
		"priorityIcon": func(p string) string {
			if _, ok := iconPaths["priority_"+p]; !ok || p == "" {
				return "priority_none"
			}
			return "priority_" + p
		},
		"tagName":  tagName,
		"hue":      func(c string) string { return strings.TrimSuffix(c, domain.TagDiamondSuffix) },
		"diamond":  func(c string) bool { return strings.HasSuffix(c, domain.TagDiamondSuffix) },
		"est":      domain.FormatEstimate,
		"estLabel": domain.EstimateLabel,
		"statuses": func() []string {
			return []string{domain.StatusTodo, domain.StatusInProgress, domain.StatusCompleted, domain.StatusCancelled}
		},
		"priorities":    func() []string { return append([]string{""}, domain.PriorityOrder...) },
		"tags":          func() []string { return append([]string{""}, domain.TagColorCycle...) },
		"presets":       func() []int { return domain.EstimatePresets },
		"finished":      finished,
		"completedFold": completedFold,
		"date": func(ts string) string {
			t, err := time.Parse(time.RFC3339, ts)
			if err != nil {
				return ts
			}
			return t.Local().Format("2 Jan 2006, 15:04")
		},
		"day": func(t time.Time) string { return t.Local().Format("2 Jan 2006") },
		"dict": func(kv ...any) map[string]any {
			m := make(map[string]any, len(kv)/2)
			for i := 0; i+1 < len(kv); i += 2 {
				m[kv[i].(string)] = kv[i+1]
			}
			return m
		},
		"imgURL": func(pid, target string) string {
			return "/p/" + pid + "/img/" + strings.TrimPrefix(target, attachments.DirName+"/")
		},
	}
}
