package dig

import (
	"reflect"
)

// resultList holds all values returned by the constructor as results.
type resultList struct {
	ctype reflect.Type

	Results []result

	// For each item at index i returned by the constructor, resultIndexes[i]
	// is the index in .Results for the corresponding result object.
	// resultIndexes[i] is -1 for errors returned by constructors.
	resultIndexes []int
}

var _ interface{ Provides() []resourceKey } = resultList{}

// var _ interface{ DotResults() []*dot.Result } = resultList{}

func (rl resultList) Provides() []resourceKey {
	var keys []resourceKey
	for _, idx := range rl.resultIndexes {
		if idx == -1 {
			continue
		}
		keys = append(keys, rl.Results[idx].Provides()...)
	}

	return keys
}

// func (rl resultList) DotResults() []*dot.Result { ... }
