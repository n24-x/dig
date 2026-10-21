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
	_ digError = errInvalidInput{}
	_ digError = errInvalidGroupOption{}
	_ digError = errResourceAddFailed{}
)

var _ func(string, error) errInvalidInput = newErrInvalidInput

type Error interface {
	error
	writeMessage(w io.Writer, v string)
}

type PanicError struct {
	//nolint:unused // TODO use
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
	_, _ = fmt.Fprint(w, e.Message)
}

func (e errInvalidInput) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}

type errInvalidGroupOption struct{ Option string }

func (e errInvalidGroupOption) Error() string { return fmt.Sprint(e) }

func (e errInvalidGroupOption) writeMessage(w io.Writer, _ string) {
	_, _ = fmt.Fprintf(w, "invalid option %q", e.Option)
}

func (e errInvalidGroupOption) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}

// errResourceAddFailed is returned when a resource cannot be added to the
// container.
//
// It corresponds to the Err field of pievent.ResourceAdded, but unlike that
// event it exists only on failure.
type errResourceAddFailed struct {
	Func   *digreflect.Func
	Reason error
}

func (e errResourceAddFailed) Error() string { return fmt.Sprint(e) }

func (e errResourceAddFailed) Unwrap() error { return e.Reason }

func (e errResourceAddFailed) writeMessage(w io.Writer, verb string) {
	_, _ = fmt.Fprintf(w, "cannot add resource "+verb, e.Func)
}

func (e errResourceAddFailed) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}
