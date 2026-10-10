package app

import (
	"strings"

	"phasionary/internal/app/components"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

type LayoutItemKind int

const (
	LayoutProject LayoutItemKind = iota
	LayoutCategory
	LayoutTask
	LayoutSeparator     // In-category divider row (domain.KindSeparator)
	LayoutDescription   // Inline description block for the task above it (own focusable row)
	LayoutEmptyCategory // "(no tasks)" placeholder
	LayoutFolded        // "(folded)" placeholder
	LayoutSpacing       // Blank lines between elements
)

type LayoutItem struct {
	Kind          LayoutItemKind
	Height        int // Screen rows this item occupies
	PositionIndex int // Index into model.positions (-1 for non-selectable)
	CategoryIndex int
	TaskIndex     int
}

type Layout struct {
	Items       []LayoutItem
	TotalHeight int
}

type LayoutConfig struct {
	FooterHeight      int
	BlankAfterProject int
	BlankBetweenCats  int
	BlankAfterCatHead int
}

func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		FooterHeight:      footerHeight,
		BlankAfterProject: blankAfterProj,
		BlankBetweenCats:  blankBetweenCat,
		BlankAfterCatHead: blankAfterCat,
	}
}

type LayoutBuilder struct {
	config             LayoutConfig
	width              int
	statusDisplay      string
	filter             *FilterState
	fold               *FoldState
	expandDescriptions bool
	limits             rowLimits
	// editHeight of 0 means no editor is open on editPos.
	editPos    int
	editHeight int
	// A cut row carries a ✂ badge that narrows its title.
	taskCut     func(taskID string) bool
	categoryCut func(categoryID string) bool
}

func NewLayoutBuilder(config LayoutConfig, width int, statusDisplay string, filter *FilterState, fold *FoldState) *LayoutBuilder {
	return &LayoutBuilder{
		config:        config,
		width:         width,
		statusDisplay: statusDisplay,
		filter:        filter,
		fold:          fold,
		limits:        defaultRowLimits(),
	}
}

func (b *LayoutBuilder) WithExpandedDescriptions(v bool) *LayoutBuilder {
	b.expandDescriptions = v
	return b
}

func (b *LayoutBuilder) WithRowLimits(l rowLimits) *LayoutBuilder {
	b.limits = l
	return b
}

func (b *LayoutBuilder) WithCutMarks(taskCut, categoryCut func(string) bool) *LayoutBuilder {
	b.taskCut, b.categoryCut = taskCut, categoryCut
	return b
}

func (b *LayoutBuilder) WithEditOverlay(posIndex, height int) *LayoutBuilder {
	b.editPos, b.editHeight = posIndex, height
	return b
}

// rowHeight is fallback unless an editor is open on posIndex, whose live buffer
// sizes the row instead.
func (b *LayoutBuilder) rowHeight(posIndex, fallback int) int {
	if b.editHeight == 0 || b.editPos != posIndex {
		return fallback
	}
	return b.editHeight
}

func (b *LayoutBuilder) Build(project domain.Project, positions []selection.Position) Layout {
	var items []LayoutItem
	totalHeight := 0
	posIndex := 0

	projectHeight := b.rowHeight(posIndex, 1)
	items = append(items, LayoutItem{
		Kind:          LayoutProject,
		Height:        projectHeight,
		PositionIndex: posIndex,
		CategoryIndex: -1,
		TaskIndex:     -1,
	})
	totalHeight += projectHeight
	posIndex++

	// Spacing after project
	if b.config.BlankAfterProject > 0 {
		items = append(items, LayoutItem{
			Kind:          LayoutSpacing,
			Height:        b.config.BlankAfterProject,
			PositionIndex: -1,
			CategoryIndex: -1,
			TaskIndex:     -1,
		})
		totalHeight += b.config.BlankAfterProject
	}

	for catIdx, category := range project.Categories {
		// Spacing between categories (not before first)
		if catIdx > 0 && b.config.BlankBetweenCats > 0 {
			items = append(items, LayoutItem{
				Kind:          LayoutSpacing,
				Height:        b.config.BlankBetweenCats,
				PositionIndex: -1,
				CategoryIndex: -1,
				TaskIndex:     -1,
			})
			totalHeight += b.config.BlankBetweenCats
		}

		catHeight := b.rowHeight(posIndex, b.countCategoryLines(category))
		items = append(items, LayoutItem{
			Kind:          LayoutCategory,
			Height:        catHeight,
			PositionIndex: posIndex,
			CategoryIndex: catIdx,
			TaskIndex:     -1,
		})
		totalHeight += catHeight
		posIndex++

		isFolded := b.fold != nil && b.fold.IsFolded(category.ID)
		if isFolded {
			items = append(items, LayoutItem{
				Kind:          LayoutFolded,
				Height:        1,
				PositionIndex: -1,
				CategoryIndex: catIdx,
				TaskIndex:     -1,
			})
			totalHeight++
			continue
		}

		visibleTaskCount := 0
		for _, task := range category.Tasks {
			if taskRowVisible(b.filter, task, category.ID) {
				visibleTaskCount++
			}
		}

		if visibleTaskCount == 0 {
			// "(no tasks)" placeholder - not selectable
			items = append(items, LayoutItem{
				Kind:          LayoutEmptyCategory,
				Height:        1,
				PositionIndex: -1,
				CategoryIndex: catIdx,
				TaskIndex:     -1,
			})
			totalHeight++
			continue
		}

		// Spacing after category header (before tasks)
		if b.config.BlankAfterCatHead > 0 {
			items = append(items, LayoutItem{
				Kind:          LayoutSpacing,
				Height:        b.config.BlankAfterCatHead,
				PositionIndex: -1,
				CategoryIndex: -1,
				TaskIndex:     -1,
			})
			totalHeight += b.config.BlankAfterCatHead
		}

		// Tasks (consecutive tasks have no blank lines between them)
		for taskIdx, task := range category.Tasks {
			if !taskRowVisible(b.filter, task, category.ID) {
				continue
			}
			if task.IsSeparator() {
				sepHeight := b.rowHeight(posIndex, 1)
				items = append(items, LayoutItem{
					Kind:          LayoutSeparator,
					Height:        sepHeight,
					PositionIndex: posIndex,
					CategoryIndex: catIdx,
					TaskIndex:     taskIdx,
				})
				totalHeight += sepHeight
				posIndex++
				continue
			}
			taskHeight := b.rowHeight(posIndex, b.countTaskLines(task))
			items = append(items, LayoutItem{
				Kind:          LayoutTask,
				Height:        taskHeight,
				PositionIndex: posIndex,
				CategoryIndex: catIdx,
				TaskIndex:     taskIdx,
			})
			totalHeight += taskHeight
			posIndex++

			if (b.expandDescriptions || b.limits.expands(task.ID)) && task.Description != "" {
				descHeight := components.DescriptionHeight(task.Description, b.width, taskTitleColumn(task, b.statusDisplay), b.limits.descriptionRows(catIdx, taskIdx, task.ID))
				items = append(items, LayoutItem{
					Kind:          LayoutDescription,
					Height:        descHeight,
					PositionIndex: posIndex,
					CategoryIndex: catIdx,
					TaskIndex:     taskIdx,
				})
				totalHeight += descHeight
				posIndex++
			}
		}
	}

	return Layout{
		Items:       items,
		TotalHeight: totalHeight,
	}
}

func (b *LayoutBuilder) countTaskLines(task domain.Task) int {
	return components.NewTaskLineRenderer(b.width, b.statusDisplay, "", false).
		WithDescriptionExpanded(b.expandDescriptions).
		WithCut(b.taskCut != nil && b.taskCut(task.ID)).
		WithRowLimits(b.limits.titleRows(task.ID), components.DescriptionPreviewLines).
		TitleRows(task)
}

func (b *LayoutBuilder) countCategoryLines(category domain.Category) int {
	if b.width <= 0 {
		return 1
	}
	cut := b.categoryCut != nil && b.categoryCut(category.ID)
	available := categoryNameWidth(b.width, category.EstimateMinutes, category.AggregateStatus(), cut)
	return ui.CountClamped(category.Name, available, ui.MaxLineRows)
}

func (m *model) buildLayout() *Layout {
	w := m.ui.Screen.Width
	limits := m.rowLimits()
	editPos, editHeight := m.editOverlay()
	cut := m.cutKey()
	c := m.ui.layout
	if c.layout != nil && c.width == w && c.limits == limits &&
		c.editPos == editPos && c.editHeight == editHeight && c.cut == cut {
		return c.layout
	}
	builder := NewLayoutBuilder(m.layoutConfig(), w, m.deps.CfgManager.Get().StatusDisplay, &m.ui.Filter, &m.ui.Fold).
		WithExpandedDescriptions(m.ui.Screen.ExpandDescriptions).
		WithRowLimits(limits).
		WithEditOverlay(editPos, editHeight).
		WithCutMarks(m.isTaskCut, m.isCategoryCut)
	layout := builder.Build(m.project, m.positions())
	m.ui.layout = layoutCache{width: w, limits: limits, editPos: editPos, editHeight: editHeight, cut: cut, layout: &layout}
	return &layout
}

// cutKey exists because the clipboard changes a row's ✂ badge without touching the project.
func (m *model) cutKey() string {
	c := m.ui.Clipboard
	if !c.IsCut {
		return ""
	}
	return c.SourceID + "|" + strings.Join(c.TaskIDs, ",") + "|" + strings.Join(c.CategoryIDs, ",")
}

// editOverlay is the row an editor is open on and the rows its live buffer
// needs, or -1, 0 when no editor is open.
func (m *model) editOverlay() (int, int) {
	if m.ui.Edit.isAdding && m.ui.Edit.input.Value() == "" {
		return -1, 0 // the placeholder is one row, whatever the field
	}
	el, ok := m.openEditRows()
	if !ok {
		return -1, 0
	}
	return m.selected(), len(el.rows)
}

// editOverhead is the width left of the edited text for the field at pos.
func (m *model) editOverhead(pos selection.Position) int {
	switch pos.Kind {
	case selection.FocusTask:
		task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
		return taskTitleColumn(task, m.deps.CfgManager.Get().StatusDisplay)
	case selection.FocusProject:
		return projectPrefixWidth
	case selection.FocusSeparator:
		return separatorPrefixWidth
	}
	return prefixWidth
}
