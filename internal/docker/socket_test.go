package docker

import (
	"strings"
	"testing"
)

func TestDetectDockerSocket(t *testing.T) {
	// This test just ensures the function doesn't panic
	socket := detectDockerSocket()
	// Socket should be empty or start with unix:// or npipe://
	if socket != "" && !strings.HasPrefix(socket, "unix://") && !strings.HasPrefix(socket, "npipe://") {
		t.Errorf("detectDockerSocket() = %q, want empty or valid socket path", socket)
	}
}
