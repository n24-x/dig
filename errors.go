package dig

import (
	"fmt"
	"io"
	"strings"

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
	_ digError = errCycleDetected{}
	_ digError = errArgumentsFailed{}
	_ digError = errValueNotFound{}
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

// errCycleDetected is returned when adding a resource would close a cycle in
// the graph.
type errCycleDetected struct {
	Message string

	// Cause is the *dag.CycleError that triggered this error.
	Cause error
}

func (e errCycleDetected) Error() string { return fmt.Sprint(e) }

func (e errCycleDetected) writeMessage(w io.Writer, _ string) {
	_, _ = fmt.Fprint(w, e.Message)
	if e.Cause == nil {
		return
	}
	_, _ = fmt.Fprintf(w, ": %s",
		strings.TrimPrefix(e.Cause.Error(), "dag: cycle detected in graph: "))
}

func (e errCycleDetected) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}

// errArgumentsFailed is returned when a function could not be run because one
// of its dependencies failed to build for any reason.
type errArgumentsFailed struct {
	Func   *digreflect.Func
	Reason error
}

func (e errArgumentsFailed) Error() string { return fmt.Sprint(e) }

func (e errArgumentsFailed) Unwrap() error { return e.Reason }

func (e errArgumentsFailed) writeMessage(w io.Writer, verb string) {
	_, _ = fmt.Fprintf(w, "cannot build arguments for function "+verb, e.Func)
}

func (e errArgumentsFailed) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}

// errValueNotFound is returned when a param asks the scope for a value that has
// not been built.
//
// This is an invariant violation rather than a user error: dag.Resolve
// guarantees that every provider of a non-optional dependency runs before its
// consumers, so by the time Build runs the value must already be in the scope.
// Optional params and passive groups do not reach this branch; they fall back
// to a zero value or an empty slice before consulting the store.
//
// In practice it fires when the set of keys a node provides and the set of keys
// it writes at extraction time disagree.
//
// It deliberately has no Unwrap: there is no cause, and formatError would print
// one a second time if it existed.
//
// TODO simplify
type errValueNotFound struct {
	key resourceType
}

func (e errValueNotFound) Error() string { return fmt.Sprint(e) }

func (e errValueNotFound) writeMessage(w io.Writer, _ string) {
	_, _ = fmt.Fprintf(w, "value not found in scope: %v", e.key)
}

func (e errValueNotFound) Format(w fmt.State, c rune) {
	formatError(e, w, c)
}
