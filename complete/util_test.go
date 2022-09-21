package complete

import "testing"

func TestInterfaces(t *testing.T) {
	s := "hello"
	if StringEntry(s).Title() != "hello" {
		t.Fatal("bad")
	}

	v := StringValues("myvals", []string{s, "world"})
	if v.NumCategories() != 1 {
		t.Fatal("bad")
	}
	if v.CategoryTitle(0) != "myvals" {
		t.Fatal("bad")
	}
	if v.NumEntries(0) != 2 {
		t.Fatal("bad")
	}
	if v.Entry(0, 0).Title() != "hello" {
		t.Fatal("bad")
	}

	v = MapValues(map[string][]string{"myvals": {s, "world"}}, nil)

	if v.NumCategories() != 1 {
		t.Fatal("bad")
	}
	if v.CategoryTitle(0) != "myvals" {
		t.Fatal("bad")
	}
	if v.NumEntries(0) != 2 {
		t.Fatal("bad")
	}
	if v.Entry(0, 0).Title() != "hello" {
		t.Fatal("bad")
	}

	v = MapValues(map[string][]myE{"myvals": {myE{}, myE{}}}, nil)

	if v.NumCategories() != 1 {
		t.Fatal("bad")
	}
	if v.CategoryTitle(0) != "myvals" {
		t.Fatal("bad")
	}
	if v.NumEntries(0) != 2 {
		t.Fatal("bad")
	}
	if v.Entry(0, 0).Title() != "hello" {
		t.Fatal("bad")
	}

	v = MapValues(map[string][]myE2{"myvals": {myE2{}, myE2{}}}, nil)

	if v.NumCategories() != 1 {
		t.Fatal("bad")
	}
	if v.CategoryTitle(0) != "myvals" {
		t.Fatal("bad")
	}
	if v.NumEntries(0) != 2 {
		t.Fatal("bad")
	}
	if v.Entry(0, 0).Title() != "hello" {
		t.Fatal("bad")
	}
}

type myE struct{}

func (myE) Title() string       { return "hello" }
func (myE) Description() string { return "" }

type myE2 struct{}

func (*myE2) Title() string       { return "hello" }
func (*myE2) Description() string { return "" }
