package computil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cockroachdb/datadriven"
)

func TestConvertToCompletionQuery(t *testing.T) {
	datadriven.RunTest(t, "testdata/convert_runes", func(t *testing.T, td *datadriven.TestData) string {
		switch td.Cmd {
		case "convert":
			if !td.HasArg("pos") {
				t.Fatalf("%s: need pos keyword arg", td.Pos)
			}
			var line, col int
			td.ScanArgs(t, "pos", &line, &col)
			rows := strings.Split(td.Input, "\n")
			var r [][]rune
			for _, row := range rows {
				r = append(r, []rune(row))
			}

			s, pos := Flatten(r, line, col)
			s = strings.ReplaceAll(s, "\n", "␤") + "🛇"
			var res strings.Builder
			fmt.Fprintln(&res, s)
			fmt.Fprintf(&res, "%*s^\n", pos, "")
			return res.String()

		default:
			t.Fatalf("%s: unknown command: %q", td.Pos, td.Cmd)
			return "" // unreachable
		}
	})
}
