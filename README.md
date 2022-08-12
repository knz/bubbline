# Bubbline

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/bubbline)
[![Go ReportCard](https://goreportcard.com/badge/knz/bubbline)](https://goreportcard.com/report/knz/bubbline)

An input line editor for line-oriented terminal applications.

Based off the [bubbletea](https://github.com/charmbracelet/bubbletea) library.

## Example use

```go
package main

import (
    "errors"
    "fmt"
    "io"
    "log"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/knz/bubbline/editline"
)

func main() {
    // Instantiate the widget.
    m := editline.New()
    for {
        // Prepare for a new input.
        m.Reset()

        // Run the widget.
        p := tea.NewProgram(m)
        if err := p.Start(); err != nil {
            log.Fatal(err)
        }
        // Handle the end of input.
        if m.Err != nil {
            if m.Err == io.EOF {
                break
            }
            if errors.Is(m.Err, editline.ErrInterrupted) {
                fmt.Println("^C")
            } else {
                fmt.Println("error: %v", m.Err)
            }
            continue
        }

        // Handle regular input.
        val := m.Value()
        fmt.Printf("\nYou have entered: %q\n", val)
        m.AddHistoryEntry(val)
    }
}
```
