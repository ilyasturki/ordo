package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ilyasturki/phasionary/internal/data"
	"github.com/ilyasturki/phasionary/internal/domain"
)

// runCLI executes the root command with an explicit --data dir and returns the
// combined output. Tests must not run in parallel: the command wires the global
// viper, so concurrent runs would race on it.
func runCLI(t *testing.T, dataDir string, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(append([]string{"--data", dataDir}, args...))
	err := cmd.Execute()
	return buf.String(), err
}

// loadDefault reads the "Default" project straight from disk, so assertions see
// persisted state rather than command output.
func loadDefault(t *testing.T, dataDir string) domain.Project {
	t.Helper()
	store := data.NewStore(filepath.Join(dataDir, "projects"))
	p, err := store.LoadProject("Default")
	if err != nil {
		t.Fatalf("load Default project: %v", err)
	}
	return p
}

func firstCategoryTasks(t *testing.T, p domain.Project, name string) []domain.Task {
	t.Helper()
	for _, c := range p.Categories {
		if c.Name == name {
			return c.Tasks
		}
	}
	t.Fatalf("category %q not found in project", name)
	return nil
}

// findTask returns the task with the given title, or ok=false. Projects are
// seeded with sample tasks, so assertions target the task under test by title
// rather than by position or count.
func findTask(tasks []domain.Task, title string) (domain.Task, bool) {
	for _, t := range tasks {
		if t.Title == title {
			return t, true
		}
	}
	return domain.Task{}, false
}

func TestTaskAdd_PersistsWithParsedEstimate(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCLI(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	if _, err := runCLI(t, dir, "task", "add", "Write docs",
		"--category", "Feature", "--priority", "high", "--estimate", "2h30m",
		"--project", "Default"); err != nil {
		t.Fatalf("task add: %v", err)
	}

	tasks := firstCategoryTasks(t, loadDefault(t, dir), "Feature")
	got, ok := findTask(tasks, "Write docs")
	if !ok {
		t.Fatal(`added task "Write docs" not found in Feature`)
	}
	if got.Priority != domain.PriorityHigh {
		t.Errorf("priority = %q, want high", got.Priority)
	}
	if got.EstimateMinutes != 150 {
		t.Errorf("estimate = %d, want 150 (2h30m)", got.EstimateMinutes)
	}
	if got.Status != domain.StatusTodo {
		t.Errorf("status = %q, want todo", got.Status)
	}
}

// The pre-refactor CLI parsed "-5" as -5 minutes and stored it. The unified
// parser must reject it — proving the bug fix at the CLI seam.
func TestTaskAdd_RejectsNegativeEstimate(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCLI(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	if _, err := runCLI(t, dir, "task", "add", "bad",
		"--category", "Feature", "--estimate", "-5", "--project", "Default"); err == nil {
		t.Fatal("expected task add to fail on negative estimate")
	}

	tasks := firstCategoryTasks(t, loadDefault(t, dir), "Feature")
	if _, ok := findTask(tasks, "bad"); ok {
		t.Fatal(`negative-estimate add persisted task "bad", want it rejected`)
	}
}

func TestTaskAdd_UnknownCategory(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCLI(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}

	if _, err := runCLI(t, dir, "task", "add", "x",
		"--category", "Nope", "--project", "Default"); err == nil {
		t.Fatal("expected task add to fail on unknown category")
	}
}

func TestTaskDescription_AddShowEditClear(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCLI(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := runCLI(t, dir, "task", "add", "Call the bank",
		"--category", "Feature", "--description", "ask about fees\nbring the contract",
		"--project", "Default"); err != nil {
		t.Fatalf("task add: %v", err)
	}

	out, err := runCLI(t, dir, "task", "show", "Call the bank", "--json", "--project", "Default")
	if err != nil {
		t.Fatalf("task show --json: %v", err)
	}
	var shown TaskDetailOutput
	if err := json.Unmarshal([]byte(out), &shown); err != nil {
		t.Fatalf("decode %q: %v", out, err)
	}
	if want := "ask about fees\nbring the contract"; shown.Task.Description != want {
		t.Errorf("json description = %q, want %q", shown.Task.Description, want)
	}

	if _, err := runCLI(t, dir, "task", "edit", "Call the bank",
		"--description", "fees only", "--project", "Default"); err != nil {
		t.Fatalf("task edit: %v", err)
	}
	out, err = runCLI(t, dir, "task", "show", "Call the bank", "--json=false", "--project", "Default")
	if err != nil {
		t.Fatalf("task show: %v", err)
	}
	if !strings.HasSuffix(out, "\n\nfees only\n") {
		t.Errorf("text show should end with the description, got:\n%s", out)
	}

	if _, err := runCLI(t, dir, "task", "edit", "Call the bank",
		"--description", "", "--project", "Default"); err != nil {
		t.Fatalf("task edit clear: %v", err)
	}
	got, _ := findTask(firstCategoryTasks(t, loadDefault(t, dir), "Feature"), "Call the bank")
	if got.Description != "" {
		t.Errorf("description = %q after clearing, want empty", got.Description)
	}
}

func TestTaskEdit_RejectsControlCharsInDescription(t *testing.T) {
	dir := t.TempDir()
	if _, err := runCLI(t, dir, "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := runCLI(t, dir, "task", "add", "Plain", "--category", "Feature", "--project", "Default"); err != nil {
		t.Fatalf("task add: %v", err)
	}
	if _, err := runCLI(t, dir, "task", "edit", "Plain",
		"--description", "bad\x1b[31m", "--project", "Default"); err == nil {
		t.Fatal("expected edit to reject an escape sequence in the description")
	}
}
