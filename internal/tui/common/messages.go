package common

// ErrMsg wraps errors from async operations
type ErrMsg struct{ Err error }

func (e ErrMsg) Error() string { return e.Err.Error() }

// WarnMsg carries non-fatal warnings
type WarnMsg struct{ Warnings []string }
