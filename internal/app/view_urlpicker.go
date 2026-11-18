package app

func (m model) urlPickerView() string {
	return m.choiceView(listChoice("Open", m.ui.URLPicker.Labels(), m.ui.URLPicker.Selected))
}
