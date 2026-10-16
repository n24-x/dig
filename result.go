package dig

import (
	"github.com/n24-x/dig/internal/dot"
)

// The result interface represents a result produced by a constructor.
type result interface {
	Provides() []resourceKey

	// TODO
	// Extract(containerWriter, bool, reflect.Value)

	DotResult() []*dot.Result
}
