package history

import (
	"bytes"
	"reflect"
	"testing"
)

func TestLoadHistory(t *testing.T) {
	testCases := []struct {
		input  string
		exp    []string
		expErr string
	}{
		{"", nil, ""},
		{"foo", nil, ""},
		{"_HiStOrY_V2_\nfoo\nbar", []string{"foo", "bar"}, ""},
		{"_HiStOrY_V2_\nfoo\\040", []string{"foo "}, ""},
		{"_HiStOrY_V2_\nfoo\\040bar", []string{"foo bar"}, ""},
		{"_HiStOrY_V2_\nfoo\\040bar\nbaz", []string{"foo bar", "baz"}, ""},
		{"_HiStOrY_V2_\nfoo\\04", nil, `invalid sequence: \04`},
		{"_HiStOrY_V2_\nfoo\\888", nil, `invalid sequence: \888`},
	}

	for _, tc := range testCases {
		buf := bytes.NewBufferString(tc.input)
		h, err := loadHistoryFromFile(buf)
		if tc.expErr != "" {
			if err == nil {
				t.Errorf("%q: expected error, got no error", tc.input)
			} else if err.Error() != tc.expErr {
				t.Errorf("%q: expected error:\n%s, got:\n%v", tc.input, tc.expErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: expected no error, got: %v", tc.input, err)
		}
		hs := []string(h)
		if !reflect.DeepEqual(tc.exp, hs) {
			t.Errorf("%q: expected:\n%+v\ngot:\n%+v", tc.input, tc.exp, hs)
		}
	}
}

func TestSaveHistory(t *testing.T) {
	testCases := []struct {
		input  []string
		exp    string
		expErr string
	}{
		{nil, "_HiStOrY_V2_\n", ""},
		{[]string{"foo", "bar"}, "_HiStOrY_V2_\nfoo\nbar\n", ""},
		{[]string{"foo "}, "_HiStOrY_V2_\nfoo\\040\n", ""},
		{[]string{"foo bar"}, "_HiStOrY_V2_\nfoo\\040bar\n", ""},
		{[]string{"foo bar", "baz"}, "_HiStOrY_V2_\nfoo\\040bar\nbaz\n", ""},
	}

	for _, tc := range testCases {
		h := History(tc.input)
		var buf bytes.Buffer
		err := saveHistoryToFile(h, &buf)
		if tc.expErr != "" {
			if err == nil {
				t.Errorf("%q: expected error, got no error", tc.input)
			} else if err.Error() != tc.expErr {
				t.Errorf("%q: expected error:\n%s, got:\n%v", tc.input, tc.expErr, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: expected no error, got: %v", tc.input, err)
		}
		if result := buf.String(); result != tc.exp {
			t.Errorf("%q: expected:\n%q\ngot:\n%q", tc.input, tc.exp, result)
		}
	}
}
