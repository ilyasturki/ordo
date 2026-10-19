package web

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

type fixture struct {
	t      *testing.T
	store  *data.Store
	auth   *Auth
	srv    *Server
	h      http.Handler
	cookie string
	pid    string
	cat    string
	ids    map[string]string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	dir := t.TempDir()
	projects := filepath.Join(dir, "projects")
	store := data.NewStore(projects)
	p, err := domain.NewProject("Phone")
	if err != nil {
		t.Fatal(err)
	}
	cat, err := p.AddCategoryUnder("", "Feature")
	if err != nil {
		t.Fatal(err)
	}
	catID := cat.ID
	other, _ := p.AddCategoryUnder("", "Fix")
	otherID := other.ID
	ids := map[string]string{"other": otherID}
	for _, title := range []string{"one", "two", "three"} {
		task, err := operations.CreateTask(&p, catID, operations.TaskFields{Title: title})
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = task.ID
	}
	imported, err := store.ImportProject(p)
	if err != nil {
		t.Fatal(err)
	}
	auth := NewAuth(dir)
	srv, err := New(store, attachments.NewStore(projects), auth, log.New(io.Discard, "", 0))
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.NewPairing()
	if err != nil {
		t.Fatal(err)
	}
	cookie, err := auth.Pair(token, "test")
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, store: store, auth: auth, srv: srv, h: srv.Handler(), cookie: cookie, pid: imported.ID, cat: catID, ids: ids}
}

func (f *fixture) do(method, path string, form url.Values, hx bool) *httptest.ResponseRecorder {
	f.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if f.cookie != "" {
		req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	}
	if hx {
		req.Header.Set("HX-Request", "true")
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func (f *fixture) action(name string, form url.Values) *httptest.ResponseRecorder {
	f.t.Helper()
	return f.do(http.MethodPost, "/p/"+f.pid+"/do/"+name, form, true)
}

func (f *fixture) project() domain.Project {
	f.t.Helper()
	p, err := f.store.LoadProjectByID(f.pid)
	if err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *fixture) task(id string) domain.Task {
	f.t.Helper()
	p := f.project()
	ci, ti, err := locate(&p, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return p.Categories[ci].Tasks[ti]
}

func snackOf(t *testing.T, rec *httptest.ResponseRecorder) snack {
	t.Helper()
	var trig map[string]snack
	if h := rec.Header().Get("HX-Trigger"); h != "" {
		if err := json.Unmarshal([]byte(h), &trig); err != nil {
			t.Fatalf("HX-Trigger %q: %v", h, err)
		}
	}
	return trig["ordo:snack"]
}

func titles(c domain.Category) []string {
	var out []string
	for _, t := range c.Tasks {
		prefix := ""
		if t.ParentID != "" {
			prefix = ">"
		}
		out = append(out, prefix+t.Title)
	}
	return out
}

func TestPagesNeedAPairedDevice(t *testing.T) {
	f := newFixture(t)
	f.cookie = ""
	if rec := f.do(http.MethodGet, "/p/"+f.pid, nil, false); rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("GET: %d %s", rec.Code, rec.Header().Get("Location"))
	}
	if rec := f.action("status.cycle", url.Values{"id": {f.ids["one"]}}); rec.Code != http.StatusUnauthorized || rec.Header().Get("HX-Redirect") != "/login" {
		t.Fatalf("POST: %d", rec.Code)
	}
	f.cookie = "forged.cookie"
	if rec := f.do(http.MethodGet, "/projects", nil, false); rec.Code != http.StatusSeeOther {
		t.Fatalf("forged cookie: %d", rec.Code)
	}
}

func TestPairingLinksWorkOnce(t *testing.T) {
	f := newFixture(t)
	token, err := f.auth.NewPairing()
	if err != nil {
		t.Fatal(err)
	}
	f.cookie = ""
	rec := f.do(http.MethodPost, "/pair", url.Values{"token": {token}, "name": {"Pixel"}}, false)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("pair: %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), deviceCookie+"=") {
		t.Fatal("no device cookie")
	}
	if rec := f.do(http.MethodPost, "/pair", url.Values{"token": {token}}, false); rec.Code != http.StatusForbidden {
		t.Fatalf("second use: %d", rec.Code)
	}
	devs, _ := f.auth.Devices()
	if len(devs) != 2 || devs[1].Name != "Pixel" {
		t.Fatalf("devices = %+v", devs)
	}
}

func TestPairingExpires(t *testing.T) {
	f := newFixture(t)
	token, _ := f.auth.NewPairing()
	f.auth.now = func() time.Time { return time.Now().Add(PairTTL + time.Second) }
	if _, err := f.auth.Pair(token, "late"); err != ErrBadPairing {
		t.Fatalf("err = %v", err)
	}
}

func TestPassphraseSignsInAndLimitsGuesses(t *testing.T) {
	f := newFixture(t)
	if err := f.auth.SetPassphrase("correct horse battery"); err != nil {
		t.Fatal(err)
	}
	f.cookie = ""
	if rec := f.do(http.MethodPost, "/login", url.Values{"passphrase": {"correct horse battery"}}, false); rec.Code != http.StatusSeeOther {
		t.Fatalf("login: %d", rec.Code)
	}
	for i := 0; i < 5; i++ {
		if rec := f.do(http.MethodPost, "/login", url.Values{"passphrase": {"nope"}}, false); rec.Code != http.StatusForbidden {
			t.Fatalf("guess %d: %d", i, rec.Code)
		}
	}
	if rec := f.do(http.MethodPost, "/login", url.Values{"passphrase": {"correct horse battery"}}, false); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after five failures: %d", rec.Code)
	}
}

func TestRevokedDeviceIsSignedOut(t *testing.T) {
	f := newFixture(t)
	id, _, _ := strings.Cut(f.cookie, ".")
	if ok, err := f.auth.Revoke(id); !ok || err != nil {
		t.Fatalf("revoke: %v %v", ok, err)
	}
	if rec := f.do(http.MethodGet, "/p/"+f.pid, nil, false); rec.Code != http.StatusSeeOther {
		t.Fatalf("revoked: %d", rec.Code)
	}
}

func TestDeviceCookieRenewsAfterADay(t *testing.T) {
	f := newFixture(t)
	if rec := f.do(http.MethodGet, "/p/"+f.pid, nil, false); strings.Contains(rec.Header().Get("Set-Cookie"), deviceCookie) {
		t.Fatal("renewed a fresh cookie")
	}
	f.auth.now = func() time.Time { return time.Now().Add(renewEvery + time.Hour) }
	rec := f.do(http.MethodGet, "/p/"+f.pid, nil, false)
	if !strings.Contains(rec.Header().Get("Set-Cookie"), deviceCookie+"=") {
		t.Fatal("no renewal")
	}
	f.auth.now = func() time.Time { return time.Now().Add(DeviceIdle + 2*renewEvery) }
	if rec := f.do(http.MethodGet, "/p/"+f.pid, nil, false); rec.Code != http.StatusSeeOther {
		t.Fatalf("idle device still in: %d", rec.Code)
	}
}

func TestCrossSitePostsAreRefused(t *testing.T) {
	f := newFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/p/"+f.pid+"/do/status.cycle", strings.NewReader("id="+f.ids["one"]))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site: %d", rec.Code)
	}
	if f.task(f.ids["one"]).Status != domain.StatusTodo {
		t.Fatal("cross-site post changed the task")
	}
}

func TestWebCoversItsActions(t *testing.T) {
	registered := make(map[string]bool)
	for _, a := range operations.Actions {
		registered[a.ID] = true
		_, handled := webActions[a.ID]
		if a.Surface&operations.Web != 0 && !handled {
			t.Errorf("%s is offered on the web but has no handler", a.ID)
		}
		if a.Surface&operations.Web == 0 && handled {
			t.Errorf("%s has a web handler but is marked TUI-only", a.ID)
		}
	}
	for id := range webActions {
		if !registered[id] {
			t.Errorf("web handler %s is missing from operations.Actions", id)
		}
	}
}

func TestStatusCycleSavesAndOffersUndo(t *testing.T) {
	f := newFixture(t)
	rec := f.action("status.cycle", url.Values{"id": {f.ids["one"]}, "back": {"/p/" + f.pid}})
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	if got := f.task(f.ids["one"]).Status; got != domain.StatusInProgress {
		t.Fatalf("status = %s", got)
	}
	sn := snackOf(t, rec)
	if sn.Msg != "Marked doing" || sn.Undo == "" {
		t.Fatalf("snack = %+v", sn)
	}
	if !strings.Contains(rec.Body.String(), `id="app"`) {
		t.Fatal("no page in the htmx answer")
	}
	rec = f.do(http.MethodPost, sn.Undo, url.Values{"back": {"/p/" + f.pid}}, true)
	if got := f.task(f.ids["one"]).Status; got != domain.StatusTodo || snackOf(t, rec).Msg != "Undone" {
		t.Fatalf("after undo: %s %+v", got, snackOf(t, rec))
	}
	rec = f.do(http.MethodPost, sn.Undo, url.Values{}, true)
	if snackOf(t, rec).Msg != "Nothing to undo" {
		t.Fatalf("second undo: %+v", snackOf(t, rec))
	}
}

func TestUndoRefusesOverALaterChange(t *testing.T) {
	f := newFixture(t)
	sn := snackOf(t, f.action("task.delete", url.Values{"id": {f.ids["two"]}}))
	if _, err := f.store.WithProjectLocked(f.pid, func(p *domain.Project) error {
		_, err := operations.SetStatuses(p, []string{f.ids["one"]}, domain.StatusCompleted)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	rec := f.do(http.MethodPost, sn.Undo, url.Values{}, true)
	if got := snackOf(t, rec).Msg; got != "Can't undo: the project changed since" {
		t.Fatalf("snack = %q", got)
	}
	if _, _, err := locate(ptr(f.project()), f.ids["two"]); err == nil {
		t.Fatal("undo restored over a later change")
	}
}

func ptr[T any](v T) *T { return &v }

func TestPlainPostsRedirectBack(t *testing.T) {
	f := newFixture(t)
	rec := f.do(http.MethodPost, "/p/"+f.pid+"/do/priority.set", url.Values{
		"id": {f.ids["one"], f.ids["two"]}, "value": {domain.PriorityHigh}, "back": {"/p/" + f.pid + "/t/" + f.ids["one"]},
	}, false)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/p/"+f.pid+"/t/"+f.ids["one"] {
		t.Fatalf("%d %s", rec.Code, rec.Header().Get("Location"))
	}
	if f.task(f.ids["two"]).Priority != domain.PriorityHigh {
		t.Fatal("batch priority not set")
	}
	rec = f.do(http.MethodPost, "/p/"+f.pid+"/do/tag.set", url.Values{"id": {f.ids["one"]}, "value": {"blue"}, "back": {"https://evil.example/p/" + f.pid}}, false)
	if rec.Header().Get("Location") != "/p/"+f.pid {
		t.Fatalf("open redirect: %s", rec.Header().Get("Location"))
	}
}

func TestTitleEditRefusesAStaleBase(t *testing.T) {
	f := newFixture(t)
	rec := f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"uno"}, "base": {"one"}})
	if f.task(f.ids["one"]).Title != "uno" || rec.Header().Get("X-Ordo-Base") != `"uno"` {
		t.Fatalf("title = %q, base header %q", f.task(f.ids["one"]).Title, rec.Header().Get("X-Ordo-Base"))
	}
	rec = f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"eins"}, "base": {"one"}})
	if snackOf(t, rec).Msg != errConflict.Msg || f.task(f.ids["one"]).Title != "uno" {
		t.Fatalf("stale base: %+v %q", snackOf(t, rec), f.task(f.ids["one"]).Title)
	}
	rec = f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"  "}, "base": {"uno"}})
	if snackOf(t, rec).Msg != "A task needs a title" {
		t.Fatalf("empty title: %+v", snackOf(t, rec))
	}
}

func TestDescriptionKeepsItsImagesLast(t *testing.T) {
	f := newFixture(t)
	img := "![shot](attachments/0123456789abcdef0123456789abcdef.png)"
	f.action("task.description", url.Values{"id": {f.ids["one"]}, "base": {""}, "text": {"first line"}, "image": {img}})
	if got := f.task(f.ids["one"]).Description; got != "first line\n"+img {
		t.Fatalf("description = %q", got)
	}
	f.action("task.description", url.Values{"id": {f.ids["one"]}, "base": {"first line\n" + img}, "text": {"first line"}, "image": {img}, "drop": {"attachments/0123456789abcdef0123456789abcdef.png"}})
	if got := f.task(f.ids["one"]).Description; got != "first line" {
		t.Fatalf("after drop = %q", got)
	}
}

func TestSplitDescriptionTakesOnlyTrailingImages(t *testing.T) {
	img := "![a](attachments/0123456789abcdef0123456789abcdef.png)"
	cases := []struct {
		desc, text string
		lines      []string
	}{
		{"", "", nil},
		{"text", "text", nil},
		{"text\n" + img, "text", []string{img}},
		{img, "", []string{img}},
		{"a\n" + img + "\nb", "a\n" + img + "\nb", nil},
		{"x\n\n" + img, "x", []string{img}},
	}
	for _, c := range cases {
		text, lines := splitDescription(c.desc)
		if text != c.text || strings.Join(lines, "|") != strings.Join(c.lines, "|") {
			t.Errorf("%q → %q %q", c.desc, text, lines)
		}
	}
}

func TestPlaceDropsARowUnderAParent(t *testing.T) {
	f := newFixture(t)
	rec := f.action("task.place", url.Values{"id": {f.ids["three"]}, "cat": {f.cat}, "parent": {f.ids["one"]}, "before": {""}})
	if got := snackOf(t, rec).Msg; got != "Nested under “one”" {
		t.Fatalf("snack %q", got)
	}
	p := f.project()
	if got := strings.Join(titles(p.Categories[0]), ","); got != "one,>three,two" {
		t.Fatalf("after nesting drop: %s", got)
	}
	f.action("task.place", url.Values{"id": {f.ids["two"]}, "cat": {f.ids["other"]}, "parent": {""}, "before": {""}})
	p = f.project()
	if got := strings.Join(titles(p.Categories[1]), ","); got != "two" {
		t.Fatalf("other category: %s", got)
	}
	f.action("task.place", url.Values{"id": {f.ids["three"]}, "cat": {f.cat}, "parent": {""}, "before": {f.ids["one"]}})
	p = f.project()
	if got := strings.Join(titles(p.Categories[0]), ","); got != "three,one" {
		t.Fatalf("un-nesting drop: %s", got)
	}
}

func TestNestAndShiftReportRefusals(t *testing.T) {
	f := newFixture(t)
	if got := snackOf(t, f.action("task.nest", url.Values{"id": {f.ids["one"]}})).Msg; got != "Nothing above to nest under" {
		t.Fatalf("nest first: %q", got)
	}
	if got := snackOf(t, f.action("task.nest", url.Values{"id": {f.ids["two"]}})).Msg; got != "Nested under “one”" {
		t.Fatalf("nest: %q", got)
	}
	if got := snackOf(t, f.action("task.shift", url.Values{"id": {f.ids["two"]}, "dir": {"up"}})).Msg; got != "Already at the top of its parent" {
		t.Fatalf("shift: %q", got)
	}
}

func TestAddTaskAndSubtask(t *testing.T) {
	f := newFixture(t)
	rec := f.action("task.add", url.Values{"cat": {f.cat}, "title": {"four"}})
	if !strings.HasPrefix(snackOf(t, rec).Msg, "Added") {
		t.Fatalf("add: %+v", snackOf(t, rec))
	}
	f.action("task.add", url.Values{"parent": {f.ids["one"]}, "title": {"sub"}})
	f.action("separator.add", url.Values{"cat": {f.cat}, "title": {"Later"}})
	if got := strings.Join(titles(f.project().Categories[0]), ","); got != "one,>sub,two,three,four,Later" {
		t.Fatalf("rows: %s", got)
	}
}

func TestMoveToCategory(t *testing.T) {
	f := newFixture(t)
	rec := f.action("task.move", url.Values{"id": {f.ids["one"], f.ids["two"]}, "value": {f.ids["other"]}})
	if got := snackOf(t, rec).Msg; got != "Moved 2 rows to Fix" {
		t.Fatalf("snack %q", got)
	}
	if got := strings.Join(titles(f.project().Categories[1]), ","); got != "one,two" {
		t.Fatalf("Fix: %s", got)
	}
}

func TestAttachStoresTheImage(t *testing.T) {
	f := newFixture(t)
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.Black)
	png.Encode(&buf, img)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("id", f.ids["one"])
	fw, _ := mw.CreateFormFile("image", "board.png")
	fw.Write(buf.Bytes())
	mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/p/"+f.pid+"/do/task.attach", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	desc := f.task(f.ids["one"]).Description
	refs := attachments.Refs(desc)
	if len(refs) != 1 || refs[0].Name != "board" {
		t.Fatalf("description = %q (snack %+v)", desc, snackOf(t, rec))
	}
	name := strings.TrimPrefix(refs[0].Target, attachments.DirName+"/")
	if rec := f.do(http.MethodGet, "/p/"+f.pid+"/img/"+name, nil, false); rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("serve image: %d %s", rec.Code, rec.Header().Get("Content-Type"))
	}
	if rec := f.do(http.MethodGet, "/p/"+f.pid+"/img/..%2F..%2Fweb.json", nil, false); rec.Code != http.StatusNotFound {
		t.Fatalf("traversal: %d", rec.Code)
	}
}

func TestPagesRender(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"/projects", "/p/" + f.pid, "/p/" + f.pid + "?f=done", "/p/" + f.pid + "/t/" + f.ids["one"], "/devices"} {
		rec := f.do(http.MethodGet, path, nil, false)
		if rec.Code != http.StatusOK {
			t.Errorf("%s: %d", path, rec.Code)
		}
		if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") {
			t.Errorf("%s: CSP %q", path, csp)
		}
	}
	if rec := f.do(http.MethodGet, "/p/"+f.pid+"/t/ghost", nil, false); rec.Code != http.StatusNotFound {
		t.Errorf("ghost task: %d", rec.Code)
	}
	f.cookie = ""
	for _, path := range []string{"/login", "/pair/abc", "/manifest.webmanifest", "/static/app.css"} {
		if rec := f.do(http.MethodGet, path, nil, false); rec.Code != http.StatusOK {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
}

func TestAsciiJSONEscapesForLatin1Headers(t *testing.T) {
	got := asciiJSON(map[string]string{"msg": "Deleted “café” 😀"})
	for _, r := range got {
		if r > 0x7f {
			t.Fatalf("non-ASCII in %q", got)
		}
	}
	var back map[string]string
	if err := json.Unmarshal([]byte(got), &back); err != nil || back["msg"] != "Deleted “café” 😀" {
		t.Fatalf("round trip: %v %q", err, back["msg"])
	}
}

func TestAuthFileStaysPrivate(t *testing.T) {
	f := newFixture(t)
	info, err := os.Stat(f.auth.Path())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("web.json mode %v", info.Mode().Perm())
	}
	raw, _ := os.ReadFile(f.auth.Path())
	_, token, _ := strings.Cut(f.cookie, ".")
	if bytes.Contains(raw, []byte(token)) {
		t.Fatal("web.json holds a raw token")
	}
}

func TestActionErrorsAnswerWithASnackNotA500(t *testing.T) {
	f := newFixture(t)
	f.action("task.add", url.Values{"parent": {f.ids["one"]}, "title": {"sub"}})
	sub := f.project().Categories[0].Tasks[1].ID
	cases := []struct {
		action string
		form   url.Values
		want   string
	}{
		{"task.add", url.Values{"parent": {sub}, "title": {"deeper"}}, "A subtask can't have subtasks: nesting goes one level deep"},
		{"task.add", url.Values{"cat": {"gone"}, "title": {"x"}}, "That category is gone"},
		{"estimate.set", url.Values{"id": {f.ids["one"]}, "value": {"soon"}}, "Pick an estimate from the list"},
		{"priority.set", url.Values{"id": {f.ids["one"]}, "value": {"urgent"}}, ""},
		{"task.place", url.Values{"id": {f.ids["two"]}, "cat": {f.cat}, "parent": {sub}}, "Already a subtask: nesting goes one level deep"},
	}
	for _, c := range cases {
		rec := f.action(c.action, c.form)
		got := snackOf(t, rec).Msg
		if rec.Code != http.StatusOK || got == "" || (c.want != "" && got != c.want) {
			t.Errorf("%s %v: %d %q", c.action, c.form, rec.Code, got)
		}
	}
}

func TestStampOnlyVouchesForACurrentPage(t *testing.T) {
	f := newFixture(t)
	send := func(page string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/p/"+f.pid+"/do/tag.set", strings.NewReader(url.Values{"id": {f.ids["one"]}, "value": {"blue"}}.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.Header.Set("X-Ordo-Page", page)
		req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
		rec := httptest.NewRecorder()
		f.h.ServeHTTP(rec, req)
		return rec
	}
	if rec := send("stale"); rec.Header().Get("X-Ordo-Stamp") != "" {
		t.Fatal("stamped the answer to a stale page")
	}
	f.action("tag.set", url.Values{"id": {f.ids["one"]}, "value": {""}})
	rec := send(f.srv.version(f.pid))
	if got := rec.Header().Get("X-Ordo-Stamp"); got == "" || got != f.srv.version(f.pid) {
		t.Fatalf("current page stamp %q, file %q", got, f.srv.version(f.pid))
	}
}

func TestStaleTitleOffersAReload(t *testing.T) {
	f := newFixture(t)
	rec := f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"uno"}, "base": {"old"}})
	if sn := snackOf(t, rec); sn.Undo != "reload" || rec.Header().Get("X-Ordo-Base") != "" {
		t.Fatalf("snack %+v base %q", sn, rec.Header().Get("X-Ordo-Base"))
	}
}

func TestBrokenFieldValuesStillRender(t *testing.T) {
	f := newFixture(t)
	if _, err := f.store.WithProjectLocked(f.pid, func(p *domain.Project) error {
		p.Categories[0].Tasks[0].Status = "someday"
		p.Categories[0].Tasks[0].Priority = "urgent"
		p.Categories[0].Tasks[0].TagColor = "-diamond"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/p/" + f.pid + "?f=all", "/p/" + f.pid + "/t/" + f.ids["one"]} {
		rec := f.do(http.MethodGet, path, nil, false)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "</html>") {
			t.Errorf("%s: %d, complete %v", path, rec.Code, strings.Contains(rec.Body.String(), "</html>"))
		}
	}
}

func TestTaskPageGroupsItsFields(t *testing.T) {
	f := newFixture(t)
	page := func(title string) string {
		t.Helper()
		rec := f.do(http.MethodGet, "/p/"+f.pid+"/t/"+f.ids[title], nil, false)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d", title, rec.Code)
		}
		return rec.Body.String()
	}
	must := func(body, want string, has bool) {
		t.Helper()
		if strings.Contains(body, want) != has {
			t.Errorf("page has %q: %v, want %v", want, !has, has)
		}
	}
	body := page("two")
	for _, sheet := range []string{"status-sheet", "priority-sheet", "tag-sheet", "estimate-sheet", "move-sheet", "task-menu"} {
		must(body, `data-sheet-open="`+sheet+`"`, true)
		must(body, `id="`+sheet+`"`, true)
	}
	must(body, `<span class="field-value">To do</span>`, true)
	must(body, `name="value" value="completed">`, true)
	must(body, `value="`+f.cat+`" aria-current="true"`, true)
	must(body, "/do/task.nest", true)
	must(body, "/do/task.unnest", false)

	f.action("status.set", url.Values{"id": {f.ids["two"]}, "value": {domain.StatusCompleted}})
	f.action("task.nest", url.Values{"id": {f.ids["two"]}})
	body = page("two")
	must(body, `name="value" value="todo">`, true)
	must(body, "Mark open", true)
	must(body, "/do/task.unnest", true)
	must(body, `id="new-subtask"`, false)
	body = page("one")
	must(body, "/do/task.nest", false)
	must(body, "/do/task.unnest", false)
	must(body, "Subtasks · 1/1", true)
}
