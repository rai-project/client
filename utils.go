package client

import (
	"fmt"
	"io"
	"strings"

	"github.com/rai-project/acl"
)

type ValidationError struct {
	Message string
}

type nopWriterCloser struct {
	io.Writer
}

func (v *ValidationError) Error() string {
	if v == nil {
		return ""
	}
	return v.Message
}

func isECE408Role(r acl.Role) bool {
	s := strings.ToLower(string(r))
	return strings.Contains(s, "ece408")
}

// Close ...
func (nopWriterCloser) Close() error { return nil }

func fprint(w io.Writer, a ...interface{}) (n int, err error) {
	if w == nil {
		return 0, nil
	}
	return fmt.Fprint(w, a...)
}

func fprintln(w io.Writer, a ...interface{}) (n int, err error) {
	if w == nil {
		return 0, nil
	}
	return fmt.Fprintln(w, a...)
}

func fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	if w == nil {
		return 0, nil
	}
	return fmt.Fprintf(w, format, a...)
}
