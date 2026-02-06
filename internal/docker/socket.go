package docker

import (
	"os"
	"path/filepath"
	"runtime"
)

// detectDockerSocket returns the Docker socket path based on platform
func detectDockerSocket() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: Check Docker Desktop locations
		paths := []string{
			filepath.Join(os.Getenv("HOME"), ".docker/run/docker.sock"),
			filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/docker.sock"),
			"/var/run/docker.sock",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return "unix://" + p
			}
		}
	case "linux":
		// Linux: Standard locations
		paths := []string{
			"/var/run/docker.sock",
			"/run/docker.sock",
			filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "docker.sock"), // Rootless
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return "unix://" + p
			}
		}
	case "windows":
		return "npipe:////./pipe/docker_engine"
	}
	return ""
}
