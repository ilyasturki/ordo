package merge

import (
	"fmt"
	"testing"

	"github.com/ilyasturki/ordo/internal/domain"
)

func benchShape(cats, perCat int) domain.Project {
	p := domain.Project{ID: "p", Name: "P"}
	for c := range cats {
		cat := domain.Category{ID: fmt.Sprintf("c%d", c), Name: fmt.Sprintf("C%d", c)}
		for t := range perCat {
			cat.Tasks = append(cat.Tasks, row(fmt.Sprintf("c%dt%d", c, t), fmt.Sprintf("Task %d", t)))
		}
		p.Categories = append(p.Categories, cat)
	}
	return p
}

func BenchmarkThree(b *testing.B) {
	for _, s := range []struct {
		name         string
		cats, perCat int
	}{{"450_tasks_3x150", 3, 150}, {"5k_tasks_50x100", 50, 100}} {
		base := benchShape(s.cats, s.perCat)
		mine := Clone(base)
		mine.Categories[0].Tasks[1].Priority = domain.PriorityHigh
		theirs := Clone(base)
		theirs.Categories[s.cats-1].Tasks[2].Title = "Retitled elsewhere"
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = Three(base, mine, theirs)
			}
		})
	}
}
