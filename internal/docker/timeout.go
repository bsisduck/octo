package docker

import "time"

const (
	TimeoutPing       = 5 * time.Second
	TimeoutList       = 30 * time.Second
	TimeoutDiskUsage  = 60 * time.Second
	TimeoutRemove     = 30 * time.Second
	TimeoutAction     = 10 * time.Second
	TimeoutPrune      = 120 * time.Second
	TimeoutWatch      = 15 * time.Second
	TimeoutLogs       = 30 * time.Second
	TimeoutStats      = 10 * time.Second
	TimeoutCommand    = 5 * time.Minute
	TimeoutExecCreate = 10 * time.Second // For exec create/attach setup
	// NOTE: No timeout for the exec session itself -- it is interactive with no predictable duration
)
