package dot

// ErrorType describes the failure state of a constructor or value group.
type ErrorType int

const (
	// noError indicates that the node was built successfully.
	noError ErrorType = iota

	// rootCauseError indicates that the node itself failed, for example because
	// a constructor returned an error or a required dependency was missing.
	rootCauseError

	// transitiveFailureError indicates that the node did not fail itself, but could
	// not be built because one of its dependencies failed.
	transitiveFailureError
)

// Color returns the display color associated with an ErrorType.
func (s ErrorType) Color() string {
	switch s {
	case rootCauseError:
		return "red"
	case transitiveFailureError:
		return "orange"
	case noError:
		return "black"
	default: // make compiler happy
		return "black"
	}
}
