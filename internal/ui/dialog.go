package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Style.Width and Style.Height are the *total* block size, so a caller that
// wants an exact content area asks for its content size plus these.
var (
	DialogChromeWidth  = HelpDialogStyle.GetHorizontalPadding() + HelpDialogStyle.GetHorizontalBorderSize()
	DialogChromeHeight = HelpDialogStyle.GetVerticalPadding() + HelpDialogStyle.GetVerticalBorderSize()

	// Both gutters, though only the left is painted: a row reaching the last
	// column phantom-wraps in some terminals.
	FullPanelChromeWidth  = 2 * FullPanelStyle.GetPaddingLeft()
	FullPanelChromeHeight = FullPanelStyle.GetVerticalPadding()
)

const (
	// One shared width, so every dialog occupies the same rectangle and — the
	// modal centering on the rendered box — never shifts as its content changes.
	dialogContentWidthMax = 70
	// Floor on a narrow terminal: overflow beats collapsing to nothing.
	dialogContentWidthMin = 10
	// Columns of the underlying view left visible on each side.
	dialogSideMargin = 2
)

func DialogContentWidth(screenWidth int) int {
	if screenWidth <= 0 {
		return dialogContentWidthMax
	}
	avail := screenWidth - DialogChromeWidth - dialogSideMargin
	return min(max(avail, dialogContentWidthMin), dialogContentWidthMax)
}

// DialogBodyHeight is the rows a dialog body gets once the frame and the
// dialog's own fixed rows are taken, floored so a short terminal overflows
// rather than collapsing to nothing.
func DialogBodyHeight(screenHeight, ownRows, minBody int) int {
	return max(screenHeight-DialogChromeHeight-ownRows, minBody)
}

// PadTo right-pads line to width cells, measuring the rendered width so styling
// sequences don't count toward it.
func PadTo(line string, width int) string {
	gap := width - ansi.StringWidth(line)
	if gap <= 0 {
		return line
	}
	return line + strings.Repeat(" ", gap)
}

func PadToStyle(line string, width int, style lipgloss.Style) string {
	gap := width - ansi.StringWidth(line)
	if gap <= 0 {
		return line
	}
	return line + style.Render(strings.Repeat(" ", gap))
}

// SplitRow lays left and right out on one row width cells wide, right flush to
// the far edge. right is dropped when it will not fit: left is the content,
// right a status decoration.
func SplitRow(left, right string, width int) string {
	leftW := ansi.StringWidth(left)
	if width > 0 && leftW+2+ansi.StringWidth(right) > width {
		right = ""
	}
	switch {
	case right == "":
		if width > 0 && leftW > width {
			return ansi.Truncate(left, width, "")
		}
		return left
	case width <= 0:
		return left + "  " + right
	}
	return left + strings.Repeat(" ", width-leftW-ansi.StringWidth(right)) + right
}
