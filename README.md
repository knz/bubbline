# Bubbline

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://pkg.go.dev/github.com/knz/bubbline)
[![Go ReportCard](https://goreportcard.com/badge/knz/bubbline)](https://goreportcard.com/report/knz/bubbline)

An input line editor for line-oriented terminal applications.

Based off the [bubbletea](https://github.com/charmbracelet/bubbletea) library.

## Features of the line editor

| Feature                                                                            | Charm `textarea` [^T] | `libedit` [^l1] /`readline` [^l2] | Bubbline (this library) |
|------------------------------------------------------------------------------------|:---------------------:|:---------------------------------:|:-----------------------:|
| Multi-line editor with both horizontal and vertical cursor navigation.             | ✅                    | ✅                                | ✅                      |
| Resizes vertically automatically as the input grows.                               | ❌                    | ✅                                | ✅                      |
| Secondary prompt for multi-line input.                                             | ❌ [^p2]              | ✅                                | ✅                      |
| Supports history navigation and search.                                            | ❌                    | ✅                                | ✅                      |
| Word navigation across input lines.                                                | ❌                    | ❌                                | ✅                      |
| Enter key conditionally ends the input.                                            | ❌                    | ✅                                | ✅                      |
| Tab completion callback.                                                           | ❌                    | ✅                                | ✅                      |
| Fancy presentation of completions with menu navigation.                            | ❌                    | ✅ [^cp]                          | ✅                      |
| Intelligent input interruption with Ctrl+C.                                        | ❌                    | ✅                                | ✅                      |
| Ctrl+Z (suspend process), Ctrl+\ (send SIGQUIT to process e.g. to get stack dump). | ❌                    | ✅                                | ✅                      |
| Uppercase/lowercase/capitalize next word, transpose characters.                    | ✅                    | ✅                                | ✅                      |
| Toggle overwrite mode.                                                             | ❌ [^p1]              | ❌                                | ✅ [^p3]                |
| Key combination to reflow the text to fit within a specific width.                 | ❌                    | ❌                                | ✅                      |
| Hide/show the prompt to simplify copy-paste from terminal.                         | ❌                    | ❌                                | ✅                      |
| Debug mode for troubleshooting.                                                    | ❌                    | ❌                                | ✅                      |
| Bracketed paste [^bp]                                                              | ❌ [^p4]              | ✅                                | ❌ [^p4]                |

[^T]: https://github.com/charmbracelet/bubbles
[^l1]: [editline/libedit](https://man.netbsd.org/editline.3)
[^l2]: [GNU readline](https://en.wikipedia.org/wiki/GNU_Readline)
[^p1]: Pending https://github.com/charmbracelet/bubbles/pull/225
[^p2]: Pending https://github.com/charmbracelet/bubbles/pull/211
[^p3]: Only Ctrl+O for now; insert key support pending https://github.com/charmbracelet/bubbletea/pull/418
[^cp]: libedit/readline's completion menu is a single line of options with wraparound.
[^bp]: https://en.wikipedia.org/wiki/Bracketed-paste
[^p4]: Pending https://github.com/charmbracelet/bubbletea/pull/397 and https://github.com/charmbracelet/bubbletea/pull/397

## Demo / explanation

[![Loom demo](https://cdn.loom.com/sessions/thumbnails/29b2effdcdda40b9a12509c2ced1de8c-with-play.gif)](https://www.loom.com/share/29b2effdcdda40b9a12509c2ced1de8c)

## Customizable key bindings

| Default keys           | Description                                                                                  | Binding name               |
|------------------------|----------------------------------------------------------------------------------------------|----------------------------|
| Ctrl+D                 | Terminate the input if the cursor is at the beginning of a line; delete character otherwise. | EndOfInput                 |
| Ctrl+C                 | Clear the input if non-empty, or interrupt input if already empty.                           | Interrupt                  |
| Tab                    | Run the `AutoComplete` callback if defined.                                                  | AutoComplete               |
| Alt+.                  | Hide/show the prompt (eases copy-paste from terminal).                                       | HideShowPrompt             |
| Ctrl+L                 | Clear the screen and re-display the current input.                                           | Refresh                    |
| Ctrl+G                 | Abort the search if currently searching; no-op otherwise.                                    | AbortSearch                |
| Ctrl+R                 | Start searching; or previous search match if already searching.                              | SearchBackward             |
| Alt+P                  | Recall previous history entry.                                                               | HistoryPrevious            |
| Alt+N                  | Recall next history entry.                                                                   | HistoryNext                |
| Ctrl+M, Enter          | Enter a new line; or terminate input if `CheckInputComplete` returns true.                   | InsertNewline              |
| Alt+Enter              | Always enter a newline; ignore input termination condition.                                  | AlwaysNewline              |
| Ctrl+F, Right          | Move one character to the right.                                                             | CharacterBackward          |
| Ctrl+B, Left           | Move one character to the left.                                                              | CharacterForward           |
| Alt+F, Alt+Right       | Move cursor to the previous word.                                                            | WordForward                |
| Alt+B, Alt+Left        | Move cursor to the next word.                                                                | WordBackward               |
| Ctrl+A, Home           | Move cursor to beginning of line.                                                            | LineNext                   |
| Ctrl+E, End            | Move cursor to end of line.                                                                  | LineEnd                    |
| Ctrl+P, Up             | Move cursor one line up, or to previous history entry if already on first line.              | LinePrevious               |
| Ctrl+N, Down           | Move cursor one line down, or to next history entry if already on last line.                 | LineStart                  |
| Ctrl+T                 | Transpose the last two characters.                                                           | TransposeCharacterBackward |
| Ctrl+O, Insert         | Toggle overwrite mode.                                                                       | ToggleOverwriteMode        |
| Alt+U                  | Make the next word uppercase.                                                                | UppercaseWordForward       |
| Alt+L                  | Make the next word lowercase.                                                                | LowercaseWordForward       |
| Alt+C                  | Capitalize the next word.                                                                    | CapitalizeWordForward      |
| Ctrl+K                 | Delete the line after the cursor.                                                            | DeleteAfterCursor          |
| Ctrl+U                 | Delete the line before the cursor.                                                           | DeleteBeforeCursor         |
| Ctrl+H, Backspace      | Delete the character before the cursor.                                                      | DeleteCharacterBackward    |
| Delete                 | Delete the character after the cursor.                                                       | DeleteCharacterForward     |
| Ctrl+W, Alt+Backspace  | Delete the word before the cursor.                                                           | DeleteWordBackward         |
| Alt+D, Alt+Delete      | Delete the word after the cursor.                                                            | DeleteWordForward          |
| Ctrl+\                 | Send SIGQUIT to process.                                                                     | SignalQuit                 |
| Ctrl+Z                 | Send SIGTSTOP to process (suspend).                                                          | SignalTTYStop              |
| Alt+?                  | Toggle display of keybindings.                                                               | MoreHelp                   |
| Alt+q                  | Reflow the current line.                                                                     | ReflowLine                 |
| Alt+Shift+Q            | Reflow the entire input.                                                                     | ReflowAll                  |
| (not bound by default) | Print debug information about the editor.                                                    | Debug                      |

## Example use

```go
package main

import (
    "errors"
    "fmt"
    "io"
    "log"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/knz/bubbline"
)

func main() {
    // Instantiate the widget.
    m := bubbline.New()

    for {
        // Read a line of input using the widget.
        val, err := m.GetLine()

        // Handle the end of input.
        if err != nil {
            if err == io.EOF {
                // No more input.
                break
            }
            if errors.Is(err, bubbline.ErrInterrupted) {
                // Entered Ctrl+C to cancel input.
                fmt.Println("^C")
            } else {
                fmt.Println("error:", err)
            }
            continue
        }

        // Handle regular input.
        fmt.Printf("\nYou have entered: %q\n", val)
        m.AddHistory(val)
    }
}
```

See the `examples` subdirectory for more examples!
