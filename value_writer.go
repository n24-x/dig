package dig

import "reflect"

// valueWriter receives the values extracted from a constructor's return.
type valueWriter interface {
	setValue(key resourceType, v reflect.Value)
	submitGroup(key resourceType, v reflect.Value)
}

type stagingValues struct {
	values map[resourceType]reflect.Value
	groups map[resourceType][]reflect.Value
}

func newStagingValues() *stagingValues {
	return &stagingValues{
		values: make(map[resourceType]reflect.Value),
		groups: make(map[resourceType][]reflect.Value),
	}
}

func (s *stagingValues) setValue(key resourceType, v reflect.Value) {
	s.values[key] = v
}

func (s *stagingValues) submitGroup(key resourceType, v reflect.Value) {
	s.groups[key] = append(s.groups[key], v)
}

// Commit applies the staged writes to w. Groups append; values overwrite.
func (s *stagingValues) Commit(w valueWriter) {
	for k, v := range s.values {
		w.setValue(k, v)
	}
	for k, vs := range s.groups {
		for _, v := range vs {
			w.submitGroup(k, v)
		}
	}
}
