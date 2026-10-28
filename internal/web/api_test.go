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
	if len(resp.Projects) != 1 || resp.Projects[0].Data == nil || resp.Projects[0].Stamp != f.srv.version(f.pid) {
		t.Fatalf("projects = %+v", resp.Projects)
	}
	if _, _, err := locate(resp.Projects[0].Data, "phone-made"); err != nil {
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
