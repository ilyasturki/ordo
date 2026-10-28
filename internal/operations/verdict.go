package operations

import (
	"errors"

	"github.com/ilyasturki/ordo/internal/domain"
)

// Verdict is what a queue makes of an applied op: everything but Applied leaves the file as it was.
type Verdict string

const (
	Applied   Verdict = "applied"
	Unchanged Verdict = "unchanged"
	Conflicts Verdict = "conflict"
	Gone      Verdict = "gone"
	Refused   Verdict = "refused"
)

func VerdictOf(err error) Verdict {
	switch {
	case err == nil:
		return Applied
	case errors.Is(err, ErrNoChange):
		return Unchanged
	case errors.As(err, new(*Conflict)):
		return Conflicts
	case errors.Is(err, domain.ErrTaskNotFound), errors.Is(err, domain.ErrCategoryNotFound):
		return Gone
	}
	return Refused
}

// Explain phrases an op's error for the user; ok is false for an error it has no words for.
func Explain(err error) (msg string, ok bool) {
	var refusal *Refusal
	switch {
	case errors.As(err, &refusal):
		return refusal.Msg, true
	case errors.As(err, new(*Conflict)):
		return err.Error(), true
	case errors.Is(err, ErrUnknownAction):
		return "This version doesn't know that edit: update ordo", true
	case errors.Is(err, domain.ErrTaskNotFound):
		return "That task is gone", true
	case errors.Is(err, domain.ErrCategoryNotFound):
		return "That category is gone", true
	case errors.Is(err, ErrTitleRequired):
		return "A task needs a title", true
	case errors.Is(err, domain.ErrSeparatorNest):
		return "A separator doesn't nest", true
	case errors.Is(err, domain.ErrNestDepth):
		return "A subtask can't have subtasks: nesting goes one level deep", true
	case errors.Is(err, ErrNegativeEstimate):
		return "Pick an estimate from the list", true
	}
	return "", false
}
