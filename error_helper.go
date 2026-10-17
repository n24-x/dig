package dig

import (
	"errors"
	"fmt"
	"io"
)

var _ func(digError, fmt.State, rune) = formatError

func formatError(e digError, w fmt.State, v rune) {
	multiline := w.Flag('+') && v == 'v'
	verb := "%v"
	if multiline {
		verb = "%+v"
	}

	// "context: " or "context:\n"
	e.writeMessage(w, verb)

	// Will route back to this function recursively if next error
	// is also wrapped and points back here
	wrappedError := errors.Unwrap(e)
	if wrappedError == nil {
		return
	}
	io.WriteString(w, ":")
	if multiline {
		io.WriteString(w, "\n")
	} else {
		io.WriteString(w, " ")
	}
	fmt.Fprintf(w, verb, wrappedError)
}
