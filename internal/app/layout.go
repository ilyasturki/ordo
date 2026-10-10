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

func NewLayoutBuilder(config LayoutConfig, width int, statusDisplay string, fold *FoldState) *LayoutBuilder {
	return &LayoutBuilder{
		config:        config,
		width:         width,
		statusDisplay: statusDisplay,
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

// Build lays out the rows positions lists, adding the blank lines and the
// placeholders between them.
func (b *LayoutBuilder) Build(project domain.Project, positions []selection.Position) Layout {
	var layout Layout
	add := func(item LayoutItem) {
		layout.Items = append(layout.Items, item)
		layout.TotalHeight += item.Height
	}
	spacing := func(height int) {
		if height > 0 {
			add(LayoutItem{Kind: LayoutSpacing, Height: height, PositionIndex: -1, CategoryIndex: -1, TaskIndex: -1})
		}
	}
	placeholder := func(kind LayoutItemKind, catIdx int) {
		add(LayoutItem{Kind: kind, Height: 1, PositionIndex: -1, CategoryIndex: catIdx, TaskIndex: -1})
	}

	seenCategory := false
	for posIndex, pos := range positions {
		item := LayoutItem{PositionIndex: posIndex, CategoryIndex: pos.CategoryIndex, TaskIndex: pos.TaskIndex}
		switch pos.Kind {
		case selection.FocusProject:
			item.Kind, item.Height = LayoutProject, b.rowHeight(posIndex, 1)
			add(item)
			spacing(b.config.BlankAfterProject)

		case selection.FocusCategory:
			if seenCategory {
				spacing(b.config.BlankBetweenCats)
			}
			seenCategory = true
			category := project.Categories[pos.CategoryIndex]
			item.Kind, item.Height = LayoutCategory, b.rowHeight(posIndex, b.countCategoryLines(&project, pos.CategoryIndex))
			add(item)
			switch {
			case b.fold != nil && b.fold.IsFolded(category.ID):
				placeholder(LayoutFolded, pos.CategoryIndex)
			case rowOfCategory(positions, posIndex+1, pos.CategoryIndex):
				if category.ParentID == "" {
					spacing(b.config.BlankAfterCatHead)
				}
			case !subCategoryOf(project, positions, posIndex+1, category.ID):
				placeholder(LayoutEmptyCategory, pos.CategoryIndex)
			}

		case selection.FocusSeparator:
			item.Kind, item.Height = LayoutSeparator, b.rowHeight(posIndex, 1)
			add(item)

		case selection.FocusTask:
			item.Kind, item.Height = LayoutTask, b.rowHeight(posIndex, b.countTaskLines(project.Categories[pos.CategoryIndex], pos.TaskIndex))
			add(item)

		case selection.FocusDescription:
			cat := project.Categories[pos.CategoryIndex]
			task := cat.Tasks[pos.TaskIndex]
			item.Kind = LayoutDescription
			column := taskTitleColumn(task, b.statusDisplay, taskIndent(cat, pos.TaskIndex))
			item.Height = components.DescriptionHeight(task.Description, b.width, column, b.limits.descriptionRows(pos.CategoryIndex, pos.TaskIndex, task.ID))
			add(item)
		}
	}
	return layout
}

// subCategoryOf reports whether positions[i] is the header of a sub-category
// of the category parentID.
func subCategoryOf(project domain.Project, positions []selection.Position, i int, parentID string) bool {
	return i < len(positions) && positions[i].Kind == selection.FocusCategory &&
		project.Categories[positions[i].CategoryIndex].ParentID == parentID
}

// rowOfCategory reports whether positions[i] is a row inside the category at
// catIdx.
func rowOfCategory(positions []selection.Position, i, catIdx int) bool {
	return i < len(positions) && positions[i].Kind != selection.FocusCategory &&
		positions[i].Kind != selection.FocusProject && positions[i].CategoryIndex == catIdx
}

func (b *LayoutBuilder) countTaskLines(cat domain.Category, taskIdx int) int {
	task := cat.Tasks[taskIdx]
	return components.NewTaskLineRenderer(b.width, b.statusDisplay, "", false).
		WithNesting(taskIndent(cat, taskIdx), taskProgress(cat, taskIdx, b.fold)).
		WithDescriptionExpanded(b.expandDescriptions).
		WithCut(b.taskCut != nil && b.taskCut(task.ID)).
		WithRowLimits(b.limits.titleRows(task.ID), components.DescriptionPreviewLines).
		TitleRows(task)
}

func (b *LayoutBuilder) countCategoryLines(project *domain.Project, catIdx int) int {
	if b.width <= 0 {
		return 1
	}
	category := project.Categories[catIdx]
	cut := b.categoryCut != nil && b.categoryCut(category.ID)
	available := categoryNameWidth(b.width, headerOf(project, catIdx, b.fold, cut))
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
	builder := NewLayoutBuilder(m.layoutConfig(), w, m.deps.CfgManager.Get().StatusDisplay, &m.ui.Fold).
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
	return strings.Join(c.TaskIDs, ",") + "|" + strings.Join(c.CategoryIDs, ",")
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
	rows := len(el.rows)
	if m.ui.Edit.err != "" {
		rows++
	}
	return m.selected(), rows
}

// editOverhead is the width left of the edited text for the field at pos.
func (m *model) editOverhead(pos selection.Position) int {
	indent := m.rowIndent(pos.CategoryIndex, pos.TaskIndex)
	switch pos.Kind {
	case selection.FocusTask:
		task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
		return taskTitleColumn(task, m.deps.CfgManager.Get().StatusDisplay, indent)
	case selection.FocusProject:
		return projectPrefixWidth
	case selection.FocusSeparator:
		return separatorPrefixWidth + indent
	}
	return prefixWidth + indent
}
