package web

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func (f *fixture) api(method, path, bearer string, body any, header ...string) *httptest.ResponseRecorder {
	f.t.Helper()
	var r io.Reader
	switch b := body.(type) {
	case nil:
	case []byte:
		r = bytes.NewReader(b)
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			f.t.Fatal(err)
		}
		r = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, r)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func (f *fixture) sync(ops []queuedOp, have map[string]string) syncResponse {
	f.t.Helper()
	rec := f.api(http.MethodPost, "/api/sync", f.cookie, syncRequest{Protocol: Protocol, Ops: ops, Have: have})
	if rec.Code != http.StatusOK {
		f.t.Fatalf("sync: %d %s", rec.Code, rec.Body)
	}
	var resp syncResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		f.t.Fatal(err)
	}
	return resp
}

func (f *fixture) queued(id string, op operations.Op) queuedOp {
	return queuedOp{ID: id, Project: f.pid, Op: op}
}

func statuses(rs []opResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = string(r.Status)
	}
	return out
}

func TestSyncAppliesTheQueueOnceAndSendsWhatChanged(t *testing.T) {
	f := newFixture(t)
	ops := []queuedOp{
		f.queued("1", operations.Op{Action: "status.set", IDs: []string{f.ids["one"]}, Value: domain.StatusCompleted}),
		f.queued("2", operations.Op{Action: "task.add", New: "phone-made", Cat: f.cat, Title: "on the train"}),
	}
	resp := f.sync(ops, nil)
	if got := strings.Join(statuses(resp.Results), ","); got != "applied,applied" {
		t.Fatalf("results = %s", got)
	}
	if len(resp.Projects) != 1 || resp.Projects[0].Data == nil || resp.Projects[0].Stamp != f.store.Stamp(f.pid) {
		t.Fatalf("projects = %+v", resp.Projects)
	}
	if _, _, err := operations.LocateTask(resp.Projects[0].Data, "phone-made"); err != nil {
		t.Fatal("the answer doesn't hold the added task")
	}
	again := f.sync(ops, map[string]string{f.pid: resp.Projects[0].Stamp})
	if got := strings.Join(statuses(again.Results), ","); got != "unchanged,unchanged" {
		t.Fatalf("replay = %s", got)
	}
	if again.Projects[0].Data != nil {
		t.Fatal("an unchanged project came back with its data")
	}
}

func TestSyncRunsPastARefusedOp(t *testing.T) {
	f := newFixture(t)
	resp := f.sync([]queuedOp{
		f.queued("1", operations.Op{Action: "task.title", IDs: []string{f.ids["one"]}, Title: "mine", Base: "not what it was"}),
		f.queued("2", operations.Op{Action: "status.cycle", IDs: []string{f.ids["one"]}}),
		f.queued("3", operations.Op{Action: "task.levitate", IDs: []string{f.ids["one"]}}),
		f.queued("4", operations.Op{Action: "status.set", IDs: []string{"gone-task"}, Value: domain.StatusCompleted}),
		f.queued("5", operations.Op{Action: "priority.set", IDs: []string{f.ids["one"]}, Value: domain.PriorityHigh}),
		{ID: "6", Project: "no-such-project", Op: operations.Op{Action: "status.set", IDs: []string{f.ids["one"]}, Value: domain.StatusCompleted}},
	}, nil)
	if got := strings.Join(statuses(resp.Results), ","); got != "conflict,refused,refused,gone,applied,gone" {
		t.Fatalf("results = %s", got)
	}
	if th := resp.Results[0].Theirs; th == nil || *th != "one" {
		t.Fatalf("theirs = %v", th)
	}
	if task := f.task(f.ids["one"]); task.Title != "one" || task.Status != domain.StatusTodo || task.Priority != domain.PriorityHigh {
		t.Fatalf("task = %+v", task)
	}
}

func TestSyncRefusesAnotherProtocol(t *testing.T) {
	f := newFixture(t)
	rec := f.api(http.MethodPost, "/api/sync", f.cookie, syncRequest{Protocol: Protocol + 1})
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), `"protocol":1`) {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestSyncGzipsForAClientThatTakesIt(t *testing.T) {
	f := newFixture(t)
	rec := f.api(http.MethodPost, "/api/sync", f.cookie, syncRequest{Protocol: Protocol}, "Accept-Encoding", "gzip")
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("encoding %q", rec.Header().Get("Content-Encoding"))
	}
	gz, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatal(err)
	}
	var resp syncResponse
	if err := json.NewDecoder(gz).Decode(&resp); err != nil || len(resp.Projects) != 1 {
		t.Fatalf("%v %+v", err, resp)
	}
}

func TestAPIPairHandsOutABearerToken(t *testing.T) {
	f := newFixture(t)
	code, err := f.auth.NewPairing()
	if err != nil {
		t.Fatal(err)
	}
	rec := f.api(http.MethodPost, "/api/pair", "", pairRequest{Token: code, Name: "Pixel"}, "Origin", "https://localhost", "Sec-Fetch-Site", "cross-site")
	var paired struct{ Token string }
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &paired) != nil || paired.Token == "" {
		t.Fatalf("pair: %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Set-Cookie") != "" {
		t.Fatal("a bearer pairing also set a cookie")
	}
	if rec := f.api(http.MethodPost, "/api/sync", paired.Token, syncRequest{Protocol: Protocol}); rec.Code != http.StatusOK {
		t.Fatalf("sync with the token: %d", rec.Code)
	}
	if rec := f.api(http.MethodPost, "/api/pair", "", pairRequest{Token: code}); rec.Code != http.StatusForbidden {
		t.Fatalf("second use: %d", rec.Code)
	}
}

func TestAPIPairCanAnswerWithACookie(t *testing.T) {
	f := newFixture(t)
	code, _ := f.auth.NewPairing()
	rec := f.api(http.MethodPost, "/api/pair", "", pairRequest{Token: code, Cookie: true})
	if !strings.Contains(rec.Header().Get("Set-Cookie"), deviceCookie+"=") || strings.Contains(rec.Body.String(), ".") {
		t.Fatalf("cookie %q, body %s", rec.Header().Get("Set-Cookie"), rec.Body)
	}
}

func TestAPIWantsADevice(t *testing.T) {
	f := newFixture(t)
	for _, bearer := range []string{"", "nobody.nothing"} {
		if rec := f.api(http.MethodPost, "/api/sync", bearer, syncRequest{Protocol: Protocol}); rec.Code != http.StatusUnauthorized {
			t.Fatalf("bearer %q: %d", bearer, rec.Code)
		}
	}
}

func TestAPIChecksTheOriginOfACookieButNotOfABearer(t *testing.T) {
	f := newFixture(t)
	body, _ := json.Marshal(syncRequest{Protocol: Protocol})
	req := httptest.NewRequest(http.MethodPost, "/api/sync", bytes.NewReader(body))
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site cookie: %d", rec.Code)
	}
	if rec := f.api(http.MethodPost, "/api/sync", f.cookie, body, "Sec-Fetch-Site", "cross-site", "Origin", "https://localhost"); rec.Code != http.StatusOK || rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("cross-site bearer: %d %q", rec.Code, rec.Header().Get("Access-Control-Allow-Origin"))
	}
	for _, site := range []string{"", "same-origin"} {
		req := httptest.NewRequest(http.MethodPost, "/api/sync", bytes.NewReader(body))
		if site != "" {
			req.Header.Set("Sec-Fetch-Site", site)
		}
		req.AddCookie(&http.Cookie{Name: deviceCookie, Value: f.cookie})
		rec := httptest.NewRecorder()
		f.h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("cookie from %q: %d", site, rec.Code)
		}
	}
	rec = f.api(http.MethodOptions, "/api/sync", "", nil, "Origin", "https://localhost", "Access-Control-Request-Method", "POST")
	if rec.Code != http.StatusNoContent || !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") {
		t.Fatalf("preflight: %d %v", rec.Code, rec.Header())
	}
}

func TestPutAttachmentWantsTheNameOfItsBytes(t *testing.T) {
	f := newFixture(t)
	var buf bytes.Buffer
	png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3, 3)))
	sum := sha256.Sum256(buf.Bytes())
	name := hex.EncodeToString(sum[:])[:32] + ".png"
	if rec := f.api(http.MethodPut, "/api/attachments/"+strings.Repeat("0", 32)+".png", f.cookie, buf.Bytes()); rec.Code != http.StatusBadRequest {
		t.Fatalf("wrong name: %d", rec.Code)
	}
	if rec := f.api(http.MethodPut, "/api/attachments/"+name, f.cookie, buf.Bytes()); rec.Code != http.StatusNoContent {
		t.Fatalf("put: %d %s", rec.Code, rec.Body)
	}
	if rec := f.api(http.MethodGet, "/api/attachments/"+name, f.cookie, nil); rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), buf.Bytes()) {
		t.Fatalf("get: %d", rec.Code)
	}
}

func TestSyncCreatesRenamesAndDeletesProjects(t *testing.T) {
	f := newFixture(t)
	op := func(id, pid string, o operations.Op) queuedOp { return queuedOp{ID: id, Project: pid, Op: o} }
	ops := []queuedOp{
		op("1", "phone-made", operations.Op{Action: "project.create", Title: " Laptop "}),
		op("2", "phone-made", operations.Op{Action: "category.add", New: "inbox", Title: "Inbox"}),
		op("3", "phone-made", operations.Op{Action: "task.add", New: "first", Cat: "inbox", Title: "first"}),
		op("4", "other", operations.Op{Action: "project.create", Title: "phone"}),
		op("5", "phone-made", operations.Op{Action: "project.rename", Title: "Desk"}),
		op("6", f.pid, operations.Op{Action: "project.rename", Title: "desk"}),
		op("7", "no-such-project", operations.Op{Action: "project.rename", Title: "x"}),
		op("8", f.pid, operations.Op{Action: "status.set", IDs: []string{f.ids["one"]}, Value: domain.StatusCompleted}),
	}
	resp := f.sync(ops, nil)
	if got := strings.Join(statuses(resp.Results), ","); got != "applied,applied,applied,refused,applied,refused,gone,applied" {
		t.Fatalf("results = %s", got)
	}
	if msg := resp.Results[3].Msg; msg != "A project with that name already exists" {
		t.Fatalf("duplicate name: %q", msg)
	}
	var made *domain.Project
	for _, p := range resp.Projects {
		if p.ID == "phone-made" {
			made = p.Data
		}
	}
	if made == nil || made.Name != "Desk" || len(made.Categories) != 1 || len(made.Categories[0].Tasks) != 1 || made.Categories[0].Tasks[0].ID != "first" {
		t.Fatalf("created project = %+v", made)
	}
	again := f.sync(ops, nil)
	if got := strings.Join(statuses(again.Results), ","); got != "unchanged,unchanged,unchanged,refused,unchanged,refused,gone,unchanged" {
		t.Fatalf("replay = %s", got)
	}
	if again.Results[0].Msg != "" {
		t.Fatalf("an unchanged op says %q", again.Results[0].Msg)
	}
	deleted := f.sync([]queuedOp{op("9", "phone-made", operations.Op{Action: "project.delete"})}, nil)
	if deleted.Results[0].Status != operations.Applied || len(deleted.Projects) != 1 {
		t.Fatalf("delete = %+v", deleted)
	}
	if again := f.sync([]queuedOp{op("9", "phone-made", operations.Op{Action: "project.delete"})}, nil); again.Results[0].Status != operations.Unchanged {
		t.Fatalf("replayed delete = %+v", again.Results[0])
	}
}

func TestAPIHelloSaysWhetherToOfferThePassphrase(t *testing.T) {
	f := newFixture(t)
	hello := func() (h struct {
		Protocol      int  `json:"protocol"`
		HasPassphrase bool `json:"has_passphrase"`
	}) {
		rec := f.api(http.MethodGet, "/api/hello", "", nil)
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &h) != nil {
			t.Fatalf("hello: %d %s", rec.Code, rec.Body)
		}
		return h
	}
	if h := hello(); h.Protocol != Protocol || h.HasPassphrase {
		t.Fatalf("hello = %+v", h)
	}
	if err := f.auth.SetPassphrase("correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if !hello().HasPassphrase {
		t.Fatal("the passphrase isn't offered")
	}
}

func TestAPILoginSignsInOnThePassphrase(t *testing.T) {
	f := newFixture(t)
	if rec := f.api(http.MethodPost, "/api/login", "", loginRequest{Passphrase: "anything"}); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "No passphrase is set") {
		t.Fatalf("without a passphrase: %d %s", rec.Code, rec.Body)
	}
	if err := f.auth.SetPassphrase("correct horse battery"); err != nil {
		t.Fatal(err)
	}
	rec := f.api(http.MethodPost, "/api/login", "", loginRequest{Passphrase: "correct horse battery", Name: "Pixel"})
	var signed struct{ Token string }
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &signed) != nil || signed.Token == "" {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	if rec := f.api(http.MethodGet, "/api/devices", signed.Token, nil); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"Pixel"`) {
		t.Fatalf("devices with the token: %d %s", rec.Code, rec.Body)
	}
	rec = f.api(http.MethodPost, "/api/login", "", loginRequest{Passphrase: "correct horse battery", Cookie: true})
	if !strings.Contains(rec.Header().Get("Set-Cookie"), deviceCookie+"=") || strings.Contains(rec.Body.String(), ".") {
		t.Fatalf("cookie %q, body %s", rec.Header().Get("Set-Cookie"), rec.Body)
	}
	for i := 0; i < 5; i++ {
		if rec := f.api(http.MethodPost, "/api/login", "", loginRequest{Passphrase: "nope"}); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Wrong passphrase") {
			t.Fatalf("guess %d: %d %s", i, rec.Code, rec.Body)
		}
	}
	if rec := f.api(http.MethodPost, "/api/login", "", loginRequest{Passphrase: "correct horse battery"}); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("after five failures: %d", rec.Code)
	}
}

type devicesAnswer struct {
	Devices []struct {
		ID   string   `json:"id"`
		Name string   `json:"name"`
		From string   `json:"from"`
		Also []string `json:"also"`
		Self bool     `json:"self"`
	} `json:"devices"`
	HasPassphrase bool `json:"has_passphrase"`
}

func (f *fixture) pairChild(name string) (id, token string) {
	f.t.Helper()
	root, _, _ := strings.Cut(f.cookie, ".")
	code, err := f.auth.NewPairingBy(root)
	if err != nil {
		f.t.Fatal(err)
	}
	token, err = f.auth.Pair(code, name)
	if err != nil {
		f.t.Fatal(err)
	}
	id, _, _ = strings.Cut(token, ".")
	return id, token
}

func TestAPIDevicesSaysWhoPairedWhom(t *testing.T) {
	f := newFixture(t)
	child, _ := f.pairChild("Laptop")
	rec := f.api(http.MethodGet, "/api/devices", f.cookie, nil)
	var got devicesAnswer
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &got) != nil || len(got.Devices) != 2 {
		t.Fatalf("devices: %d %s", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "token") {
		t.Fatal("the answer carries a token hash")
	}
	root, laptop := got.Devices[0], got.Devices[1]
	if !root.Self || root.Name != "test" || strings.Join(root.Also, ",") != "Laptop" || root.From != "" {
		t.Fatalf("this device = %+v", root)
	}
	if laptop.Self || laptop.ID != child || laptop.From != "test" || len(laptop.Also) != 0 {
		t.Fatalf("the laptop = %+v", laptop)
	}
	if rec := f.api(http.MethodGet, "/api/devices", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: %d", rec.Code)
	}
}

func TestAPIPairMintsALinkAfterThePassphrase(t *testing.T) {
	f := newFixture(t)
	var minted struct {
		Link    string `json:"link"`
		QR      string `json:"qr"`
		Minutes int    `json:"minutes"`
	}
	rec := f.api(http.MethodPost, "/api/devices/pair", f.cookie, nil)
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &minted) != nil {
		t.Fatalf("pair: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(minted.Link, "/pair/") || !strings.HasPrefix(minted.QR, "data:image/png;base64,") || minted.Minutes != 10 {
		t.Fatalf("minted %+v", minted)
	}
	if _, err := f.auth.Pair(minted.Link, "Tablet"); err != nil {
		t.Fatalf("the link doesn't pair: %v", err)
	}
	if err := f.auth.SetPassphrase("correct horse battery"); err != nil {
		t.Fatal(err)
	}
	if rec := f.api(http.MethodPost, "/api/devices/pair", f.cookie, map[string]string{"passphrase": "nope"}); rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "Wrong passphrase") {
		t.Fatalf("wrong passphrase: %d %s", rec.Code, rec.Body)
	}
	if rec := f.api(http.MethodPost, "/api/devices/pair", f.cookie, map[string]string{"passphrase": "correct horse battery"}); rec.Code != http.StatusOK {
		t.Fatalf("right passphrase: %d %s", rec.Code, rec.Body)
	}
}

func TestAPIRevokeSaysWhetherTheCallerLostAccess(t *testing.T) {
	f := newFixture(t)
	root, _, _ := strings.Cut(f.cookie, ".")
	other, _ := f.pairChild("Laptop")
	_, phone := f.pairChild("Phone")
	var answer struct{ Self bool }
	rec := f.api(http.MethodPost, "/api/devices/"+other+"/revoke", phone, nil)
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &answer) != nil || answer.Self {
		t.Fatalf("revoke another: %d %s", rec.Code, rec.Body)
	}
	rec = f.api(http.MethodPost, "/api/devices/"+root+"/revoke", phone, nil)
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &answer) != nil || !answer.Self {
		t.Fatalf("revoke the device that paired this one: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("cookie kept: %q", rec.Header().Get("Set-Cookie"))
	}
	if rec := f.api(http.MethodGet, "/api/devices", phone, nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after losing access: %d", rec.Code)
	}
	if rec := f.api(http.MethodPost, "/api/devices/"+other+"/revoke", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous revoke: %d", rec.Code)
	}
}

func TestAPILogoutSignsTheCallerOut(t *testing.T) {
	f := newFixture(t)
	_, laptop := f.pairChild("Laptop")
	if rec := f.api(http.MethodPost, "/api/logout", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous logout: %d", rec.Code)
	}
	rec := f.api(http.MethodPost, "/api/logout", laptop, nil)
	if rec.Code != http.StatusNoContent || !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") {
		t.Fatalf("logout: %d %q", rec.Code, rec.Header().Get("Set-Cookie"))
	}
	if rec := f.api(http.MethodPost, "/api/sync", laptop, syncRequest{Protocol: Protocol}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("after logout: %d", rec.Code)
	}
	if rec := f.api(http.MethodPost, "/api/sync", f.cookie, syncRequest{Protocol: Protocol}); rec.Code != http.StatusOK {
		t.Fatalf("the device that paired it: %d", rec.Code)
	}
}
