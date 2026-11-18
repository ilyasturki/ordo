package components

// YankItem is one selectable entry in the yank picker: Label is shown in the
// list, Value is what gets written to the clipboard when chosen.
type YankItem struct {
	Label string
	Value string
}

type YankPickerState struct {
	Items    []YankItem
	Selected int
}

func NewYankPickerState(items []YankItem) YankPickerState {
	return YankPickerState{Items: items, Selected: 0}
}

func (y *YankPickerState) Labels() []string {
	out := make([]string, len(y.Items))
	for i, it := range y.Items {
		out[i] = it.Label
	}
	return out
}

func (y *YankPickerState) SelectedItem() (YankItem, bool) {
	if y.Selected < 0 || y.Selected >= len(y.Items) {
		return YankItem{}, false
	}
	return y.Items[y.Selected], true
}
