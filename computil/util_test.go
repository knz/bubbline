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

func TestFindLongestCommonPrefix(t *testing.T) {
	td := []struct {
		a, b string
		ci   bool
		exp  string
	}{
		{``, ``, false, ``},
		{`a`, ``, false, ``},
		{``, `b`, false, ``},
		{`aab`, `ab`, false, `a`},
		{`aab`, `ab`, true, `a`},
		{`aab`, `aa`, false, `aa`},
		{`aab`, `aa`, true, `aa`},
		{`aab`, `Aba`, false, `a`},
		{`aab`, `Aba`, true, ``},
		{"\xc3\xb8", "\xc3\x98", true, ""},
		{"\xc3\xb8", "\xc3\x98", false, "ø"},
	}

	for i, tc := range td {
		p := FindLongestCommonPrefix(tc.a, tc.b, tc.ci)
		if p != tc.exp {
			t.Fatalf("%d: expected %q, got %q", i, tc.exp, p)
		}
	}
}

func TestFindWord(t *testing.T) {
	text := [][]rune{
		[]rune("there's no place"),
		[]rune("like home"),
	}

	word, s, e := FindWord(text, 1, 6)
	if word != "home" || s != 5 || e != 9 {
		t.Fatal("bad")
	}
	word, s, e = FindWord(text, 1, 2)
	if word != "like" || s != 0 || e != 4 {
		t.Fatal("bad")
	}
}
