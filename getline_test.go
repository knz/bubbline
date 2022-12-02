package bubbline

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
)

func TestTeaInputPreservedAcrossRestarts1(t *testing.T) {
	const expectedCount = 100

	// Prepare 100 inputs in a single continuous buffer.
	input := strings.Repeat("hello\r", expectedCount)
	inputBuf := bytes.NewBuffer([]byte(input))
	var outBuf bytes.Buffer

	// ed will be our editor.
	ed := &testModel1{}

	var count int
	for count = 0; count < 100; count++ {
		// Reset the editor.
		ed.msg = ""
		// Run the bubbletea interaction until it stops.
		p := tea.NewProgram(ed, tea.WithInput(inputBuf), tea.WithOutput(&outBuf))
		if err := p.Start(); err != nil {
			t.Fatal(err)
		}
		// At this point bubbletea has shut down; get the remaining
		// payload.
		msg := ed.msg

		// Is this what we expect?
		t.Logf("msg: %q", msg)
		if msg != "hello" {
			// No: stop.
			t.Errorf("corrupted input: %q", msg)
			break
		}
	}
	// Did we consume all the input?
	if count != expectedCount {
		t.Errorf("expected %d inputs, got %d", expectedCount, count)
	}
}

type testModel1 struct {
	msg string
}

func (m *testModel1) Init() tea.Cmd { return nil }
func (m *testModel1) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := imsg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			return m, tea.Quit
		default:
			m.msg += string(msg.Runes)
		}
	}
	return m, nil
}
func (m *testModel1) View() string { return "" }

func TestTeaInputPreservedAcrossRestarts2(t *testing.T) {
	const expectedCount = 100

	// Prepare 100 inputs in a single continuous buffer.
	input := strings.Repeat("hello\r", expectedCount)
	inputBuf := bytes.NewBuffer([]byte(input))
	var outBuf bytes.Buffer

	// ed will be our editor.
	ed := &testModel2{
		ready: make(chan string),
		cont:  make(chan tea.Cmd),
	}

	// In this test, we start bubbletea just once.
	p := tea.NewProgram(ed, tea.WithInput(inputBuf), tea.WithOutput(&outBuf))
	go func() {
		// The event loop runs asynchronously, in a separate goroutine.
		if err := p.Start(); err != nil {
			os.Exit(1)
		}
	}()

	// Wait for the async goroutine to call our Init() method.
	<-ed.ready

	var count int
	for count = 0; count < 100; count++ {
		// Reset the editor.
		ed.msg = ""
		// Tell our model to resume execution - at this point it is either
		// blocked in Init() or in Update(). This will resume execution
		// in the bubbletea event loop.
		ed.cont <- nil

		// Wait for the next input to be recognized.
		msg := <-ed.ready

		// Is this what we expect?
		t.Logf("msg: %q", msg)
		if msg != "hello" {
			// No: stop.
			// Don't forget to tell our async goroutine to terminate.
			ed.cont <- tea.Quit
			t.Errorf("corrupted input: %q", msg)
			break
		}
	}
	if count != expectedCount {
		t.Errorf("expected %d inputs, got %d", expectedCount, count)
	}
}

type testModel2 struct {
	// msg is the last recognized input.
	msg string

	// cont will deliver instructions from the Test loop
	// above to continue the bubbletea event loop.
	cont chan tea.Cmd
	// ready is written by the Update method below to
	// tell the Test loop that an input is ready.
	ready chan string
}

func (m *testModel2) Init() tea.Cmd {
	m.ready <- ""
	return <-m.cont
}
func (m *testModel2) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := imsg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Tell the Test loop we have some input ready.
			m.ready <- m.msg
			// Wait for the Test loop to look at the input
			// and tell us what to do next.
			return m, <-m.cont
		default:
			m.msg += string(msg.Runes)
		}
	}
	return m, nil
}
func (m *testModel2) View() string { return "" }

func TestTeaInputPreservedAcrossRestarts3(t *testing.T) {
	const expectedCount = 100

	// Prepare 100 inputs in a single continuous buffer.
	input := strings.Repeat("hello\r", expectedCount)
	inputBuf := bytes.NewBuffer([]byte(input))
	var outBuf bytes.Buffer

	// ed will be our editor.
	ed := &testModel3{
		cont:  make(chan tea.Cmd),
		ready: make(chan string),
	}

	// In this test, we start bubbletea just once.
	p := tea.NewProgram(ed, tea.WithInput(inputBuf), tea.WithOutput(&outBuf))

	ed.completeFn = func(msg string) tea.Cmd {
		// Release the terminal, so the Test loop below can do sane I/O.
		p.ReleaseTerminal()
		// Give some time to the input reader to finish processing.
		time.Sleep(100 * time.Millisecond)
		ed.ready <- msg
		cmd := <-ed.cont
		// Re-take the terminal in Bubbline.
		p.RestoreTerminal()
		return cmd
	}

	go func() {
		// The event loop runs asynchronously, in a separate goroutine.
		if err := p.Start(); err != nil {
			os.Exit(1)
		}
	}()

	// Wait for the async goroutine to call our Init() method.
	<-ed.ready

	var count int
	for count = 0; count < 100; count++ {
		// Reset the editor.
		ed.msg = ""
		// Tell our model to resume execution - at this point it is either
		// blocked in Init() or in Update(). This will resume execution
		// in the bubbletea event loop.
		ed.cont <- nil

		// Wait for the next input to be recognized.
		msg := <-ed.ready

		// Is this what we expect?
		t.Logf("msg: %q", msg)
		if msg != "hello" {
			// No: stop.
			// Don't forget to tell our async goroutine to terminate.
			ed.cont <- tea.Quit
			t.Errorf("corrupted input: %q", msg)
			break
		}
	}
	if count != expectedCount {
		t.Errorf("expected %d inputs, got %d", expectedCount, count)
	}
}

type testModel3 struct {
	// msg is the last recognized input.
	msg string

	// cont will deliver instructions from the Test loop
	// above to continue the bubbletea event loop.
	cont chan tea.Cmd
	// ready is written by the Update method below to
	// tell the Test loop that an input is ready.
	ready chan string

	// completeFn is called when an input has been recognized.
	completeFn func(msg string) tea.Cmd
}

func (m *testModel3) Init() tea.Cmd {
	m.ready <- ""
	return <-m.cont
}
func (m *testModel3) Update(imsg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := imsg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			cmd := m.completeFn(m.msg)
			return m, cmd
		default:
			m.msg += string(msg.Runes)
		}
	}
	return m, nil
}
func (m *testModel3) View() string { return "" }

func TestInputPreservedAcrossRestarts(t *testing.T) {
	const expectedCount = 100
	input := strings.Repeat("hello\n", expectedCount)
	inputBuf := bytes.NewBuffer([]byte(input))
	var outBuf bytes.Buffer

	ed := New()
	ed.CursorMode = cursor.CursorStatic
	ed.Start(tea.WithInput(inputBuf), tea.WithOutput(&outBuf))
	var count int
	for count = 0; count < 100; count++ {
		line, err := ed.GetLine()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("line: %q", line)
		if line != "hello" {
			t.Errorf("corrupted input: %q", line)
			break
		}
	}
	if count != expectedCount {
		t.Errorf("expected %d inputs, got %d", expectedCount, count)
	}
}
