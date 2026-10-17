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

	kTyp1 := resourceKey{valueType: reflect.TypeFor[*typ1]()}
	kTyp2 := resourceKey{valueType: reflect.TypeFor[*typ2]()}
	kTyp3 := resourceKey{valueType: reflect.TypeFor[*typ3]()}
	kTyp2Named := resourceKey{valueType: reflect.TypeFor[*typ2](), valueName: "lee"}
	kTyp1Group := resourceKey{valueType: reflect.TypeFor[*typ1](), groupName: "rain"}
	kTyp1Slice := resourceKey{valueType: reflect.TypeFor[[]*typ1]()}
	kInt := resourceKey{valueType: reflect.TypeFor[int]()}
	kError := resourceKey{valueType: reflect.TypeFor[error]()}

	tests := []struct {
		name         string
		input        any
		opts         constructorOptions
		wantRequires []resourceKey
		wantProvides []resourceKey
	}{
		{
			name:         "single result",
			input:        func() *typ3 { return nil },
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "two params",
			input:        func(*typ1, *typ2) *typ3 { return nil },
			wantRequires: []resourceKey{kTyp1, kTyp2},
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "error result is not a resource",
			input:        func() (*typ1, error) { return nil, nil },
			wantProvides: []resourceKey{kTyp1},
		},
		{
			name:         "requiredParamsGoodData1: required struct is flattened",
			input:        func(requiredParamsGoodData1) *typ3 { return nil },
			wantRequires: []resourceKey{kTyp1, kTyp2Named},
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "optionalParamsGoodData2: optional param is dropped from requires",
			input:        func(optionalParamsGoodData2) *typ3 { return nil },
			wantRequires: []resourceKey{kTyp2},
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "groupParamsGoodData3: param group uses the element type",
			input:        func(groupParamsGoodData3) *typ3 { return nil },
			wantRequires: []resourceKey{kTyp1Group},
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "passiveGroupParamsGoodData4: passive group is dropped from requires",
			input:        func(passiveGroupParamsGoodData4) *typ3 { return nil },
			wantRequires: []resourceKey{kInt},
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "ignoreUnexportedParamsGoodData5: unexported field is skipped",
			input:        func(ignoreUnexportedParamsGoodData5) *typ3 { return nil },
			wantRequires: []resourceKey{kTyp1Slice},
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "paramsGoodData6: error field is a dependency",
			input:        func(paramsGoodData6) *typ3 { return nil },
			wantRequires: []resourceKey{kTyp1, kError},
			wantProvides: []resourceKey{kTyp3},
		},
		{
			name:         "providedResultsGoodData7: provided struct is flattened",
			input:        func() providedResultsGoodData7 { return providedResultsGoodData7{} },
			wantProvides: []resourceKey{kTyp1, kTyp2Named},
		},
		{
			name:         "groupResultsGoodData8: result group keeps the value type",
			input:        func() groupResultsGoodData8 { return groupResultsGoodData8{} },
			wantProvides: []resourceKey{kTyp1Group},
		},
		{
			name:         "flattenResultsGoodData9: result group with flatten uses the element type",
			input:        func() flattenResultsGoodData9 { return flattenResultsGoodData9{} },
			wantProvides: []resourceKey{kTyp1Group},
		},
		{
			name:         "As replaces the provided type",
			input:        func() *bytes.Buffer { return nil },
			opts:         constructorOptions{ResultAs: []any{new(io.Reader)}},
			wantProvides: []resourceKey{{valueType: reflect.TypeFor[io.Reader]()}},
		},
		{
			name:         "ResultName names the provided value",
			input:        func() *typ1 { return nil },
			opts:         constructorOptions{ResultName: "foo"},
			wantProvides: []resourceKey{{valueType: reflect.TypeFor[*typ1](), valueName: "foo"}},
		},
		{
			name:         "ResultGroup puts the provided value in a group",
			input:        func() *typ1 { return nil },
			opts:         constructorOptions{ResultGroup: "rain"},
			wantProvides: []resourceKey{kTyp1Group},
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

	tests := []struct {
		name    string
		input   any
		opts    constructorOptions
		wantErr string
	}{
		{
			name:    "param is a pointer to a required struct",
			input:   func(*requiredParamsGoodData1) *typ3 { return nil },
			wantErr: "cannot use a pointer as required struct",
		},
		{
			name:    "result is a pointer to a provided struct",
			input:   func() *providedResultsGoodData7 { return nil },
			wantErr: "cannot use pointer as provided struct",
		},
		{
			name:    "paramsBadData1: optional value is not a valid bool",
			input:   func(paramsBadData1) *typ3 { return nil },
			wantErr: `invalid value "janus" for "optional" tag on field A`,
		},
		{
			name:    "resultsBadData2: optional is not allowed",
			input:   func() resultsBadData2 { return resultsBadData2{} },
			wantErr: "cannot use optional struct tag in provided struct",
		},
		{
			name:    "paramsBadData3: group can only be used on slices",
			input:   func(paramsBadData3) *typ3 { return nil },
			wantErr: "only slice field can use group tag",
		},
		{
			name:    "paramsBadData4: group conflicts with name",
			input:   func(paramsBadData4) *typ3 { return nil },
			wantErr: "cannot use name struct tag with group tag",
		},
		{
			name:    "resultsBadData5: group conflicts with name/optional",
			input:   func() resultsBadData5 { return resultsBadData5{} },
			wantErr: "cannot use optional struct tag in provided struct",
		},
		{
			name:    "resultsBadData6: error is not a valid result field",
			input:   func() resultsBadData6 { return resultsBadData6{} },
			wantErr: "cannot return an error here",
		},
		{
			name:    "paramsBadData7: unexported field",
			input:   func(paramsBadData7) *typ3 { return nil },
			wantErr: "unexported fields not allowed in dig.Required",
		},
		{
			name:    "resultsBadData8: unexported field",
			input:   func() resultsBadData8 { return resultsBadData8{} },
			wantErr: "unexported fields not allowed in provided struct",
		},
		{
			name:    "paramsBadData9: embeds *Required",
			input:   func(paramsBadData9) *typ3 { return nil },
			wantErr: "required struct cannot embed *dig.Required",
		},
		{
			name:    "resultsBadData10: embeds *Provided",
			input:   func() resultsBadData10 { return resultsBadData10{} },
			wantErr: "provided struct cannot embed *dig.Provided",
		},
		{
			name:    "paramsBadData11: a Provided struct used as a param",
			input:   func(paramsBadData11) *typ3 { return nil },
			wantErr: "expected required struct, but got provided struct",
		},
		{
			name:    "resultsBadData12: a Required struct used as a result",
			input:   func() resultsBadData12 { return resultsBadData12{} },
			wantErr: "expected provided struct, but got required struct",
		},
		{
			name:    "ResultGroup with passive is rejected",
			input:   func() *typ1 { return nil },
			opts:    constructorOptions{ResultGroup: "rain,passive"},
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
