package dig

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/n24-x/dig/internal/digreflect"
)

type Option[T any] func(T)

// ConstructorOptions carries the settings that shape a parsed constructorNode.
// See [Option] for more info.
//
// TODO change to unexport
type ConstructorOptions struct {
	result          resultOptions
	buildScope      *Scope
	registeredScope *Scope
	location        *digreflect.Func
}

// DecorateOptions holds the accumulated effects of the Option values passed
// to Decorate. See [Option] for more info.
type DecorateOptions struct {
	info     *DecoratorReceiptInfo
	observer Observer
}

// resultOptions carries the settings that shape a parsed result.
type resultOptions struct {
	// If set, this is the name of the associated result value.
	//
	// For Return Objects, name:".." tags on fields override this.
	nameTag  string
	groupTag string
	asTypes  []any
}

// ResourceOptions holds the accumulated effects of the ResourceOption values
// passed to Resource. See [Option] for more info.
type ResourceOptions struct {
	nameTag         string
	groupTag        string
	asTypes         []any
	global          bool
	registeredScope *Scope
	location        *digreflect.Func
	info            *ResourceReceiptInfo
	observer        Observer
}

var _ interface{ Validate() error } = (*ResourceOptions)(nil)

// Validate reports whether the option combination is usable. Options are
// validated as a set, because some constraints span multiple fields (Name and
// Group are mutually exclusive).
func (ro *ResourceOptions) Validate() error {
	if ro.nameTag != "" && ro.groupTag != "" {
		return newErrInvalidInput(
			fmt.Sprintf("cannot use name-tag and group-tag together: name:%q group:%q",
				ro.nameTag, ro.groupTag), nil)
	}

	// Names must be representable inside a backquoted string. The only
	// limitation for raw string literals as per
	// https://golang.org/ref/spec#raw_string_lit is that they cannot contain
	// backquotes.
	if strings.ContainsRune(ro.nameTag, '`') {
		return newErrInvalidInput(
			fmt.Sprintf("invalid dig.Name(%q): names cannot contain backquotes", ro.nameTag), nil)
	}
	if strings.ContainsRune(ro.groupTag, '`') {
		return newErrInvalidInput(
			fmt.Sprintf("invalid dig.Group(%q): group names cannot contain backquotes", ro.groupTag), nil)
	}

	duplicateAsType := make(map[reflect.Type]struct{})
	for _, v := range ro.asTypes {
		asType := reflect.TypeOf(v)

		if asType == nil {
			return newErrInvalidInput("invalid dig.As(nil): argument must be a pointer to an interface", nil)
		}

		if asType.Kind() != reflect.Pointer {
			return newErrInvalidInput(
				fmt.Sprintf("invalid dig.As(%v): argument must be a pointer to an interface", asType), nil)
		}

		pointingTo := asType.Elem()
		if pointingTo.Kind() != reflect.Interface {
			return newErrInvalidInput(
				fmt.Sprintf("invalid dig.As(*%v): argument must be a pointer to an interface", pointingTo), nil)
		}

		if isErrorType(pointingTo) {
			return newErrInvalidInput("invalid dig.As(new(error)): cannot use error in As()", nil)
		}

		_, ok := duplicateAsType[asType]
		if ok {
			return newErrInvalidInput(
				fmt.Sprintf("duplicate type in As: %v", asType), nil)
		}
		duplicateAsType[asType] = struct{}{}
	}

	return nil
}
