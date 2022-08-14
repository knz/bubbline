# Bubbline

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/bubbline)
[![Go ReportCard](https://goreportcard.com/badge/knz/bubbline)](https://goreportcard.com/report/knz/bubbline)

An input line editor for line-oriented terminal applications.

Based off the [bubbletea](https://github.com/charmbracelet/bubbletea) library.

## Features of the line editor

- Adds [editline/libedit](https://man.netbsd.org/editline.3)/[readline](https://en.wikipedia.org/wiki/GNU_Readline)
  goodies to [bubbles](https://github.com/charmbracelet/bubbles)'
  original `textarea` widget:
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

## Demo / explanation

[![Loom demo](https://cdn.loom.com/sessions/thumbnails/29b2effdcdda40b9a12509c2ced1de8c-with-play.gif)](https://www.loom.com/share/29b2effdcdda40b9a12509c2ced1de8c)

## Customizable key bindings

| Default keys           | Description                                                                     | Binding name               |
|------------------------|---------------------------------------------------------------------------------|----------------------------|
| Ctrl+D                 | Terminate the input if the cursor is at the beginning of a line.                | EndOfInput                 |
| Ctrl+C                 | Clear the input if non-empty, or interrupt input if already empty.              | Interrupt                  |
| Tab                    | Run the `AutoComplete` callback if defined.                                     | AutoComplete               |
| Alt+.                  | Hide/show the prompt (eases copy-paste from terminal).                          | HideShowPrompt             |
| Ctrl+L                 | Clear the screen and re-display the current input.                              | Refresh                    |
| Ctrl+G                 | Abort the search if currently searching; no-op otherwise.                       | AbortSearch                |
| Ctrl+R                 | Start searching; or previous search match if already searching.                 | SearchBackward             |
| Alt+P                  | Recall previous history entry.                                                  | HistoryPrevious            |
| Alt+N                  | Recall next history entry.                                                      | HistoryNext                |
| Ctrl+M, Enter          | Enter a new line; or terminate input if `CheckInputComplete` returns true.      | InsertNewline              |
| Alt+Enter              | Always enter a newline.                                                         | AlwaysNewline              |
| Ctrl+F, Right          | Move one character to the right.                                                | CharacterBackward          |
| Ctrl+B, Left           | Move one character to the left.                                                 | CharacterForward           |
| Alt+F, Alt+Right       | Move cursor to the previous word.                                               | WordForward                |
| Alt+B, Alt+Left        | Move cursor to the next word.                                                   | WordBackward               |
| Ctrl+A, Home           | Move cursor to beginning of line.                                               | LineNext                   |
| Ctrl+E, End            | Move cursor to end of line.                                                     | LineEnd                    |
| Ctrl+P, Up             | Move cursor one line up, or to previous history entry if already on first line. | LinePrevious               |
| Ctrl+N, Down           | Move cursor one line down, or to next history entry if already on last line.    | LineStart                  |
| Ctrl+T                 | Transpose the last two characters.                                              | TransposeCharacterBackward |
| Alt+U                  | Make the next word uppercase.                                                   | UppercaseWordForward       |
| Alt+L                  | Make the next word lowercase.                                                   | LowercaseWordForward       |
| Alt+C                  | Capitalize the next word.                                                       | CapitalizeWordForward      |
| Ctrl+K                 | Delete the line after the cursor.                                               | DeleteAfterCursor          |
| Ctrl+U                 | Delete the line before the cursor.                                              | DeleteBeforeCursor         |
| Ctrl+H, Backspace      | Delete the character before the cursor.                                         | DeleteCharacterBackward    |
| Delete                 | Delete the character after the cursor.                                          | DeleteCharacterForward     |
| Ctrl+W, Alt+Backspace  | Delete the word before the cursor.                                              | DeleteWordBackward         |
| Alt+D, Alt+Delete      | Delete the word after the cursor.                                               | DeleteWordForward          |
| Ctrl+\                 | Send SIGQUIT to process.                                                        | SignalQuit                 |
| Ctrl+Z                 | Send SIGTSTOP to process (suspend).                                             | SignalTTYStop              |
| (not bound by default) | Print debug information about the editor.                                       | Debug                      |

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
