package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"phasionary/internal/config"
	"phasionary/internal/data"
	"phasionary/internal/domain"
)

// argHint returns a short description of what an argument placeholder accepts,
// or "" if no extra hint is useful.
func argHint(name string) string {
	switch name {
	case "task":
		return "task title or ID (full or 4+ char prefix)"
	case "category":
		return "category name or ID (full or 4+ char prefix)"
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
		CategoryName:  project.Categories[cIdx].Name,
		CategoryIndex: cIdx,
		TaskIndex:     tIdx,
	}, nil
}

func resolveCategory(project domain.Project, selector string) (*domain.Category, int, error) {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil, -1, ErrNotFound
	}

	needle := domain.NormalizeName(selector)
	minPrefixLen := 4

	for cIdx := range project.Categories {
		cat := &project.Categories[cIdx]

		if cat.ID == selector {
			return cat, cIdx, nil
		}

		if len(selector) >= minPrefixLen && strings.HasPrefix(strings.ToLower(cat.ID), strings.ToLower(selector)) {
			return cat, cIdx, nil
		}

		if domain.NormalizeName(cat.Name) == needle {
			return cat, cIdx, nil
		}
	}

	return nil, -1, ErrNotFound
}
