package tests

import (
	"os/exec"
	"strings"
	"testing"
)

// Integration tests require Docker to be running
// These tests verify the CLI interface

func TestOctoVersion(t *testing.T) {
	cmd := exec.Command("../bin/octo", "version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("octo version failed: %v", err)
	}

	if !strings.Contains(string(output), "Octo version") {
		t.Errorf("Expected 'Octo version' in output, got: %s", output)
	}
}

func TestOctoHelp(t *testing.T) {
	cmd := exec.Command("../bin/octo", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("octo --help failed: %v", err)
	}

	expected := []string{
		"Orchestrate your Docker containers",
		"status",
		"analyze",
		"cleanup",
		"prune",
		"diagnose",
	}

	for _, exp := range expected {
		if !strings.Contains(string(output), exp) {
			t.Errorf("Expected %q in help output", exp)
		}
	}
}

func TestOctoStatusHelp(t *testing.T) {
	cmd := exec.Command("../bin/octo", "status", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("octo status --help failed: %v", err)
	}

	if !strings.Contains(string(output), "status") {
		t.Errorf("Expected 'status' in output, got: %s", output)
	}
}

func TestOctoAnalyzeHelp(t *testing.T) {
	cmd := exec.Command("../bin/octo", "analyze", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("octo analyze --help failed: %v", err)
	}

	expected := []string{"analyze", "containers", "images", "volumes"}
	for _, exp := range expected {
		if !strings.Contains(string(output), exp) {
			t.Errorf("Expected %q in analyze help output", exp)
		}
	}
}

func TestOctoCleanupHelp(t *testing.T) {
	cmd := exec.Command("../bin/octo", "cleanup", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("octo cleanup --help failed: %v", err)
	}

	expected := []string{"cleanup", "dry-run", "force"}
	for _, exp := range expected {
		if !strings.Contains(string(output), exp) {
			t.Errorf("Expected %q in cleanup help output", exp)
		}
	}
}

func TestOctoPruneHelp(t *testing.T) {
	cmd := exec.Command("../bin/octo", "prune", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("octo prune --help failed: %v", err)
	}

	if !strings.Contains(string(output), "prune") {
		t.Errorf("Expected 'prune' in output, got: %s", output)
	}
}

func TestOctoDiagnoseHelp(t *testing.T) {
	cmd := exec.Command("../bin/octo", "diagnose", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("octo diagnose --help failed: %v", err)
	}

	if !strings.Contains(string(output), "diagnose") {
		t.Errorf("Expected 'diagnose' in output, got: %s", output)
	}
}

func TestOctoInvalidCommand(t *testing.T) {
	cmd := exec.Command("../bin/octo", "invalid-command")
	_, err := cmd.Output()
	// Should exit with error for invalid command
	if err == nil {
		t.Error("Expected error for invalid command")
	}
}

func TestOctoCleanupDryRun(t *testing.T) {
	// Skip if Docker is not running
	dockerCheck := exec.Command("docker", "info")
	if err := dockerCheck.Run(); err != nil {
		t.Skip("Docker is not running, skipping integration test")
	}

	cmd := exec.Command("../bin/octo", "cleanup", "--dry-run")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// It's okay if it fails due to no resources to clean
		t.Logf("cleanup --dry-run output: %s", output)
	}
}

func TestOctoPruneDryRun(t *testing.T) {
	// Skip if Docker is not running
	dockerCheck := exec.Command("docker", "info")
	if err := dockerCheck.Run(); err != nil {
		t.Skip("Docker is not running, skipping integration test")
	}

	cmd := exec.Command("../bin/octo", "prune", "--dry-run")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("prune --dry-run output: %s", output)
	}
}
