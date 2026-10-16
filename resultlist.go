package dig

import (
	"reflect"

	"github.com/n24-x/dig/internal/dot"
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

//var _ interface{ DotResults() []*dot.Result } = resultList{}

// Provides returns the resourceKeys produced by the constructor.
//
// By the time Provides is called, error returns have already been removed
// from Results (see newResultList), so there is no need to skip anything here.
func (rl resultList) Provides() []resourceKey {
	var keys []resourceKey
	for _, r := range rl.Results {
		keys = append(keys, r.Provides()...)
	}

	return keys
}

func (rl resultList) DotResults() []*dot.Result {
	var dotResults []*dot.Result
	for _, r := range rl.Results {
		dotResults = append(dotResults, r.DotResults()...)
	}
	return dotResults
}
