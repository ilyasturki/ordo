package components

import (
	"github.com/ilyasturki/ordo/internal/domain"
)

type EstimatePickerState struct {
	Selected int
}

func NewEstimatePickerState(currentEstimate int) EstimatePickerState {
	selected := 0
	for i, preset := range domain.EstimatePresets {
		if preset == currentEstimate {
			selected = i
			break
		}
	}
	return EstimatePickerState{Selected: selected}
}

func (e *EstimatePickerState) SelectedValue() int {
	if e.Selected >= 0 && e.Selected < len(domain.EstimatePresets) {
		return domain.EstimatePresets[e.Selected]
	}
	return 0
}
