package web

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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
	ids := map[string]string{"other": other.ID}
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

func (f *fixture) withCookie(method, path string, body any) *httptest.ResponseRecorder {
	f.t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func (f *fixture) task(id string) domain.Task {
	f.t.Helper()
	p, err := f.store.LoadProjectByID(f.pid)
	if err != nil {
		f.t.Fatal(err)
	}
	ci, ti, err := operations.LocateTask(&p, id)
	if err != nil {
		f.t.Fatal(err)
	}
	return p.Categories[ci].Tasks[ti]
}

func TestPairingCodesWorkOnce(t *testing.T) {
	f := newFixture(t)
	code, err := f.auth.NewPairing()
	if err != nil {
		t.Fatal(err)
	}
	if rec := f.api(http.MethodPost, "/api/pair", "", pairRequest{Token: code, Name: "Pixel"}); rec.Code != http.StatusOK {
		t.Fatalf("pair: %d %s", rec.Code, rec.Body)
	}
	if rec := f.api(http.MethodPost, "/api/pair", "", pairRequest{Token: code}); rec.Code != http.StatusForbidden {
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

func TestADeviceMintsOneLiveCodeThatRecordsWhoPaired(t *testing.T) {
	f := newFixture(t)
	minter, _, _ := strings.Cut(f.cookie, ".")
	mint := func() string {
		var minted struct{ Link string }
		rec := f.api(http.MethodPost, "/api/devices/pair", f.cookie, nil)
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &minted) != nil {
			t.Fatalf("mint: %d %s", rec.Code, rec.Body)
		}
		return minted.Link
	}
	first, second := mint(), mint()
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

func TestPassphraseSignsInAndLimitsGuesses(t *testing.T) {
	f := newFixture(t)
	if err := f.auth.SetPassphrase("correct horse battery"); err != nil {
		t.Fatal(err)
	}
	login := func(phrase string) int {
		return f.api(http.MethodPost, "/api/login", "", map[string]string{"passphrase": phrase}).Code
	}
	if code := login("correct horse battery"); code != http.StatusOK {
		t.Fatalf("login: %d", code)
	}
	for i := range 5 {
		if code := login("nope"); code != http.StatusForbidden {
			t.Fatalf("guess %d: %d", i, code)
		}
	}
	if code := login("correct horse battery"); code != http.StatusTooManyRequests {
		t.Fatalf("after five failures: %d", code)
	}
}

func TestRevokedDeviceIsSignedOut(t *testing.T) {
	f := newFixture(t)
	id, _, _ := strings.Cut(f.cookie, ".")
	if gone, err := f.auth.Revoke(id); len(gone) != 1 || err != nil {
		t.Fatalf("revoke: %v %v", gone, err)
	}
	if rec := f.api(http.MethodPost, "/api/sync", f.cookie, syncRequest{Protocol: Protocol}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("revoked: %d", rec.Code)
	}
}

func TestDeviceCookieRenewsAfterADay(t *testing.T) {
	f := newFixture(t)
	sync := func() *httptest.ResponseRecorder {
		return f.withCookie(http.MethodPost, "/api/sync", syncRequest{Protocol: Protocol})
	}
	if rec := sync(); rec.Code != http.StatusOK || strings.Contains(rec.Header().Get("Set-Cookie"), deviceCookie) {
		t.Fatalf("a fresh cookie: %d %q", rec.Code, rec.Header().Get("Set-Cookie"))
	}
	f.auth.now = func() time.Time { return time.Now().Add(renewEvery + time.Hour) }
	if rec := sync(); !strings.Contains(rec.Header().Get("Set-Cookie"), deviceCookie+"=") {
		t.Fatal("no renewal")
	}
	f.auth.now = func() time.Time { return time.Now().Add(DeviceIdle + 2*renewEvery) }
	if rec := sync(); rec.Code != http.StatusUnauthorized {
		t.Fatalf("idle device still in: %d", rec.Code)
	}
}

func TestCrossSiteCookieEditsAreRefused(t *testing.T) {
	f := newFixture(t)
	body, _ := json.Marshal(syncRequest{Protocol: Protocol, Ops: []queuedOp{f.queued("q1", operations.Op{Action: "status.set", IDs: []string{f.ids["one"]}, Value: domain.StatusCompleted})}})
	req := httptest.NewRequest(http.MethodPost, "/api/sync", bytes.NewReader(body))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site: %d", rec.Code)
	}
	if f.task(f.ids["one"]).Status != domain.StatusTodo {
		t.Fatal("a cross-site request changed the task")
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

func appServer(t *testing.T, files fstest.MapFS) http.Handler {
	t.Helper()
	a, err := loadApp(files)
	if err != nil {
		t.Fatal(err)
	}
	return secureHeaders(a.scripts, a)
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestTheAppRoutesItself(t *testing.T) {
	const boot = "\n\t\t\t\timport('/_app/start.js');\n\t\t\t"
	h := appServer(t, fstest.MapFS{
		"spa/index.html":              {Data: []byte("<html><script>" + boot + "</script></html>")},
		"spa/placeholder.html":        {Data: []byte("placeholder")},
		"spa/_app/immutable/start.js": {Data: []byte("start()")},
		"spa/ordo-core.wasm":          {Data: []byte("\x00asm")},
		"spa/_app/version.json":       {Data: []byte("{}")},
	})
	for _, path := range []string{"/", "/p/x/t/y", "/pair/CODE", "/devices", "/missing.js"} {
		rec := get(h, path)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "import(") || rec.Header().Get("Cache-Control") != "no-cache" {
			t.Errorf("%s: %d %q %q", path, rec.Code, rec.Body, rec.Header().Get("Cache-Control"))
		}
	}
	if rec := get(h, "/_app/immutable/start.js"); rec.Body.String() != "start()" || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("hashed asset: %q %q", rec.Body, rec.Header().Get("Cache-Control"))
	}
	if rec := get(h, "/ordo-core.wasm"); rec.Header().Get("Content-Type") != "application/wasm" || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Fatalf("wasm: %q %q", rec.Header().Get("Content-Type"), rec.Header().Get("Cache-Control"))
	}
	sum := sha256.Sum256([]byte(boot))
	csp := get(h, "/").Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'") || !strings.Contains(csp, "'wasm-unsafe-eval'") {
		t.Fatalf("csp = %q", csp)
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /: %d", rec.Code)
	}
}

func TestABuildWithoutTheAppShowsThePlaceholder(t *testing.T) {
	h := appServer(t, fstest.MapFS{"spa/placeholder.html": {Data: []byte("built without its web app")}})
	if rec := get(h, "/p/x"); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "without its web app") {
		t.Fatalf("placeholder: %d %q", rec.Code, rec.Body)
	}
}

func TestEventsNameTheProjectThatChanged(t *testing.T) {
	f := newFixture(t)
	srv := httptest.NewServer(f.h)
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/api/events", nil)
	req.Header.Set("Authorization", "Bearer "+f.cookie)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	lines := bufio.NewScanner(resp.Body)
	for lines.Scan() && lines.Text() != "" {
	}
	f.srv.hub.publish(f.pid)
	for lines.Scan() {
		if pid, ok := strings.CutPrefix(lines.Text(), "data: "); ok {
			if pid != f.pid {
				t.Fatalf("data = %q", pid)
			}
			return
		}
	}
	t.Fatal("the stream ended without an event")
}
