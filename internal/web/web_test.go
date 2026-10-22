package web

import (
	"bytes"
	"encoding/json"
	"html"
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

// A pairing link opened in a browser tab, outside the installed app, must leave the token for the app to spend.
func TestOpeningAPairLinkLeavesItUnspent(t *testing.T) {
	f := newFixture(t)
	token, _ := f.auth.NewPairing()
	f.cookie = ""
	rec := f.do(http.MethodGet, "/pair/"+token, nil, false)
	if body := rec.Body.String(); rec.Code != http.StatusOK || !strings.Contains(body, "data-browser-hint") || !strings.Contains(body, "data-copy") {
		t.Fatalf("pair page: %d, hint %v", rec.Code, strings.Contains(body, "data-browser-hint"))
	}
	link := "http://127.0.0.1:7319/pair/" + token
	if rec := f.do(http.MethodPost, "/pair", url.Values{"token": {link}, "from": {"login"}}, false); rec.Code != http.StatusSeeOther {
		t.Fatalf("pasting the tab's link in the app: %d", rec.Code)
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

func TestPairTokenTakesALinkOrAPastedCode(t *testing.T) {
	const token = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for _, in := range []string{
		token,
		" abcdefghijklmnopqrstuvwxyz\n",
		"ABCDE-FGHIJ-KLMNO-PQRST-UVWXYZ",
		"abcde fghij klmno pqrst uvwxyz",
		"https://ordo.example.com/pair/" + token,
		"http://localhost:7319/pair/" + strings.ToLower(token) + "?utm=x#y",
		"ordo.example.com/pair/" + token + "/",
	} {
		if got := pairToken(in); got != token {
			t.Errorf("pairToken(%q) = %q", in, got)
		}
	}
}

func TestPastedLinkPairsFromTheLoginPage(t *testing.T) {
	f := newFixture(t)
	f.cookie = ""
	for _, paste := range []func(string) string{
		func(tok string) string { return "https://elsewhere.example/pair/" + tok },
		strings.ToLower,
	} {
		token, _ := f.auth.NewPairing()
		rec := f.do(http.MethodPost, "/pair", url.Values{"token": {paste(token)}, "from": {"login"}}, false)
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("pair %q: %d", paste(token), rec.Code)
		}
	}
	rec := f.do(http.MethodPost, "/pair", url.Values{"token": {"not a code"}, "from": {"login"}}, false)
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Pairing link or code") {
		t.Fatalf("a bad code should show the login page again: %d", rec.Code)
	}
}

func TestPairingANewDeviceNeedsADevice(t *testing.T) {
	f := newFixture(t)
	f.cookie = ""
	if rec := f.do(http.MethodPost, "/devices/pair", url.Values{}, false); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous mint: %d", rec.Code)
	}
}

func mintedToken(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `src="data:image/png;base64,`) {
		t.Fatalf("mint: %d, no QR", rec.Code)
	}
	_, after, ok := strings.Cut(body, "/pair/")
	if !ok {
		t.Fatal("mint: no link")
	}
	return after[:strings.IndexAny(after, `"<`)]
}

func TestADeviceMintsOneLiveCodeThatRecordsWhoPaired(t *testing.T) {
	f := newFixture(t)
	minter, _, _ := strings.Cut(f.cookie, ".")
	first := mintedToken(t, f.do(http.MethodPost, "/devices/pair", url.Values{}, false))
	second := mintedToken(t, f.do(http.MethodPost, "/devices/pair", url.Values{}, false))
	if _, err := f.auth.Pair(first, "stale"); err != ErrBadPairing {
		t.Fatalf("a second code should replace the first: %v", err)
	}
	if _, err := f.auth.Pair(second, "Laptop"); err != nil {
		t.Fatal(err)
	}
	devs, _ := f.auth.Devices()
	if len(devs) != 2 || devs[1].PairedBy != minter {
		t.Fatalf("devices = %+v", devs)
	}
	if body := f.do(http.MethodGet, "/devices", nil, false).Body.String(); !strings.Contains(body, "Paired from test") || !strings.Contains(body, "also signs out Laptop") {
		t.Fatal("devices page hides who paired whom")
	}
}

func TestPairingANewDeviceAsksThePassphrase(t *testing.T) {
	f := newFixture(t)
	if err := f.auth.SetPassphrase("correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if rec := f.do(http.MethodPost, "/devices/pair", url.Values{}, false); rec.Code != http.StatusForbidden {
		t.Fatalf("no passphrase: %d", rec.Code)
	}
	mintedToken(t, f.do(http.MethodPost, "/devices/pair", url.Values{"passphrase": {"correct horse battery"}}, false))
	for i := 0; i < 4; i++ {
		f.do(http.MethodPost, "/devices/pair", url.Values{"passphrase": {"nope"}}, false)
	}
	if rec := f.do(http.MethodPost, "/devices/pair", url.Values{"passphrase": {"correct horse battery"}}, false); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after five failures: %d", rec.Code)
	}
	f.cookie = ""
	if rec := f.do(http.MethodPost, "/login", url.Values{"passphrase": {"correct horse battery"}}, false); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("login after five mint failures: %d", rec.Code)
	}
}

func TestRevokingADeviceRevokesTheDevicesItPaired(t *testing.T) {
	f := newFixture(t)
	root, _, _ := strings.Cut(f.cookie, ".")
	pairFrom := func(by, name string) string {
		token, err := f.auth.NewPairingBy(by)
		if err != nil {
			t.Fatal(err)
		}
		cookie, err := f.auth.Pair(token, name)
		if err != nil {
			t.Fatal(err)
		}
		id, _, _ := strings.Cut(cookie, ".")
		return id
	}
	child := pairFrom(root, "child")
	grandchild := pairFrom(child, "grandchild")
	pending, _ := f.auth.NewPairingBy(grandchild)
	other := pairFrom("", "other")

	gone, err := f.auth.Revoke(child)
	if err != nil {
		t.Fatal(err)
	}
	if len(gone) != 2 || gone[0].ID != child || gone[1].ID != grandchild {
		t.Fatalf("revoked %+v", gone)
	}
	devs, _ := f.auth.Devices()
	var left []string
	for _, d := range devs {
		left = append(left, d.ID)
	}
	if strings.Join(left, ",") != root+","+other {
		t.Fatalf("left %v", left)
	}
	if _, err := f.auth.Pair(pending, "late"); err != ErrBadPairing {
		t.Fatalf("a revoked device's code still pairs: %v", err)
	}
}

func TestRevokingTheDeviceThatPairedThisOneSignsItOut(t *testing.T) {
	f := newFixture(t)
	root, _, _ := strings.Cut(f.cookie, ".")
	token, _ := f.auth.NewPairingBy(root)
	cookie, err := f.auth.Pair(token, "child")
	if err != nil {
		t.Fatal(err)
	}
	f.cookie = cookie
	rec := f.do(http.MethodPost, "/devices/"+root+"/revoke", url.Values{}, false)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("revoke parent: %d %q", rec.Code, rec.Header().Get("Location"))
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("cookie kept: %q", rec.Header().Get("Set-Cookie"))
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
	if gone, err := f.auth.Revoke(id); len(gone) != 1 || err != nil {
		t.Fatalf("revoke: %v %v", gone, err)
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
	if f.task(f.ids["one"]).Title != "uno" || rec.Header().Get("X-Ordo-Saved") != "1" {
		t.Fatalf("title = %q, saved header %q", f.task(f.ids["one"]).Title, rec.Header().Get("X-Ordo-Saved"))
	}
	rec = f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"eins"}, "base": {"one"}})
	if rec.Header().Get("X-Ordo-Conflict") != "1" || rec.Header().Get("X-Ordo-Saved") != "" || f.task(f.ids["one"]).Title != "uno" {
		t.Fatalf("stale base: conflict %q, saved %q, title %q", rec.Header().Get("X-Ordo-Conflict"), rec.Header().Get("X-Ordo-Saved"), f.task(f.ids["one"]).Title)
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

func TestPlaceLandsWhereTheDragSlotShows(t *testing.T) {
	f := newFixture(t)
	place := func(id, parent, before string) {
		t.Helper()
		if rec := f.action("task.place", url.Values{"id": {f.ids[id]}, "cat": {f.cat}, "parent": {f.ids[parent]}, "before": {f.ids[before]}}); rec.Code != http.StatusOK {
			t.Fatalf("place %s: %d", id, rec.Code)
		}
	}
	steps := []struct {
		id, parent, before, want string
	}{
		{"two", "one", "", "one,>two,three"},
		{"three", "one", "two", "one,>three,>two"},
		{"three", "", "", "one,>two,three"},
		{"two", "", "three", "one,two,three"},
		{"two", "one", "", "one,>two,three"},
		{"one", "", "", "three,one,>two"},
	}
	for _, s := range steps {
		place(s.id, s.parent, s.before)
		if got := strings.Join(titles(f.project().Categories[0]), ","); got != s.want {
			t.Fatalf("place %s under %q before %q: %s, want %s", s.id, s.parent, s.before, got, s.want)
		}
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
	f.action("separator.add", url.Values{"id": {f.ids["one"]}})
	if got := strings.Join(titles(f.project().Categories[0]), ","); got != "Later,one,>sub,,two,three,four" {
		t.Fatalf("rows: %s", got)
	}
}

func (f *fixture) partial(name string, form url.Values) *httptest.ResponseRecorder {
	f.t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/p/"+f.pid+"/do/"+name, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("X-Ordo-Want", "row")
	req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func TestInsertAnswersWithTheNewRowAlone(t *testing.T) {
	f := newFixture(t)
	rec := f.partial("task.insert", url.Values{"cat": {f.cat}, "title": {"first"}})
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.HasPrefix(strings.TrimSpace(body), `<li class="row s-todo"`) || strings.Contains(body, `id="app"`) {
		t.Fatalf("partial answer %d: %s", rec.Code, body)
	}
	if snackOf(t, rec).Msg != "" {
		t.Fatalf("an inline add raised a snack: %+v", snackOf(t, rec))
	}
	first := f.project().Categories[0].Tasks[0]
	if first.Title != "first" || !strings.Contains(body, `data-id="`+first.ID+`"`) {
		t.Fatalf("top row %q, body %s", first.Title, body)
	}
	f.partial("task.insert", url.Values{"cat": {f.cat}, "id": {first.ID}, "title": {"second"}})
	if got := strings.Join(titles(f.project().Categories[0]), ","); got != "first,second,one,two,three" {
		t.Fatalf("batch order: %s", got)
	}
	rec = f.partial("task.insert", url.Values{"cat": {f.cat}, "title": {"  "}})
	if strings.TrimSpace(rec.Body.String()) != "" || snackOf(t, rec).Msg != "A task needs a title" {
		t.Fatalf("refused add: %q %+v", rec.Body.String(), snackOf(t, rec))
	}
	rec = f.partial("separator.add", url.Values{"cat": {f.cat}})
	if !strings.Contains(rec.Body.String(), "data-sep") || snackOf(t, rec).Undo == "" {
		t.Fatalf("separator answer: %s %+v", rec.Body.String(), snackOf(t, rec))
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
	for _, path := range []string{"/projects", "/p/" + f.pid, "/p/" + f.pid + "/t/" + f.ids["one"], "/devices"} {
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

func rowNames(rows []rowView) string {
	var out []string
	for _, r := range rows {
		out = append(out, r.Title)
		for _, c := range r.Children {
			out = append(out, ">"+c.Title)
		}
	}
	return strings.Join(out, ",")
}

func TestCompletedTakesOnlyFinishedBlocks(t *testing.T) {
	p, err := domain.NewProject("Phone")
	if err != nil {
		t.Fatal(err)
	}
	cat, _ := p.AddCategoryUnder("", "Feature")
	add := func(f operations.TaskFields) string {
		task, err := operations.CreateTask(&p, cat.ID, f)
		if err != nil {
			t.Fatal(err)
		}
		return task.ID
	}
	one := add(operations.TaskFields{Title: "one"})
	sub := add(operations.TaskFields{Title: "sub", ParentID: one})
	two := add(operations.TaskFields{Title: "two"})
	add(operations.TaskFields{Kind: domain.KindSeparator, Title: "Later"})
	three := add(operations.TaskFields{Title: "three"})
	if _, err := operations.SetStatuses(&p, []string{one, two}, domain.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	if _, err := operations.SetStatuses(&p, []string{three}, domain.StatusCancelled); err != nil {
		t.Fatal(err)
	}
	c := buildCategories(&p, nil)[0]
	if got := rowNames(c.Rows); got != "one,>sub,Later" {
		t.Errorf("open rows: %s", got)
	}
	if got := rowNames(c.Completed); got != "two,three" {
		t.Errorf("completed: %s", got)
	}
	if !c.OpenTasks() {
		t.Error("a finished parent with an open subtask counts as open")
	}
	if _, err := operations.SetStatuses(&p, []string{sub}, domain.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	c = buildCategories(&p, nil)[0]
	if got := rowNames(c.Rows); got != "Later" {
		t.Errorf("open rows once sub is done: %s", got)
	}
	if got := rowNames(c.Completed); got != "one,>sub,two,three" {
		t.Errorf("completed once sub is done: %s", got)
	}
	if c.OpenTasks() {
		t.Error("a lone separator counts as an open task")
	}
}

func TestProjectPagesShowNoProgress(t *testing.T) {
	f := newFixture(t)
	body := f.do(http.MethodGet, "/p/"+f.pid, nil, false).Body.String()
	if !strings.Contains(body, "No tasks yet") || strings.Contains(body, "Completed (") {
		t.Error("an empty category says No tasks yet, with no Completed fold")
	}
	f.action("status.set", url.Values{"id": {f.ids["one"], f.ids["two"], f.ids["three"]}, "value": {domain.StatusCompleted}})
	body = f.do(http.MethodGet, "/p/"+f.pid, nil, false).Body.String()
	for _, want := range []string{"All tasks complete", "Completed (3)", `<details class="completed">`} {
		if !strings.Contains(body, want) {
			t.Errorf("project page lacks %q", want)
		}
	}
	for _, gone := range []string{"<progress", "of 3 done", "hidden by the filter", "?f="} {
		if strings.Contains(body, gone) {
			t.Errorf("project page still shows %q", gone)
		}
	}
	body = f.do(http.MethodGet, "/projects", nil, false).Body.String()
	if !strings.Contains(body, "0 open") || strings.Contains(body, " done") || strings.Contains(body, "%<") {
		t.Errorf("projects page counts: %s", body[strings.Index(body, "supporting"):][:80])
	}
}

func TestCompletedFoldOpensFromTheCookie(t *testing.T) {
	f := newFixture(t)
	f.action("status.set", url.Values{"id": {f.ids["one"]}, "value": {domain.StatusCompleted}})
	if rec := f.do(http.MethodPost, "/fold/"+completedFold(f.cat), nil, true); rec.Code != http.StatusNoContent {
		t.Fatalf("fold: %d", rec.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/p/"+f.pid, nil)
	req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	req.AddCookie(&http.Cookie{Name: foldCookie, Value: completedFold(f.cat)})
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), `<details class="completed" open>`) {
		t.Fatal("the Completed fold stays closed under its cookie key")
	}
}

func TestPreviewDropsImagesAndKeepsTwoLines(t *testing.T) {
	img := "![a](attachments/0123456789abcdef0123456789abcdef.png)"
	cases := map[string]string{
		"":                                     "",
		"one":                                  "one",
		img + "\n\nfirst  line\nsecond\nthird": "first line\nsecond",
		"text " + img + " more":                "text more",
		img:                                    "",
	}
	for desc, want := range cases {
		if got := preview(desc); got != want {
			t.Errorf("preview(%q) = %q, want %q", desc, got, want)
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

// The page keeps the typed text and offers Keep mine or Use theirs, reading theirs from the page this answers with.
func TestStaleTitleFlagsAConflictAndRendersTheirs(t *testing.T) {
	f := newFixture(t)
	rec := f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"uno"}, "base": {"old"}, "back": {"/p/" + f.pid + "/t/" + f.ids["one"]}})
	if sn := snackOf(t, rec); sn.Msg != "" || rec.Header().Get("X-Ordo-Conflict") != "1" {
		t.Fatalf("snack %+v conflict %q", sn, rec.Header().Get("X-Ordo-Conflict"))
	}
	if !strings.Contains(rec.Body.String(), `name="base" value="one" data-base`) {
		t.Fatal("the answer doesn't render their title as the field's base")
	}
	f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"uno"}, "base": {"one"}})
	if title := f.task(f.ids["one"]).Title; title != "uno" {
		t.Fatalf("keep mine on their base: %q", title)
	}
}

// Keep mine resends the typed details on their base, keeping the images they hold.
func TestStaleDetailsKeepMineOnTheirBase(t *testing.T) {
	f := newFixture(t)
	img := "![shot](attachments/0123456789abcdef0123456789abcdef.png)"
	if _, err := f.store.WithProjectLocked(f.pid, func(p *domain.Project) error {
		p.Categories[0].Tasks[0].Description = "their text\n" + img
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rec := f.action("task.description", url.Values{"id": {f.ids["one"]}, "base": {""}, "text": {"my text"}, "back": {"/p/" + f.pid + "/t/" + f.ids["one"]}})
	body := rec.Body.String()
	if rec.Header().Get("X-Ordo-Conflict") != "1" || !strings.Contains(body, ">their text</textarea>") || !strings.Contains(body, `name="image" value="`+html.EscapeString(img)+`"`) {
		t.Fatalf("conflict %q, rendered theirs %v", rec.Header().Get("X-Ordo-Conflict"), strings.Contains(body, "their text"))
	}
	if f.task(f.ids["one"]).Description != "their text\n"+img {
		t.Fatalf("on disk %q", f.task(f.ids["one"]).Description)
	}
	f.action("task.description", url.Values{"id": {f.ids["one"]}, "base": {"their text\n" + img}, "text": {"my text"}, "image": {img}})
	if desc := f.task(f.ids["one"]).Description; desc != "my text\n"+img {
		t.Fatalf("keep mine = %q", desc)
	}
}

// A proxy refuses an answer whose headers outgrow its buffer: nginx's default is 4 or 8 KB.
func TestLongDetailsNeverRideInAHeader(t *testing.T) {
	f := newFixture(t)
	long := strings.Repeat("a long line of details that keeps going\n", 500)
	assertSmallHeaders := func(rec *httptest.ResponseRecorder) {
		t.Helper()
		for name, values := range rec.Header() {
			for _, v := range values {
				if len(v) > 512 {
					t.Fatalf("%s carries %d bytes", name, len(v))
				}
			}
		}
	}
	rec := f.action("task.description", url.Values{"id": {f.ids["one"]}, "base": {""}, "text": {long}})
	assertSmallHeaders(rec)
	if rec.Header().Get("X-Ordo-Saved") != "1" || f.task(f.ids["one"]).Description != strings.TrimRight(long, "\n") {
		t.Fatalf("saved %q, %d bytes on disk", rec.Header().Get("X-Ordo-Saved"), len(f.task(f.ids["one"]).Description))
	}
	rec = f.action("task.description", url.Values{"id": {f.ids["one"]}, "base": {""}, "text": {"stale " + long}, "back": {"/p/" + f.pid + "/t/" + f.ids["one"]}})
	assertSmallHeaders(rec)
	if rec.Header().Get("X-Ordo-Conflict") != "1" || !strings.Contains(rec.Body.String(), "a long line of details") {
		t.Fatalf("conflict %q", rec.Header().Get("X-Ordo-Conflict"))
	}
}

func TestDeletedTaskPageLandsOnItsProject(t *testing.T) {
	f := newFixture(t)
	f.action("task.delete", url.Values{"id": {f.ids["one"]}})
	path := "/p/" + f.pid + "/t/" + f.ids["one"]
	rec := f.do(http.MethodGet, path, nil, true)
	if rec.Code != http.StatusOK || rec.Header().Get("HX-Push-Url") != "/p/"+f.pid || !strings.Contains(rec.Body.String(), `data-page="project"`) {
		t.Fatalf("refresh: %d, push %q", rec.Code, rec.Header().Get("HX-Push-Url"))
	}
}

func TestTextSaveOnADeletedTaskFlagsItGone(t *testing.T) {
	f := newFixture(t)
	f.action("task.delete", url.Values{"id": {f.ids["one"]}})
	rec := f.action("task.title", url.Values{"id": {f.ids["one"]}, "title": {"typed"}, "base": {"one"}})
	if rec.Header().Get("X-Ordo-Gone") != "1" || snackOf(t, rec).Msg != "That task is gone" {
		t.Fatalf("gone %q, snack %+v", rec.Header().Get("X-Ordo-Gone"), snackOf(t, rec))
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
