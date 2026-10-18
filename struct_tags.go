package dig

import (
	"fmt"
	"reflect"
	"strconv"
)

const (
	_optionalTag         = "optional"
	_nameTag             = "name"
	_ignoreUnexportedTag = "ignore-unexported"
	_groupTag            = "group"
)

// Checks if ignoring unexported files in a dig.Required field is allowed.
//
// The struct field MUST be an _requiredType.
func isIgnoreUnexportedSet(f reflect.StructField) (bool, error) {
	tag := f.Tag.Get(_ignoreUnexportedTag)
	if tag == "" {
		return false, nil
	}

	allowed, err := strconv.ParseBool(tag)
	if err != nil {
		err = newErrInvalidInput(
			fmt.Sprintf("invalid value %q for %q tag on field %v", tag, _ignoreUnexportedTag, f.Name), err)
	}

	return allowed, err
}

// Checks if a field of a required struct is optional.
func isFieldOptional(f reflect.StructField) (bool, error) {
	tag := f.Tag.Get(_optionalTag)
	if tag == "" {
		return false, nil
	}

	optional, err := strconv.ParseBool(tag)
	if err != nil {
		err = newErrInvalidInput(
			fmt.Sprintf(
				"invalid value %q for %q tag on field %v",
				tag,
				_optionalTag,
				f.Name,
			),
			err,
		)
	}

	return optional, err
}

var (
	_ func(reflect.StructField) (bool, error) = isIgnoreUnexportedSet
	_ func(reflect.StructField) (bool, error) = isFieldOptional
)
