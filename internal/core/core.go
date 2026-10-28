// Package core holds the projects the phone and browser app opened and runs their edits
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
}

// Perform runs a user's edit; its answer's Queue is what to send.
func (s *Session) Perform(pid string, rawOp []byte) Answer {
	return s.run(pid, rawOp, func(p *domain.Project, op operations.Op) (operations.Outcome, *operations.Op, error) {
		out, queue, err := operations.Perform(p, op)
		return out, &queue, err
	})
}

// Replay runs an op from the queue again, on a copy the server just sent.
func (s *Session) Replay(pid string, rawOp []byte) Answer {
	return s.run(pid, rawOp, func(p *domain.Project, op operations.Op) (operations.Outcome, *operations.Op, error) {
		if !operations.Replayable(op.Action) {
			return operations.Outcome{}, nil, operations.ErrUnknownAction
		}
		out, err := operations.Apply(p, op)
		return out, nil, err
	})
}

func (s *Session) run(pid string, rawOp []byte, fn func(*domain.Project, operations.Op) (operations.Outcome, *operations.Op, error)) Answer {
	p, ok := s.projects[pid]
	if !ok {
		return Answer{Status: operations.Gone, Msg: "That project is gone"}
	}
	var op operations.Op
	if err := json.Unmarshal(rawOp, &op); err != nil {
		return Answer{Status: operations.Refused, Msg: "Can't read that edit: " + err.Error()}
	}
	out, queue, err := fn(p, op)
	a := Answer{Status: operations.VerdictOf(err), Msg: out.Msg, Row: out.Row, Saved: out.Saved}
	if err != nil {
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
	if err == nil {
		a.Queue = queue
	}
	return a
}
