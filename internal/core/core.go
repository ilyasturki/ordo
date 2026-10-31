// Package core holds the projects the Android app opened and runs their edits
// through operations, the code `ordo serve` replays them with.
package core

import (
	"encoding/json"
	"errors"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

type Session struct {
	projects map[string]*domain.Project
}

func New() *Session {
	return &Session{projects: make(map[string]*domain.Project)}
}

// Open holds the project in raw, read the way ordo reads a project file, and returns its ID.
func (s *Session) Open(raw []byte) (string, error) {
	var p domain.Project
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", err
	}
	if err := domain.ValidateID(p.ID); err != nil {
		return "", err
	}
	if err := domain.CheckSchema(p); err != nil {
		return "", err
	}
	domain.StripProjectText(&p)
	domain.NormalizeNesting(&p)
	domain.NormalizeTags(&p)
	s.projects[p.ID] = &p
	return p.ID, nil
}

func (s *Session) Close(pid string) { delete(s.projects, pid) }

// Project is the held project as JSON, nil when it isn't open.
func (s *Session) Project(pid string) []byte {
	p, ok := s.projects[pid]
	if !ok {
		return nil
	}
	raw, _ := json.Marshal(p)
	return raw
}

type Answer struct {
	Status operations.Verdict `json:"status"`
	Msg    string             `json:"msg,omitempty"`
	Row    string             `json:"row,omitempty"`
	Saved  bool               `json:"saved,omitempty"`
	Theirs *string            `json:"theirs,omitempty"`
	// Queue is the op that carries the edit to the server, set when it applied.
	Queue *operations.Op `json:"queue,omitempty"`
	// Undo are the ops that take the edit back, performed like any edit, set when it applied.
	Undo []operations.Op `json:"undo,omitempty"`
}

var errGone = errors.New("project not held")

// Perform runs a user's edit; its answer's Queue is what to send. A project.create given no pid
// mints one and answers it as Row.
func (s *Session) Perform(pid string, rawOp []byte) Answer {
	var op operations.Op
	if err := json.Unmarshal(rawOp, &op); err != nil {
		return unreadable(err)
	}
	if op.Action == "project.create" && pid == "" {
		id, err := domain.NewID()
		if err != nil {
			return answer(operations.Outcome{}, err)
		}
		pid = id
	}
	before := operations.Clone(s.held(pid))
	out, wire, err := s.perform(pid, op)
	a := answer(out, err)
	if err == nil {
		a.Queue = &wire
		a.Undo = operations.Undo(before, s.held(pid), wire)
	}
	return a
}

func (s *Session) perform(pid string, op operations.Op) (operations.Outcome, operations.Op, error) {
	if operations.ProjectAction(op.Action) {
		out, err := s.projectOp(pid, op)
		return out, op, err
	}
	p, ok := s.projects[pid]
	if !ok {
		return operations.Outcome{}, operations.Op{}, errGone
	}
	return operations.Perform(p, op)
}

// Replay runs an op from the queue again, on a copy the server just sent.
func (s *Session) Replay(pid string, rawOp []byte) Answer {
	var op operations.Op
	if err := json.Unmarshal(rawOp, &op); err != nil {
		return unreadable(err)
	}
	if operations.ProjectAction(op.Action) {
		return answer(s.projectOp(pid, op))
	}
	if !operations.Replayable(op.Action) {
		return answer(operations.Outcome{}, operations.ErrUnknownAction)
	}
	p, ok := s.projects[pid]
	if !ok {
		return answer(operations.Outcome{}, errGone)
	}
	return answer(operations.Apply(p, op))
}

// held is the project as the session holds it, the zero Project when it holds none.
func (s *Session) held(pid string) domain.Project {
	if p, ok := s.projects[pid]; ok {
		return *p
	}
	return domain.Project{}
}

// projectOp creates, renames or deletes a project among the held ones, by the rules ordo serve
// applies to its files.
func (s *Session) projectOp(pid string, op operations.Op) (operations.Outcome, error) {
	p, held := s.projects[pid]
	if op.Action == "project.delete" {
		if !held {
			return operations.Outcome{}, operations.ErrNoChange
		}
		s.Close(pid)
		return operations.Outcome{Msg: operations.ProjectMsg(op.Action, p.Name)}, nil
	}
	name, err := operations.ProjectName(op.Title)
	if err != nil {
		return operations.Outcome{}, err
	}
	switch {
	case op.Action == "project.create" && held:
		return operations.Outcome{}, operations.ErrNoChange
	case op.Action == "project.rename" && !held:
		return operations.Outcome{}, errGone
	case op.Action == "project.rename" && p.Name == name:
		return operations.Outcome{}, operations.ErrNoChange
	}
	for id, other := range s.projects {
		if id != pid && domain.NormalizeName(other.Name) == domain.NormalizeName(name) {
			return operations.Outcome{}, operations.ErrProjectNameTaken
		}
	}
	now := domain.NowTimestamp()
	if held {
		p.Name, p.UpdatedAt = name, now
		return operations.Outcome{Msg: operations.ProjectMsg(op.Action, name)}, nil
	}
	if err := domain.ValidateID(pid); err != nil {
		return operations.Outcome{}, err
	}
	s.projects[pid] = &domain.Project{ID: pid, Name: name, CreatedAt: now, UpdatedAt: now, Categories: []domain.Category{}}
	return operations.Outcome{Msg: operations.ProjectMsg(op.Action, name), Row: pid}, nil
}

func unreadable(err error) Answer {
	return Answer{Status: operations.Refused, Msg: "Can't read that edit: " + err.Error()}
}

func answer(out operations.Outcome, err error) Answer {
	if errors.Is(err, errGone) {
		return Answer{Status: operations.Gone, Msg: "That project is gone"}
	}
	a := Answer{Status: operations.VerdictOf(err), Msg: out.Msg, Row: out.Row, Saved: out.Saved}
	switch {
	case a.Status == operations.Unchanged:
		a.Msg = ""
	case err != nil:
		msg, ok := operations.Explain(err)
		if !ok {
			msg = "Can't do that: " + err.Error()
		}
		a.Msg = msg
	}
	var conflict *operations.Conflict
	if errors.As(err, &conflict) {
		a.Theirs = &conflict.Theirs
	}
	return a
}
