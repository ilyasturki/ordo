package web

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

// Protocol is the version of /api/sync; docs/sync.md says when it goes up.
const Protocol = 1

const maxSyncBody = 16 << 20

func (s *Server) api() http.Handler {
	authed := http.NewServeMux()
	authed.HandleFunc("POST /api/sync", s.sync)
	authed.HandleFunc("GET /api/attachments/{name}", s.image)
	authed.HandleFunc("PUT /api/attachments/{name}", s.putAttachment)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/pair", s.apiPair)
	mux.Handle("/api/", s.apiDevice(authed))
	return cors(mux)
}

// cors lets the phone app, served from its own origin, call the API with its bearer token.
// The wildcard never reaches a cookie: browsers send none to a wildcard origin.
func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// apiDevice takes the phone app's bearer token, or the browser app's cookie under the
// same cross-origin check as the pages.
func (s *Server) apiDevice(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, bearer := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !bearer {
			if err := s.csrf.Check(r); err != nil {
				apiError(w, http.StatusForbidden, "Cross-origin request refused")
				return
			}
			if c, err := r.Cookie(deviceCookie); err == nil {
				token = c.Value
			}
		}
		dev, renew, ok := s.auth.Check(token)
		if !ok {
			apiError(w, http.StatusUnauthorized, "Sign in again")
			return
		}
		if renew && !bearer {
			s.setCookie(w, r, deviceCookie, token, cookieAge)
		}
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, dev)))
	})
}

func apiError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"error": msg, "protocol": Protocol})
}

// writeJSON gzips the answer when the client takes it: a project is a few hundred KB on a slow line.
func writeJSON(w http.ResponseWriter, r *http.Request, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Add("Vary", "Accept-Encoding")
	var out io.Writer = w
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		out = gz
	}
	json.NewEncoder(out).Encode(v)
}

type pairRequest struct {
	Token string `json:"token"`
	Name  string `json:"name"`
	// Cookie asks for the device token as an HttpOnly cookie instead of in the answer.
	Cookie bool `json:"cookie"`
}

func (s *Server) apiPair(w http.ResponseWriter, r *http.Request) {
	if s.limiter.blocked("pair:" + clientIP(r)) {
		apiError(w, http.StatusTooManyRequests, "Too many attempts. Wait a few minutes.")
		return
	}
	var req pairRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "Bad request")
		return
	}
	name := truncate(strings.TrimSpace(req.Name), 60)
	if name == "" {
		name = deviceName(r)
	}
	token, err := s.auth.Pair(req.Token, name)
	if err != nil {
		s.limiter.fail("pair:" + clientIP(r))
		if !errors.Is(err, ErrBadPairing) {
			s.log.Printf("pair: %v", err)
		}
		apiError(w, http.StatusForbidden, "This pairing code is used or expired. Get a new one from Devices on a signed-in device, or run ordo serve pair.")
		return
	}
	if req.Cookie {
		s.setCookie(w, r, deviceCookie, token, cookieAge)
		token = ""
	}
	writeJSON(w, r, map[string]any{"token": token, "protocol": Protocol})
}

type syncRequest struct {
	Protocol int        `json:"protocol"`
	Ops      []queuedOp `json:"ops"`
	// Have is the stamp of each project the client holds; a project whose file still has it comes back without data.
	Have map[string]string `json:"have"`
}

type queuedOp struct {
	ID      string `json:"id"`
	Project string `json:"project"`
	operations.Op
}

type opResult struct {
	ID     string             `json:"id"`
	Status operations.Verdict `json:"status"`
	Msg    string             `json:"msg,omitempty"`
	Theirs *string            `json:"theirs,omitempty"`
}

type projectState struct {
	ID    string          `json:"id"`
	Name  string          `json:"name,omitempty"`
	Stamp string          `json:"stamp"`
	Data  *domain.Project `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

type syncResponse struct {
	Protocol int            `json:"protocol"`
	Results  []opResult     `json:"results"`
	Projects []projectState `json:"projects"`
}

func (s *Server) sync(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxSyncBody)).Decode(&req); err != nil {
		apiError(w, http.StatusBadRequest, "Bad request")
		return
	}
	if req.Protocol != Protocol {
		apiError(w, http.StatusConflict, "The app and ordo serve speak different versions: update the older one")
		return
	}
	results := make([]opResult, len(req.Ops))
	for i := 0; i < len(req.Ops); {
		j := i + 1
		for j < len(req.Ops) && req.Ops[j].Project == req.Ops[i].Project {
			j++
		}
		s.applyRun(req.Ops[i:j], results[i:j])
		i = j
	}
	snaps, err := s.store.Snapshots(req.Have)
	if err != nil {
		s.log.Printf("sync: %v", err)
		apiError(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	resp := syncResponse{Protocol: Protocol, Results: results, Projects: make([]projectState, 0, len(snaps))}
	for _, snap := range snaps {
		st := projectState{ID: snap.Project.ID, Name: snap.Project.Name, Stamp: snap.Stamp}
		switch {
		case snap.Err != nil:
			st.Error = snap.Err.Error()
		case !snap.Held:
			st.Data = &snap.Project
		}
		resp.Projects = append(resp.Projects, st)
	}
	writeJSON(w, r, resp)
}

// applyRun applies one project's consecutive ops under one lock and one save; a refused op
// leaves the project as it was and the next one runs.
func (s *Server) applyRun(ops []queuedOp, results []opResult) {
	changed := false
	_, err := s.store.WithProjectLocked(ops[0].Project, func(p *domain.Project) error {
		for k, op := range ops {
			results[k] = applyQueued(p, op)
			changed = changed || results[k].Status == operations.Applied
		}
		if !changed {
			return operations.ErrNoChange
		}
		return nil
	})
	if err == nil || errors.Is(err, operations.ErrNoChange) {
		return
	}
	status, msg := operations.Gone, "That project is gone"
	if !errors.Is(err, data.ErrProjectNotFound) {
		s.log.Printf("sync %s: %v", ops[0].Project, err)
		status, msg = operations.Refused, "The server couldn't save this"
	}
	for k := range results {
		results[k] = opResult{ID: ops[k].ID, Status: status, Msg: msg}
	}
}

func applyQueued(p *domain.Project, op queuedOp) opResult {
	res := opResult{ID: op.ID}
	if !operations.Replayable(op.Action) {
		res.Status, res.Msg = operations.Refused, "This ordo serve doesn't know "+op.Action+": update it"
		return res
	}
	out, err := operations.Apply(p, op.Op)
	res.Status, res.Msg = operations.VerdictOf(err), out.Msg
	if err != nil {
		res.Msg, _ = userMessage(err, true)
	}
	var conflict *operations.Conflict
	if errors.As(err, &conflict) {
		res.Theirs = &conflict.Theirs
	}
	return res
}

// putAttachment stores an image the phone attached offline, under the name it computed from the bytes.
func (s *Server) putAttachment(w http.ResponseWriter, r *http.Request) {
	want := attachments.DirName + "/" + r.PathValue("name")
	if !attachments.Internal(want) {
		apiError(w, http.StatusBadRequest, "Not an image name")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, attachments.MaxSize))
	if err != nil {
		apiError(w, http.StatusRequestEntityTooLarge, "That image is too large")
		return
	}
	target, err := attachments.Target(raw)
	if err != nil {
		apiError(w, http.StatusBadRequest, err.Error())
		return
	}
	if target != want {
		apiError(w, http.StatusBadRequest, "The image doesn't match its name")
		return
	}
	if _, err := s.images.Add(raw); err != nil {
		s.log.Printf("store image: %v", err)
		apiError(w, http.StatusInternalServerError, "The image wasn't stored")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
