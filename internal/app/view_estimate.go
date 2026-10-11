package app

import (
	"github.com/ilyasturki/phasionary/internal/app/components"
	"github.com/ilyasturki/phasionary/internal/ui"
)

var estimatePresetLabels = []string{
	"None",
	"15 minutes",
	"30 minutes",
	"1 hour",
	"2 hours",
	"4 hours",
	"1 day",
	"2 days",
	"3 days",
	"5 days",
}

func (m model) estimatePickerView() string {
	return components.RenderListPicker(
		"Time Estimate",
		estimatePresetLabels,
		m.ui.EstimatePicker.Selected,
		[]ui.Hint{{Key: "enter", Label: "select"}, {Key: "esc", Label: "cancel"}},
	)
}
