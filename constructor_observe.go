package dig

import "time"

type ObserveInfo struct {
	Name        string
	Error       error
	ElapsedTime time.Duration
}

type Observer struct {
	// Announce is called before the constructor runs. The name matches
	// the Name field passed to Report. It is Optional.
	Announce func(string)
	Report   func(ObserveInfo)
}

func WithObserver(ob Observer) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.Observer = ob
	}
}
