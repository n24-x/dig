package dig

import (
	"fmt"
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

var (
	_ interface{ Provides() []resourceType }  = resultList{}
	_ interface{ DotResults() []*dot.Result } = resultList{}
)

var _ func(reflect.Type, resultOptions) (resultList, error) = newResultList

// Provides returns the resourceTypes produced by the constructor.
//
// By the time Provides is called, error returns have already been removed
// from Results (see newResultList), so there is no need to skip anything here.
func (rl resultList) Provides() []resourceType {
	var keys []resourceType
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

func newResultList(ctype reflect.Type, opts resultOptions) (resultList, error) {
	numReturnParameters := ctype.NumOut()
	results := make([]result, 0, numReturnParameters)
	resultIndexes := make([]int, numReturnParameters)

	resultIndex := 0
	for idx := range numReturnParameters {
		typ := ctype.Out(idx)
		if isError(typ) {
			resultIndexes[idx] = -1
			continue // skip error
		}
		r, err := newResult(typ, opts)
		if err != nil {
			return resultList{}, newErrInvalidInput(
				fmt.Sprintf("bad result %d (%v)", idx+1, typ), err)
		}

		results = append(results, r)
		resultIndexes[idx] = resultIndex
		resultIndex++
	}

	return resultList{
		ctype:         ctype,
		Results:       results,
		resultIndexes: resultIndexes,
	}, nil
}
