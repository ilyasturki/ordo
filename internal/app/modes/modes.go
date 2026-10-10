package modes

type Mode int

const (
	ModeNormal Mode = iota
	ModeEdit
	ModeHelp
	ModeConfirmDelete
	ModeOptions
	ModeProjectPicker
	ModeFilter
	ModeExternalEdit
	ModeInfo
	ModeEstimatePicker
	ModeURLPicker
	ModeVisual
	ModeDescriptionEdit
	ModeYankPicker
	ModeSearch
	ModeTagEdit
	ModeCommandPrompt
	ModePasteChooser
)

type Action int

const (
	ActionNavigate Action = iota
	ActionToggleTask
	ActionDeleteItem
	ActionEditItem
	ActionAddTask
	ActionAddCategory
	ActionChangePriority
	ActionChangeTag
	ActionChangeEstimate
	ActionMoveItem
	ActionCopy
	ActionOpenPicker
	ActionOpenOptions
	ActionOpenHelp
)

type Machine struct {
	current Mode
}

func NewMachine(initial Mode) *Machine {
	return &Machine{current: initial}
}

func (m *Machine) Current() Mode {
	return m.current
}

func (m *Machine) IsNormal() bool {
	return m.current == ModeNormal
}

func (m *Machine) IsEdit() bool {
	return m.current == ModeEdit
}

func (m *Machine) IsHelp() bool {
	return m.current == ModeHelp
}

func (m *Machine) IsConfirmDelete() bool {
	return m.current == ModeConfirmDelete
}

func (m *Machine) IsOptions() bool {
	return m.current == ModeOptions
}

func (m *Machine) IsProjectPicker() bool {
	return m.current == ModeProjectPicker
}

func (m *Machine) IsFilter() bool {
	return m.current == ModeFilter
}

func (m *Machine) IsExternalEdit() bool {
	return m.current == ModeExternalEdit
}

func (m *Machine) IsInfo() bool {
	return m.current == ModeInfo
}

func (m *Machine) IsEstimatePicker() bool {
	return m.current == ModeEstimatePicker
}

func (m *Machine) TransitionTo(mode Mode) bool {
	if !m.canTransition(mode) {
		return false
	}
	m.current = mode
	return true
}

func (m *Machine) canTransition(target Mode) bool {
	switch m.current {
	case ModeNormal:
		return true
	case ModeEdit:
		return target == ModeNormal || target == ModePasteChooser
	case ModeHelp:
		return target == ModeNormal
	case ModeConfirmDelete:
		return target == ModeNormal || target == ModeProjectPicker
	case ModeOptions:
		return target == ModeNormal
	case ModeProjectPicker:
		return target == ModeNormal || target == ModeConfirmDelete
	case ModeFilter:
		return target == ModeNormal
	case ModeExternalEdit:
		return target == ModeNormal
	case ModeInfo:
		return target == ModeNormal
	case ModeEstimatePicker:
		return target == ModeNormal
	case ModeURLPicker:
		return target == ModeNormal
	case ModeVisual:
		return target == ModeNormal
	case ModeDescriptionEdit:
		return target == ModeNormal
	case ModeYankPicker:
		return target == ModeNormal
	case ModeSearch:
		return target == ModeNormal
	case ModeTagEdit:
		return target == ModeNormal
	case ModeCommandPrompt:
		return target == ModeNormal
	case ModePasteChooser:
		return target == ModeNormal || target == ModeEdit
	}
	return false
}

func (m *Machine) CanPerformAction(action Action) bool {
	switch m.current {
	case ModeNormal:
		return true
	case ModeEdit:
		return false
	case ModeHelp:
		return action == ActionOpenHelp
	case ModeConfirmDelete:
		return false
	case ModeOptions:
		return false
	case ModeProjectPicker:
		return false
	case ModeFilter:
		return false
	case ModeEstimatePicker:
		return false
	case ModeURLPicker:
		return false
	case ModeVisual:
		return action == ActionNavigate || action == ActionCopy
	}
	return false
}

func (m *Machine) ToNormal() {
	m.current = ModeNormal
}

func (m *Machine) ToEdit() bool {
	return m.TransitionTo(ModeEdit)
}

func (m *Machine) ToHelp() bool {
	return m.TransitionTo(ModeHelp)
}

func (m *Machine) ToConfirmDelete() bool {
	return m.TransitionTo(ModeConfirmDelete)
}

func (m *Machine) ToOptions() bool {
	return m.TransitionTo(ModeOptions)
}

func (m *Machine) ToProjectPicker() bool {
	return m.TransitionTo(ModeProjectPicker)
}

func (m *Machine) ToFilter() bool {
	return m.TransitionTo(ModeFilter)
}

func (m *Machine) ToExternalEdit() bool {
	return m.TransitionTo(ModeExternalEdit)
}

func (m *Machine) ToggleHelp() {
	if m.current == ModeHelp {
		m.current = ModeNormal
	} else if m.current == ModeNormal {
		m.current = ModeHelp
	}
}

func (m *Machine) ToInfo() bool {
	return m.TransitionTo(ModeInfo)
}

func (m *Machine) ToggleInfo() {
	if m.current == ModeInfo {
		m.current = ModeNormal
	} else if m.current == ModeNormal {
		m.current = ModeInfo
	}
}

func (m *Machine) ToEstimatePicker() bool {
	return m.TransitionTo(ModeEstimatePicker)
}

func (m *Machine) IsURLPicker() bool {
	return m.current == ModeURLPicker
}

func (m *Machine) ToURLPicker() bool {
	return m.TransitionTo(ModeURLPicker)
}

func (m *Machine) IsVisual() bool {
	return m.current == ModeVisual
}

func (m *Machine) ToVisual() bool {
	return m.TransitionTo(ModeVisual)
}

func (m *Machine) IsDescriptionEdit() bool {
	return m.current == ModeDescriptionEdit
}

func (m *Machine) ToDescriptionEdit() bool {
	return m.TransitionTo(ModeDescriptionEdit)
}

func (m *Machine) IsYankPicker() bool {
	return m.current == ModeYankPicker
}

func (m *Machine) ToYankPicker() bool {
	return m.TransitionTo(ModeYankPicker)
}

func (m *Machine) IsSearch() bool {
	return m.current == ModeSearch
}

func (m *Machine) ToSearch() bool {
	return m.TransitionTo(ModeSearch)
}

func (m *Machine) IsTagEdit() bool {
	return m.current == ModeTagEdit
}

func (m *Machine) ToTagEdit() bool {
	return m.TransitionTo(ModeTagEdit)
}

func (m *Machine) IsCommandPrompt() bool {
	return m.current == ModeCommandPrompt
}

func (m *Machine) ToCommandPrompt() bool {
	return m.TransitionTo(ModeCommandPrompt)
}

func (m *Machine) IsPasteChooser() bool {
	return m.current == ModePasteChooser
}

func (m *Machine) ToPasteChooser() bool {
	return m.TransitionTo(ModePasteChooser)
}
