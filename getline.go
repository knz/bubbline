package bubbline

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/knz/bubbline/editline"
	"github.com/knz/bubbline/history"
)

// Editor represents an input line editor.
type Editor struct {
	*editline.Model

	autoSaveHistory bool
	histFile        string
}

// New instantiates an editor.
func New() *Editor {
	return &Editor{
		Model: editline.New(0, 0),
	}
}

var _ tea.Model = (*Editor)(nil)

// Update is part of the tea.Model interface.
func (m *Editor) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := imsg.(type) {
	case tea.WindowSizeMsg:
		m.Model.SetSize(msg.Width, msg.Height)
	}
	_, next := m.Model.Update(imsg)
	return m, next
}

// Close should be called when the editor is not used any more.
func (m *Editor) Close() {}

// ErrInterrupted is returned when the input was interrupted with
// e.g. Ctrl+C.
var ErrInterrupted = editline.ErrInterrupted

// Getline runs the editor and returns the line that was read.
func (m *Editor) GetLine() (string, error) {
	p := tea.NewProgram(m)
	m.Reset()
	if err := p.Start(); err != nil {
		return "", err
	}
	return m.Value(), m.Err
}

// AddHistory adds a history entry and optionally saves
// the history to file.
func (m *Editor) AddHistory(line string) error {
	m.AddHistoryEntry(line)
	if m.autoSaveHistory && m.histFile != "" {
		return m.SaveHistory()
	}
	return nil
}

// LoadHistory loads the entry history from file.
func (m *Editor) LoadHistory(file string) error {
	h, err := history.LoadHistory(file)
	if err != nil {
		return err
	}
	m.SetHistory(h)
	return nil
}

// SaveHistory saves the current history to the file
// previously configured with SetAutoSaveHistory.
func (m *Editor) SaveHistory() error {
	if m.histFile == "" {
		return errors.New("no savefile configured")
	}
	h := m.GetHistory()
	if h == nil {
		return errors.New("history not configured")
	}
	return history.SaveHistory(h, m.histFile)
}

// SetAutoSaveHistory enables/disables auto-saving of entered lines
// to the history.
func (m *Editor) SetAutoSaveHistory(file string, autoSave bool) {
	m.autoSaveHistory = autoSave
	m.histFile = file
}
