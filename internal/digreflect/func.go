package digreflect

import (
	"fmt"
	"io"
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

// String returns a single-line representation of the function, including its
// file and line number. It is equivalent to fmt's %v.
//
// Use %s for the compact package.name identifier without quoting or position.
func (f *Func) String() string {
	return fmt.Sprint(f)
}

// Dotted returns the package-qualified function name, <package>.<function>,
// with no quoting and no position. It is what the %s verb prints.
func (f *Func) Dotted() string {
	return f.Package + "." + f.Name
}

// Format implements fmt.Formatter for Func, so a function can be rendered in
// three ways:
//
//   - %s: compact, unquoted package.name, without the file and line.
//   - %v: package.name followed by the file and line in parentheses.
//   - %+v: package.name on one line, file and line indented on the next.
func (f *Func) Format(w fmt.State, c rune) {
	if c == 's' {
		// package.name
		_, _ = io.WriteString(w, f.Dotted())
	} else if w.Flag('+') && c == 'v' {
		// "path/to/package".MyFunction
		// 	path/to/file.go:42
		_, _ = fmt.Fprintf(w, "%q.%v", f.Package, f.Name)
		_, _ = fmt.Fprintf(w, "\n\t%v:%v", f.File, f.Line)
	} else {
		// "path/to/package".MyFunction (path/to/file.go:42)
		_, _ = fmt.Fprintf(
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
