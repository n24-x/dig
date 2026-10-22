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

	kTyp1 := resourceType{valueType: reflect.TypeFor[*typ1]()}
	kTyp2 := resourceType{valueType: reflect.TypeFor[*typ2]()}
	kTyp3 := resourceType{valueType: reflect.TypeFor[*typ3]()}
	kTyp2Named := resourceType{valueType: reflect.TypeFor[*typ2](), valueName: "lee"}
	kTyp1Group := resourceType{valueType: reflect.TypeFor[*typ1](), groupName: "rain"}
	kTyp1Slice := resourceType{valueType: reflect.TypeFor[[]*typ1]()}
	kInt := resourceType{valueType: reflect.TypeFor[int]()}

	tests := []struct {
		name         string
		input        any
		opts         constructorOptions
		wantRequires []resourceType
		wantProvides []resourceType
	}{
		{
			name:         "single result",
			input:        func() *typ3 { return nil },
			wantProvides: []resourceType{kTyp3},
		},
		{
			name:         "two params",
			input:        func(*typ1, *typ2) *typ3 { return nil },
			wantRequires: []resourceType{kTyp1, kTyp2},
			wantProvides: []resourceType{kTyp3},
		},
		{
			name:         "error result is not a resource",
			input:        func() (*typ1, error) { return nil, nil },
			wantProvides: []resourceType{kTyp1},
		},
		{
			name:         "requiredParamsGoodData1: parameter-struct is flattened",
			input:        func(requiredParamsGoodData1) *typ3 { return nil },
			wantRequires: []resourceType{kTyp1, kTyp2Named},
			wantProvides: []resourceType{kTyp3},
		},
		{
			name:         "optionalParamsGoodData2: optional param is dropped from requires",
			input:        func(optionalParamsGoodData2) *typ3 { return nil },
			wantRequires: []resourceType{kTyp2},
			wantProvides: []resourceType{kTyp3},
		},
		{
			name:         "groupParamsGoodData3: param group uses the element type",
			input:        func(groupParamsGoodData3) *typ3 { return nil },
			wantRequires: []resourceType{kTyp1Group},
			wantProvides: []resourceType{kTyp3},
		},
		{
			name:         "passiveGroupParamsGoodData4: passive group is dropped from requires",
			input:        func(passiveGroupParamsGoodData4) *typ3 { return nil },
			wantRequires: []resourceType{kInt},
			wantProvides: []resourceType{kTyp3},
		},
		{
			name:         "ignoreUnexportedParamsGoodData5: unexported field is skipped",
			input:        func(ignoreUnexportedParamsGoodData5) *typ3 { return nil },
			wantRequires: []resourceType{kTyp1Slice},
			wantProvides: []resourceType{kTyp3},
		},
		{
			name:         "providedResultsGoodData7: return-struct is flattened",
			input:        func() providedResultsGoodData7 { return providedResultsGoodData7{} },
			wantProvides: []resourceType{kTyp1, kTyp2Named},
		},
		{
			name:         "groupResultsGoodData8: result group keeps the value type",
			input:        func() groupResultsGoodData8 { return groupResultsGoodData8{} },
			wantProvides: []resourceType{kTyp1Group},
		},
		{
			name:         "flattenResultsGoodData9: result group with flatten uses the element type",
			input:        func() flattenResultsGoodData9 { return flattenResultsGoodData9{} },
			wantProvides: []resourceType{kTyp1Group},
		},
		{
			name:         "duplicateGroupResultsGoodData10: same type may join the same group twice",
			input:        func() duplicateGroupResultsGoodData10 { return duplicateGroupResultsGoodData10{} },
			wantProvides: []resourceType{kTyp1Group, kTyp1Group},
		},
		{
			name:  "mixedGroupResultsGoodData11: the value type is part of the group key",
			input: func() mixedGroupResultsGoodData11 { return mixedGroupResultsGoodData11{} },
			wantProvides: []resourceType{
				{valueType: reflect.TypeFor[*typ1](), groupName: "rain"},
				{valueType: reflect.TypeFor[*typ2](), groupName: "rain"},
			},
		},
		{
			name:         "As replaces the provided type",
			input:        func() *bytes.Buffer { return nil },
			opts:         constructorOptions{Result: resultOptions{As: []any{new(io.Reader)}}},
			wantProvides: []resourceType{{valueType: reflect.TypeFor[io.Reader]()}},
		},
		{
			name:         "ValueName names the provided value",
			input:        func() *typ1 { return nil },
			opts:         constructorOptions{Result: resultOptions{ValueName: "foo"}},
			wantProvides: []resourceType{{valueType: reflect.TypeFor[*typ1](), valueName: "foo"}},
		},
		{
			name:         "GroupName puts the provided value in a group",
			input:        func() *typ1 { return nil },
			opts:         constructorOptions{Result: resultOptions{GroupName: "rain"}},
			wantProvides: []resourceType{kTyp1Group},
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
			name:    "param is a pointer to a parameter-struct",
			input:   func(*requiredParamsGoodData1) *typ3 { return nil },
			wantErr: "cannot use a pointer as parameter-struct",
		},
		{
			name:    "result is a pointer to a return-struct",
			input:   func() *providedResultsGoodData7 { return nil },
			wantErr: "cannot use pointer as return-struct",
		},
		{
			name:    "paramsBadData1: optional value is not a valid bool",
			input:   func(paramsBadData1) *typ3 { return nil },
			wantErr: `invalid value "janus" for "optional" tag on field A`,
		},
		{
			name:    "resultsBadData2: optional is not allowed",
			input:   func() resultsBadData2 { return resultsBadData2{} },
			wantErr: "cannot use optional-tag in return-struct",
		},
		{
			name:    "paramsBadData3: group can only be used on slices",
			input:   func(paramsBadData3) *typ3 { return nil },
			wantErr: "only slice field can use group-tag",
		},
		{
			name:    "paramsBadData4: group conflicts with name",
			input:   func(paramsBadData4) *typ3 { return nil },
			wantErr: "cannot use name-tag with group-tag",
		},
		{
			name:    "resultsBadData5: group conflicts with name/optional",
			input:   func() resultsBadData5 { return resultsBadData5{} },
			wantErr: "cannot use optional-tag in return-struct",
		},
		{
			name:    "resultsBadData6: error is not a valid result field",
			input:   func() resultsBadData6 { return resultsBadData6{} },
			wantErr: "cannot handle error here",
		},
		{
			name:    "paramsBadData7: unexported field",
			input:   func(paramsBadData7) *typ3 { return nil },
			wantErr: "unexported fields not allowed in dig.Required",
		},
		{
			name:    "resultsBadData8: unexported field",
			input:   func() resultsBadData8 { return resultsBadData8{} },
			wantErr: "unexported fields not allowed in return-struct",
		},
		{
			name:    "paramsBadData9: embeds *Required",
			input:   func(paramsBadData9) *typ3 { return nil },
			wantErr: "parameter-struct cannot embed *dig.Required",
		},
		{
			name:    "resultsBadData10: embeds *Provided",
			input:   func() resultsBadData10 { return resultsBadData10{} },
			wantErr: "return-struct cannot embed *dig.Provided",
		},
		{
			name:    "paramsBadData11: a return-struct used as a param",
			input:   func(paramsBadData11) *typ3 { return nil },
			wantErr: "expected parameter-struct, but got return-struct",
		},
		{
			name:    "resultsBadData12: a parameter-struct used as a result",
			input:   func() resultsBadData12 { return resultsBadData12{} },
			wantErr: "expected return-struct, but got parameter-struct",
		},
		{
			name:    "paramsBadData13: flatten is not allowed in parameter-struct",
			input:   func(paramsBadData13) *typ3 { return nil },
			wantErr: "cannot use group-tag with flatten in parameter-struct",
		},
		{
			name:    "paramsBadData14: error is not a valid dependency",
			input:   func(paramsBadData14) *typ3 { return nil },
			wantErr: "cannot handle error here",
		},
		{
			name:    "paramsBadData15: error nested in a parameter-struct field",
			input:   func(paramsBadData15) *typ3 { return nil },
			wantErr: `bad field "Inner" of dig.paramsBadData15: bad field "ErrorType" of dig.paramsBadData15Inner: cannot handle error here`,
		},
		{
			name:    "resultsBadData16: error nested in a return-struct field",
			input:   func() resultsBadData16 { return resultsBadData16{} },
			wantErr: `bad field "Inner" of dig.resultsBadData16: bad field "ErrorType" of dig.resultsBadData16Inner: cannot handle error here`,
		},
		{
			name:    "GroupName with passive is rejected",
			input:   func() *typ1 { return nil },
			opts:    constructorOptions{Result: resultOptions{GroupName: "rain,passive"}},
			wantErr: "passive was incorrectly used",
		},
		{
			name:    "As target is not implemented",
			input:   func() *bytes.Buffer { return nil },
			opts:    constructorOptions{Result: resultOptions{As: []any{new(io.Closer)}}},
			wantErr: "does not implement",
		},
		{
			name:    "no results at all",
			input:   func() {},
			wantErr: "must provide at least one non-error type",
		},
		{
			name:    "only an error result",
			input:   func() error { return nil },
			wantErr: "must provide at least one non-error type",
		},
		{
			name:    "only error results",
			input:   func() (error, error) { return nil, nil },
			wantErr: "must provide at least one non-error type",
		},
		{
			name:    "emptyResultsBadData17: return-struct without fields",
			input:   func() emptyResultsBadData17 { return emptyResultsBadData17{} },
			wantErr: "must provide at least one non-error type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newConstructorNode(tt.input, tt.opts)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
