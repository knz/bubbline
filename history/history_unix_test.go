//go:build !windows
// +build !windows

package history

import (
	"fmt"
	"io/ioutil"
	"os"
)

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
