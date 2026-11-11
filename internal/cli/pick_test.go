package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func pickProjects(names ...string) []domain.Project {
	projects := make([]domain.Project, len(names))
	for i, n := range names {
		projects[i] = domain.Project{ID: "id-" + n, Name: n}
	}
	return projects
}

func TestMatchPick(t *testing.T) {
	tests := []struct {
		name      string
		projects  []string
		query     string
		wantID    string
		wantQuery string
		wantErr   bool
	}{
		{name: "unique prefix", projects: []string{"Ferry", "Billing"}, query: "fer", wantID: "id-Ferry"},
		{name: "prefix ignores case", projects: []string{"ferry", "Billing"}, query: "BIL", wantID: "id-Billing"},
		{name: "unique substring", projects: []string{"Ferry", "Billing"}, query: "lli", wantID: "id-Billing"},
		{name: "unique prefix beats other substrings", projects: []string{"Ferry", "Sea ferry"}, query: "fer", wantID: "id-Ferry"},
		{name: "substring is smartcase", projects: []string{"Sea Ferry", "Sea ferry"}, query: "Fer", wantID: "id-Sea Ferry"},
		{name: "uppercase prefix narrowed by smartcase", projects: []string{"ferry", "Ferret"}, query: "Fer", wantID: "id-Ferret"},
		{name: "several prefixes", projects: []string{"Ferry", "Ferret"}, query: "fer", wantQuery: "fer"},
		{name: "several substrings", projects: []string{"Old ferry", "New ferry"}, query: "ferry", wantQuery: "ferry"},
		{name: "several prefixes none matching smartcase", projects: []string{"ferry", "ferret"}, query: "FER", wantQuery: "fer"},
		{name: "query is trimmed", projects: []string{"Ferry", "Billing"}, query: "  fer ", wantID: "id-Ferry"},
		{name: "no match", projects: []string{"Ferry"}, query: "zzz", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := matchPick(pickProjects(tt.projects...), tt.query)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), `no project matches "`+tt.query+`"`) {
					t.Fatalf("err = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.projectID != tt.wantID || got.query != tt.wantQuery {
				t.Fatalf("got %+v, want id %q query %q", got, tt.wantID, tt.wantQuery)
			}
		})
	}
}

func TestResolvePickExactNameBeatsLooserMatches(t *testing.T) {
	dir := initDefault(t)
	store := data.NewStore(filepath.Join(dir, "projects"))
	longer, err := store.CreateProject("Default Two")
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolvePick(store, " default ")
	if err != nil {
		t.Fatal(err)
	}
	if got.projectID != loadDefault(t, dir).ID || got.projectID == longer.ID {
		t.Fatalf("got %+v", got)
	}
}

func TestResolvePickByPrefixAndNoMatch(t *testing.T) {
	dir := initDefault(t)
	store := data.NewStore(filepath.Join(dir, "projects"))
	got, err := resolvePick(store, "def")
	if err != nil || got.projectID != loadDefault(t, dir).ID {
		t.Fatalf("got %+v, err %v", got, err)
	}
	if _, err := resolvePick(store, "nothing like it"); err == nil || err.Error() != `no project matches "nothing like it"` {
		t.Fatalf("err = %v", err)
	}
}

func TestPickWithNoMatchExitsWithAnError(t *testing.T) {
	dir := initDefault(t)
	_, err := runCLI(t, dir, "pick", "zzz")
	if err == nil || err.Error() != `no project matches "zzz"` {
		t.Fatalf("err = %v", err)
	}
}
