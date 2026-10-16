package digreflect

import (
	"fmt"
	"reflect"
	"runtime"
)

// Func contains runtime information about a function.
type Func struct {
	// Name of the function.
	Name string

	// Name of the package in which this function is defined.
	Package string

	// Path to the file in which this function is defined.
	File string

	// Line number in the file at which this function is defined.
	Line int
}

// String returns a string representation of the function.
func (f *Func) String() string {
	return fmt.Sprint(f)
}

// Format implements fmt.Formatter for Func, printing a single-line
// representation for %v and a multi-line one for %+v.
func (f *Func) Format(w fmt.State, c rune) {
	if w.Flag('+') && c == 'v' {
		// "path/to/package".MyFunction
		// 	path/to/file.go:42
		fmt.Fprintf(w, "%q.%v", f.Package, f.Name)
		fmt.Fprintf(w, "\n\t%v:%v", f.File, f.Line)
	} else {
		// "path/to/package".MyFunction (path/to/file.go:42)
		fmt.Fprintf(
			w,
			"%q.%v (%v:%v)",
			f.Package,
			f.Name,
			f.File,
			f.Line,
		)
	}
}

// InspectFunc inspects and returns runtime information about the given
// function.
func InspectFunc(function any) *Func {
	fptr := reflect.ValueOf(function).Pointer()
	return InspectFuncPC(fptr)
}

// InspectFuncPC inspects and returns runtime information about the function
// at the given program counter address.
func InspectFuncPC(pc uintptr) *Func {
	f := runtime.FuncForPC(pc)
	if f == nil {
		return nil
	}

	pkgName, funcName := splitFuncName(f.Name())
	fileSource, lineNum := f.FileLine(pc)

	return &Func{
		Name:    funcName,
		Package: pkgName,
		File:    fileSource,
		Line:    lineNum,
	}
}
