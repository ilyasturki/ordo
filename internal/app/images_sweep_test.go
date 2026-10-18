package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

// imagesSweepProject puts chips of every shape on the sweep's rows: long names
// that must be cut, references inline in prose, and several on one task.
func imagesSweepProject() domain.Project {
	p := sweepProject(false)
	ref := func(name string, n int) string {
		return fmt.Sprintf("![%s](attachments/%032x.png)", name, n)
	}
	long := strings.Repeat("a very long screenshot name ", 4)
	n := 0
	for c := range p.Categories {
		for i := range p.Categories[c].Tasks {
			t := &p.Categories[c].Tasks[i]
			n++
			switch n % 4 {
			case 0:
				t.Description = operations.JoinDescription(t.Description, ref(long, n))
			case 1:
				t.Description = operations.JoinDescription(t.Description, "see "+ref("shot", n)+" and then "+ref("other", n+100)+" after it")
			case 2:
				t.Description = ref("only", n)
			}
		}
	}
	return p
}

// Chips and badges change row heights, which the layout must count as drawn,
// whether the image's file is there or not.
func TestLayoutSweep_ImagesKeepTheCursorWhole(t *testing.T) {
	for _, base := range sweepCases() {
		if base.name != "mixed" {
			continue
		}
		c := base
		c.name, c.project = "images", imagesSweepProject
		m := c.model(t)
		m.deps.Attachments = attachments.NewStore(t.TempDir())
		for _, k := range []string{"j", "k", "z", "d", "j", "k"} {
			steps := 1
			if k == "j" || k == "k" {
				steps = m.ui.Selection.Count() + 1
			}
			for i := range steps {
				next, _ := m.Update(sweepKey(k))
				m = asModel(next)
				if !cursorWhole(t, m, fmt.Sprintf("%s %s #%d", c, k, i), true) {
					return
				}
			}
		}
	}
}
