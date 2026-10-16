package dig

import (
	"fmt"
	"io"

	"github.com/n24-x/dig/internal/digreflect"
)

type digError interface {
	Error
	fmt.Formatter
}

var (
	_ digError = (*errInvalidInput)(nil)
	_ digError = (*errInvalidGroupOption)(nil)
)

type Error interface {
	error
	writeMessage(w io.Writer, v string)
}

type PanicError struct {
	fn    *digreflect.Func
	Panic any
}

// errInvalidInput represents an error caused by invalid input.
//
// TODO maybe it need rename
type errInvalidInput struct {
	Message string
	Cause   error
}

func newErrInvalidInput(msg string, cause error) errInvalidInput {
	return errInvalidInput{msg, cause}
}

func (e errInvalidInput) Error() string { return fmt.Sprint(e) }
func (e errInvalidInput) Unwrap() error { return e.Cause }

func (e errInvalidInput) writeMessage(w io.Writer, _ string) {
	fmt.Fprint(w, e.Message)
}

func (e errInvalidInput) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}

type errInvalidGroupOption struct{ Option string }

func (e errInvalidGroupOption) Error() string { return fmt.Sprint(e) }

func (e errInvalidGroupOption) writeMessage(w io.Writer, v string) {
	fmt.Fprintf(w, "invalid option %q", e.Option)
}

func (e errInvalidGroupOption) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}
