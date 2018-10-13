package client

import (
	"fmt"
	"io"
)

type ValidationError struct {
	Message string
}

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
