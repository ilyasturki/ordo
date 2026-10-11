package app

import (
	"github.com/ilyasturki/phasionary/internal/app/components"
	"github.com/ilyasturki/phasionary/internal/ui"
)

func (m model) yankPickerView() string {
	return components.RenderListPicker(
		"Yank",
		m.ui.YankPicker.Labels(),
		m.ui.YankPicker.Selected,
		[]ui.Hint{{Key: "enter", Label: "copy"}, {Key: "esc", Label: "cancel"}},
	)
}
