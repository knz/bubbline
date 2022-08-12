package textarea

// EmptyValue returns true iff the value is empty.
func (m *Model) EmptyValue() bool {
	return len(m.value) == 0 || (len(m.value) == 1 && len(m.value[0]) == 0)
}

// NumLinesInValue returns the number of logical lines in the value.
func (m *Model) NumLinesInValue() int {
	return len(m.value)
}

// CursorPos retrieves the position of the cursor inside the input.
func (m *Model) CursorPos() int {
	return m.col
}

// AtBeginningOfLine returns true if the cursor is at the beginning of
// a line.
func (m *Model) AtBeginningOfLine() bool {
	return m.col == 0
}

// AtFirstLineOfInputAndView returns true if the cursor is on the first line
// of the input and viewport.
func (m *Model) AtFirstLineOfInputAndView() bool {
	li := m.LineInfo()
	return m.row == 0 && li.RowOffset == 0
}

// AtEndOfInput returns true if the cursor is on the last line of the input and viewport.
func (m *Model) AtLastLineOfInputAndView() bool {
	li := m.LineInfo()
	return m.row >= len(m.value)-1 && li.RowOffset == li.Height-1
}

// ResetViewCursorDown scrolls the viewport so that the cursor
// is position on the bottom line.
func (m Model) ResetViewCursorDown() {
	row := m.cursorLineNumber()
	m.viewport.SetYOffset(row - m.viewport.Height)
}

// LogicalHeight returns the number of lines needed in a viewport to
// show the entire value.
func (m Model) LogicalHeight() int {
	logicalHeight := 0
	nl := m.LineCount()
	for row := 0; row < nl; row++ {
		li := m.LineInfoAt(row, 0)
		logicalHeight += li.Height
	}
	return logicalHeight
}
