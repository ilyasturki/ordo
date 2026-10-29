package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func TestCycleTag_UntaggedTakesFirstMarkFreeAmongOpenTasks(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[1].TagColor = domain.TagGreen // t2, in progress
	p.Categories[0].Tasks[2].TagColor = domain.TagBlue  // t3, completed: frees blue
	p.Categories[1].Tasks[0].TagColor = domain.TagBlue  // other category
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2) // t1

	m.cycleTag()
	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[0].TagColor)

	m.cycleTag()
	assert.Equal(t, domain.TagMagenta, m.project.Categories[0].Tasks[0].TagColor, "a tagged task keeps cycling")
}

func TestVisualCycleTag_IgnoresMarksInsideTheBlock(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[1].TagColor = domain.TagGreen // t2
	p.Categories[1].Tasks[0].TagColor = domain.TagBlue  // t4
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(3) // t2, t3, t4

	m.visualCycleTag()

	for _, task := range []domain.Task{
		m.project.Categories[0].Tasks[0],
		m.project.Categories[0].Tasks[1],
		m.project.Categories[0].Tasks[2],
		m.project.Categories[1].Tasks[0],
	} {
		assert.Equal(t, domain.TagGreen, task.TagColor, task.ID)
	}
	assert.Equal(t, "", m.project.Categories[1].Tasks[1].TagColor, "t5 sits outside the block")
}

func TestVisualCycleTag_SkipsMarksOpenOutsideTheBlock(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[1].TagColor = domain.TagGreen // t2, in progress
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2) // t1 alone
	m.enterVisualMode()

	m.visualCycleTag()
	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[0].TagColor)
}

func TestTagEdit_BatchRecoloursEveryTaskSharingTheMarkInOneUndo(t *testing.T) {
	p := sampleProject()
	for i := range p.Categories[0].Tasks {
		p.Categories[0].Tasks[i].TagColor = domain.TagGreen
	}
	p.Categories[1].Tasks[0].TagColor = domain.TagGreen
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2) // t1

	m.startTagEdit()
	require.Len(t, m.ui.TagEdit.batchIDs, 3)
	m.handleTagEditKey(tea.KeyPressMsg{Code: tea.KeySpace})
	require.True(t, m.ui.TagEdit.batch)
	m.ui.TagEdit.colorIdx = tagColorIndex(domain.TagCyan)
	m.handleTagEditKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	for _, task := range m.project.Categories[0].Tasks {
		assert.Equal(t, domain.TagCyan, task.TagColor, task.ID)
	}
	assert.Equal(t, domain.TagGreen, m.project.Categories[1].Tasks[0].TagColor, "other categories keep the mark")

	m.undo()
	for _, task := range m.project.Categories[0].Tasks {
		assert.Equal(t, domain.TagGreen, task.TagColor, task.ID)
	}
}

func TestTagEdit_WithoutBatchOnlyTheTaskChanges(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagGreen
	p.Categories[0].Tasks[1].TagColor = domain.TagGreen
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2)

	m.startTagEdit()
	m.ui.TagEdit.colorIdx = tagColorIndex(domain.TagBlue)
	m.finishTagEdit()

	assert.Equal(t, domain.TagBlue, m.project.Categories[0].Tasks[0].TagColor)
	assert.Equal(t, domain.TagGreen, m.project.Categories[0].Tasks[1].TagColor)
}

func TestTagEdit_NoBatchOfferForALoneMark(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].TagColor = domain.TagGreen
	m := newTestModel(t, p)
	m.ui.Selection.SetSelected(2)

	m.startTagEdit()
	assert.Empty(t, m.ui.TagEdit.batchIDs)
	m.handleTagEditKey(tea.KeyPressMsg{Code: tea.KeySpace})
	assert.False(t, m.ui.TagEdit.batch)
}
