package file

import (
	"fmt"
	"io/ioutil"
)

// WriteBytesToFile writes given bytes to a file.
func WriteBytesToFile(dst string, b []byte) error {
	if err := ioutil.WriteFile(dst, b, 0644); err != nil {
		return fmt.Errorf("%s: writing file: %v", dst, err)
	}
	return nil
}

// ContentToString returns the string content of the
// given file.
func ContentToString(fpath string) (string, error) {
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return "", fmt.Errorf("%s: reading file: %v", fpath, err)
	}
	return string(b), nil
}
