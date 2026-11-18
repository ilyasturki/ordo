package operations

import "errors"

// ErrNoChange signals that a verb succeeded but nothing actually changed (for
// example, moving the first task up). Front doors treat it as "skip the save"
// rather than as a failure, so a boundary tap doesn't bump UpdatedAt.
var ErrNoChange = errors.New("no change")

// ErrTitleRequired is returned when a task title is empty or whitespace.
var ErrTitleRequired = errors.New("title is required")

// ErrNameRequired is returned when a category name is empty or whitespace.
var ErrNameRequired = errors.New("name is required")

// ErrSeparatorFieldNotAllowed is returned when a write targets a field a
// separator does not have. A separator is a divider: it carries a label (Title)
// and nothing else, which is why the TUI blocks status, priority,
// description and tags on one. The operations layer enforces the same rule so
// no caller can turn a divider into a half-task.
var ErrSeparatorFieldNotAllowed = errors.New("a separator only has a label")

// ErrInvalidKind is returned when a create request names a row kind that is
// neither an ordinary task nor a separator.
var ErrInvalidKind = errors.New("invalid kind")
