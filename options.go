package dig

import (
	"fmt"
	"strings"

	"github.com/n24-x/dig/internal/digreflect"
)

type Option[T any] func(T)

// constructorOptions carries the settings that shape a parsed constructorNode.
type constructorOptions struct {
	Result   resultOptions
	Location *digreflect.Func

	// TODO
	// Callback
	// BeforeCallback
}

// resultOptions carries the settings that shape a parsed result.
type resultOptions struct {
	// If set, this is the name of the associated result value.
	//
	// For Return Objects, name:".." tags on fields override this.
	ValueName string
	GroupName string
	As        []any
}

// resourceOptions holds the accumulated effects of the ResourceOption values
// passed to Resource.
type resourceOptions struct {
	ValueName string
	GroupName string
	As        []any
	Location  *digreflect.Func

	// TODO
	// Info *ResourceInfo
}

// TODO: Validate reports whether the option combination is usable. Options are
// validated as a set, because some constraints span multiple fields (Name and
// Group are mutually exclusive).
func (ro *resourceOptions) Validate() error {
	if ro.ValueName != "" && ro.GroupName != "" {
		return newErrInvalidInput(
			fmt.Sprintf("cannot use name-tag and group-tag together: name:%q group:%q",
				ro.ValueName, ro.GroupName), nil)
	}

	// Names must be representable inside a backquoted string. The only
	// limitation for raw string literals as per
	// https://golang.org/ref/spec#raw_string_lit is that they cannot contain
	// backquotes.
	if strings.ContainsRune(ro.ValueName, '`') {
		return newErrInvalidInput(
			fmt.Sprintf("invalid dig.Name(%q): names cannot contain backquotes", ro.ValueName), nil)
	}
	if strings.ContainsRune(ro.GroupName, '`') {
		return newErrInvalidInput(
			fmt.Sprintf("invalid dig.Group(%q): group names cannot contain backquotes", ro.GroupName), nil)
	}

	// TODO
	// As Validate

	return nil
}
