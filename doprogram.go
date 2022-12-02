package bubbline

import "io"

func doProgram(fn func(io.Writer) error) *doProgramC {
	return &doProgramC{fn: fn}
}

type doProgramC struct {
	fn  func(io.Writer) error
	out io.Writer
}

// Run is part of the tea.ExecCommand interface.
func (d *doProgramC) Run() error {
	return d.fn(d.out)
}

// SetStdin is part of the tea.ExecCommand interface.
func (d *doProgramC) SetStdin(io.Reader) {}

// SetStdout is part of the tea.ExecCommand interface.
func (d *doProgramC) SetStdout(out io.Writer) { d.out = out }

// SetStderr is part of the tea.ExecCommand interface.
func (d *doProgramC) SetStderr(io.Writer) {}
