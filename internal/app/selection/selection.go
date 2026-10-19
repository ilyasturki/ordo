package selection

type FocusKind int

const (
	FocusProject FocusKind = iota
	FocusCategory
	FocusTask
	FocusDescription
	// FocusSeparator is a separator row living in a category's Tasks slice
	// (domain.KindSeparator). It shares the (CategoryIndex, TaskIndex) addressing
	// of a task but gates out task-only actions (status, priority, estimate…).
	FocusSeparator
	// FocusHidden is a category's "N hidden by the filter" row: CategoryIndex and ID are the category's, TaskIndex is -1.
	FocusHidden
)

type Position struct {
	Kind          FocusKind
	CategoryIndex int
	TaskIndex     int
	// ID is the row's category or task ID, stable across rebuilds; "" on the project row.
	ID string
}

type Manager struct {
	positions []Position
	selected  int
}

func NewManager(positions []Position, initialSelection int) *Manager {
	m := &Manager{
		positions: positions,
		selected:  initialSelection,
	}
	m.clamp()
	return m
}

func (m *Manager) Positions() []Position {
	return m.positions
}

func (m *Manager) Selected() int {
	return m.selected
}

func (m *Manager) SetSelected(index int) {
	m.selected = index
	m.clamp()
}

func (m *Manager) SetPositions(positions []Position) {
	m.positions = positions
	m.clamp()
}

func (m *Manager) Count() int {
	return len(m.positions)
}

func (m *Manager) IsEmpty() bool {
	return len(m.positions) == 0
}

func (m *Manager) SelectedPosition() (Position, bool) {
	if m.selected < 0 || m.selected >= len(m.positions) {
		return Position{}, false
	}
	return m.positions[m.selected], true
}

func (m *Manager) MoveBy(delta int) bool {
	if len(m.positions) == 0 {
		return false
	}
	prev := m.selected
	m.selected += delta
	m.clamp()
	return m.selected != prev
}

func (m *Manager) MoveTo(index int) bool {
	if len(m.positions) == 0 {
		return false
	}
	prev := m.selected
	m.selected = index
	m.clamp()
	return m.selected != prev
}

func (m *Manager) JumpToFirst() bool {
	if len(m.positions) == 0 {
		return false
	}
	prev := m.selected
	m.selected = 0
	return m.selected != prev
}

func (m *Manager) JumpToLast() bool {
	if len(m.positions) == 0 {
		return false
	}
	prev := m.selected
	m.selected = len(m.positions) - 1
	return m.selected != prev
}

func (m *Manager) FindPositionIndex(predicate func(Position) bool) int {
	for i, pos := range m.positions {
		if predicate(pos) {
			return i
		}
	}
	return -1
}

func (m *Manager) SelectByPredicate(predicate func(Position) bool) bool {
	idx := m.FindPositionIndex(predicate)
	if idx >= 0 {
		m.selected = idx
		return true
	}
	return false
}

func (m *Manager) clamp() {
	if len(m.positions) == 0 {
		m.selected = -1
		return
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(m.positions) {
		m.selected = len(m.positions) - 1
	}
}
