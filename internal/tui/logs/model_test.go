package logs

import (
	"context"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bsisduck/octo/internal/docker"
)

var testTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func makeEntries(n int, stream string) []docker.LogEntry {
	entries := make([]docker.LogEntry, n)
	for i := 0; i < n; i++ {
		entries[i] = docker.LogEntry{
			Timestamp: testTime.Add(time.Duration(i) * time.Second),
			Stream:    stream,
			Content:   fmt.Sprintf("log line %d", i),
		}
	}
	return entries
}

func mockService(entries []docker.LogEntry) *docker.MockDockerService {
	return &docker.MockDockerService{
		GetContainerLogsFn: func(_ context.Context, _ string, _ int) ([]docker.LogEntry, error) {
			return entries, nil
		},
		StreamContainerLogsFn: func(_ context.Context, _ string) (<-chan docker.LogEntry, <-chan error, func()) {
			logCh := make(chan docker.LogEntry)
			errCh := make(chan error)
			close(logCh)
			return logCh, errCh, func() {}
		},
	}
}

func TestLogsModelInit(t *testing.T) {
	mock := mockService(nil)
	m := New(mock, "abc123", "test-container")

	if !m.following {
		t.Error("expected following=true on init")
	}
	if m.buffer.Capacity() != DefaultCapacity {
		t.Errorf("buffer capacity = %d, want %d", m.buffer.Capacity(), DefaultCapacity)
	}
	if m.containerID != "abc123" {
		t.Errorf("containerID = %q, want %q", m.containerID, "abc123")
	}
	if m.containerName != "test-container" {
		t.Errorf("containerName = %q, want %q", m.containerName, "test-container")
	}

	cmd := m.Init()
	if cmd == nil {
		t.Error("Init() returned nil cmd, expected fetch command")
	}
}

func TestLogsModelScrolling(t *testing.T) {
	// Use 50 entries with small height so viewport is smaller than content
	entries := makeEntries(50, "stdout")
	mock := mockService(entries)
	m := New(mock, "abc123", "test-container")
	m.width = 80
	m.height = 20 // viewport = 20 - 7 = 13 lines, content = 50 lines

	// Simulate receiving initial logs
	model, _ := m.Update(InitialLogsMsg{Entries: entries})
	m = model.(Model)

	if len(m.viewLines) != 50 {
		t.Fatalf("viewLines count = %d, want 50", len(m.viewLines))
	}

	// Following should have scrolled to bottom
	expectedBottomOffset := len(m.viewLines) - m.viewportHeight()
	if m.offset != expectedBottomOffset {
		t.Errorf("offset = %d after init, want %d (bottom)", m.offset, expectedBottomOffset)
	}

	// Scroll up with 'k'
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = model.(Model)
	if m.following {
		t.Error("expected following=false after scroll up")
	}
	if m.offset != expectedBottomOffset-1 {
		t.Errorf("offset = %d after 'k', want %d", m.offset, expectedBottomOffset-1)
	}

	// Jump to top with 'g'
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = model.(Model)
	if m.offset != 0 {
		t.Errorf("offset = %d after 'g', want 0", m.offset)
	}
	if m.following {
		t.Error("expected following=false after 'g'")
	}

	// Jump to bottom with 'G'
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = model.(Model)
	if !m.following {
		t.Error("expected following=true after 'G'")
	}
	if m.offset != expectedBottomOffset {
		t.Errorf("offset = %d after 'G', want %d", m.offset, expectedBottomOffset)
	}

	// Scroll down with 'j' from top
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = model.(Model) // first go to top
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = model.(Model)
	if m.offset != 1 {
		t.Errorf("offset = %d after 'j' from top, want 1", m.offset)
	}
}

func TestLogsModelFollowMode(t *testing.T) {
	entries := makeEntries(5, "stdout")
	mock := mockService(entries)
	m := New(mock, "abc123", "test-container")
	m.width = 80
	m.height = 30

	// Receive initial logs -- should follow
	model, _ := m.Update(InitialLogsMsg{Entries: entries})
	m = model.(Model)

	if !m.following {
		t.Error("expected following=true after init")
	}

	// Scroll up disables following
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(Model)
	if m.following {
		t.Error("expected following=false after scroll up")
	}

	// 'G' restores following
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = model.(Model)
	if !m.following {
		t.Error("expected following=true after 'G'")
	}

	// 'f' toggles following off
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = model.(Model)
	if m.following {
		t.Error("expected following=false after 'f' toggle")
	}

	// 'f' toggles following back on
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = model.(Model)
	if !m.following {
		t.Error("expected following=true after second 'f' toggle")
	}
}

func TestLogsModelFilter(t *testing.T) {
	entries := make([]docker.LogEntry, 10)
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("info line %d", i)
		if i%3 == 0 {
			content = fmt.Sprintf("error line %d", i)
		}
		entries[i] = docker.LogEntry{
			Timestamp: testTime.Add(time.Duration(i) * time.Second),
			Stream:    "stdout",
			Content:   content,
		}
	}

	mock := mockService(entries)
	m := New(mock, "abc123", "test-container")
	m.width = 80
	m.height = 30

	// Load entries
	model, _ := m.Update(InitialLogsMsg{Entries: entries})
	m = model.(Model)

	totalLines := len(m.viewLines)
	if totalLines != 10 {
		t.Fatalf("viewLines = %d, want 10", totalLines)
	}

	// Enter filter mode with '/'
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = model.(Model)
	if !m.filtering {
		t.Error("expected filtering=true after '/'")
	}

	// Type "error"
	for _, ch := range "error" {
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = model.(Model)
	}
	if m.filterText != "error" {
		t.Errorf("filterText = %q, want %q", m.filterText, "error")
	}

	// Press enter to apply filter
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if m.filtering {
		t.Error("expected filtering=false after enter")
	}

	// Should have fewer lines (only those matching "error")
	if len(m.viewLines) >= totalLines {
		t.Errorf("filtered viewLines = %d, should be less than %d", len(m.viewLines), totalLines)
	}
	if len(m.viewLines) == 0 {
		t.Error("filtered viewLines should not be empty")
	}

	// Each filtered line should contain "error"
	for i, line := range m.viewLines {
		if !containsCaseInsensitive(line, "error") {
			t.Errorf("filtered line %d = %q, does not contain 'error'", i, line)
		}
	}
}

func containsCaseInsensitive(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && findCaseInsensitive(s, substr))
}

func findCaseInsensitive(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return indexOf(s, substr) >= 0
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestLogsModelRegexFilter(t *testing.T) {
	entries := make([]docker.LogEntry, 10)
	for i := 0; i < 10; i++ {
		content := fmt.Sprintf("info request %dms", i*100)
		if i%2 == 0 {
			content = fmt.Sprintf("error timeout %dms", i*100)
		}
		entries[i] = docker.LogEntry{
			Timestamp: testTime.Add(time.Duration(i) * time.Second),
			Stream:    "stdout",
			Content:   content,
		}
	}

	mock := mockService(entries)
	m := New(mock, "abc123", "test-container")
	m.width = 80
	m.height = 30

	// Load entries
	model, _ := m.Update(InitialLogsMsg{Entries: entries})
	m = model.(Model)

	// Toggle regex mode with ctrl+r
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = model.(Model)
	if !m.filtering {
		t.Error("expected filtering=true after ctrl+r")
	}
	if !m.useRegex {
		t.Error("expected useRegex=true after ctrl+r")
	}

	// Type regex pattern
	for _, ch := range "err.*timeout" {
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = model.(Model)
	}

	// Apply with enter
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = model.(Model)

	if m.compiledRegex == nil {
		t.Fatal("expected compiledRegex to be set after valid regex pattern")
	}

	// Should have filtered results
	if len(m.viewLines) == 0 {
		t.Error("expected some filtered results for 'err.*timeout' pattern")
	}
	if len(m.viewLines) >= 10 {
		t.Errorf("expected fewer than 10 filtered lines, got %d", len(m.viewLines))
	}
}

func TestLogsModelTruncationWarning(t *testing.T) {
	mock := mockService(nil)
	m := New(mock, "abc123", "test-container")
	// Use a small buffer to trigger truncation
	m.buffer = NewRingBuffer(5)
	m.width = 80
	m.height = 30

	// Create 10 entries -- will overflow buffer of capacity 5
	entries := makeEntries(10, "stdout")

	model, _ := m.Update(InitialLogsMsg{Entries: entries})
	m = model.(Model)

	if m.buffer.Dropped() != 5 {
		t.Errorf("Dropped() = %d, want 5", m.buffer.Dropped())
	}
	if m.buffer.Len() != 5 {
		t.Errorf("Len() = %d, want 5", m.buffer.Len())
	}

	// Simulate a stream message to trigger truncation warning update
	streamEntry := docker.LogEntry{
		Timestamp: testTime.Add(11 * time.Second),
		Stream:    "stdout",
		Content:   "new streamed line",
	}
	model, _ = m.Update(StreamLogMsg{Entry: streamEntry})
	m = model.(Model)

	if m.truncationWarning == "" {
		t.Error("expected truncation warning to be set")
	}
	expectedWarning := "Logs truncated: oldest 6 lines dropped"
	if m.truncationWarning != expectedWarning {
		t.Errorf("truncationWarning = %q, want %q", m.truncationWarning, expectedWarning)
	}

	// Verify the warning appears in the View output
	view := m.View()
	if !containsStr(view, "Logs truncated") {
		t.Error("View() should contain truncation warning")
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func TestLogsModelView(t *testing.T) {
	entries := makeEntries(3, "stdout")
	mock := mockService(entries)
	m := New(mock, "abc123", "test-container")
	m.width = 80
	m.height = 30

	// Load entries
	model, _ := m.Update(InitialLogsMsg{Entries: entries})
	m = model.(Model)

	view := m.View()

	// Check header
	if !containsStr(view, "Logs: test-container") {
		t.Error("View should contain container name in header")
	}
	if !containsStr(view, "[FOLLOWING]") {
		t.Error("View should show [FOLLOWING] when following=true")
	}

	// Check footer keybindings
	if !containsStr(view, "scroll") {
		t.Error("View should contain scroll keybinding hint")
	}
	if !containsStr(view, "export") {
		t.Error("View should contain export keybinding hint")
	}
}

func TestLogsModelStreamError(t *testing.T) {
	mock := &docker.MockDockerService{
		GetContainerLogsFn: func(_ context.Context, _ string, _ int) ([]docker.LogEntry, error) {
			return nil, nil
		},
		StreamContainerLogsFn: func(_ context.Context, _ string) (<-chan docker.LogEntry, <-chan error, func()) {
			logCh := make(chan docker.LogEntry)
			errCh := make(chan error)
			close(logCh)
			return logCh, errCh, func() {}
		},
	}
	m := New(mock, "abc123", "test-container")
	m.width = 80
	m.height = 30

	// Send a stream error
	model, _ := m.Update(StreamErrMsg{Err: fmt.Errorf("connection lost")})
	m = model.(Model)

	if !containsStr(m.statusMessage, "Stream error") {
		t.Errorf("statusMessage = %q, expected to contain 'Stream error'", m.statusMessage)
	}
}
