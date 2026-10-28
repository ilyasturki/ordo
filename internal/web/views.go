package web

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/domain"
)

func finished(status string) bool {
	return status == domain.StatusCompleted || status == domain.StatusCancelled
}

type imageView struct {
	Name   string
	Target string
	Served bool
}

type rowView struct {
	domain.Task
	Children   []rowView
	ChildDone  int
	ChildTotal int
	Images     []imageView
	Preview    string
}

func (r rowView) Est() string { return domain.FormatEstimate(r.EstimateMinutes) }

type catView struct {
	ID            string
	Name          string
	ParentName    string
	Sub           bool
	Est           string
	Folded        bool
	Rows          []rowView
	Completed     []rowView
	CompletedOpen bool
}

// OpenTasks reports whether Rows holds a task, not only separators.
func (c catView) OpenTasks() bool {
	for _, r := range c.Rows {
		if !r.IsSeparator() {
			return true
		}
	}
	return false
}

type countsView struct {
	Open, Doing int
}

func imagesOf(description string) []imageView {
	var out []imageView
	for _, ref := range attachments.Refs(description) {
		out = append(out, imageView{Name: ref.Name, Target: ref.Target, Served: ref.Internal()})
	}
	return out
}

func newRow(t domain.Task) rowView {
	return rowView{
		Task:    t,
		Images:  imagesOf(t.Description),
		Preview: preview(t.Description),
	}
}

// preview is the description's first two non-blank lines, without its images.
func preview(description string) string {
	var lines []string
	for _, line := range strings.Split(attachments.Strip(description), "\n") {
		if line = strings.Join(strings.Fields(line), " "); line != "" {
			lines = append(lines, line)
		}
		if len(lines) == 2 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

// buildRows nests each category's rows, moving a top-level task to completed once it and its subtasks are all finished.
func buildRows(cat domain.Category) (open, completed []rowView) {
	for i := 0; i < len(cat.Tasks); {
		end := domain.BlockEnd(cat.Tasks, i)
		row := newRow(cat.Tasks[i])
		row.ChildDone, row.ChildTotal = cat.ChildProgress(i)
		for _, child := range cat.Tasks[i+1 : end] {
			row.Children = append(row.Children, newRow(child))
		}
		if !row.IsSeparator() && finished(row.Status) && row.ChildDone == row.ChildTotal {
			completed = append(completed, row)
		} else {
			open = append(open, row)
		}
		i = end
	}
	return open, completed
}

// completedFold is the fold-cookie key present while a category's Completed fold is open.
func completedFold(categoryID string) string { return "done-" + categoryID }

func buildCategories(p *domain.Project, folded map[string]bool) []catView {
	cats := make([]catView, 0, len(p.Categories))
	for i, c := range p.Categories {
		view := catView{
			ID:            c.ID,
			Name:          c.Name,
			Sub:           c.ParentID != "",
			Est:           domain.FormatEstimate(c.EstimateMinutes),
			Folded:        folded[c.ID],
			CompletedOpen: folded[completedFold(c.ID)],
		}
		if parent := domain.ParentIndex(p.Categories, i); parent >= 0 {
			view.ParentName = p.Categories[parent].Name
		}
		view.Rows, view.Completed = buildRows(c)
		cats = append(cats, view)
	}
	return cats
}

func projectCounts(p *domain.Project) countsView {
	c := p.StatusCounts()
	return countsView{Open: c.Todo + c.InProgress, Doing: c.InProgress}
}

// splitDescription separates the trailing image-only lines from the text, when that split loses nothing.
func splitDescription(description string) (text string, images []string) {
	lines := strings.Split(description, "\n")
	cut := len(lines)
	for cut > 0 {
		line := strings.TrimSpace(lines[cut-1])
		if line == "" {
			cut--
			continue
		}
		pieces := attachments.Split(line)
		if len(pieces) != 1 || pieces[0].Ref == nil {
			break
		}
		cut--
	}
	for _, line := range lines[cut:] {
		if strings.TrimSpace(line) != "" {
			images = append(images, strings.TrimSpace(line))
		}
	}
	return strings.TrimRight(strings.Join(lines[:cut], "\n"), "\n"), images
}

// descriptionFields splits a description into the details field and its image lines, or
// leaves it whole when joining them back would not give the same description.
func descriptionFields(description string) (text string, images []string, lossless bool) {
	text, images = splitDescription(description)
	if joinDescription(text, images) != strings.TrimRight(description, "\n") {
		return description, nil, false
	}
	return text, images, true
}

func joinDescription(text string, images []string) string {
	text = strings.TrimRight(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	parts := make([]string, 0, len(images)+1)
	if strings.TrimSpace(text) != "" {
		parts = append(parts, text)
	}
	parts = append(parts, images...)
	return strings.Join(parts, "\n")
}
