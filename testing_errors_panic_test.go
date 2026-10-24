package dig

import (
	"errors"
	"fmt"
	"testing"

	"github.com/n24-x/dig/internal/digreflect"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPanicErrorFormatting(t *testing.T) {
	t.Parallel()

	fn := digreflect.InspectFunc(TestPanicErrorFormatting)
	stringPanic := PanicError{Func: fn, Panic: "boom"}

	t.Run("short", func(t *testing.T) {
		assert.Equal(t,
			fmt.Sprintf("panic: %q in func: %v", "boom", fn),
			stringPanic.Error())
	})

	t.Run("expanded", func(t *testing.T) {
		got := fmt.Sprintf("%+v", stringPanic)
		assert.Contains(t, got, `panic: "boom" in func: `)
		assert.Contains(t, got, "testing_errors_panic_test.go:") // file:line of this func
	})

	t.Run("non string panic", func(t *testing.T) {
		// %q would print a non-string as a character (42 => '*') and leak a
		// formatting error for floats and bools, so only strings are quoted.
		pe := PanicError{Func: fn, Panic: 42}
		assert.Contains(t, pe.Error(), "panic: 42 in func:")
	})

	t.Run("float panic", func(t *testing.T) {
		pe := PanicError{Func: fn, Panic: 3.14}
		got := pe.Error()
		assert.Contains(t, got, "panic: 3.14 in func:")
		assert.NotContains(t, got, "%!q", "must not leak a fmt error into the message")
	})

	t.Run("error panic", func(t *testing.T) {
		pe := PanicError{Func: fn, Panic: errors.New("nope")}
		got := pe.Error()
		assert.Contains(t, got, "panic: nope in func:")
		assert.NotContains(t, got, "%!q")
	})

	t.Run("nil func", func(t *testing.T) {
		pe := PanicError{Panic: "boom"}
		assert.NotPanics(t, func() { _ = pe.Error() })
	})
}

// PanicError must not be a dig.Error: it represents a panic raised by user
// code, and sealing it would make RootCause skip past it.
func TestPanicErrorIsNotDigError(t *testing.T) {
	t.Parallel()

	var err error = PanicError{Panic: "boom"}

	var de Error
	require.False(t, errors.As(err, &de), "PanicError must not satisfy dig.Error")

	var pe PanicError
	require.True(t, errors.As(err, &pe), "PanicError must be discoverable")
}
