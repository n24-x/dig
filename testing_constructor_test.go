package dig

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConstructorNodeGoodCases(t *testing.T) {
	t.Parallel()

	type t1 struct{}
	type t2 struct{}
	type t3 struct{}

	type requiredParams struct {
		Required

		A *t1
		B *t2 `name:"bee"`
	}

	type optionalParams struct {
		Required

		A *t1 `optional:"true"`
		B *t2
	}

	type groupParams struct {
		Required

		Ts []*t1 `group:"ts"`
	}

	type passiveGroupParams struct {
		Required
		Ts []*t1 `group:"ts,passive"`
	}

	type providedResults struct {
		Provided

		A *t1
		B *t2 `name:"bee"`
	}

	type groupResults struct {
		Provided

		Ts []*t1 `group:"ts"`
	}

	type flattenResults struct {
		Provided

		Ts []*t1 `group:"ts,flatten"`
	}

	kT1 := resourceKey{valueType: reflect.TypeFor[*t1]()}
	kT2 := resourceKey{valueType: reflect.TypeFor[*t2]()}
	kT3 := resourceKey{valueType: reflect.TypeFor[*t3]()}
	kT2Named := resourceKey{valueType: reflect.TypeFor[*t2](), valueName: "bee"}
	kT1Group := resourceKey{valueType: reflect.TypeFor[*t1](), groupName: "ts"}
	kT1SliceGroup := resourceKey{valueType: reflect.TypeFor[[]*t1](), groupName: "ts"}

	tests := []struct {
		name         string
		input        any
		opts         constructorOptions
		wantRequires []resourceKey
		wantProvides []resourceKey
	}{
		{
			name:         "single result",
			input:        func() *t3 { return nil },
			wantProvides: []resourceKey{kT3},
		},
		{
			name:         "two params",
			input:        func(*t1, *t2) *t3 { return nil },
			wantRequires: []resourceKey{kT1, kT2},
			wantProvides: []resourceKey{kT3},
		},
		{
			name:         "error result is not a resource",
			input:        func() (*t1, error) { return nil, nil },
			wantProvides: []resourceKey{kT1},
		},
		{
			name:         "required struct is flattened",
			input:        func(requiredParams) *t3 { return nil },
			wantRequires: []resourceKey{kT1, kT2Named},
			wantProvides: []resourceKey{kT3},
		},
		{
			name:         "optional param is dropped from requires",
			input:        func(optionalParams) *t3 { return nil },
			wantRequires: []resourceKey{kT2},
			wantProvides: []resourceKey{kT3},
		},
		{
			name:         "param group uses the element type",
			input:        func(groupParams) *t3 { return nil },
			wantRequires: []resourceKey{kT1Group},
			wantProvides: []resourceKey{kT3},
		},
		{
			name:         "passive group is dropped from requires",
			input:        func(passiveGroupParams) *t3 { return nil },
			wantProvides: []resourceKey{kT3},
		},
		{
			name:         "provided struct is flattened",
			input:        func() providedResults { return providedResults{} },
			wantProvides: []resourceKey{kT1, kT2Named},
		},
		{
			name:         "result group without flatten keeps the slice type",
			input:        func() groupResults { return groupResults{} },
			wantProvides: []resourceKey{kT1SliceGroup},
		},
		{
			name:         "result group with flatten uses the element type",
			input:        func() flattenResults { return flattenResults{} },
			wantProvides: []resourceKey{kT1Group},
		},
		{
			name:         "As replaces the provided type",
			input:        func() *bytes.Buffer { return nil },
			opts:         constructorOptions{ResultAs: []any{new(io.Reader)}},
			wantProvides: []resourceKey{{valueType: reflect.TypeFor[io.Reader]()}},
		},
		{
			name:         "ResultName names the provided value",
			input:        func() *t1 { return nil },
			opts:         constructorOptions{ResultName: "foo"},
			wantProvides: []resourceKey{{valueType: reflect.TypeFor[*t1](), valueName: "foo"}},
		},
		{
			name:         "ResultGroup puts the provided value in a group",
			input:        func() *t1 { return nil },
			opts:         constructorOptions{ResultGroup: "ts"},
			wantProvides: []resourceKey{kT1Group},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := newConstructorNode(tt.input, tt.opts)
			require.NoError(t, err)

			require.Equal(t, tt.wantRequires, n.Requires(), "unexpected requires")
			require.Equal(t, tt.wantProvides, n.Provides(), "unexpected provides")
		})
	}
}

func TestNewConstructorNodeBadCases(t *testing.T) {
	t.Parallel()

	type t1 struct{}

	type requiredParams struct {
		Required
		A *t1
	}

	type providedResults struct {
		Provided
		A *t1
	}

	type embedsRequiredPtr struct{ *Required }

	type embedsProvidedPtr struct{ *Provided }

	type groupNotSlice struct {
		Required
		A *t1 `group:"a"`
	}

	type groupWithName struct {
		Required
		A []*t1 `group:"a" name:"x"`
	}

	type groupWithOptional struct {
		Required
		A []*t1 `group:"a" optional:"true"`
	}

	type resultWithError struct {
		Provided
		E error
	}

	type flattenNonSlice struct {
		Provided
		A *t1 `group:"a,flatten"`
	}

	type passiveResult struct {
		Provided
		A []*t1 `group:"a,passive"`
	}

	tests := []struct {
		name    string
		input   any
		opts    constructorOptions
		wantErr string
	}{
		{
			name:    "param is a provided struct",
			input:   func(providedResults) *t1 { return nil },
			wantErr: "expected required struct, but got provided struct",
		},
		{
			name:    "param embeds *Required",
			input:   func(embedsRequiredPtr) *t1 { return nil },
			wantErr: "required struct cannot embed *dig.Required",
		},
		{
			name:    "param is a pointer to a required struct",
			input:   func(*requiredParams) *t1 { return nil },
			wantErr: "cannot use a pointer as required struct",
		},
		{
			name:    "param group is not a slice",
			input:   func(groupNotSlice) *t1 { return nil },
			wantErr: "only slice field can use group tag",
		},
		{
			name:    "param group uses a name tag",
			input:   func(groupWithName) *t1 { return nil },
			wantErr: "cannot use name struct tag with group tag",
		},
		{
			name:    "param group uses an optional tag",
			input:   func(groupWithOptional) *t1 { return nil },
			wantErr: "cannot use optional struct tag with group tag",
		},
		{
			name:    "result is a required struct",
			input:   func() requiredParams { return requiredParams{} },
			wantErr: "expected provided struct, but got required struct",
		},
		{
			name:    "result embeds *Provided",
			input:   func() embedsProvidedPtr { return embedsProvidedPtr{} },
			wantErr: "provided struct cannot embed *dig.Provided",
		},
		{
			name:    "result is a pointer to a provided struct",
			input:   func() *providedResults { return nil },
			wantErr: "cannot use pointer as provided struct",
		},
		{
			name:    "result struct has an error field",
			input:   func() resultWithError { return resultWithError{} },
			wantErr: "cannot return an error here",
		},
		{
			name:    "result group flattens a non-slice",
			input:   func() flattenNonSlice { return flattenNonSlice{} },
			wantErr: "only slice field can use group struct tag with flatten",
		},
		{
			name:    "result group is passive",
			input:   func() passiveResult { return passiveResult{} },
			wantErr: "passive was incorrectly used",
		},
		{
			name:    "As target is not implemented",
			input:   func() *bytes.Buffer { return nil },
			opts:    constructorOptions{ResultAs: []any{new(io.Closer)}},
			wantErr: "does not implement",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newConstructorNode(tt.input, tt.opts)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
