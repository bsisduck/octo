package docker

import "time"

const (
	TimeoutPing        = 5 * time.Second
	TimeoutList        = 30 * time.Second
	TimeoutDiskUsage   = 60 * time.Second
	TimeoutRemove      = 30 * time.Second
	TimeoutAction      = 10 * time.Second
	TimeoutPrune       = 120 * time.Second
	TimeoutWatch       = 15 * time.Second
	TimeoutCommand     = 5 * time.Minute
)
