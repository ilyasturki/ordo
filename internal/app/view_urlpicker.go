package app

import (
	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/ui"
)

func (m model) urlPickerView() string {
	return components.RenderListPicker(
		"Open",
		m.ui.URLPicker.Labels(),
		m.ui.URLPicker.Selected,
		[]ui.Hint{{Key: "enter", Label: "open"}, {Key: "esc", Label: "cancel"}},
	)
}
