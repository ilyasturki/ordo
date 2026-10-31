package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func projectUpdated(name, updated string) domain.Project {
	return domain.Project{ID: name, Name: name, UpdatedAt: updated}
}

func TestOrderProjects(t *testing.T) {
	const (
		jan = "2026-01-01T00:00:00Z"
		feb = "2026-02-01T00:00:00Z"
		mar = "2026-03-01T00:00:00Z"
		apr = "2026-04-01T00:00:00Z"
	)
	cases := []struct {
		name     string
		projects []domain.Project
		opened   map[string]string
		want     []string
	}{
		{
			name:     "opened here after its last edit leads",
			projects: []domain.Project{projectUpdated("Alpha", feb), projectUpdated("Beta", mar)},
			opened:   map[string]string{"Alpha": apr},
			want:     []string{"Alpha", "Beta"},
		},
		{
			name:     "edited elsewhere after it was opened here leads",
			projects: []domain.Project{projectUpdated("Alpha", jan), projectUpdated("Beta", feb)},
			opened:   map[string]string{"Alpha": jan, "Beta": jan},
			want:     []string{"Beta", "Alpha"},
		},
		{
			name:     "edited elsewhere beats an older open",
			projects: []domain.Project{projectUpdated("Alpha", mar), projectUpdated("Beta", feb)},
			opened:   map[string]string{"Alpha": jan, "Beta": feb},
			want:     []string{"Alpha", "Beta"},
		},
		{
			name:     "never opened here ranks by updated_at alone",
			projects: []domain.Project{projectUpdated("Alpha", jan), projectUpdated("Beta", feb), projectUpdated("Gamma", mar)},
			opened:   map[string]string{"Alpha": "not a time"},
			want:     []string{"Gamma", "Beta", "Alpha"},
		},
		{
			name:     "a tie falls to the name, ignoring case",
			projects: []domain.Project{projectUpdated("beta", feb), projectUpdated("Alpha", feb), projectUpdated("Cedar", feb)},
			want:     []string{"Alpha", "beta", "Cedar"},
		},
		{
			name:     "an open and an edit at the same instant fall to the name",
			projects: []domain.Project{projectUpdated("Beta", feb), projectUpdated("Alpha", jan)},
			opened:   map[string]string{"Alpha": feb},
			want:     []string{"Alpha", "Beta"},
		},
		{
			name:     "a fraction of a second still orders two opens",
			projects: []domain.Project{projectUpdated("Alpha", jan), projectUpdated("Beta", jan)},
			opened:   map[string]string{"Alpha": "2026-05-01T00:00:00.100Z", "Beta": "2026-05-01T00:00:00.200Z"},
			want:     []string{"Beta", "Alpha"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pickerNames(orderProjects(tc.projects, tc.opened)))
		})
	}
}

func TestOrderProjectsLeavesItsInputAlone(t *testing.T) {
	in := []domain.Project{projectUpdated("Beta", "2026-01-01T00:00:00Z"), projectUpdated("Alpha", "2026-02-01T00:00:00Z")}
	orderProjects(in, nil)
	assert.Equal(t, []string{"Beta", "Alpha"}, pickerNames(in))
}

func TestSwitchingProjectsStampsItAsOpened(t *testing.T) {
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	alpha, err := store.CreateProject("Alpha")
	require.NoError(t, err)
	beta, err := store.CreateProject("Beta")
	require.NoError(t, err)

	m := newTestModel(t, alpha)
	sm := withStubStateManager(m)
	m.deps.Store = store
	m.ui.Picker = ProjectPickerState{projects: []domain.Project{alpha, beta}, selected: 1}
	m.selectProject()

	require.Equal(t, beta.ID, m.project.ID)
	assert.Contains(t, sm.opened, beta.ID)
	assert.NotContains(t, sm.opened, alpha.ID)
}

func TestCreatingProjectStampsItAsOpened(t *testing.T) {
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	alpha, err := store.CreateProject("Alpha")
	require.NoError(t, err)

	m := newTestModel(t, alpha)
	sm := withStubStateManager(m)
	m.deps.Store = store
	m.createPickedProject("Beta")

	assert.Contains(t, sm.opened, m.project.ID)
}
