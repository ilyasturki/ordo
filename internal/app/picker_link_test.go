package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

// linkTrackingState records dir-link reads/writes so tests can assert whether
// selecting a project persisted the directory link.
type linkTrackingState struct {
	*stubStateManager
	link     string
	setCalls []string
}

func newLinkTrackingState(initial string) *linkTrackingState {
	return &linkTrackingState{stubStateManager: newStubStateManager(), link: initial}
}

func (s *linkTrackingState) GetProjectForDir() string { return s.link }

func (s *linkTrackingState) SetProjectForDir(id string) error {
	s.link = id
	s.setCalls = append(s.setCalls, id)
	return nil
}

// newPickerTestModel wires a real store (two projects) and a link-tracking
// state manager, with the picker open and `target` selected.
func newPickerTestModel(t *testing.T, link string) (*model, *linkTrackingState, domain.Project, domain.Project) {
	t.Helper()
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	alpha, err := store.CreateProject("Alpha")
	require.NoError(t, err)
	beta, err := store.CreateProject("Beta")
	require.NoError(t, err)

	m := newTestModel(t, alpha)
	state := newLinkTrackingState(link)
	m.deps.Store = store
	m.deps.StateManager = state
	m.deps.CfgManager = &stubConfigReader{cfg: config.DefaultConfig()}

	m.ui.Picker = ProjectPickerState{
		projects: []domain.Project{alpha, beta},
		selected: 1, // Beta
	}
	return m, state, alpha, beta
}

func TestSelectProject_LinkedDir_DoesNotOverride(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "Alpha-id-placeholder")
	state.link = alpha.ID // dir already linked to Alpha

	m.selectProject()

	assert.Equal(t, beta.ID, m.project.ID, "session should switch to the picked project")
	assert.Equal(t, alpha.ID, state.link, "existing dir link must be preserved")
	assert.Empty(t, state.setCalls, "selecting must not persist over an existing link")
}

func TestSelectProject_UnlinkedDir_EstablishesLink(t *testing.T) {
	m, state, _, beta := newPickerTestModel(t, "")

	m.selectProject()

	assert.Equal(t, beta.ID, m.project.ID)
	assert.Equal(t, beta.ID, state.link, "an unlinked dir should adopt the picked project")
	assert.Equal(t, []string{beta.ID}, state.setCalls)
}
