package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

type actionResult struct {
	Msg   string
	Focus string
	Base  *string
}

type actionFunc func(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error)

var errConflict = &operations.Refusal{Msg: "Changed on another device: reload to see it"}

// webActions is the web side of operations.Actions; TestWebCoversItsActions keeps the two in step.
var webActions = map[string]actionFunc{
	"task.add":         addTask,
	"separator.add":    addSeparator,
	"task.title":       editTitle,
	"task.description": editDescription,
	"task.attach":      attachImages,
	"task.delete":      deleteTasks,
	"status.cycle":     cycleStatus,
	"status.set":       setStatus,
	"priority.set":     setPriority,
	"tag.set":          setTag,
	"estimate.set":     setEstimate,
	"task.nest":        nest(true),
	"task.unnest":      nest(false),
	"task.shift":       shift,
	"task.move":        moveToCategory,
	"task.place":       place,
}

func (s *Server) do(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("action")
	fn, ok := webActions[name]
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
	var (
		res       actionResult
		before    domain.Project
		actionErr error
		current   bool
	)
	after, err := s.store.WithProjectLocked(pid, func(p *domain.Project) error {
		page := r.Header.Get("X-Ordo-Page")
		current = page != "" && page == s.version(pid)
		before = cloneProject(*p)
		res, actionErr = fn(s, p, r, ids)
		return actionErr
	})
	if err != nil {
		s.respond(w, r, pid, res, err, reply{userErr: err == actionErr})
		return
	}
	out := reply{undo: s.undo.put(after.ID, before, after)}
	if current {
		out.stamp = s.version(pid)
	}
	s.respond(w, r, after.ID, res, nil, out)
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
	var refusal *operations.Refusal
	switch {
	case errors.As(err, &refusal):
		return refusal.Msg, true
	case errors.Is(err, domain.ErrTaskNotFound):
		return "That task is gone", true
	case errors.Is(err, domain.ErrCategoryNotFound):
		return "That category is gone", true
	case errors.Is(err, operations.ErrTitleRequired):
		return "A task needs a title", true
	case errors.Is(err, domain.ErrSeparatorNest):
		return "A separator doesn't nest", true
	case errors.Is(err, domain.ErrNestDepth):
		return "A subtask can't have subtasks: nesting goes one level deep", true
	case errors.Is(err, operations.ErrNegativeEstimate):
		return "Pick an estimate from the list", true
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
		if err == error(errConflict) {
			sn.Undo = "reload"
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
	if res.Base != nil && err == nil {
		w.Header().Set("X-Ordo-Base", asciiJSON(*res.Base))
	}
	if out.stamp != "" {
		w.Header().Set("X-Ordo-Stamp", out.stamp)
	}
	w.Header().Set("HX-Push-Url", "false")
	s.renderBack(w, r, back)
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
		if p, err := s.store.LoadProjectByID(parts[1]); err == nil {
			if _, _, err := locate(&p, parts[3]); err == nil {
				s.task(w, get)
				return
			}
		}
		get.URL = &url.URL{Path: "/p/" + parts[1]}
		w.Header().Set("HX-Push-Url", get.URL.Path)
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

func cloneProject(p domain.Project) domain.Project {
	p.Categories = slices.Clone(p.Categories)
	for i := range p.Categories {
		p.Categories[i].Tasks = slices.Clone(p.Categories[i].Tasks)
	}
	return p
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

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return fmt.Sprintf(many, n)
}

func needIDs(ids []string) error {
	if len(ids) == 0 {
		return operations.ErrNoChange
	}
	return nil
}

func addTask(s *Server, p *domain.Project, r *http.Request, _ []string) (actionResult, error) {
	cat := r.FormValue("cat")
	f := operations.TaskFields{Title: r.FormValue("title"), ParentID: r.FormValue("parent")}
	if f.ParentID != "" {
		if ci, _, err := locate(p, f.ParentID); err == nil {
			cat = p.Categories[ci].ID
		}
	}
	t, err := operations.CreateTask(p, cat, f)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{Msg: "Added " + quote(t.Title), Focus: "t-" + t.ID}, nil
}

func addSeparator(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	after := ""
	if len(ids) > 0 {
		after = ids[0]
	}
	cat := r.FormValue("cat")
	if after != "" {
		if ci, _, err := locate(p, after); err == nil {
			cat = p.Categories[ci].ID
		}
	}
	t, err := operations.CreateTaskAfter(p, cat, operations.TaskFields{Kind: domain.KindSeparator, Title: r.FormValue("title")}, after)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{Msg: "Added a separator", Focus: "t-" + t.ID}, nil
}

func taskByID(p *domain.Project, id string) (*domain.Task, string, error) {
	ci, ti, err := locate(p, id)
	if err != nil {
		return nil, "", err
	}
	return &p.Categories[ci].Tasks[ti], p.Categories[ci].ID, nil
}

func editTitle(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	if err := needIDs(ids); err != nil {
		return actionResult{}, err
	}
	t, cat, err := taskByID(p, ids[0])
	if err != nil {
		return actionResult{}, err
	}
	title := strings.TrimSpace(r.FormValue("title"))
	if title == t.Title {
		return actionResult{}, operations.ErrNoChange
	}
	if t.Title != r.FormValue("base") {
		return actionResult{}, errConflict
	}
	if _, err := operations.UpdateTask(p, cat, t.ID, operations.TaskUpdate{Title: &title}); err != nil {
		return actionResult{}, err
	}
	return actionResult{Base: &title}, nil
}

func editDescription(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	if err := needIDs(ids); err != nil {
		return actionResult{}, err
	}
	t, cat, err := taskByID(p, ids[0])
	if err != nil {
		return actionResult{}, err
	}
	if strings.TrimRight(t.Description, "\n") != strings.TrimRight(strings.ReplaceAll(r.FormValue("base"), "\r\n", "\n"), "\n") {
		return actionResult{}, errConflict
	}
	drop := r.FormValue("drop")
	images := slices.DeleteFunc(slices.Clone(r.Form["image"]), func(line string) bool {
		refs := attachments.Refs(line)
		return drop != "" && len(refs) == 1 && refs[0].Target == drop
	})
	desc := joinDescription(r.FormValue("text"), images)
	if desc == strings.TrimRight(t.Description, "\n") {
		return actionResult{}, operations.ErrNoChange
	}
	if _, err := operations.UpdateTask(p, cat, t.ID, operations.TaskUpdate{Description: &desc}); err != nil {
		return actionResult{}, err
	}
	if drop != "" {
		return actionResult{Msg: "Image removed"}, nil
	}
	return actionResult{Base: &desc}, nil
}

func attachImages(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	if err := needIDs(ids); err != nil {
		return actionResult{}, err
	}
	if r.MultipartForm == nil || len(r.MultipartForm.File["image"]) == 0 {
		return actionResult{}, operations.ErrNoChange
	}
	var lines []string
	for _, fh := range r.MultipartForm.File["image"] {
		f, err := fh.Open()
		if err != nil {
			return actionResult{}, err
		}
		raw, err := io.ReadAll(io.LimitReader(f, attachments.MaxSize+1))
		f.Close()
		if err != nil {
			return actionResult{}, err
		}
		target, err := s.images.Add(raw)
		if err != nil {
			return actionResult{}, err
		}
		name := strings.TrimSuffix(fh.Filename, "."+strings.ToLower(lastExt(fh.Filename)))
		if name == "" || name == "image" {
			name = "photo"
		}
		lines = append(lines, attachments.Line(name, target))
	}
	if _, err := operations.AppendDescription(p, ids[0], strings.Join(lines, "\n")); err != nil {
		return actionResult{}, err
	}
	return actionResult{Msg: plural(len(lines), "Image attached", "%d images attached")}, nil
}

func lastExt(name string) string {
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return ""
}

func deleteTasks(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	if err := needIDs(ids); err != nil {
		return actionResult{}, err
	}
	removed, err := operations.DeleteTasks(p, ids, r.FormValue("keep") == "1")
	if err != nil {
		return actionResult{}, err
	}
	tasks, seps := 0, 0
	for _, t := range removed {
		if t.IsSeparator() {
			seps++
		} else {
			tasks++
		}
	}
	if tasks == 0 {
		return actionResult{Msg: plural(seps, "Separator deleted", "%d separators deleted")}, nil
	}
	if len(removed) == 1 {
		return actionResult{Msg: "Deleted " + quote(removed[0].Title)}, nil
	}
	return actionResult{Msg: plural(tasks, "1 task deleted", "%d tasks deleted")}, nil
}

func statusResult(status string, n int) actionResult {
	if n == 1 {
		return actionResult{Msg: "Marked " + strings.ToLower(statusLabels[status])}
	}
	return actionResult{Msg: fmt.Sprintf("%d tasks marked %s", n, strings.ToLower(statusLabels[status]))}
}

func cycleStatus(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	next, n, err := operations.CycleStatus(p, ids, r.FormValue("reverse") == "1")
	if err != nil {
		return actionResult{}, err
	}
	return statusResult(next, n), nil
}

func setStatus(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	status := r.FormValue("value")
	n, err := operations.SetStatuses(p, ids, status)
	if err != nil {
		return actionResult{}, err
	}
	return statusResult(status, n), nil
}

func setPriority(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	priority := r.FormValue("value")
	n, err := operations.SetPriorities(p, ids, priority)
	if err != nil {
		return actionResult{}, err
	}
	what := "Priority " + strings.ToLower(priorityLabels[priority])
	if priority == "" {
		what = "Priority cleared"
	}
	if n > 1 {
		what += fmt.Sprintf(" on %d tasks", n)
	}
	return actionResult{Msg: what}, nil
}

func setTag(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	color := r.FormValue("value")
	n, err := operations.SetTags(p, ids, color)
	if err != nil {
		return actionResult{}, err
	}
	what := "Tagged " + strings.ToLower(tagName(color))
	if color == "" {
		what = "Tag removed"
	}
	if n > 1 {
		what += fmt.Sprintf(" on %d tasks", n)
	}
	return actionResult{Msg: what}, nil
}

func setEstimate(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	minutes, err := strconv.Atoi(r.FormValue("value"))
	if err != nil {
		return actionResult{}, operations.ErrNegativeEstimate
	}
	n, err := operations.SetEstimates(p, ids, minutes)
	if err != nil {
		return actionResult{}, err
	}
	what := "Estimate " + domain.FormatEstimate(minutes)
	if minutes == 0 {
		what = "Estimate cleared"
	}
	if n > 1 {
		what += fmt.Sprintf(" on %d tasks", n)
	}
	return actionResult{Msg: what}, nil
}

func nest(in bool) actionFunc {
	return func(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
		if err := needIDs(ids); err != nil {
			return actionResult{}, err
		}
		parent, err := operations.NestTasks(p, ids, in, operations.AllShown)
		if err != nil {
			return actionResult{}, err
		}
		if !in {
			return actionResult{Msg: "Un-nested", Focus: "t-" + ids[0]}, nil
		}
		t, _, _ := taskByID(p, parent)
		return actionResult{Msg: "Nested under " + quote(t.Title), Focus: "t-" + ids[0]}, nil
	}
}

func shift(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	if err := needIDs(ids); err != nil {
		return actionResult{}, err
	}
	dir := 1
	if r.FormValue("dir") == "up" {
		dir = -1
	}
	if err := operations.ShiftTasks(p, ids, dir, operations.AllShown); err != nil {
		return actionResult{}, err
	}
	return actionResult{Focus: "t-" + ids[0]}, nil
}

func moveToCategory(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	if err := needIDs(ids); err != nil {
		return actionResult{}, err
	}
	dst, err := p.FindCategoryByID(r.FormValue("value"))
	if err != nil {
		return actionResult{}, &operations.Refusal{Msg: "That category is gone"}
	}
	if ci, _, err := locate(p, ids[0]); err == nil && ci == dst && len(ids) == 1 {
		return actionResult{}, operations.ErrNoChange
	}
	rows, err := operations.MoveRows(p, ids, p.Categories[dst].ID, len(p.Categories[dst].Tasks), "")
	if err != nil {
		return actionResult{}, err
	}
	what := "Moved"
	if len(rows) > 1 {
		what = fmt.Sprintf("Moved %d rows", len(rows))
	}
	return actionResult{Msg: what + " to " + p.CategoryPath(dst)}, nil
}

// place drops the rows before the row named "before", or last in its list when that is empty.
func place(s *Server, p *domain.Project, r *http.Request, ids []string) (actionResult, error) {
	if err := needIDs(ids); err != nil {
		return actionResult{}, err
	}
	ci, err := p.FindCategoryByID(r.FormValue("cat"))
	if err != nil {
		return actionResult{}, &operations.Refusal{Msg: "That category is gone"}
	}
	tasks := p.Categories[ci].Tasks
	parent := r.FormValue("parent")
	index := len(tasks)
	if parent != "" {
		pi := slices.IndexFunc(tasks, func(t domain.Task) bool { return t.ID == parent })
		if pi < 0 {
			return actionResult{}, domain.ErrTaskNotFound
		}
		index = domain.BlockEnd(tasks, pi)
	}
	if before := r.FormValue("before"); before != "" {
		if bi := slices.IndexFunc(tasks, func(t domain.Task) bool { return t.ID == before }); bi >= 0 {
			index = bi
		}
	}
	rows, err := operations.MoveRows(p, ids, p.Categories[ci].ID, index, parent)
	if err != nil {
		return actionResult{}, err
	}
	if t, _, err := taskByID(p, rows[0].ParentID); err == nil && rows[0].ParentID != "" {
		return actionResult{Msg: "Nested under " + quote(t.Title)}, nil
	}
	return actionResult{Msg: "Moved"}, nil
}

func quote(s string) string {
	r := []rune(s)
	if len(r) > 40 {
		s = string(r[:39]) + "…"
	}
	return "“" + s + "”"
}
