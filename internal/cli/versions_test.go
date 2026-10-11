package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedVersions(t *testing.T) string {
	t.Helper()
	dir := initDefault(t)
	for _, args := range [][]string{
		{"category", "add", "v0.2.0 - polish"},
		{"category", "add", "v0.1.0"},
		{"category", "add", "ui", "--parent", "v0.2.0 - polish"},
		{"task", "add", "Ship the menu", "-C", "v0.2.0 - polish"},
		{"task", "add", "Dark theme", "-C", "v0.2.0 - polish/ui"},
		{"task", "add", "Login", "-C", "v0.1.0"},
	} {
		_, err := runCLI(t, dir, append(args, "--project", "Default")...)
		require.NoError(t, err, args)
	}
	_, err := runCLI(t, dir, "task", "status", "Dark theme", "in_progress", "--project", "Default")
	require.NoError(t, err)
	return dir
}

func TestVersionsListsOnlyVersionsWithSubCategoryCounts(t *testing.T) {
	dir := seedVersions(t)
	out, err := runCLI(t, dir, "versions", "--project", "Default", "--json")
	require.NoError(t, err)
	var got VersionsOutput
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Len(t, got.Versions, 2)
	assert.Equal(t, "v0.2.0 - polish", got.Versions[0].Name)
	assert.Equal(t, 2, got.Versions[0].TaskCount)
	assert.Equal(t, 1, got.Versions[0].InProgress)
	assert.Empty(t, got.Versions[0].ReleasedAt)
	assert.Equal(t, "v0.1.0", got.Versions[1].Name)
}

func TestCategoryShipStampsAndUnshipClears(t *testing.T) {
	dir := seedVersions(t)
	out, err := runCLI(t, dir, "category", "ship", "v0.1.0", "--project", "Default")
	require.NoError(t, err)
	assert.Contains(t, out, "Shipped version: v0.1.0")

	p := loadDefault(t, dir)
	_, idx, err := resolveCategory(p, "v0.1.0")
	require.NoError(t, err)
	assert.NotEmpty(t, p.Categories[idx].ReleasedAt)
	assert.Equal(t, 3, p.Schema, "a ship date needs schema 3")

	out, err = runCLI(t, dir, "versions", "--project", "Default")
	require.NoError(t, err)
	assert.Regexp(t, `v0\.1\.0\s+0/1\s+0\s+\d{4}-\d{2}-\d{2} \(1 open\)`, out)

	_, err = runCLI(t, dir, "category", "ship", "v0.1.0", "--project", "Default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already shipped")

	_, err = runCLI(t, dir, "category", "unship", "v0.1.0", "--project", "Default")
	require.NoError(t, err)
	p = loadDefault(t, dir)
	assert.Empty(t, p.Categories[idx].ReleasedAt)
	assert.Equal(t, 2, p.Schema, "back to the nesting schema once nothing is shipped")
}

func TestCategoryShipRefusesNonVersions(t *testing.T) {
	dir := seedVersions(t)
	for _, name := range []string{"Feature", "v0.2.0 - polish/ui"} {
		_, err := runCLI(t, dir, "category", "ship", name, "--project", "Default")
		require.Error(t, err, name)
		assert.True(t, strings.Contains(err.Error(), "is not a version"), err.Error())
	}
	_, err := runCLI(t, dir, "category", "unship", "Feature", "--project", "Default")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not shipped")
}

func TestVersionsWithoutAny(t *testing.T) {
	dir := initDefault(t)
	out, err := runCLI(t, dir, "versions", "--project", "Default")
	require.NoError(t, err)
	assert.Contains(t, out, "No versions found")
	out, err = runCLI(t, dir, "versions", "--project", "Default", "--json")
	require.NoError(t, err)
	assert.JSONEq(t, `{"versions":[]}`, out)
}
