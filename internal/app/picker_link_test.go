package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	link        string
	setCalls    []string
	unlinkCalls int
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

func (s *linkTrackingState) UnlinkDir() (string, error) {
	prev := s.link
	s.link = ""
	s.unlinkCalls++
	return prev, nil
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

func TestLinkSelectedProject_RelinksAndSwitches(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = alpha.ID

	m.linkSelectedProject()

	assert.Equal(t, beta.ID, m.project.ID)
	assert.Equal(t, []string{beta.ID}, state.setCalls, "l must override an existing link")
	assert.Equal(t, "Linked this directory to Beta", m.ui.Screen.StatusMsg)
}

func TestLinkSelectedProject_CurrentProject(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = beta.ID
	m.ui.Picker.selected = 0

	m.linkSelectedProject()

	assert.Equal(t, alpha.ID, m.project.ID)
	assert.Equal(t, []string{alpha.ID}, state.setCalls)
	assert.Equal(t, "Linked this directory to Alpha", m.ui.Screen.StatusMsg)
}

func TestLinkSelectedProject_NewProjectRowIsNoop(t *testing.T) {
	m, state, alpha, _ := newPickerTestModel(t, "")
	m.ui.Picker.onNew = true

	m.linkSelectedProject()

	assert.Equal(t, alpha.ID, m.project.ID)
	assert.Empty(t, state.setCalls)
}

func TestPickerRow_MarksLinkedProject(t *testing.T) {
	m, state, _, beta := newPickerTestModel(t, "")
	state.link = beta.ID

	body := ansi.Strip(strings.Join(m.pickerBody(60), "\n"))

	assert.Contains(t, body, "Alpha (current)")
	assert.Contains(t, body, "Beta (linked)")
}

func pressPickerKey(t *testing.T, m *model, key string) {
	t.Helper()
	after, _ := m.handleProjectPickerKey(tea.KeyPressMsg{Code: rune(key[0]), Text: key})
	*m = after
}

func TestPickerL_OnLinkedRowUnlinksAndStays(t *testing.T) {
	for _, fullscreen := range []bool{false, true} {
		m, state, alpha, beta := newPickerTestModel(t, "")
		state.link = beta.ID
		m.ui.Picker.fullscreen = fullscreen
		m.ui.Modes.ToProjectPicker()

		pressPickerKey(t, m, "l")

		assert.Equal(t, 1, state.unlinkCalls)
		assert.Empty(t, state.link)
		assert.Empty(t, state.setCalls)
		assert.Equal(t, alpha.ID, m.project.ID, "unlinking does not open the row")
		assert.True(t, m.ui.Modes.IsProjectPicker(), "unlinking stays in the picker")
		assert.Equal(t, "Unlinked this directory from Beta", m.ui.Screen.StatusMsg)
		assert.NotContains(t, ansi.Strip(m.projectPickerView()), "(linked)")
	}
}

func TestPickerL_OffLinkedRowStillLinks(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = alpha.ID

	pressPickerKey(t, m, "l")

	assert.Zero(t, state.unlinkCalls)
	assert.Equal(t, []string{beta.ID}, state.setCalls)
	assert.Equal(t, beta.ID, m.project.ID)
}

func TestPickerUnlink_SticksWhenPickingAfterwards(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = beta.ID
	pressPickerKey(t, m, "l")

	m.ui.Picker.selected = 1
	m.selectProject()
	assert.Equal(t, beta.ID, m.project.ID)
	assert.Empty(t, state.link, "picking after an unlink must not relink the directory")

	m.ui.Picker = ProjectPickerState{projects: []domain.Project{alpha, beta}, selected: 0}
	m.selectProject()
	assert.Equal(t, alpha.ID, m.project.ID)
	assert.Empty(t, state.setCalls, "the unlink holds for the whole session")
}

func TestPickerUnlink_SticksWhenCreatingAProject(t *testing.T) {
	m, state, _, beta := newPickerTestModel(t, "")
	state.link = beta.ID
	pressPickerKey(t, m, "l")

	m.createPickedProject("Gamma")

	assert.Equal(t, "Gamma", m.project.Name)
	assert.Empty(t, state.link)
	assert.Empty(t, state.setCalls)
}

func TestPickerLinkAfterUnlink_ResumesLinking(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = beta.ID
	pressPickerKey(t, m, "l")

	m.ui.Picker.selected = 0
	pressPickerKey(t, m, "l")
	assert.Equal(t, alpha.ID, state.link, "l off the linked row links deliberately")
	assert.False(t, m.ui.KeepDirUnlinked)
}

func TestCreatePickedProject_KeepsAnExistingLink(t *testing.T) {
	m, state, alpha, _ := newPickerTestModel(t, "")
	state.link = alpha.ID

	m.createPickedProject("Gamma")

	assert.Equal(t, "Gamma", m.project.Name)
	assert.Equal(t, alpha.ID, state.link, "creating opens the project for the session, like picking one")
}

func TestCreatePickedProject_LinksAnUnlinkedDir(t *testing.T) {
	m, state, _, _ := newPickerTestModel(t, "")

	m.createPickedProject("Gamma")

	assert.Equal(t, m.project.ID, state.link)
}

func TestDeleteCurrentProject_KeepsALinkToAnotherProject(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = beta.ID
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteProject, ProjectID: alpha.ID}

	m.confirmDeleteProject()

	assert.Equal(t, beta.ID, m.project.ID)
	assert.Equal(t, beta.ID, state.link)
	assert.Empty(t, state.setCalls)
}

func TestDeleteLinkedProject_DropsTheLink(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = beta.ID
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteProject, ProjectID: beta.ID}

	m.confirmDeleteProject()

	assert.Equal(t, alpha.ID, m.project.ID)
	assert.Empty(t, state.link, "a link must not outlive its project")
}

func TestDeleteCurrentLinkedProject_LinksTheNextOne(t *testing.T) {
	m, state, alpha, beta := newPickerTestModel(t, "")
	state.link = alpha.ID
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteProject, ProjectID: alpha.ID}

	m.confirmDeleteProject()

	assert.Equal(t, beta.ID, m.project.ID)
	assert.Equal(t, beta.ID, state.link)
}

func TestPickerFooter_NamesUnlinkOnTheLinkedRow(t *testing.T) {
	for _, fullscreen := range []bool{false, true} {
		m, state, _, beta := newPickerTestModel(t, "")
		state.link = beta.ID
		m.ui.Picker.fullscreen = fullscreen
		m.ui.Screen.Width, m.ui.Screen.Height = 100, 30

		m.ui.Picker.selected = 1
		onLinked := ansi.Strip(m.projectPickerView())
		rowsOn, visibleOn := m.pickerHintRows(), m.pickerVisibleCount()
		m.ui.Picker.selected = 0
		offLinked := ansi.Strip(m.projectPickerView())

		assert.Contains(t, onLinked, "l unlink")
		assert.Contains(t, offLinked, "l link")
		assert.NotContains(t, offLinked, "unlink")
		assert.Equal(t, rowsOn, m.pickerHintRows())
		assert.Equal(t, visibleOn, m.pickerVisibleCount())
		assert.Equal(t, lipgloss.Height(onLinked), lipgloss.Height(offLinked))
	}
}
