package operations

import (
	"strings"

	"github.com/ilyasturki/ordo/internal/domain"
)

// ProjectAction reports whether action creates, renames or deletes a whole project. Apply doesn't
// take one: the store runs it on the server, the session on the phone, for the project its queue
// entry names.
func ProjectAction(action string) bool {
	switch action {
	case "project.create", "project.rename", "project.delete":
		return true
	}
	return false
}

// ErrProjectNameTaken refuses a project name another project goes by.
var ErrProjectNameTaken error = &Refusal{Msg: "A project with that name already exists"}

// ProjectName is title as a project's name: trimmed, on one line, not empty.
func ProjectName(title string) (string, error) {
	name := strings.TrimSpace(title)
	if name == "" {
		return "", refuse("A project needs a name")
	}
	return name, domain.ValidateLine(name)
}

// ProjectMsg phrases an applied project op on the project it named name.
func ProjectMsg(action, name string) string {
	switch action {
	case "project.create":
		return "Created " + quote(name)
	case "project.rename":
		return "Renamed to " + quote(name)
	}
	return "Deleted " + quote(name)
}
