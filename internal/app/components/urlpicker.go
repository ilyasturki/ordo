package components

type Link struct {
	Label  string
	Target string
}

type URLPickerState struct {
	Links    []Link
	Selected int
}

func NewURLPickerState(links []Link) URLPickerState {
	return URLPickerState{Links: links, Selected: 0}
}

func (u *URLPickerState) MoveUp() {
	if u.Selected > 0 {
		u.Selected--
	}
}

func (u *URLPickerState) MoveDown() {
	if u.Selected < len(u.Links)-1 {
		u.Selected++
	}
}

func (u *URLPickerState) SelectedLink() (Link, bool) {
	if u.Selected < 0 || u.Selected >= len(u.Links) {
		return Link{}, false
	}
	return u.Links[u.Selected], true
}

func (u *URLPickerState) Labels() []string {
	labels := make([]string, len(u.Links))
	for i, l := range u.Links {
		labels[i] = l.Label
	}
	return labels
}
