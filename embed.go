package dig

import "reflect"

// Required may be embedded into structs to request dig to treat them as special
// parameter structs. When a constructor accepts such a struct, instead of the
// struct becoming a dependency for that constructor, all its fields become
// dependencies instead.
//
// Fields of the struct may optionally be tagged to customize the behavior of
// dig. The following tags are supported,
//
//   - name: Requests a value with the same name and type from the container. See
//     Named Values for more information.
//
//   - optional: If set to true, indicates that the dependency is optional and
//     the constructor gracefully handles its absence.
//
//   - group: Name of the Value Group from which this field will be filled. The
//     field must be a slice type.
type Required struct{ _ digSentinel }

// Provided is the inverse of Required: it marks a struct as a special return
// struct. Instead of providing the struct itself as a dependency, all its
// fields are provided as dependencies.
//
// It can use name and group tag.
type Provided struct{ _ digSentinel }

// Placeholder type placed in dig.Required/dig.Provided to make their
// special nature obvious in godocs. Otherwise they will appear as
// plain empty structs.
type digSentinel struct{}

var (
	//nolint:unused // TODO use
	_noValue         reflect.Value
	_errType         = reflect.TypeFor[error]()
	_requiredType    = reflect.TypeFor[Required]()
	_requiredPtrType = reflect.TypeFor[*Required]()
	_providedType    = reflect.TypeFor[Provided]()
	_providedPtrType = reflect.TypeFor[*Provided]()
)

var (
	_ func(any) bool = IsRequired
	_ func(any) bool = IsProvided
)

func IsRequired(value any) bool {
	return embedsType(value, _requiredType)
}

func IsProvided(value any) bool {
	return embedsType(value, _providedType)
}
