package cli

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/attachments"
)

func pngFile(t *testing.T) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 3, 3))))
	path := filepath.Join(t.TempDir(), "crash.png")
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o600))
	return path
}

func addShotTask(t *testing.T) string {
	t.Helper()
	dir := initDefault(t)
	_, err := runCLI(t, dir, "task", "add", "Crash on save", "--description", "Seen twice.", "--category", "Feature", "--project", "Default")
	require.NoError(t, err)
	return dir
}

var storedRef = regexp.MustCompile(`!\[([^\]]*)\]\((attachments/[0-9a-f]{32}\.png)\)`)

func shotDescription(t *testing.T, dir string) string {
	t.Helper()
	task, ok := findTask(firstCategoryTasks(t, loadDefault(t, dir), "Feature"), "Crash on save")
	require.True(t, ok)
	return task.Description
}

func TestTaskAttach_StoresTheFileAndReferencesIt(t *testing.T) {
	dir := addShotTask(t)

	out, err := runCLI(t, dir, "task", "attach", "Crash on save", pngFile(t), "--project", "Default")

	require.NoError(t, err)
	desc := shotDescription(t, dir)
	m := storedRef.FindStringSubmatch(desc)
	require.NotNil(t, m, desc)
	assert.True(t, strings.HasPrefix(desc, "Seen twice.\n!["), "the reference goes on a line of its own")
	assert.Equal(t, "crash.png", m[1])
	stored := filepath.Join(dir, "projects", m[2])
	assert.FileExists(t, stored)
	assert.Contains(t, out, "Attached crash.png to Crash on save: "+stored)
}

func TestTaskAttach_ReadsStdin(t *testing.T) {
	dir := addShotTask(t)
	shot, err := os.ReadFile(pngFile(t))
	require.NoError(t, err)
	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetIn(bytes.NewReader(shot))
	cmd.SetArgs([]string{"--data", dir, "task", "attach", "Crash on save", "-", "--name", "the [crash]", "--project", "Default"})

	require.NoError(t, cmd.Execute())

	m := storedRef.FindStringSubmatch(shotDescription(t, dir))
	require.NotNil(t, m)
	assert.Equal(t, "the crash", m[1])
}

func TestTaskAttach_RefusesWhatIsNotAnImage(t *testing.T) {
	dir := addShotTask(t)
	text := filepath.Join(t.TempDir(), "notes.txt")
	require.NoError(t, os.WriteFile(text, []byte("not an image"), 0o600))

	_, err := runCLI(t, dir, "task", "attach", "Crash on save", text, "--project", "Default")
	assert.ErrorIs(t, err, attachments.ErrNotImage)

	_, err = runCLI(t, dir, "task", "attach", "No such task", pngFile(t), "--project", "Default")
	assert.ErrorContains(t, err, "not found")
	assert.NoDirExists(t, filepath.Join(dir, "projects", attachments.DirName), "nothing stored for a task that isn't there")
	assert.Equal(t, "Seen twice.", shotDescription(t, dir))
}

func TestTaskShow_ListsTheImagesWithTheirPath(t *testing.T) {
	dir := addShotTask(t)
	_, err := runCLI(t, dir, "task", "attach", "Crash on save", pngFile(t), "--project", "Default")
	require.NoError(t, err)
	desc := shotDescription(t, dir)
	stored := filepath.Join(dir, "projects", storedRef.FindStringSubmatch(desc)[2])
	gone := "attachments/0123456789abcdef0123456789abcdef.png"
	desc += "\n![later](" + gone + ")"
	_, err = runCLI(t, dir, "task", "edit", "Crash on save", "--description", desc, "--project", "Default")
	require.NoError(t, err)

	out, err := runCLI(t, dir, "task", "show", "Crash on save", "--project", "Default")
	require.NoError(t, err)
	assert.Contains(t, out, "![crash.png]("+stored+")", "the description points at the file too")
	assert.Regexp(t, `Images:\n  crash\.png +`+regexp.QuoteMeta(stored)+`\n  later +`+regexp.QuoteMeta(filepath.Join(dir, "projects", gone))+` \(missing\)\n`, out)

	out, err = runCLI(t, dir, "task", "show", "Crash on save", "--project", "Default", "--json")
	require.NoError(t, err)
	var got TaskDetailOutput
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Equal(t, desc, got.Task.Description, "JSON keeps the description as stored")
	assert.Equal(t, []TaskImage{
		{Name: "crash.png", Path: stored},
		{Name: "later", Path: filepath.Join(dir, "projects", gone), Missing: true},
	}, got.Task.Images)
}

func TestAttachmentsPath_PrintsTheDirectory(t *testing.T) {
	dir := initDefault(t)
	out, err := runCLI(t, dir, "attachments", "path")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "projects", attachments.DirName)+"\n", out)
}

func TestExport_MarkdownPointsImagesAtTheirFileJSONStaysRaw(t *testing.T) {
	dir := addShotTask(t)
	_, err := runCLI(t, dir, "task", "attach", "Crash on save", pngFile(t), "--project", "Default")
	require.NoError(t, err)
	target := storedRef.FindStringSubmatch(shotDescription(t, dir))[2]

	out, err := runCLI(t, dir, "export", "-f", "markdown", "--project", "Default")
	require.NoError(t, err)
	assert.Contains(t, out, "    ![crash.png]("+filepath.Join(dir, "projects", target)+")\n")

	out, err = runCLI(t, dir, "export", "-f", "json", "--project", "Default")
	require.NoError(t, err)
	assert.Contains(t, out, "]("+target+")")
}
