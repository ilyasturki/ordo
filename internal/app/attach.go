package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/clipboard"
	"github.com/ilyasturki/ordo/internal/domain"
)

// pendingImage is an image on its way into the store: the clipboard's bytes,
// or a file read from path.
type pendingImage struct {
	name string
	data []byte
	path string
}

func clipboardImage(data []byte) pendingImage {
	return pendingImage{name: "Screenshot " + time.Now().Format("2006-01-02 15.04.05"), data: data}
}

func imageFiles(paths []string) []pendingImage {
	images := make([]pendingImage, len(paths))
	for i, p := range paths {
		images[i] = pendingImage{name: filepath.Base(p), path: p}
	}
	return images
}

type imagesStoredMsg struct {
	taskID      string
	retiresCopy bool
	refs        []attachments.Ref
	largest     int
	failed      string
	err         error
}

// attachImages stores images off the event loop, before the save that
// references them.
func (m *model) attachImages(images []pendingImage, retiresCopy bool) tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok || (pos.Kind != selection.FocusTask && pos.Kind != selection.FocusDescription) {
		m.ui.Screen.StatusMsg = "Focus a task to attach an image"
		return nil
	}
	taskID := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID
	return storeImages(m.deps.Attachments, imagesStoredMsg{taskID: taskID, retiresCopy: retiresCopy}, images)
}

func (m *model) attachImagesInEditor(images []pendingImage) tea.Cmd {
	return storeImages(m.deps.Attachments, imagesStoredMsg{taskID: m.descriptionEditTaskID()}, images)
}

var errNoStore = errors.New("no attachments directory")

func storeImages(store *attachments.Store, msg imagesStoredMsg, images []pendingImage) tea.Cmd {
	return func() tea.Msg {
		for _, img := range images {
			data, err := img.data, error(nil)
			if data == nil {
				data, err = attachments.ReadImage(img.path)
			}
			var target string
			switch {
			case err != nil:
			case store == nil:
				err = errNoStore
			default:
				target, err = store.Add(data)
			}
			if err != nil {
				msg.failed, msg.err = img.name, err
				return msg
			}
			msg.refs = append(msg.refs, attachments.Ref{Name: img.name, Target: target})
			if len(data) > attachments.WarnSize {
				msg.largest = max(msg.largest, len(data))
			}
		}
		return msg
	}
}

func (m *model) handleImagesStored(msg imagesStoredMsg) tea.Cmd {
	if msg.err != nil {
		m.ui.Screen.StatusMsg = attachFailure(msg.failed, msg.err)
		return nil
	}
	lines := make([]string, len(msg.refs))
	for i, r := range msg.refs {
		lines[i] = attachments.Line(r.Name, r.Target)
	}
	block := strings.Join(lines, "\n")
	ci, ti := m.findTaskByID(msg.taskID)
	if ci < 0 {
		m.ui.Screen.StatusMsg = "The task is gone: nothing attached"
		return nil
	}
	task := &m.project.Categories[ci].Tasks[ti]
	status := attachedStatus(msg, task.Title)
	if m.ui.Modes.IsDescriptionEdit() && m.descriptionEditTaskID() == msg.taskID {
		var cmd tea.Cmd
		m.ui.DescriptionEdit.textarea, cmd = m.ui.DescriptionEdit.textarea.Update(tea.PasteMsg{Content: block})
		m.ui.Screen.StatusMsg = status
		return cmd
	}
	// An editor or a visual range holds indexes and an undo step of its own.
	if !m.ui.Modes.IsNormal() {
		m.ui.Screen.StatusMsg = "Image not attached: it arrived mid-edit, press p again"
		return nil
	}
	m.recordHistory()
	if msg.retiresCopy {
		m.ui.Clipboard = ClipboardState{}
	}
	task.Description = appendDescription(task.Description, block)
	task.UpdatedAt = domain.NowTimestamp()
	m.rebuildPositions()
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = status
	return nil
}

func attachedStatus(msg imagesStoredMsg, title string) string {
	what := msg.refs[0].Name
	if len(msg.refs) > 1 {
		what = fmt.Sprintf("%d images", len(msg.refs))
	}
	status := "Attached " + what + " to " + title
	if msg.largest > 0 {
		status += " (large: " + attachments.FormatSize(int64(msg.largest)) + ")"
	}
	return status
}

func attachFailure(name string, err error) string {
	switch {
	case errors.Is(err, attachments.ErrTooLarge):
		return fmt.Sprintf("%s is over %d MB: not attached", name, attachments.MaxSize>>20)
	case errors.Is(err, attachments.ErrNotImage):
		return name + ": " + err.Error()
	}
	return fmt.Sprintf("Can't attach %s: %v", name, err)
}

// pasteTextOrImages attaches pasted text naming nothing but image files, as a
// terminal's drop of them does, and pastes any other text as rows.
func (m *model) pasteTextOrImages(text string, retiresCopy bool) tea.Cmd {
	if paths, ok := droppedImages(text); ok {
		return m.attachImages(imageFiles(paths), retiresCopy)
	}
	m.pasteText(text, retiresCopy)
	return nil
}

// droppedImages reads text as dropped files: words in shell quoting, or
// file:// URIs. ok only when every word is an image file.
func droppedImages(text string) ([]string, bool) {
	words, ok := shellWords(text)
	if !ok || len(words) == 0 {
		return nil, false
	}
	paths := make([]string, len(words))
	for i, w := range words {
		if paths[i], ok = dropPath(w); !ok {
			return nil, false
		}
	}
	return paths, allImageFiles(paths)
}

func allImageFiles(paths []string) bool {
	for _, p := range paths {
		if !attachments.IsImageFile(p) {
			return false
		}
	}
	return len(paths) > 0
}

func dropPath(word string) (string, bool) {
	if strings.HasPrefix(word, "file://") {
		return clipboard.FilePath(word)
	}
	if rest, ok := strings.CutPrefix(word, "~/"); ok {
		home, err := os.UserHomeDir()
		return filepath.Join(home, rest), err == nil
	}
	return word, filepath.IsAbs(word)
}

// shellWords splits text the way a shell would, which is how terminals quote
// a dropped path; ok is false for an unclosed quote.
func shellWords(text string) ([]string, bool) {
	var words []string
	var word strings.Builder
	inWord := false
	var quote rune
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		switch {
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case quote == '"':
			switch {
			case r == '"':
				quote = 0
			case r == '\\' && i+1 < len(runes) && strings.ContainsRune("\"\\$`", runes[i+1]):
				i++
				word.WriteRune(runes[i])
			default:
				word.WriteRune(r)
			}
		case r == '\\' && i+1 < len(runes):
			i++
			if runes[i] != '\n' {
				word.WriteRune(runes[i])
				inWord = true
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case unicode.IsSpace(r):
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, false
	}
	if inWord {
		words = append(words, word.String())
	}
	return words, true
}
