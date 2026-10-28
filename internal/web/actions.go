package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

type actionResult struct {
	Msg   string
	Focus string
	// Saved tells a text field's autosave that the file now holds what it sent.
	Saved bool
	// Row names the row a request sent with X-Ordo-Want: row gets back alone, instead of the page.
	Row string
}

// formOp reads a form into the op it asks for; msg replaces an applied op's own message.
type formOp func(s *Server, r *http.Request, ids []string) (op operations.Op, msg string, err error)

// webActions is the web side of operations.Actions; TestWebCoversItsActions keeps the two in step.
var webActions = map[string]formOp{
	"task.add": fields(func(r *http.Request, _ []string) operations.Op {
		return operations.Op{Cat: r.FormValue("cat"), Title: r.FormValue("title"), Parent: r.FormValue("parent")}
	}),
	"task.insert":   fields(rowBelow),
	"separator.add": fields(rowBelow),
	"task.title": fields(func(r *http.Request, ids []string) operations.Op {
		return operations.Op{IDs: ids, Title: r.FormValue("title"), Base: r.FormValue("base")}
	}),
	"task.description": editDescription,
	"task.attach":      attachImages,
	"task.delete": fields(func(r *http.Request, ids []string) operations.Op {
		return operations.Op{IDs: ids, Keep: r.FormValue("keep") == "1"}
	}),
	"status.cycle": fields(func(r *http.Request, ids []string) operations.Op {
		return operations.Op{IDs: ids, Reverse: r.FormValue("reverse") == "1"}
	}),
	"status.set":   fields(withValue),
	"priority.set": fields(withValue),
	"tag.set":      fields(withValue),
	"estimate.set": fields(withValue),
	"task.nest":    fields(onRows),
	"task.unnest":  fields(onRows),
	"task.shift": fields(func(r *http.Request, ids []string) operations.Op {
		return operations.Op{IDs: ids, Up: r.FormValue("dir") == "up"}
	}),
	"task.move": fields(func(r *http.Request, ids []string) operations.Op {
		return operations.Op{IDs: ids, Cat: r.FormValue("value")}
	}),
	"task.place": fields(func(r *http.Request, ids []string) operations.Op {
		return operations.Op{IDs: ids, Cat: r.FormValue("cat"), Parent: r.FormValue("parent"), Before: r.FormValue("before")}
	}),
}

// focusesRow names the actions whose answer puts the cursor on the row they made or moved.
var focusesRow = map[string]bool{"task.add": true, "task.insert": true, "separator.add": true, "task.nest": true, "task.unnest": true, "task.shift": true}

func fields(build func(r *http.Request, ids []string) operations.Op) formOp {
	return func(_ *Server, r *http.Request, ids []string) (operations.Op, string, error) {
		return build(r, ids), "", nil
	}
}

func rowBelow(r *http.Request, ids []string) operations.Op {
	return operations.Op{IDs: ids, Cat: r.FormValue("cat"), Title: r.FormValue("title")}
}

func withValue(r *http.Request, ids []string) operations.Op {
	return operations.Op{IDs: ids, Value: r.FormValue("value")}
}

func onRows(_ *http.Request, ids []string) operations.Op {
	return operations.Op{IDs: ids}
}

func (s *Server) do(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("action")
	build, ok := webActions[name]
	if !ok {
		http.NotFound(w, r)
		return
	}
	pid := r.PathValue("pid")
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			s.respond(w, r, pid, actionResult{}, &operations.Refusal{Msg: "The upload didn't come through"}, reply{})
			return
		}
	} else if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	ids := slices.DeleteFunc(r.Form["id"], func(id string) bool { return domain.ValidateID(id) != nil })
	op, msg, err := build(s, r, ids)
	if err != nil {
		s.respond(w, r, pid, actionResult{}, err, reply{userErr: true})
		return
	}
	op.Action = name
	var (
		out       operations.Outcome
		before    domain.Project
		actionErr error
		current   bool
	)
	after, err := s.store.WithProjectLocked(pid, func(p *domain.Project) error {
		page := r.Header.Get("X-Ordo-Page")
		current = page != "" && page == s.version(pid)
		before = operations.Clone(*p)
		out, actionErr = operations.Apply(p, op)
		return actionErr
	})
	res := actionResult{Msg: out.Msg, Saved: out.Saved, Row: out.Row}
	if msg != "" {
		res.Msg = msg
	}
	if focusesRow[name] && out.Row != "" {
		res.Focus = "t-" + out.Row
	}
	if err != nil {
		s.respond(w, r, pid, res, err, reply{userErr: err == actionErr})
		return
	}
	sent := reply{undo: s.undo.put(after.ID, before, after)}
	if current {
		sent.stamp = s.version(pid)
	}
	s.respond(w, r, after.ID, res, nil, sent)
}

const maxUpload = 64 << 20

type reply struct {
	undo    string
	stamp   string
	userErr bool
}

type snack struct {
	Msg  string `json:"msg"`
	Undo string `json:"undo,omitempty"`
}

// userMessage phrases an action's own error; ok is false for a failure of the store itself.
func userMessage(err error, fromAction bool) (msg string, ok bool) {
	if msg, ok := operations.Explain(err); ok {
		return msg, true
	}
	switch {
	case errors.Is(err, attachments.ErrNotImage), errors.Is(err, attachments.ErrTooLarge):
		return err.Error(), true
	case fromAction:
		return "Can't do that: " + err.Error(), true
	}
	return "", false
}

func (s *Server) respond(w http.ResponseWriter, r *http.Request, pid string, res actionResult, err error, out reply) {
	sn := snack{}
	switch {
	case err == nil:
		sn.Msg = res.Msg
		if out.undo != "" {
			sn.Undo = "/p/" + pid + "/undo/" + out.undo
		}
	case errors.Is(err, operations.ErrNoChange):
	default:
		msg, ok := userMessage(err, out.userErr)
		if !ok {
			s.fail(w, r, err)
			return
		}
		sn.Msg = msg
		// Flags, not text: a description can outgrow a proxy's header buffer.
		switch {
		case errors.As(err, new(*operations.Conflict)):
			// The page reads their text from the page this answers with, and asks.
			w.Header().Set("X-Ordo-Conflict", "1")
			sn = snack{}
		case errors.Is(err, domain.ErrTaskNotFound):
			w.Header().Set("X-Ordo-Gone", "1")
		}
	}
	back := backTo(r, pid)
	if r.Header.Get("HX-Request") == "" {
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	if sn.Msg != "" {
		w.Header().Set("HX-Trigger", asciiJSON(map[string]snack{"ordo:snack": sn}))
	}
	if res.Focus != "" && err == nil {
		w.Header().Set("HX-Trigger-After-Settle", asciiJSON(map[string]string{"ordo:focus": res.Focus}))
	}
	if res.Saved && (err == nil || errors.Is(err, operations.ErrNoChange)) {
		w.Header().Set("X-Ordo-Saved", "1")
	}
	if out.stamp != "" {
		w.Header().Set("X-Ordo-Stamp", out.stamp)
	}
	w.Header().Set("HX-Push-Url", "false")
	if r.Header.Get("X-Ordo-Want") == "row" {
		s.renderRow(w, pid, res.Row)
		return
	}
	s.renderBack(w, r, back)
}

// renderRow answers with one row's markup, or an empty body when the action made none.
func (s *Server) renderRow(w http.ResponseWriter, pid, id string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if id == "" {
		return
	}
	p, err := s.store.LoadProjectByID(pid)
	if err != nil {
		return
	}
	ci, ti, err := locate(&p, id)
	if err != nil {
		return
	}
	cat := p.Categories[ci]
	row := newRow(cat.Tasks[ti])
	row.ChildDone, row.ChildTotal = cat.ChildProgress(ti)
	if err := s.pages["project"].ExecuteTemplate(w, "row", map[string]any{"R": row, "PID": p.ID, "Back": "/p/" + p.ID}); err != nil {
		s.log.Printf("render row %s: %v", id, err)
	}
}

// renderBack answers an htmx action with the page it came from, the project page once its task is gone.
func (s *Server) renderBack(w http.ResponseWriter, r *http.Request, back string) {
	u, _ := url.Parse(back)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	get := r.Clone(r.Context())
	get.Method = http.MethodGet
	get.URL = u
	get.Form, get.PostForm = nil, nil
	get.SetPathValue("pid", parts[1])
	if len(parts) == 4 && parts[2] == "t" {
		get.SetPathValue("tid", parts[3])
		s.task(w, get)
		return
	}
	s.project(w, get)
}

func (s *Server) undoAction(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	entry, ok := s.undo.take(r.PathValue("key"))
	if !ok || entry.project != r.PathValue("pid") {
		s.respond(w, r, r.PathValue("pid"), actionResult{}, &operations.Refusal{Msg: "Nothing to undo"}, reply{})
		return
	}
	_, err := s.store.WithProjectLocked(entry.project, func(p *domain.Project) error {
		if !sameContent(*p, entry.after) {
			return &operations.Refusal{Msg: "Can't undo: the project changed since"}
		}
		*p = entry.before
		return nil
	})
	s.respond(w, r, entry.project, actionResult{Msg: "Undone"}, err, reply{})
}

func sameContent(a, b domain.Project) bool {
	a.UpdatedAt, b.UpdatedAt, a.Schema, b.Schema = "", "", 0, 0
	ja, err1 := json.Marshal(a)
	jb, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && string(ja) == string(jb)
}

// asciiJSON escapes every non-ASCII rune: browsers read response headers as Latin-1.
func asciiJSON(v any) string {
	b, _ := json.Marshal(v)
	var out strings.Builder
	for _, r := range string(b) {
		if r < 0x80 {
			out.WriteRune(r)
			continue
		}
		for _, u := range utf16.Encode([]rune{r}) {
			fmt.Fprintf(&out, `\u%04x`, u)
		}
	}
	return out.String()
}

func editDescription(_ *Server, r *http.Request, ids []string) (operations.Op, string, error) {
	drop := r.FormValue("drop")
	images := slices.DeleteFunc(slices.Clone(r.Form["image"]), func(line string) bool {
		refs := attachments.Refs(line)
		return drop != "" && len(refs) == 1 && refs[0].Target == drop
	})
	op := operations.Op{IDs: ids, Text: joinDescription(r.FormValue("text"), images), Base: r.FormValue("base")}
	if drop != "" {
		return op, "Image removed", nil
	}
	return op, "", nil
}

func attachImages(s *Server, r *http.Request, ids []string) (operations.Op, string, error) {
	if len(ids) == 0 || r.MultipartForm == nil || len(r.MultipartForm.File["image"]) == 0 {
		return operations.Op{}, "", operations.ErrNoChange
	}
	var lines []string
	for _, fh := range r.MultipartForm.File["image"] {
		f, err := fh.Open()
		if err != nil {
			return operations.Op{}, "", err
		}
		raw, err := io.ReadAll(io.LimitReader(f, attachments.MaxSize+1))
		f.Close()
		if err != nil {
			return operations.Op{}, "", err
		}
		target, err := s.images.Add(raw)
		if err != nil {
			return operations.Op{}, "", err
		}
		name := strings.TrimSuffix(fh.Filename, "."+strings.ToLower(lastExt(fh.Filename)))
		if name == "" || name == "image" {
			name = "photo"
		}
		lines = append(lines, attachments.Line(name, target))
	}
	return operations.Op{IDs: ids, Lines: lines}, "", nil
}

func lastExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return ""
}
