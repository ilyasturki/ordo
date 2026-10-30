package operations

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

var StatusLabels = map[string]string{
	domain.StatusTodo:       "Todo",
	domain.StatusInProgress: "In Progress",
	domain.StatusCompleted:  "Completed",
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
	hue := domain.CanonicalTagColor(color)
	return strings.ToUpper(hue[:1]) + hue[1:]
}
