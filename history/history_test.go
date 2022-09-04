package history

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
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
		{"_HiStOrY_V2_\nfoo\nbar\n", []string{"foo", "bar"}, ""},
		{"_HiStOrY_V2_\nfo\\?o\n\134b\\01ar\\", []string{"fo\\?o", "\\b\\01ar\\"}, ""},
		{"_HiStOrY_V2_\nfoo\\040", []string{"foo "}, ""},
		{"_HiStOrY_V2_\nfoo\\040bar", []string{"foo bar"}, ""},
		{"_HiStOrY_V2_\nfoo\\040bar\nbaz", []string{"foo bar", "baz"}, ""},
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
		if !reflect.DeepEqual(tc.exp, h) {
			t.Errorf("%q: expected:\n%+v\ngot:\n%+v", tc.input, tc.exp, h)
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
		{[]string{"fo\\03o", "bar"}, "_HiStOrY_V2_\nfo\\13403o\nbar\n", ""},
		{[]string{"foo "}, "_HiStOrY_V2_\nfoo\\040\n", ""},
		{[]string{"foo bar"}, "_HiStOrY_V2_\nfoo\\040bar\n", ""},
		{[]string{"foo bar", "baz"}, "_HiStOrY_V2_\nfoo\\040bar\nbaz\n", ""},
	}

	for _, tc := range testCases {
		var buf bytes.Buffer
		err := saveHistoryToFile(tc.input, &buf)
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

func Example_history() {
	_, err := LoadHistory("notexist")
	fmt.Println(err)
	_, err = LoadHistory("/dev/null/notvalid")
	fmt.Println(err)
	err = SaveHistory(nil, "/dev/null/notvalid")
	fmt.Println(err)

	f, err := ioutil.TempFile("", "test")
	if err != nil {
		fmt.Println(err)
		return
	}
	fname := f.Name()
	f.Close()
	defer os.Remove(fname)
	err = SaveHistory(nil, fname)
	fmt.Println(err)
	_, err = LoadHistory(fname)
	fmt.Println(err)

	// Output:
	// <nil>
	// open /dev/null/notvalid: not a directory
	// open /dev/null/notvalid: not a directory
	// <nil>
	// <nil>
}
