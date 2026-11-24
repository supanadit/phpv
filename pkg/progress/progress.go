// Package progress defines the callback used to report build progress.
package progress

// Func receives human-readable status updates during long-running work.
// A nil Func disables reporting.
type Func func(stage, message string)
