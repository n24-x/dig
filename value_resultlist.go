package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// resultList holds all values returned by the constructor as results.
type resultList struct {
	ctype reflect.Type

	results []result

	// For each item at index i returned by the constructor, resultIndexes[i]
	// is the index in .results for the corresponding result object.
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
// from results (see newResultList), so there is no need to skip anything here.
func (rl resultList) Provides() []resourceType {
	var keys []resourceType
	for _, r := range rl.results {
		keys = append(keys, r.Provides()...)
	}

	return keys
}

func (rl resultList) DotResults() []*dot.Result {
	var dotResults []*dot.Result
	for _, r := range rl.results {
		dotResults = append(dotResults, r.DotResults()...)
	}
	return dotResults
}

func newResultList(ctype reflect.Type, opts resultOptions) (resultList, error) {
	numReturnResults := ctype.NumOut()
	results := make([]result, 0, numReturnResults)
	resultIndexes := make([]int, numReturnResults)

	resultIndex := 0
	for idx := range numReturnResults {
		typ := ctype.Out(idx)
		if isErrorType(typ) {
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
		results:       results,
		resultIndexes: resultIndexes,
	}, nil
}

func (rl resultList) ExtractList(w valueWriter, values []reflect.Value) error {
	for i, v := range values {
		if idx := rl.resultIndexes[i]; idx >= 0 {
			rl.results[idx].Extract(w, v)
			continue
		}

		// ctor error
		if err, _ := reflect.TypeAssert[error](v); err != nil {
			return err
		}
	}
	return nil
}
