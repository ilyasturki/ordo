package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestServeAnswersEachCallOnItsOwnLine(t *testing.T) {
	calls := []call{
		{Call: "open", Arg: `{"id":"p1","name":"Phone","categories":[{"id":"c1","name":"Feature","tasks":[]}]}`},
		{Call: "perform", Pid: "p1", Arg: `{"action":"task.add","cat":"c1","title":"Retry \"uploads\""}`},
		{Call: "project", Pid: "nope"},
		{Call: "open", Arg: `not json`},
		{Call: "fly"},
	}
	var in bytes.Buffer
	for _, c := range calls {
		raw, _ := json.Marshal(c)
		in.Write(append(raw, '\n'))
	}
	var out bytes.Buffer
	if err := serve(&in, &out); err != nil {
		t.Fatal(err)
	}
	var replies []reply
	lines := bufio.NewScanner(&out)
	for lines.Scan() {
		var r reply
		if err := json.Unmarshal(lines.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		replies = append(replies, r)
	}
	if len(replies) != len(calls) {
		t.Fatalf("replies = %+v", replies)
	}
	if replies[0].Out != "p1" || !strings.Contains(replies[1].Out, `"status":"applied"`) {
		t.Fatalf("replies = %+v", replies)
	}
	if replies[2] != (reply{}) || replies[3].Error == "" || replies[4].Error == "" {
		t.Fatalf("replies = %+v", replies)
	}
}
