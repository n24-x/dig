package dig

import "reflect"

type Required struct{ _ digSentinel }

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

func IsRequired(value any) bool {
	return embedsType(value, _requiredType)
}

func IsProvided(value any) bool {
	return embedsType(value, _providedType)
}

var (
	_ func(any) bool = IsRequired
	_ func(any) bool = IsProvided
)
