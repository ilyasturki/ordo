package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectGroup_SetsShowsAndClears(t *testing.T) {
	dir := initDefault(t)
	stamp := loadDefault(t, dir).UpdatedAt

	out, err := runCLI(t, dir, "project", "group", "Default", " Work ")
	require.NoError(t, err)
	assert.Contains(t, out, "Group set to “Work”")
	got := loadDefault(t, dir)
	assert.Equal(t, "Work", got.Group)
	assert.Equal(t, stamp, got.UpdatedAt, "a group moves no project in the recency order")

	out, err = runCLI(t, dir, "project", "group", "Default")
	require.NoError(t, err)
	assert.Equal(t, "Work\n", out)

	out, err = runCLI(t, dir, "project", "show", "Default")
	require.NoError(t, err)
	assert.Contains(t, out, "Group:      Work\n")

	_, err = runCLI(t, dir, "project", "group", "Default", "")
	require.NoError(t, err)
	assert.Empty(t, loadDefault(t, dir).Group)
	out, err = runCLI(t, dir, "project", "group", "Default")
	require.NoError(t, err)
	assert.Equal(t, "Default has no group.\n", out)
}

func TestProjectGroup_RefusesTwoLines(t *testing.T) {
	dir := initDefault(t)
	_, err := runCLI(t, dir, "project", "group", "Default", "Work\nPerso")
	require.Error(t, err)
	assert.Empty(t, loadDefault(t, dir).Group)
}

func TestProjects_ListByGroupOnceOneIsSet(t *testing.T) {
	dir := initDefault(t)
	out, err := runCLI(t, dir, "projects")
	require.NoError(t, err)
	assert.Equal(t, "Default\n", out, "with no group the list stays one name per line")

	for _, name := range []string{"Ferry", "Garden"} {
		_, err = runCLI(t, dir, "project", "add", name)
		require.NoError(t, err)
	}
	_, err = runCLI(t, dir, "project", "group", "Default", "Work")
	require.NoError(t, err)
	_, err = runCLI(t, dir, "project", "group", "Garden", "Perso")
	require.NoError(t, err)

	out, err = runCLI(t, dir, "project", "list")
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.Len(t, lines, 4)
	assert.Equal(t, []string{"NAME", "GROUP"}, strings.Fields(lines[0]))
	assert.Equal(t, []string{"Ferry"}, strings.Fields(lines[3]), "the ungrouped come last")

	out, err = runCLI(t, dir, "projects", "--json")
	require.NoError(t, err)
	var got ProjectsOutput
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	groups := map[string]string{}
	for _, p := range got.Projects {
		groups[p.Name] = p.Group
	}
	assert.Equal(t, map[string]string{"Default": "Work", "Garden": "Perso", "Ferry": ""}, groups)
}
