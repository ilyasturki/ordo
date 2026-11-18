package app

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
	return m.choiceView(listChoice("Time Estimate", estimatePresetLabels, m.ui.EstimatePicker.Selected))
}
