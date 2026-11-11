package app

import (
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type lineField struct {
	textinput.Model
	edits fieldEdits
}

type areaField struct {
	textarea.Model
	edits fieldEdits
}

// killRing is shared by every field, as readline's is by every line.
var killRing string

const fieldUndoDepth = 100

type fieldSnap struct {
	value     string
	line, col int
}

type fieldRun int

const (
	runNone fieldRun = iota
	runTyping
	runRubout
)

type fieldEdits struct {
	undo []fieldSnap
	run  fieldRun
	// runEnd is where the cursor stood after the run's last key; a cursor
	// moved since, by any means, starts a new run.
	runEnd     fieldSnap
	runOnSpace bool
	killing    bool
}

type fieldBuffer interface {
	snap() fieldSnap
	restore(fieldSnap)
	currentLine() ([]rune, int)
	setCurrentLine([]rune, int)
	insert(string)
}

type lineOp func(r []rune, pos int) ([]rune, int)

var fieldLineOps = map[string]lineOp{
	"ctrl+t": transposeChars,
	"alt+t":  transposeWords,
	"alt+u":  func(r []rune, pos int) ([]rune, int) { return caseWord(r, pos, caseUpper) },
	"alt+l":  func(r []rune, pos int) ([]rune, int) { return caseWord(r, pos, caseLower) },
	"alt+c":  func(r []rune, pos int) ([]rune, int) { return caseWord(r, pos, caseCapital) },
	// bubbles' textinput takes the next word too when the cursor is on a word's last letter.
	"alt+d":       killWordForward,
	"alt+delete":  killWordForward,
	"ctrl+delete": killWordForward,
}

func isUndoKey(key string) bool {
	switch key {
	case "ctrl+_", "ctrl+/", "ctrl+shift+-":
		return true
	}
	return false
}

// killKeys maps each kill to whether it kills backward, so a run of kills
// gathers into one yank in reading order.
var killKeys = map[string]bool{
	"ctrl+w": true, "alt+backspace": true, "ctrl+backspace": true, "ctrl+u": true,
	"ctrl+k": false, "alt+d": false, "alt+delete": false, "ctrl+delete": false,
}

func newTextInput() lineField {
	ti := textinput.New()
	// bubbles' own ctrl+v reads through atotto, whose wl-paste can't be killed; handleKeyMsg reads it through internal/clipboard.
	ti.KeyMap.Paste.SetEnabled(false)
	return lineField{Model: ti}
}

func (f lineField) Update(msg tea.Msg) (lineField, tea.Cmd) {
	if !f.Focused() {
		return f, nil
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "alt+<":
			f.CursorStart()
			f.edits.moved()
			return f, nil
		case "alt+>":
			f.CursorEnd()
			f.edits.moved()
			return f, nil
		}
	}
	if f.edits.apply(&f, msg) {
		return f, nil
	}
	before := f.snap()
	var cmd tea.Cmd
	f.Model, cmd = f.Model.Update(msg)
	f.edits.record(msg, before, f.snap())
	return f, cmd
}

func (f *lineField) snap() fieldSnap {
	return fieldSnap{value: f.Value(), col: f.Position()}
}

func (f *lineField) restore(s fieldSnap) {
	f.SetValue(s.value)
	f.SetCursor(s.col)
}

func (f *lineField) currentLine() ([]rune, int) {
	return []rune(f.Value()), f.Position()
}

func (f *lineField) setCurrentLine(r []rune, pos int) {
	f.restore(fieldSnap{value: string(r), col: pos})
}

func (f *lineField) insert(s string) {
	r, pos := f.currentLine()
	ins := []rune(s)
	f.setCurrentLine(append(append(append([]rune{}, r[:pos]...), ins...), r[pos:]...), pos+len(ins))
}

func (f *lineField) replace(s string) {
	if s != f.Value() {
		f.edits.push(f.snap())
	}
	f.edits.moved()
	f.SetValue(s)
	f.CursorEnd()
}

func (f areaField) Update(msg tea.Msg) (areaField, tea.Cmd) {
	if !f.Focused() {
		return f, nil
	}
	if f.edits.apply(&f, msg) {
		return f, nil
	}
	before := f.snap()
	var cmd tea.Cmd
	f.Model, cmd = f.Model.Update(msg)
	f.edits.record(msg, before, f.snap())
	return f, cmd
}

func (f *areaField) snap() fieldSnap {
	return fieldSnap{value: f.Value(), line: f.Line(), col: f.Column()}
}

func (f *areaField) restore(s fieldSnap) {
	f.SetValue(s.value)
	f.MoveToBegin()
	for steps := f.Length() + f.LineCount(); f.Line() < s.line && f.Line() < f.LineCount()-1 && steps > 0; steps-- {
		f.CursorDown()
	}
	f.SetCursorColumn(s.col)
}

func (f *areaField) currentLine() ([]rune, int) {
	lines := strings.Split(f.Value(), "\n")
	return []rune(lines[f.Line()]), f.Column()
}

func (f *areaField) setCurrentLine(r []rune, pos int) {
	lines := strings.Split(f.Value(), "\n")
	line := f.Line()
	lines[line] = string(r)
	f.restore(fieldSnap{value: strings.Join(lines, "\n"), line: line, col: pos})
}

func (f *areaField) insert(s string) {
	f.InsertString(s)
}

func (f *areaField) replace(s string) {
	if s != f.Value() {
		f.edits.push(f.snap())
	}
	f.edits.moved()
	f.SetValue(s)
	f.MoveToEnd()
	// The textarea fills its viewport only in Update, so a scroll outside one lands on stale content.
	f.Model, _ = f.Model.Update(nil)
}

func (e *fieldEdits) apply(buf fieldBuffer, msg tea.Msg) bool {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return false
	}
	key := k.String()
	switch {
	case isUndoKey(key):
		e.undoLast(buf)
		return true
	case key == "ctrl+y":
		if killRing != "" {
			e.push(buf.snap())
			buf.insert(killRing)
		}
		e.run, e.killing = runNone, false
		return true
	}
	op, ok := fieldLineOps[key]
	if !ok {
		return false
	}
	before := buf.snap()
	r, pos := buf.currentLine()
	next, nextPos := op(append([]rune{}, r...), pos)
	if string(next) == string(r) {
		if nextPos != pos {
			buf.setCurrentLine(next, nextPos)
		}
		e.run, e.killing = runNone, false
		return true
	}
	e.push(before)
	buf.setCurrentLine(next, nextPos)
	e.run = runNone
	if backward, isKill := killKeys[key]; isKill {
		e.kill(killed(before, buf.snap()), backward)
	} else {
		e.killing = false
	}
	return true
}

func (e *fieldEdits) record(msg tea.Msg, before, after fieldSnap) {
	k, isKey := msg.(tea.KeyPressMsg)
	if before.value == after.value {
		if isKey {
			e.killing = false
			if before.line != after.line || before.col != after.col {
				e.run = runNone
			}
		}
		return
	}
	key := ""
	if isKey {
		key = k.String()
	}
	run := runNone
	switch {
	case isKey && k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0:
		run = runTyping
	case key == "backspace" || key == "ctrl+h" || key == "delete" || key == "ctrl+d":
		run = runRubout
	}
	onSpace := run == runTyping && strings.TrimSpace(k.Text) == ""
	joins := run != runNone && e.run == run && e.runEnd == before &&
		!(run == runTyping && e.runOnSpace && !onSpace)
	if !joins {
		e.push(before)
	}
	e.run, e.runEnd, e.runOnSpace = run, after, onSpace
	if backward, isKill := killKeys[key]; isKill {
		e.kill(killed(before, after), backward)
	} else {
		e.killing = false
	}
}

func (e *fieldEdits) moved() {
	e.run, e.killing = runNone, false
}

func (e *fieldEdits) push(s fieldSnap) {
	if len(e.undo) == fieldUndoDepth {
		e.undo = append(e.undo[:0:0], e.undo[1:]...)
	}
	e.undo = append(e.undo, s)
}

func (e *fieldEdits) undoLast(buf fieldBuffer) {
	e.run, e.killing = runNone, false
	if len(e.undo) == 0 {
		return
	}
	last := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	buf.restore(last)
}

func (e *fieldEdits) kill(text string, backward bool) {
	switch {
	case !e.killing:
		killRing = text
	case backward:
		killRing = text + killRing
	default:
		killRing += text
	}
	e.killing = true
}

// killed is what a kill took out of before: the span at the cursor it left.
func killed(before, after fieldSnap) string {
	b := []rune(before.value)
	start := after.col
	for _, l := range strings.Split(after.value, "\n")[:after.line] {
		start += len([]rune(l)) + 1
	}
	end := start + len(b) - len([]rune(after.value))
	if start < 0 || end > len(b) || start > end {
		return ""
	}
	return string(b[start:end])
}

func wordEnd(r []rune, i int) int {
	for i < len(r) && unicode.IsSpace(r[i]) {
		i++
	}
	for i < len(r) && !unicode.IsSpace(r[i]) {
		i++
	}
	return i
}

func wordStart(r []rune, i int) int {
	for i > 0 && unicode.IsSpace(r[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(r[i-1]) {
		i--
	}
	return i
}

// transposeChars swaps the characters around the cursor and steps past them;
// at the end of the line it swaps the last two.
func transposeChars(r []rune, pos int) ([]rune, int) {
	if pos == 0 || len(r) < 2 {
		return r, pos
	}
	if pos == len(r) {
		pos--
	}
	r[pos-1], r[pos] = r[pos], r[pos-1]
	return r, pos + 1
}

// transposeWords drags the word before the cursor past the one after it;
// at the end of the line it swaps the last two words.
func transposeWords(r []rune, pos int) ([]rune, int) {
	end2 := wordEnd(r, pos)
	start2 := wordStart(r, end2)
	start1 := wordStart(r, start2)
	end1 := wordEnd(r, start1)
	if start1 == start2 || end1 > start2 {
		return r, pos
	}
	out := make([]rune, 0, len(r))
	out = append(out, r[:start1]...)
	out = append(out, r[start2:end2]...)
	out = append(out, r[end1:start2]...)
	out = append(out, r[start1:end1]...)
	out = append(out, r[end2:]...)
	return out, end2
}

type wordCase int

const (
	caseUpper wordCase = iota
	caseLower
	caseCapital
)

func caseWord(r []rune, pos int, c wordCase) ([]rune, int) {
	end := wordEnd(r, pos)
	first := true
	for i := pos; i < end; i++ {
		if unicode.IsSpace(r[i]) {
			continue
		}
		switch {
		case c == caseUpper || (c == caseCapital && first):
			r[i] = unicode.ToUpper(r[i])
		default:
			r[i] = unicode.ToLower(r[i])
		}
		first = false
	}
	return r, end
}

func killWordForward(r []rune, pos int) ([]rune, int) {
	end := wordEnd(r, pos)
	return append(r[:pos:pos], r[end:]...), pos
}
