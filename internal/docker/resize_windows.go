//go:build windows

package docker

import (
	"os"
)

// startResizeListener is a no-op on Windows since SIGWINCH is not available.
func startResizeListener(_ chan<- os.Signal) {
	// SIGWINCH is Unix-only; on Windows, terminal resize is handled differently.
}
