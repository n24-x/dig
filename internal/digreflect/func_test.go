package digreflect

import (
	"fmt"
	"strings"
	"testing"

	myrepository "github.com/n24-x/dig/internal/digreflect/tests/myrepository.git"
	mypackage "github.com/n24-x/dig/internal/digreflect/tests/myrepository.git/mypackage"

	"github.com/stretchr/testify/assert"
)

func SomeExportedFunction() {}

func unexportedFunction() {}

func nestedFunctions() (nested1, nested2, nested3 func()) {
	// we call the functions to satisfy the linter.
	nested1 = func() {}
	nested2 = func() {
		nested3 = func() {}
	}
	nested2() // set nested3
	return
}

func TestInspectFunc(t *testing.T) {
	t.Parallel()

	nested1, nested2, nested3 := nestedFunctions()

	tests := []struct {
		name        string
		input       any
		wantName    string
		wantPackage string

		// We don't match the exact file name because $GOPATH can be anywhere
		// on someone's system. Instead we'll match the suffix.
		wantFileSuffix string
	}{
		{
			name:           "exported function",
			input:          SomeExportedFunction,
			wantName:       "SomeExportedFunction",
			wantPackage:    "github.com/n24-x/dig/internal/digreflect",
			wantFileSuffix: "/internal/digreflect/func_test.go",
		},
		{
			name:           "unexported function",
			input:          unexportedFunction,
			wantName:       "unexportedFunction",
			wantPackage:    "github.com/n24-x/dig/internal/digreflect",
			wantFileSuffix: "/internal/digreflect/func_test.go",
		},
		{
			name:           "nested function",
			input:          nested1,
			wantName:       "nestedFunctions.func1",
			wantPackage:    "github.com/n24-x/dig/internal/digreflect",
			wantFileSuffix: "/internal/digreflect/func_test.go",
		},
		{
			name:           "second nested function",
			input:          nested2,
			wantName:       "nestedFunctions.func2",
			wantPackage:    "github.com/n24-x/dig/internal/digreflect",
			wantFileSuffix: "/internal/digreflect/func_test.go",
		},
		{
			name:           "nested inside a nested function",
			input:          nested3,
			wantName:       "nestedFunctions.func2.1",
			wantPackage:    "github.com/n24-x/dig/internal/digreflect",
			wantFileSuffix: "/internal/digreflect/func_test.go",
		},
		{
			name:           "inside a .git package",
			input:          myrepository.Hello,
			wantName:       "Hello",
			wantPackage:    "github.com/n24-x/dig/internal/digreflect/tests/myrepository.git",
			wantFileSuffix: "/internal/digreflect/tests/myrepository.git/hello.go",
		},
		{
			name:           "subpackage of a .git package",
			input:          mypackage.Add,
			wantName:       "Add",
			wantPackage:    "github.com/n24-x/dig/internal/digreflect/tests/myrepository.git/mypackage",
			wantFileSuffix: "/internal/digreflect/tests/myrepository.git/mypackage/add.go",
		},
		{
			name:           "dependency",
			input:          assert.Contains,
			wantName:       "Contains",
			wantPackage:    "github.com/stretchr/testify/assert",
			wantFileSuffix: "/assert/assertions.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := InspectFunc(tt.input)
			assert.Equal(t, tt.wantName, f.Name, "function name did not match")
			assert.Equal(t, tt.wantPackage, f.Package, "package name did not match")

			assert.True(t, strings.HasSuffix(f.File, tt.wantFileSuffix),
				"file path %q does not end with src/%v", f.File, tt.wantFileSuffix)
		})
	}
}

func TestSplitFunc(t *testing.T) {
	t.Parallel()

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()

		pname, fname := splitFuncName("")
		assert.Empty(t, pname, "package name must be empty")
		assert.Empty(t, fname, "function name must be empty")
	})

	t.Run("vendored dependency", func(t *testing.T) {
		t.Parallel()

		pname, fname := splitFuncName("github.com/n24-x/dig/vendor/example.com/foo/bar.Baz")
		assert.Equal(t, "example.com/foo/bar", pname)
		assert.Equal(t, "Baz", fname)
	})
}

func TestFuncFormatting(t *testing.T) {
	t.Parallel()

	f := Func{
		Package: "foo/bar/baz",
		Name:    "Qux",
		File:    "src/foo/bar/baz/qux.go",
		Line:    42,
	}

	assert.Equal(t,
		`"foo/bar/baz".Qux (src/foo/bar/baz/qux.go:42)`,
		f.String(), "%v did not match")

	assert.Equal(t, `"foo/bar/baz".Qux
	src/foo/bar/baz/qux.go:42`, fmt.Sprintf("%+v", &f), "%v did not match")
}
