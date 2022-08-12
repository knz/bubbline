package history

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type History []string

const cookie = "_HiStOrY_V2_"

// LoadHistory loadsa a history from a file loaded from the specified path.
func LoadHistory(fileName string) (History, error) {
	f, err := os.Open(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return LoadHistoryFromFile(f)
}

// LoadHistoryFromFile loads a history from the specified file.
func LoadHistoryFromFile(f io.Reader) (History, error) {
	var buf [len(cookie) + 1]byte
	n, err := f.Read(buf[:])
	if err == io.EOF {
		// empty file.
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sl := buf[:n]
	if !bytes.Equal(sl, []byte(cookie+"\n")) {
		// Cookie not recognized. No-op.
		return nil, nil
	}
	// Read the remainder of the file.
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if len(contents) > 0 && contents[len(contents)-1] == '\n' {
		contents = contents[:len(contents)-1]
	}
	lines := bytes.Split(contents, []byte("\n"))
	var hist History
	for _, line := range lines {
		// Unescape octal codes.
		resultEnd := 0
		for c := 0; c < len(line); c++ {
			if line[c] == '\\' {
				if c+3 >= len(line) {
					return hist, fmt.Errorf("invalid sequence: %s", line[c:])
				}
				var b byte
				for i := 1; i <= 3; i++ {
					digit := line[c+i]
					if digit < '0' || digit > '7' {
						return hist, fmt.Errorf("invalid sequence: %s", line[c:c+4])
					}
					b = (b << 3) | (digit - '0')
				}
				line[resultEnd] = b
				c += 3
			} else {
				line[resultEnd] = line[c]
			}
			resultEnd++
		}
		result := line[:resultEnd]
		hist = append(hist, string(result))
	}
	return hist, nil
}

// SaveHistory saves a history to the specified file.
func SaveHistory(h History, fileName string) (retErr error) {
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := f.Close()
		if retErr == nil {
			retErr = closeErr
		}
	}()

	return SaveHistoryToFile(h, f)
}

// SaveHistoryToFile saves the history to the specified file.
func SaveHistoryToFile(h History, f io.Writer) error {
	w := bufio.NewWriter(f)
	_, err := w.Write([]byte(cookie + "\n"))
	if err != nil {
		return err
	}
	for _, entry := range h {
		var buf bytes.Buffer
		for c := 0; c < len(entry); c++ {
			if b := entry[c]; b == ' ' || b == '\t' || b == '\n' {
				buf.WriteByte('\\')
				buf.WriteByte((b>>6)&7 + '0')
				buf.WriteByte((b>>3)&7 + '0')
				buf.WriteByte((b>>0)&7 + '0')
			} else {
				buf.WriteByte(b)
			}
		}
		buf.WriteByte('\n')
		_, err := w.Write(buf.Bytes())
		if err != nil {
			return err
		}
	}
	return w.Flush()
}
