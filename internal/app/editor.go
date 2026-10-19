package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

type editorFinishedMsg struct {
	err      error
	tempFile string
}

type ExternalEditState struct {
	TempFilePath  string
	ItemType      selection.FocusKind
	CategoryIndex int
	TaskIndex     int
}

func (e *ExternalEditState) reset() {
	e.TempFilePath = ""
	e.ItemType = selection.FocusProject
	e.CategoryIndex = -1
	e.TaskIndex = -1
}

func getEditorCmd() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vim"
}

func formatTaskForEdit(task domain.Task) string {
	if task.Description == "" {
		return task.Title
	}
	return task.Title + "\n\n" + task.Description
}

func parseTaskEdit(content string) (string, string) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	lines := strings.Split(content, "\n")

	titleIdx := -1
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			titleIdx = i
			break
		}
	}
	if titleIdx < 0 {
		return "", ""
	}
	title := strings.TrimSpace(lines[titleIdx])

	if titleIdx+1 >= len(lines) {
		return title, ""
	}
	body := strings.Trim(strings.Join(lines[titleIdx+1:], "\n"), "\n")
	if strings.TrimSpace(body) == "" {
		return title, ""
	}
	return title, body
}

func formatCategoryForEdit(category domain.Category) string {
	return category.Name
}

func formatProjectForEdit(project domain.Project) string {
	return project.Name
}

func (m *model) startExternalEdit() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok {
		return nil
	}
	// Separators and the hidden row carry no editable body; the external editor is a no-op on them.
	if pos.Kind == selection.FocusSeparator || pos.Kind == selection.FocusHidden {
		return nil
	}

	itemType := pos.Kind
	var content string
	switch pos.Kind {
	case selection.FocusProject:
		content = formatProjectForEdit(m.project)
	case selection.FocusCategory:
		content = formatCategoryForEdit(m.project.Categories[pos.CategoryIndex])
	case selection.FocusTask:
		content = formatTaskForEdit(m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex])
	case selection.FocusDescription:
		// `e` from a description row edits the whole task externally
		// (title + description), matching the user-facing intent.
		content = formatTaskForEdit(m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex])
		itemType = selection.FocusTask
	}

	tempFile, err := os.CreateTemp("", "ordo-edit-*.txt")
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Failed to create temp file: %v", err)
		return nil
	}

	if _, err := tempFile.WriteString(content); err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		m.ui.Screen.StatusMsg = fmt.Sprintf("Failed to write temp file: %v", err)
		return nil
	}
	tempFile.Close()

	m.ui.ExternalEdit = ExternalEditState{
		TempFilePath:  tempFile.Name(),
		ItemType:      itemType,
		CategoryIndex: pos.CategoryIndex,
		TaskIndex:     pos.TaskIndex,
	}

	m.ui.Modes.ToExternalEdit()

	editor := getEditorCmd()
	c := exec.Command(editor, tempFile.Name())

	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err, tempFile: tempFile.Name()}
	})
}

func (m *model) handleEditorFinished(msg editorFinishedMsg) {
	// The editor owned the terminal, so a copy made in it or a focus loss sent no BlurMsg.
	m.distrustClipboardWrites()
	defer func() {
		os.Remove(msg.tempFile)
		m.ui.ExternalEdit.reset()
		m.ui.Modes.ToNormal()
	}()

	if msg.err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Editor error: %v", msg.err)
		return
	}

	content, err := os.ReadFile(msg.tempFile)
	if err != nil {
		m.ui.Screen.StatusMsg = fmt.Sprintf("Failed to read edited file: %v", err)
		return
	}

	switch m.ui.ExternalEdit.ItemType {
	case selection.FocusProject:
		m.applyProjectEdit(string(content))
	case selection.FocusCategory:
		m.applyCategoryEdit(string(content))
	case selection.FocusTask:
		m.applyTaskEdit(string(content))
	}
}

func (m *model) applyProjectEdit(content string) {
	name := strings.TrimSpace(content)
	if name == "" {
		m.ui.Screen.StatusMsg = "Project name cannot be empty"
		return
	}

	if name == m.project.Name {
		return
	}

	if err := m.renameProject(name); err != nil {
		if errors.Is(err, data.ErrDuplicateProjectName) {
			m.ui.Screen.StatusMsg = "A project with that name already exists"
			return
		}
		m.ui.Screen.StatusMsg = fmt.Sprintf("Failed to save: %v", err)
		return
	}
	m.ui.Screen.StatusMsg = "Project updated"
}

func (m *model) applyCategoryEdit(content string) {
	idx := m.ui.ExternalEdit.CategoryIndex
	if idx < 0 || idx >= len(m.project.Categories) {
		m.ui.Screen.StatusMsg = "Category no longer exists"
		return
	}

	name := strings.TrimSpace(content)
	if name == "" {
		m.ui.Screen.StatusMsg = "Category name cannot be empty"
		return
	}

	if name == m.project.Categories[idx].Name {
		return
	}

	m.recordHistory()
	if err := m.project.RenameCategory(idx, name); err != nil {
		m.discardLastHistory()
		if errors.Is(err, domain.ErrDuplicateCategoryName) {
			m.ui.Screen.StatusMsg = "A category with that name already exists"
			return
		}
		m.ui.Screen.StatusMsg = fmt.Sprintf("Rename failed: %v", err)
		return
	}
	m.storeTaskUpdate()
	if m.ui.Screen.StatusMsg == "" {
		m.ui.Screen.StatusMsg = "Category updated"
	}
}

func (m *model) applyTaskEdit(content string) {
	catIdx := m.ui.ExternalEdit.CategoryIndex
	taskIdx := m.ui.ExternalEdit.TaskIndex

	if catIdx < 0 || catIdx >= len(m.project.Categories) {
		m.ui.Screen.StatusMsg = "Category no longer exists"
		return
	}
	if taskIdx < 0 || taskIdx >= len(m.project.Categories[catIdx].Tasks) {
		m.ui.Screen.StatusMsg = "Task no longer exists"
		return
	}

	title, description := parseTaskEdit(content)
	if title == "" {
		m.ui.Screen.StatusMsg = "Task title cannot be empty"
		return
	}

	task := &m.project.Categories[catIdx].Tasks[taskIdx]
	if title == task.Title && description == task.Description {
		return
	}

	m.recordHistory()
	task.Title = title
	task.Description = description
	task.UpdatedAt = domain.NowTimestamp()
	m.storeTaskUpdate()
	if m.ui.Screen.StatusMsg == "" {
		m.ui.Screen.StatusMsg = "Task updated"
	}
}
