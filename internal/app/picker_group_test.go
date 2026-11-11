package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func grouped(name, group, updated string) domain.Project {
	p := projectUpdated(name, updated)
	p.Group = group
	return p
}

func TestOrderProjectsGathersGroupsByTheirMostRecentProject(t *testing.T) {
	const (
		jan = "2026-01-01T00:00:00Z"
		feb = "2026-02-01T00:00:00Z"
		mar = "2026-03-01T00:00:00Z"
		apr = "2026-04-01T00:00:00Z"
		may = "2026-05-01T00:00:00Z"
	)
	projects := []domain.Project{
		grouped("Loose", "", may),
		grouped("Ferry", "Work", jan),
		grouped("Garden", "Perso", apr),
		grouped("Api", "work ", mar),
		grouped("Books", "Perso", feb),
		grouped("Old", "", jan),
	}
	assert.Equal(t,
		[]string{"Garden", "Books", "Api", "Ferry", "Loose", "Old"},
		pickerNames(orderProjects(projects, nil)),
		"Perso leads on Garden, Work follows on Api whatever its case, the ungrouped come last")
}

func pickerText(m *model) string {
	return ansi.Strip(strings.Join(m.pickerBody(60), "\n"))
}

func TestPickerHeadsEachGroupWithASeparator(t *testing.T) {
	m := newPickerModel(t, 0, 80, 40)
	m.ui.Picker.projects = orderProjects([]domain.Project{
		grouped("Ferry", "Work", "2026-03-01T00:00:00Z"),
		grouped("Garden", "Perso", "2026-02-01T00:00:00Z"),
		grouped("Loose", "", "2026-04-01T00:00:00Z"),
	}, nil)

	lines := strings.Split(pickerText(m), "\n")[4:]
	require.Len(t, lines, 6)
	assert.Contains(t, lines[0], "━━ Work ━")
	assert.Contains(t, lines[1], "Ferry")
	assert.Contains(t, lines[2], "━━ Perso ━")
	assert.Contains(t, lines[3], "Garden")
	assert.Equal(t, strings.TrimSpace(lines[4]), strings.Repeat("━", len([]rune(strings.TrimSpace(lines[4])))), "the ungrouped take a bare rule")
	assert.Contains(t, lines[5], "Loose")
}

func TestPickerWithoutGroupsDrawsNoSeparator(t *testing.T) {
	m := newPickerModel(t, 3, 80, 40)
	body := pickerText(m)
	assert.NotContains(t, body, "━")
	assert.Len(t, pickerRows(m.ui.Picker.projects), 3)
}

// The window counts separators as rows, and scrolling up to a group's first project shows its heading.
func TestPickerScrollsOverSeparators(t *testing.T) {
	var projects []domain.Project
	for i, g := range []string{"A", "A", "A", "B", "B", "B", "C", "C", "C"} {
		p := makeProjects(9)[i]
		p.Group = g
		projects = append(projects, p)
	}
	p := &ProjectPickerState{projects: projects}
	const visible = 5

	p.selected = 8
	p.ensureVisible(visible)
	assert.Equal(t, 12-visible, p.scrollOffset, "the last project sits flush, three headings above it")

	p.selected = 3
	p.ensureVisible(visible)
	assert.Equal(t, 3, p.scrollOffset, "B's heading shows above its first project, a row of context above it")
	assert.Equal(t, -1, pickerRows(projects)[p.scrollOffset+1].project)
}

func TestPickerSearchCrossesGroups(t *testing.T) {
	m := newPickerModel(t, 0, 80, 40)
	m.ui.Picker.projects = []domain.Project{
		grouped("Ferry work", "Work", ""), grouped("Garden", "Perso", ""), grouped("Ferry home", "Perso", ""),
	}
	m.ui.Picker.startFiltering()
	m.ui.Picker.query = "ferry"
	m.ui.Picker.applyFilter(10)
	assert.Equal(t, []string{"Ferry work", "Ferry home"}, pickerNames(m.ui.Picker.projects))
	body := pickerText(m)
	assert.Contains(t, body, "Work")
	assert.Contains(t, body, "Perso")
}

func groupStoreModel(t *testing.T) (*model, *data.Store, domain.Project, domain.Project) {
	t.Helper()
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	alpha, err := store.CreateProject("Alpha")
	require.NoError(t, err)
	beta, err := store.CreateProject("Beta")
	require.NoError(t, err)
	alpha, err = store.LoadProjectByID(alpha.ID)
	require.NoError(t, err)
	m := newTestModel(t, alpha)
	withStubStateManager(m)
	m.deps.Store = store
	m.rebuildPositions()
	m.openProjectPicker()
	return m, store, alpha, beta
}

func typeIn(t *testing.T, m *model, text string) {
	t.Helper()
	for _, r := range text {
		pressPickerKey(t, m, string(r))
	}
}

func TestGroupKeySetsAnotherProjectsGroupAndKeepsItsStamp(t *testing.T) {
	m, store, _, beta := groupStoreModel(t)
	stamp := onDisk(t, store, beta.ID).UpdatedAt
	m.ui.Picker.selected = indexOf(m.ui.Picker.projects, beta.ID)

	pressPickerKey(t, m, "m")
	require.True(t, m.ui.Picker.grouping)
	typeIn(t, m, "Work")
	hit(m, "enter")

	got := onDisk(t, store, beta.ID)
	assert.Equal(t, "Work", got.Group)
	assert.Equal(t, stamp, got.UpdatedAt)
	assert.False(t, m.ui.Picker.grouping)
	assert.Equal(t, "Group set to “Work”", m.ui.Screen.StatusMsg)
	assert.Equal(t, beta.ID, m.ui.Picker.projects[m.ui.Picker.selected].ID, "the cursor follows the project")
	assert.Contains(t, pickerText(m), "━━ Work ━")
}

// The open project takes the file back, so its next save keeps the group.
func TestGroupOnTheOpenProjectSurvivesItsNextEdit(t *testing.T) {
	m, store, alpha, _ := groupStoreModel(t)
	m.ui.Picker.selected = indexOf(m.ui.Picker.projects, alpha.ID)

	pressPickerKey(t, m, "m")
	typeIn(t, m, "Perso")
	hit(m, "enter")
	assert.Equal(t, "Perso", m.project.Group)

	task := tasksIn(m.project, 1)[0]
	retitle(&m.project, task.ID, "Edited after")
	m.storeTaskUpdate()
	require.NoError(t, m.ui.Save.Err)

	got := onDisk(t, store, alpha.ID)
	assert.Equal(t, "Perso", got.Group)
	edited, _ := findTask(got, task.ID)
	assert.Equal(t, "Edited after", edited.Title)
}

func TestGroupFieldCompletesAndCyclesTheGroupsInUse(t *testing.T) {
	m := newPickerModel(t, 0, 80, 40)
	m.ui.Picker.projects = []domain.Project{
		grouped("Ferry", "Work", ""), grouped("Garden", "Perso", ""), {ID: "loose", Name: "Loose"},
	}
	m.ui.Picker.selected = 2
	pressPickerKey(t, m, "m")
	p := &m.ui.Picker
	assert.Equal(t, "Work", p.groupHint(), "an empty field offers the most recent group")

	typeIn(t, m, "P")
	assert.Equal(t, "erso", p.groupHint())
	hit(m, "tab")
	assert.Equal(t, "Perso", p.input.Value())

	p.input.SetValue("Work")
	p.groupCycling = false
	hit(m, "tab")
	assert.Equal(t, "Perso", p.input.Value(), "a field naming a group cycles to the next")
	hit(m, "tab")
	assert.Equal(t, "Work", p.input.Value())

	hit(m, "esc")
	assert.False(t, p.grouping)
}

func TestGroupFieldEmptiedClearsTheGroup(t *testing.T) {
	m, store, alpha, _ := groupStoreModel(t)
	_, err := store.WithProjectLocked(alpha.ID, func(p *domain.Project) error {
		p.Group = "Work"
		return nil
	})
	require.NoError(t, err)
	m.openProjectPicker()
	m.ui.Picker.selected = indexOf(m.ui.Picker.projects, alpha.ID)

	pressPickerKey(t, m, "m")
	assert.Equal(t, "Work", m.ui.Picker.input.Value(), "the field starts on the project's group")
	hit(m, "ctrl+u", "enter")
	assert.Empty(t, onDisk(t, store, alpha.ID).Group)
	assert.Equal(t, "Group cleared", m.ui.Screen.StatusMsg)
}

func indexOf(projects []domain.Project, id string) int {
	for i, p := range projects {
		if p.ID == id {
			return i
		}
	}
	return -1
}
