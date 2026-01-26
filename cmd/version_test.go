package cmd

import (
	"testing"
)

func TestVersionVariables(t *testing.T) {
	// These are set at build time, but we can verify they exist
	// Default values should be empty or "dev"
	if Version == "" {
		Version = "dev" // Set default for test
	}

	if Version != "dev" && Version != "" {
		// If set, should be a valid version string
		t.Logf("Version: %s", Version)
	}
}

func TestBuildInfo(t *testing.T) {
	// Test that version variables are accessible
	_ = Version
	_ = BuildTime
	_ = GitCommit
}
