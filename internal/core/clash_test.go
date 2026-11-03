package core_test

import (
	"testing"

	"github.com/ilyasturki/ordo/internal/operations"
)

func TestATakenCategoryNameAnswersTheNamesThatFreeIt(t *testing.T) {
	s, pid := opened(t)
	add := s.Perform(pid, op(operations.Op{Action: "category.add", Title: "feature"}))
	if add.Status != operations.Refused || add.Clash == nil || add.Queue != nil {
		t.Fatalf("add = %+v", add)
	}
	if want := (operations.Clash{Taken: "c1", This: "feature (2)", Other: "Feature (2)"}); *add.Clash != want {
		t.Fatalf("clash = %+v, want %+v", *add.Clash, want)
	}
	s.Perform(pid, op(operations.Op{Action: "category.add", New: "c2", Title: "Fix"}))
	rename := s.Perform(pid, op(operations.Op{Action: "category.rename", Cat: "c2", Title: "Feature"}))
	if rename.Clash == nil || rename.Clash.Taken != "c1" || rename.Clash.This != "Feature (2)" {
		t.Fatalf("rename = %+v", rename)
	}
	if free := s.Perform(pid, op(operations.Op{Action: "category.add", Title: "Docs"})); free.Clash != nil {
		t.Fatalf("a free name = %+v", free)
	}
}
