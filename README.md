# Bubbline

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/bubbline)
[![Go ReportCard](https://goreportcard.com/badge/knz/bubbline)](https://goreportcard.com/report/knz/bubbline)

An input line editor for line-oriented terminal applications.

Based off the [bubbletea](https://github.com/charmbracelet/bubbletea) library.

## Features of the line editor

- Adds libedit/readline goodies to [bubbles](https://github.com/charmbracelet/bubbles)' original `textarea` widget:
  - Resizes horizontally to terminal width.
  - Resizes vertically automatically as the input grows.
  - Supports history navigation and search.
  - Uppercase/lowercase/capitalize next word, transpose characters.
  - Word navigation across input lines.
  - Enter key conditionally ends the input.
  - Tab completion callback.
  - Intelligent input interruption with Ctrl+C.
  - Ctrl+Z (suspend process), Ctrl+\ (send SIGQUIT to process e.g. to get stack dump).

- Additional features not in the original libedit or textarea:
  - Hide/show the prompt to simplify copy-paste from terminal.
  - Secondary prompt for multi-line input.
  - Debug mode for troubleshooting.

## Customizable key bindings

| Name                       | Default keys           | Description                                                                     |
|----------------------------|------------------------|---------------------------------------------------------------------------------|
| EndOfInput                 | Ctrl+D                 | Terminate the input if the cursor is at the beginning of a line.                |
| Interrupt                  | Ctrl+C                 | Clear the input if non-empty, or interrupt input if already empty.              |
| AutoComplete               | Tab                    | Run the `AutoComplete` callback if defined.                                     |
| HideShowPrompt             | Alt+.                  | Hide/show the prompt (eases copy-paste from terminal).                          |
| Refresh                    | Ctrl+L                 | Clear the screen and re-display the current input.                              |
| AbortSearch                | Ctrl+G                 | Abort the search if currently searching; no-op otherwise.                       |
| SearchBackward             | Ctrl+R                 | Start searching; or previous search match if already searching.                 |
| HistoryPrevious            | Alt+P                  | Recall previous history entry.                                                  |
| HistoryNext                | Alt+N                  | Recall next history entry.                                                      |
| InsertNewline              | Enter, Ctrl+M          | Enter a new line; or terminate input if `CheckInputComplete` returns true.      |
| CharacterBackward          | Right, Ctrl+F          | Move one character to the right.                                                |
| CharacterForward           | Left, Ctrl+B           | Move one character to the left.                                                 |
| WordForward                | Alt+Right, Alt+F       | Move cursor to the previous word.                                               |
| WordBackward               | Alt+Left, Alt+B        | Move cursor to the next word.                                                   |
| LineNext                   | Home, Ctrl+A           | Move cursor to beginning of line.                                               |
| LineEnd                    | End, Ctrl+E            | Move cursor to end of line.                                                     |
| LinePrevious               | Up, Ctrl+P             | Move cursor one line up, or to previous history entry if already on first line. |
| LineStart                  | Down, Ctrl+N           | Move cursor one line down, or to next history entry if already on last line.    |
| TransposeCharacterBackward | Ctrl+T                 | Transpose the last two characters.                                              |
| UppercaseWordForward       | Alt+U                  | Make the next word uppercase.                                                   |
| LowercaseWordForward       | Alt+L                  | Make the next word lowercase.                                                   |
| CapitalizeWordForward      | Alt+C                  | Capitalize the next word.                                                       |
| DeleteAfterCursor          | Ctrl+K                 | Delete the line after the cursor.                                               |
| DeleteBeforeCursor         | Ctrl+U                 | Delete the line before the cursor.                                              |
| DeleteCharacterBackward    | Backspace, Ctrl+H      | Delete the character before the cursor.                                         |
| DeleteCharacterForward     | Delete                 | Delete the character after the cursor.                                          |
| DeleteWordBackward         | Alt+Backspace, Ctrl+W  | Delete the word before the cursor.                                              |
| DeleteWordForward          | Alt+Delete, Alt+D      | Delete the word after the cursor.                                               |
| SignalQuit                 | Ctrl+\                 | Send SIGQUIT to process.                                                        |
| SignalTTYStop              | Ctrl+Z                 | Send SIGTSTOP to process (suspend).                                             |
| Debug                      | (not bound by default) | Print debug information about the editor.                                       |

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
