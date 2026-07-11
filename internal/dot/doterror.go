// Copyright (c) 2019 Uber Technologies, Inc.
// Copyright (c) 2026 k2 <skrik2@outlook.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dot

// ErrorType describes the failure state of a constructor or value group.
type ErrorType int

const (
	// noError indicates that the node was built successfully.
	noError ErrorType = iota

	// rootCause indicates that the node itself failed, for example because
	// a constructor returned an error or a required dependency was missing.
	rootCause

	// transitiveFailure indicates that the node did not fail itself, but could
	// not be built because one of its dependencies failed.
	transitiveFailure
)

// Color returns the display color associated with an ErrorType.
func (s ErrorType) Color() string {
	switch s {
	case rootCause:
		return "red"
	case transitiveFailure:
		return "orange"
	default:
		return "black"
	}
}
