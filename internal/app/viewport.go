package app

const (
	scrollMoreAbove = "  ↑ more above"
	scrollMoreBelow = "  ↓ more below"
)

type Viewport struct {
	Layout       *Layout
	Config       LayoutConfig
	TopRow       int
	ScreenHeight int

	VisibleStart int
	VisibleEnd   int
	RowOffset    int // rows of VisibleStart scrolled off the top
	HasMoreAbove bool
	HasMoreBelow bool

	usedHeight  int // Rows consumed by fully-visible layout items
	availHeight int // Content rows available (excludes scroll indicators + footer)
}

func NewViewport(layout *Layout, screenHeight int, config LayoutConfig) *Viewport {
	return &Viewport{
		Layout:       layout,
		Config:       config,
		ScreenHeight: screenHeight,
	}
}

// rowRange is the row span [start, end) the item at posIndex occupies.
func (l *Layout) rowRange(posIndex int) (int, int, bool) {
	row := 0
	for _, item := range l.Items {
		if item.PositionIndex == posIndex {
			return row, row + item.Height, true
		}
		row += item.Height
	}
	return 0, 0, false
}

func (v *Viewport) availableHeight() int {
	if v.ScreenHeight <= v.Config.FooterHeight {
		return 1
	}
	return v.ScreenHeight - v.Config.FooterHeight
}

func (v *Viewport) contentHeight(reserveMoreBelow bool) int {
	availHeight := v.availableHeight()
	if v.TopRow > 0 {
		availHeight-- // "more above" indicator
	}
	if reserveMoreBelow {
		availHeight-- // "more below" indicator
	}
	if availHeight < 1 {
		availHeight = 1
	}
	return availHeight
}

// windowAt is the content rows a view starting at top would draw, indicators
// included.
func (v *Viewport) windowAt(top int) int {
	height := v.availableHeight()
	if top > 0 {
		height--
	}
	if v.Layout != nil && height < v.Layout.TotalHeight-top {
		height--
	}
	return max(height, 1)
}

func (v *Viewport) ComputeVisibility(topRow int) {
	v.HasMoreBelow = false
	v.VisibleStart = -1
	v.VisibleEnd = -1
	v.RowOffset = 0

	if v.Layout == nil || len(v.Layout.Items) == 0 {
		v.TopRow = 0
		v.HasMoreAbove = false
		return
	}

	v.TopRow = min(max(topRow, 0), max(v.Layout.TotalHeight-1, 0))
	row := 0
	for i, item := range v.Layout.Items {
		if row+item.Height > v.TopRow {
			v.VisibleStart, v.RowOffset = i, v.TopRow-row
			break
		}
		row += item.Height
	}
	if v.VisibleStart < 0 {
		v.VisibleStart, v.RowOffset, v.TopRow = 0, 0, 0
	}
	v.HasMoreAbove = v.TopRow > 0

	v.computeVisibleRange(false)
	if v.HasMoreBelow {
		v.computeVisibleRange(true)
	}
}

func (v *Viewport) computeVisibleRange(reserveMoreBelow bool) {
	availHeight := v.contentHeight(reserveMoreBelow)
	usedHeight := 0

	v.HasMoreBelow = false
	v.VisibleEnd = -1
	v.availHeight = availHeight

	for i := v.VisibleStart; i < len(v.Layout.Items); i++ {
		height := v.Layout.Items[i].Height
		if i == v.VisibleStart {
			height -= v.RowOffset
		}

		if usedHeight+height > availHeight {
			v.HasMoreBelow = true
			v.VisibleEnd = i
			break
		}

		usedHeight += height
		v.VisibleEnd = i + 1
	}

	v.usedHeight = usedHeight
	if v.VisibleEnd < len(v.Layout.Items) {
		v.HasMoreBelow = true
	}
}

// RemainingContentHeight is the content rows left after the fully visible
// items, which the next item can be drawn a partial slice of.
func (v *Viewport) RemainingContentHeight() int {
	remaining := v.availHeight - v.usedHeight
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ShowRows returns the top row that brings [start, end) into view, moving the
// view as little as it can. Two passes settle the window, whose height depends
// on the offset it is being solved for through the scroll indicators.
func (v *Viewport) ShowRows(start, end int) int {
	if v.Layout == nil || len(v.Layout.Items) == 0 {
		return 0
	}
	top := v.TopRow
	for range 2 {
		// A row taller than the view keeps any top that shows only its rows.
		lo := end - v.windowAt(top)
		top = min(max(v.TopRow, min(lo, start)), max(lo, start))
	}
	return max(top, 0)
}

// EnsureVisible returns the top row showing the item at posIndex whole, or as
// much of it as fits when it is taller than the view.
func (v *Viewport) EnsureVisible(posIndex int) int {
	start, end, ok := v.Layout.rowRange(posIndex)
	if !ok {
		return v.TopRow
	}
	return v.ShowRows(start, end)
}

func (v *Viewport) BottomOnPosition(posIndex int) int {
	start, end, ok := v.Layout.rowRange(posIndex)
	if !ok {
		return v.TopRow
	}
	return max(min(end-v.windowAt(end-v.availableHeight()), start), 0)
}

func (v *Viewport) CenterOnPosition(posIndex int) int {
	start, end, ok := v.Layout.rowRange(posIndex)
	if !ok {
		return 0
	}
	top := start - max(v.windowAt(start)-(end-start), 0)/2
	return max(top, 0)
}
