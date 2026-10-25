package dig

import "time"

// ObserveInfo is what an [Observer.Report] gets once a constructor has run.
type ObserveInfo struct {
	// Name identifies the constructor in the form <package>.<function>,
	// the same string [Observer.Announce] received.
	Name string

	// Error is what the run reported: the constructor's own error wrapped in
	// [errConstructorFailed], or a [PanicError] if it panicked under
	// RecoverFromPanics. It is nil on success.
	//
	// Without RecoverFromPanics a panic never reaches the return that assigns
	// this field, so Error is nil even though the run did not succeed.
	Error error

	// ElapsedTime is how long the constructor took to run.
	ElapsedTime time.Duration
}

// Observer watches one constructor: Announce when it starts, Report when it
// finishes. Either field may be nil.
//
// Observers see constructors only. The function passed to [Scope.Setup] runs
// without announcing or reporting; use its error return to learn how that went.
type Observer struct {
	// Announce is called just before the constructor runs, with the name
	// [ObserveInfo.Name] will carry. It is optional.
	//
	// It exists for the case the run never finishes - the constructor blocks,
	// or the process dies - where a Report will never arrive.
	Announce func(string)

	// Report is called after the constructor runs, including when it returns
	// an error or panics. It is optional.
	Report func(ObserveInfo)
}

var _ func(Observer) Option[*ResourceOptions] = WithObserver

// WithObserver makes the container report what happens to the constructor
// passed to [Scope.Resource].
//
// Both halves are registered at once because they describe the same run.
// Passing a zero Observer is legal and observes nothing.
func WithObserver(ob Observer) Option[*ResourceOptions] {
	return func(ro *ResourceOptions) {
		if ro == nil {
			return
		}
		ro.observer = ob
	}
}
