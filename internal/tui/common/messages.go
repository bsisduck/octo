package common

// ErrMsg wraps errors from async operations
type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

// WarnMsg carries non-fatal warnings
type WarnMsg struct{ Warnings []string }

// ClipboardMsg reports the result of an async clipboard copy operation
type ClipboardMsg struct {
	Success bool
	Text    string // what was copied (for status display)
	Err     error  // nil on success
}

// ClearStatusMsg signals that the status message should be cleared
type ClearStatusMsg struct{}
