package app

func (m model) yankPickerView() string {
	return m.choiceView(listChoice("Yank", m.ui.YankPicker.Labels(), m.ui.YankPicker.Selected))
}
