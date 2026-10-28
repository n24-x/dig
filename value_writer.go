package dig

import "reflect"

// valueWriter receives the values extracted from a constructor/decorator return.
type valueWriter interface {
	setValue(key resourceType, v reflect.Value)
	setDecoratedValue(key resourceType, v reflect.Value)

	// v is an element of the group.
	submitGroupValue(key resourceType, v reflect.Value)
	// slice represents the entire group
	submitDecoratedGroup(key resourceType, slice reflect.Value)
}

type stagingValues struct {
	values          map[resourceType]reflect.Value
	decoratedValues map[resourceType]reflect.Value

	groups          map[resourceType][]reflect.Value
	decoratedGroups map[resourceType]reflect.Value
}

var (
	_ valueWriter                        = (*stagingValues)(nil)
	_ interface{ Commit(w valueWriter) } = (*stagingValues)(nil)
	_ func() *stagingValues              = newStagingValues
)

func newStagingValues() *stagingValues {
	return &stagingValues{
		values:          make(map[resourceType]reflect.Value),
		decoratedValues: make(map[resourceType]reflect.Value),
		groups:          make(map[resourceType][]reflect.Value),
		decoratedGroups: make(map[resourceType]reflect.Value),
	}
}

func (s *stagingValues) setValue(key resourceType, v reflect.Value) {
	s.values[key] = v
}

func (s *stagingValues) setDecoratedValue(key resourceType, v reflect.Value) {
	s.decoratedValues[key] = v
}

func (s *stagingValues) submitGroupValue(key resourceType, v reflect.Value) {
	s.groups[key] = append(s.groups[key], v)
}

func (s *stagingValues) submitDecoratedGroup(key resourceType, slice reflect.Value) {
	s.decoratedGroups[key] = slice
}

// Commit applies the staged writes to w. Group members append; everything
// else overwrites.
func (s *stagingValues) Commit(w valueWriter) {
	for k, v := range s.values {
		w.setValue(k, v)
	}
	for k, v := range s.decoratedValues {
		w.setDecoratedValue(k, v)
	}
	for k, vs := range s.groups {
		for _, v := range vs {
			w.submitGroupValue(k, v)
		}
	}
	for k, slice := range s.decoratedGroups {
		w.submitDecoratedGroup(k, slice)
	}
}
