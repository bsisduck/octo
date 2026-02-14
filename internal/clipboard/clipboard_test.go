package clipboard

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// saveAndRestore saves the current test variables and returns a restore function.
func saveAndRestore() func() {
	origGoos := goos
	origLookPath := lookPath
	origGetenv := getenv
	return func() {
		goos = origGoos
		lookPath = origLookPath
		getenv = origGetenv
	}
}

func TestClipboardCmdDarwin(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "darwin"

	name, args, err := clipboardCmd()
	require.NoError(t, err)
	assert.Equal(t, "pbcopy", name)
	assert.Nil(t, args)
}

func TestClipboardCmdWindows(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "windows"

	name, args, err := clipboardCmd()
	require.NoError(t, err)
	assert.Equal(t, "clip.exe", name)
	assert.Nil(t, args)
}

func TestClipboardCmdLinuxWayland(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "linux"
	getenv = func(key string) string {
		if key == "WAYLAND_DISPLAY" {
			return "wayland-0"
		}
		return ""
	}
	lookPath = func(file string) (string, error) {
		if file == "wl-copy" {
			return "/usr/bin/wl-copy", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}

	name, args, err := clipboardCmd()
	require.NoError(t, err)
	assert.Equal(t, "/usr/bin/wl-copy", name)
	assert.Nil(t, args)
}

func TestClipboardCmdLinuxX11Xclip(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "linux"
	getenv = func(key string) string { return "" } // No WAYLAND_DISPLAY
	lookPath = func(file string) (string, error) {
		if file == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}

	name, args, err := clipboardCmd()
	require.NoError(t, err)
	assert.Equal(t, "/usr/bin/xclip", name)
	assert.Equal(t, []string{"-selection", "clipboard"}, args)
}

func TestClipboardCmdLinuxX11Xsel(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "linux"
	getenv = func(key string) string { return "" }
	lookPath = func(file string) (string, error) {
		if file == "xsel" {
			return "/usr/bin/xsel", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}

	name, args, err := clipboardCmd()
	require.NoError(t, err)
	assert.Equal(t, "/usr/bin/xsel", name)
	assert.Equal(t, []string{"--clipboard", "--input"}, args)
}

func TestClipboardCmdLinuxWSLFallback(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "linux"
	getenv = func(key string) string { return "" }
	lookPath = func(file string) (string, error) {
		if file == "clip.exe" {
			return "/mnt/c/Windows/System32/clip.exe", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}

	name, args, err := clipboardCmd()
	require.NoError(t, err)
	assert.Equal(t, "/mnt/c/Windows/System32/clip.exe", name)
	assert.Nil(t, args)
}

func TestClipboardCmdLinuxNoToolFound(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "linux"
	getenv = func(key string) string { return "" }
	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("not found: %s", file)
	}

	_, _, err := clipboardCmd()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no clipboard tool found")
	assert.Contains(t, err.Error(), "wl-copy")
	assert.Contains(t, err.Error(), "xclip")
	assert.Contains(t, err.Error(), "xsel")
}

func TestClipboardCmdUnsupportedOS(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "freebsd"

	_, _, err := clipboardCmd()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clipboard not supported on freebsd")
	assert.Contains(t, err.Error(), "install xclip")
}

func TestClipboardCmdLinuxWaylandFallsBackToX11(t *testing.T) {
	restore := saveAndRestore()
	defer restore()

	goos = "linux"
	getenv = func(key string) string {
		if key == "WAYLAND_DISPLAY" {
			return "wayland-0"
		}
		return ""
	}
	// wl-copy not found, falls back to xclip
	lookPath = func(file string) (string, error) {
		if file == "xclip" {
			return "/usr/bin/xclip", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}

	name, args, err := clipboardCmd()
	require.NoError(t, err)
	assert.Equal(t, "/usr/bin/xclip", name)
	assert.Equal(t, []string{"-selection", "clipboard"}, args)
}
