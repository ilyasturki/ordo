package operations

// Surface is a bit set of the front ends offering an action.
type Surface uint8

const (
	TUI Surface = 1 << iota
	Web
	Both = TUI | Web
)

// Action names one user-facing edit; the web's dispatch and its parity test key off Actions.
type Action struct {
	ID      string
	Label   string
	Surface Surface
}

var Actions = []Action{
	{"task.add", "Add task", Both},
	{"task.insert", "Add task at the top or below a row", Both},
	{"separator.add", "Add separator", Both},
	{"task.title", "Edit title", Both},
	{"task.description", "Edit description", Both},
	{"task.attach", "Attach image", Both},
	{"task.delete", "Delete", Both},
	{"status.cycle", "Cycle status", Both},
	{"status.set", "Set status", Web},
	{"priority.step", "Raise or lower priority", TUI},
	{"priority.set", "Set priority", Web},
	{"tag.cycle", "Cycle tag", TUI},
	{"tag.set", "Set tag", Both},
	{"tag.paste", "Paste tag", TUI},
	{"estimate.set", "Set estimate", Both},
	{"task.nest", "Nest under the task above", Both},
	{"task.unnest", "Un-nest", Both},
	{"task.shift", "Move up or down", Both},
	{"task.move", "Move to category", Both},
	{"task.place", "Drag to a position", Both},
	{"task.paste", "Paste a copy", TUI},
	{"version.bump", "Bump a version", TUI},
	{"category.add", "Add category", TUI},
	{"category.rename", "Rename category", TUI},
	{"category.merge", "Merge categories", TUI},
	{"category.delete", "Delete category", TUI},
	{"category.estimate", "Set category estimate", TUI},
	{"category.nest", "Nest category", TUI},
	{"category.shift", "Move category", TUI},
	{"categories.reverse", "Reverse categories", TUI},
	{"project.manage", "Create, rename or delete a project", TUI},
}
