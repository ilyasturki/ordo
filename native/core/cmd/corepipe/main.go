// Command corepipe serves ordocore over stdin and stdout, one JSON call per line, for the app's
// JVM tests, which can't load the Android library.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ilyasturki/ordo/native/core/ordocore"
)

type call struct {
	Call string `json:"call"`
	Pid  string `json:"pid"`
	Arg  string `json:"arg"`
}

type reply struct {
	Out   string `json:"out"`
	Error string `json:"error,omitempty"`
}

func main() {
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "corepipe:", err)
		os.Exit(1)
	}
}

func serve(r io.Reader, w io.Writer) error {
	c := ordocore.New()
	in := bufio.NewScanner(r)
	in.Buffer(make([]byte, 0, 1<<20), 64<<20)
	out := json.NewEncoder(w)
	for in.Scan() {
		var req call
		if err := json.Unmarshal(in.Bytes(), &req); err != nil {
			return err
		}
		if err := out.Encode(run(c, req)); err != nil {
			return err
		}
	}
	return in.Err()
}

func run(c *ordocore.Core, req call) reply {
	switch req.Call {
	case "open":
		id, err := c.Open(req.Arg)
		if err != nil {
			return reply{Error: err.Error()}
		}
		return reply{Out: id}
	case "perform":
		return reply{Out: c.Perform(req.Pid, req.Arg)}
	case "replay":
		return reply{Out: c.Replay(req.Pid, req.Arg)}
	case "project":
		return reply{Out: c.Project(req.Pid)}
	case "close":
		c.Close(req.Pid)
		return reply{}
	}
	return reply{Error: "unknown call " + req.Call}
}
