package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

const (
	shotTarget = "attachments/0123456789abcdef0123456789abcdef.png"
	goneTarget = "attachments/fedcba9876543210fedcba9876543210.png"
)

func missingOnly(targets ...string) func(string) bool {
	return func(target string) bool {
		for _, t := range targets {
			if t == target {
				return true
			}
		}
		return false
	}
}

func plainLines(s string) []string {
	return strings.Split(ansi.Strip(s), "\n")
}

func TestRenderDescription_ShowsAnImageAsAChip(t *testing.T) {
	task := domain.Task{Title: "T", Status: domain.StatusTodo, Description: "the overflow here:\n" +
		"![Screenshot 10.10](" + shotTarget + ")\n" +
		"and ![old](" + goneTarget + ") in the middle\n" +
		"![ext](/nowhere/shot.png)"}
	r := NewTaskLineRenderer(80, "text", "full", true).
		WithRowLimits(Uncapped, Uncapped).
		WithImages(missingOnly(goneTarget, "/nowhere/shot.png"))

	got := plainLines(r.RenderDescription(task, 4, false))

	assert.Equal(t, []string{
		"    ▎ the overflow here:",
		"    ▎ ▣ Screenshot 10.10",
		"    ▎ and",
		"    ▎ ▣ old (not synced yet)",
		"    ▎ in the middle",
		"    ▎ ▣ ext (missing)",
	}, got)
	assert.NotContains(t, strings.Join(got, "\n"), "0123456789abcdef", "the hash never shows")
}

// The layout measures a description once; the file can sync in before it is drawn.
func TestDescriptionHeight_DoesNotDependOnTheFileBeingThere(t *testing.T) {
	long := strings.Repeat("very long name ", 10)
	desc := "![" + long + "](" + shotTarget + ")\nsee ![a](" + shotTarget + ") and more words after it"
	task := domain.Task{Title: "T", Status: domain.StatusTodo, Description: desc}
	for _, width := range []int{30, 50, 80, 0} {
		height := DescriptionHeight(desc, width, 4, Uncapped)
		for name, missing := range map[string]func(string) bool{
			"present": missingOnly(),
			"missing": missingOnly(shotTarget),
		} {
			r := NewTaskLineRenderer(width, "text", "full", true).
				WithRowLimits(Uncapped, Uncapped).
				WithImages(missing)
			rendered := plainLines(r.RenderDescription(task, 4, false))
			assert.Len(t, rendered, height, "width %d, %s", width, name)
			if width > 0 {
				for _, l := range rendered {
					assert.LessOrEqual(t, ansi.StringWidth(l), width, "width %d, %s: %q", width, name, l)
				}
			}
		}
	}
}

func TestRenderDescription_SearchFindsTheChipsName(t *testing.T) {
	desc := "one\ntwo\nthree\nfour\n![needle shot](" + shotTarget + ")"
	task := domain.Task{Title: "T", Status: domain.StatusTodo, Description: desc}
	r := NewTaskLineRenderer(80, "text", "full", true).WithRowLimits(Uncapped, 2)
	r.WithSearch("needle", r.selectedStyle())

	got := ansi.Strip(r.RenderDescription(task, 0, false))

	assert.Contains(t, got, "▣ needle shot", "the window scrolls to the chip holding the match")
}

func TestTitleRow_CountsImagesEvenWithDescriptionsExpanded(t *testing.T) {
	one := domain.Task{Title: "Fix the bar", Status: domain.StatusTodo, Description: "![a](" + shotTarget + ")"}
	two := domain.Task{Title: "Fix the bar", Status: domain.StatusTodo,
		Description: "text\n![a](" + shotTarget + ")\n![b](/tmp/b.png)"}
	plain := domain.Task{Title: "Fix the bar", Status: domain.StatusTodo, Description: "just text"}

	for expanded, suffix := range map[bool]string{false: "Fix the bar ¶ ▣", true: "Fix the bar ▣"} {
		r := NewTaskLineRenderer(80, "text", "full", true).WithDescriptionExpanded(expanded)
		assert.True(t, strings.HasSuffix(ansi.Strip(r.Render(one, false)), suffix), "expanded=%v", expanded)
		assert.Contains(t, ansi.Strip(r.Render(two, true)), "▣2")
		assert.NotContains(t, ansi.Strip(r.Render(plain, false)), "▣")
	}
}

func TestTitleRows_MakeRoomForTheImageBadge(t *testing.T) {
	desc := "![a](" + shotTarget + ")\n![b](" + goneTarget + ")"
	for width := 20; width <= 40; width++ {
		task := domain.Task{Title: strings.Repeat("x", width-12), Status: domain.StatusTodo, Description: desc}
		r := NewTaskLineRenderer(width, "text", "full", true).WithDescriptionExpanded(true)
		rendered := plainLines(r.Render(task, false))
		require.Len(t, rendered, r.TitleRows(task), "width %d", width)
		for _, l := range rendered {
			assert.LessOrEqual(t, ansi.StringWidth(l), width, "width %d: %q", width, l)
		}
	}
}
