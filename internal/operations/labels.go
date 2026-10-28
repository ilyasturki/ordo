package operations

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

var StatusLabels = map[string]string{
	domain.StatusTodo:       "To do",
	domain.StatusInProgress: "Doing",
	domain.StatusCompleted:  "Done",
	domain.StatusCancelled:  "Cancelled",
}

var PriorityLabels = map[string]string{
	"":                      "None",
	domain.PriorityCritical: "Critical",
	domain.PriorityHigh:     "High",
	domain.PriorityMedium:   "Medium",
	domain.PriorityLow:      "Low",
	domain.PriorityTrivial:  "Trivial",
}

func TagName(color string) string {
	if color == "" {
		return "None"
	}
	hue, diamond := strings.CutSuffix(color, domain.TagDiamondSuffix)
	if hue == "" {
		return color
	}
	name := strings.ToUpper(hue[:1]) + hue[1:]
	if diamond {
		name += " diamond"
	}
	return name
}
