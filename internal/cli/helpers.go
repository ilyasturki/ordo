package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

// argHint returns a short description of what an argument placeholder accepts,
// or "" if no extra hint is useful.
func argHint(name string) string {
	switch name {
	case "task":
		return "task title or ID (full or 4+ char prefix)"
	case "category":
		return "category name, parent/name or ID (full or 4+ char prefix)"
	case "project":
		return "project name or ID"
	}
	return ""
}

// exactArgs validates positional argument count and produces a clearer error
// than cobra.ExactArgs when args are missing or extra. Each name describes the
// expected positional in order, e.g. exactArgs("task", "status").
func exactArgs(names ...string) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == len(names) {
			return nil
		}
		if len(args) < len(names) {
			missing := names[len(args):]
			placeholders := make([]string, len(missing))
			var hints []string
			for i, n := range missing {
				placeholders[i] = "<" + n + ">"
				if h := argHint(n); h != "" {
					hints = append(hints, "  <"+n+">: "+h)
				}
			}
			msg := fmt.Sprintf("missing argument: %s", strings.Join(placeholders, " "))
			if len(hints) > 0 {
				msg += "\n\n" + strings.Join(hints, "\n")
			}
			msg += "\n\nUsage:\n  " + cmd.UseLine()
			return errors.New(msg)
		}
		return fmt.Errorf("too many arguments: expected %d, got %d\n\nUsage:\n  %s", len(names), len(args), cmd.UseLine())
	}
}

var ErrNotFound = errors.New("not found")

func projectSelector(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	if flag := viper.GetString("project"); flag != "" {
		return flag
	}
	if mgr, err := stateManagerFromViper(); err == nil {
		if id := mgr.GetProjectForDir(); id != "" {
			return id
		}
	}
	return ""
}

// stateManagerFromViper builds a StateManager scoped to the current working
// directory, using the same paths the root command resolves.
func stateManagerFromViper() (*data.StateManager, error) {
	dataDir, err := config.ResolveDataDir(viper.GetString("data"))
	if err != nil {
		return nil, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	mgr := data.NewStateManager(filepath.Dir(dataDir), cwd)
	if err := mgr.Load(); err != nil {
		return nil, err
	}
	return mgr, nil
}

func loadStoreAndProject(selector string) (*data.Store, domain.Project, error) {
	store, err := storeFromViper()
	if err != nil {
		return nil, domain.Project{}, err
	}
	project, err := store.LoadProject(selector)
	if err != nil {
		return nil, domain.Project{}, err
	}
	return store, project, nil
}

func withProject(selector string, fn func(*data.Store, *domain.Project) error) error {
	store, err := storeFromViper()
	if err != nil {
		return err
	}
	// Resolve the (possibly fuzzy) selector to an exact ID first, then load
	// and save under the project flock so a concurrent writer (the TUI, a
	// second CLI) can't clobber our update.
	initial, err := store.LoadProject(selector)
	if err != nil {
		return err
	}
	_, err = store.WithProjectLocked(initial.ID, func(p *domain.Project) error {
		return fn(store, p)
	})
	return err
}

func viewProject(selector string, fn func(domain.Project) error) error {
	_, project, err := loadStoreAndProject(selector)
	if err != nil {
		return err
	}
	return fn(project)
}

type TaskRef struct {
	Task          *domain.Task
	CategoryName  string
	CategoryIndex int
	TaskIndex     int
}

func resolveTask(project domain.Project, selector string) (TaskRef, error) {
	cIdx, tIdx, err := project.FindTask(selector)
	if err != nil {
		return TaskRef{}, ErrNotFound
	}
	return TaskRef{
		Task:          &project.Categories[cIdx].Tasks[tIdx],
		CategoryName:  cliPath(project, cIdx),
		CategoryIndex: cIdx,
		TaskIndex:     tIdx,
	}, nil
}

// resolveCategory tries an ID, the path cliPath prints, a sub-category's own name, then an ID prefix; a tie at one step is refused.
func resolveCategory(project domain.Project, selector string) (*domain.Category, int, error) {
	trimmed := strings.TrimSpace(selector)
	notFound := fmt.Errorf("category %q not found", selector)
	if trimmed == "" {
		return nil, -1, notFound
	}
	cats := project.Categories
	for i := range cats {
		if cats[i].ID == trimmed {
			return &cats[i], i, nil
		}
	}

	for _, named := range [][]int{pathMatches(project, trimmed), nameMatches(project, trimmed, true)} {
		switch len(named) {
		case 0:
			continue
		case 1:
			return &cats[named[0]], named[0], nil
		}
		ids := make([]string, len(named))
		for j, i := range named {
			ids[j] = fmt.Sprintf("%s (%s)", cats[i].ID, cliPath(project, i))
		}
		return nil, -1, fmt.Errorf("%d categories are named %q; pass a path or an ID instead: %s",
			len(named), trimmed, strings.Join(ids, ", "))
	}

	const minPrefixLen = 4
	if len(trimmed) >= minPrefixLen {
		for i := range cats {
			if strings.HasPrefix(strings.ToLower(cats[i].ID), strings.ToLower(trimmed)) {
				return &cats[i], i, nil
			}
		}
	}
	return nil, -1, notFound
}

// nameMatches lists the categories named name: the sub-categories with sub, else the top-level ones.
func nameMatches(project domain.Project, name string, sub bool) []int {
	needle := domain.NormalizeName(name)
	var out []int
	for i, c := range project.Categories {
		if (c.ParentID != "") == sub && domain.NormalizeName(c.Name) == needle {
			out = append(out, i)
		}
	}
	return out
}

// pathMatches tries every "/" as the parent/name split, since a name may hold one too.
func pathMatches(project domain.Project, path string) []int {
	out := nameMatches(project, path, false)
	for _, split := range domain.SplitCategoryPath(path) {
		parent, name := domain.NormalizeName(split[0]), domain.NormalizeName(split[1])
		for i, c := range project.Categories {
			p := domain.ParentIndex(project.Categories, i)
			if p >= 0 && domain.NormalizeName(project.Categories[p].Name) == parent && domain.NormalizeName(c.Name) == name {
				out = append(out, i)
			}
		}
	}
	return out
}

// cliPath is the path the CLI resolves back to the category at i.
func cliPath(project domain.Project, i int) string {
	if p := domain.ParentIndex(project.Categories, i); p >= 0 {
		return project.Categories[p].Name + domain.CategoryPathSep + project.Categories[i].Name
	}
	return project.Categories[i].Name
}
