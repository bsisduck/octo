package clipboard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Testable variables for platform-specific behavior.
// Tests can override these to simulate different OS/environment conditions.
var (
	goos     = runtime.GOOS
	lookPath = exec.LookPath
	getenv   = os.Getenv
)

// Copy writes text to the system clipboard.
// Returns nil on success, or a descriptive error (always non-fatal).
func Copy(text string) error {
	name, args, err := clipboardCmd()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("clipboard timed out (is a display server running?)")
		}
		return fmt.Errorf("clipboard command failed: %w", err)
	}
	return nil
}

func clipboardCmd() (string, []string, error) {
	switch goos {
	case "darwin":
		return "pbcopy", nil, nil
	case "linux":
		return linuxClipboardCmd()
	case "windows":
		return "clip.exe", nil, nil
	default:
		return "", nil, fmt.Errorf(
			"clipboard not supported on %s; install xclip, xsel, or wl-copy", goos)
	}
}

func linuxClipboardCmd() (string, []string, error) {
	// Wayland first (if WAYLAND_DISPLAY is set)
	if getenv("WAYLAND_DISPLAY") != "" {
		if path, err := lookPath("wl-copy"); err == nil {
			return path, nil, nil
		}
	}
	// X11 fallback
	if path, err := lookPath("xclip"); err == nil {
		return path, []string{"-selection", "clipboard"}, nil
	}
	if path, err := lookPath("xsel"); err == nil {
		return path, []string{"--clipboard", "--input"}, nil
	}
	// WSL fallback (linux but with clip.exe available)
	if path, err := lookPath("clip.exe"); err == nil {
		return path, nil, nil
	}
	return "", nil, fmt.Errorf(
		"no clipboard tool found; install wl-copy (Wayland), xclip, or xsel (X11)")
}
