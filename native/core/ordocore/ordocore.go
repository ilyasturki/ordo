// Package ordocore is internal/core for the Android app, bound with gomobile, with JSON strings in and out.
package ordocore

import (
	"encoding/json"

	"github.com/ilyasturki/ordo/internal/core"
)

// Core is not safe for concurrent use: the app calls it from one thread.
type Core struct{ s *core.Session }

func New() *Core { return &Core{s: core.New()} }

func (c *Core) Open(raw string) (string, error) { return c.s.Open([]byte(raw)) }

// Perform answers core.Answer as JSON.
func (c *Core) Perform(pid, op string) string { return answer(c.s.Perform(pid, []byte(op))) }

// Replay answers core.Answer as JSON.
func (c *Core) Replay(pid, op string) string { return answer(c.s.Replay(pid, []byte(op))) }

// Project is "" when the project isn't held.
func (c *Core) Project(pid string) string { return string(c.s.Project(pid)) }

func (c *Core) Close(pid string) { c.s.Close(pid) }

func answer(a core.Answer) string {
	raw, _ := json.Marshal(a)
	return string(raw)
}
