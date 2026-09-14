package app

import (
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"

	"phasionary/internal/app/components"
	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/config"
	"phasionary/internal/data"
	"phasionary/internal/domain"
)

type DescriptionEditState struct {
	textarea      textarea.Model
	categoryIndex int
	taskIndex     int
	original      string
	creating      bool // true when the task had no description before this edit
	// cursorRow holds the textarea cursor's current display row, shared by
	// pointer with the prompt func so it can draw the ">" gutter on that line.
	// Refreshed each render in descriptionEditView.
	cursorRow *int
}

type ClipboardState struct {
	Task     *domain.Task
	IsCut    bool
	SourceID string

	Tasks       []domain.Task
	TaskIDs     []string
	Categories  []domain.Category
	CategoryIDs []string
}

type VisualState struct {
	Active           bool
	Kind             selection.FocusKind
	AnchorCategoryID string
	AnchorTaskID     string
	// AnchorOnDescription pins the anchor to the task's description row rather
	// than the task row itself — both carry the same task ID, so the ID pair
	// alone cannot tell them apart when re-resolving the anchor.
	AnchorOnDescription bool
}

// TagClipboard holds a copied tag for the "tag painter": gt grabs the focused
// task's tag, p (or p in visual mode) applies it elsewhere. Set distinguishes
// "nothing copied yet" from a copied empty tag (which pastes as a clear).
type TagClipboard struct {
	Set   bool
	Color string
	Label string
}

type Screen struct {
	Width  int
	Height int
	// TopRow counts the layout's rows, not its items: an edited title wraps to
	// as many rows as it takes, so an item can outgrow the screen.
	TopRow             int
	StatusMsg          string
	SyncOffline        bool
	PendingKey         rune
	WindowFocused      bool
	ExpandDescriptions bool
	// WheelAccum accumulates mouse-wheel events between scroll steps. Trackpads
	// and hi-res wheels emit several events per physical notch, so we only move
	// a line once wheelScrollDivisor events pile up (see wheelTick). Sign tracks
	// direction; a reversal resets it.
	WheelAccum int
	// PendingCenter defers centering the restored cursor until the first
	// WindowSizeMsg. Height is still 0 when Run restores it, and centering
	// against a zero height pins the row to the top — the very thing centering
	// exists to avoid. Cleared once consumed, so later resizes go back to
	// ensureVisible rather than yanking the view out from under the user.
	PendingCenter bool
}

// InfoState holds the info dialog's scroll position. The dialog shows fields in
// full that the list clamps, so its body can outgrow the screen.
type InfoState struct {
	ScrollOffset int
}

type HelpState struct {
	Focused      int
	ScrollOffset int
	// Filtering is true while the `/` incremental filter is active; Filter holds
	// its query. The filter narrows the shortcut list, matching each row against
	// both its key and its label.
	Filtering bool
	Filter    textinput.Model
}

type UIState struct {
	Selection      *selection.Manager
	Modes          *modes.Machine
	Screen         Screen
	Edit           EditState
	Picker         ProjectPickerState
	ConfirmDelete  ConfirmDeleteState
	Options        OptionsState
	Filter         FilterState
	Fold           FoldState
	ExternalEdit   ExternalEditState
	EstimatePicker components.EstimatePickerState
	URLPicker      components.URLPickerState
	YankPicker     components.YankPickerState
	Help           HelpState
	Info           InfoState
	Clipboard      ClipboardState
	TagClip        TagClipboard
	// TagCopiedLast records whether the most recent copy gesture was a tag copy
	// (gt) rather than a task/category cut or copy. It routes the normal-mode `p`:
	// true → paint the copied tag, false → paste the clipboard task/category.
	TagCopiedLast   bool
	Visual          VisualState
	DescriptionEdit DescriptionEditState
	TagEdit         TagEditState
	History         HistoryState
	Search          SearchState
	Cursor          cursorTracker
	layout          layoutCache
}

// projectSaver persists project snapshots asynchronously so mutations never
// block the event loop on fsync. Satisfied by *data.Saver; nil in tests, where
// storeTaskUpdate falls back to a synchronous save.
type projectSaver interface {
	Enqueue(domain.Project)
	Results() <-chan error
	Close()
}

// cursorSaver records the focused row off the event loop, coalescing bursts of
// movement into one write. Satisfied by *data.CursorSaver; nil in tests, where
// trackCursor becomes a no-op.
type cursorSaver interface {
	Set(projectID string, cursor data.Cursor)
	Drop(projectID string)
}

// cursorTracker remembers what was last handed to the cursor saver, so the many
// events that leave the cursor where it was cost nothing. The project ID is part
// of it: the same row means a different thing after a project switch.
type cursorTracker struct {
	projectID string
	cursor    data.Cursor
}

// layoutCache memoizes the built Layout so navigation and scroll math don't
// rewalk every task each frame (buildLayout runs on every render and on every
// ensureVisible). A nil layout means "no cached value"; invalidated on any
// content or structural change. Keyed on width plus the cursor's description
// row (cursorCat/cursorTask, -1/-1 when the cursor is elsewhere), since that
// row lays out at full height while the others are capped to their preview.
type layoutCache struct {
	width      int
	cursorCat  int
	cursorTask int
	editPos    int
	editHeight int
	layout     *Layout
}

type Dependencies struct {
	Store        data.ProjectRepository
	CfgManager   config.Reader
	StateManager data.StateRepository
	Saver        projectSaver
	CursorSaver  cursorSaver
}

func NewUIState(sel *selection.Manager, modeMachine *modes.Machine) *UIState {
	return &UIState{
		Selection: sel,
		Modes:     modeMachine,
		Screen:    Screen{WindowFocused: true},
		Filter:    NewFilterState(),
		Fold:      NewFoldState(),
		History:   NewHistoryState(),
	}
}

func NewDependencies(store data.ProjectRepository, cfgManager config.Reader, stateManager data.StateRepository) *Dependencies {
	return &Dependencies{
		Store:        store,
		CfgManager:   cfgManager,
		StateManager: stateManager,
	}
}
