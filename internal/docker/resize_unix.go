//go:build !windows

package docker

import (
	"os"
	"os/signal"
	"syscall"
)

// startResizeListener registers SIGWINCH signal notifications on Unix systems.
func startResizeListener(sigCh chan<- os.Signal) {
	signal.Notify(sigCh, syscall.SIGWINCH)
}
