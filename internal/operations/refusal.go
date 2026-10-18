package operations

import (
	"errors"
	"fmt"

	"github.com/ilyasturki/ordo/internal/domain"
)

// Refusal is a request the rules turn down; its message is written for the user.
type Refusal struct{ Msg string }

func (r *Refusal) Error() string { return r.Msg }

func refuse(format string, args ...any) error {
	return &Refusal{Msg: fmt.Sprintf(format, args...)}
}

// NestRefusal phrases a nesting error from domain for a row of kind, already nested or not.
func NestRefusal(err error, nested bool, kind, child, children string) error {
	switch {
	case errors.Is(err, domain.ErrNothingToNest):
		return refuse("Nothing above to nest under")
	case errors.Is(err, domain.ErrSeparatorNest):
		return refuse("A separator doesn't nest")
	case errors.Is(err, domain.ErrNestDepth) && nested:
		return refuse("Already a %s: nesting goes one level deep", child)
	case errors.Is(err, domain.ErrNestDepth):
		return refuse("A %s with %s can't nest: nesting goes one level deep", kind, children)
	case errors.Is(err, domain.ErrNotNested):
		return refuse("Already at the top level")
	case errors.Is(err, domain.ErrMixedLevels):
		return refuse("Can't: the selection mixes levels")
	}
	return err
}

func EdgeRefusal(dir int, nested bool) error {
	switch {
	case dir > 0 && nested:
		return refuse("Already at the bottom of its parent")
	case dir > 0:
		return refuse("Already at the bottom")
	case nested:
		return refuse("Already at the top of its parent")
	}
	return refuse("Already at the top")
}
