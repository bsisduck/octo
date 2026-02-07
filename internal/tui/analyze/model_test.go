package analyze

import (
	"context"
	"testing"

	"github.com/bsisduck/octo/internal/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestAnalyze_NewCreatesModel tests New creates a model
func TestAnalyze_NewCreatesModel(t *testing.T) {
	mock := &docker.MockDockerService{}
	opts := Options{TypeFilter: "", Dangling: false}

	m := New(mock, opts)

	assert.Equal(t, ResourceAll, m.filterType)
	assert.False(t, m.showDangling)
	assert.True(t, m.loading)
	assert.Equal(t, 0, m.selected)
}

// TestAnalyze_InitReturnsCmd tests Init returns a command
func TestAnalyze_InitReturnsCmd(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	cmd := m.Init()

	assert.NotNil(t, cmd)
}

// TestAnalyze_UpdateWithDataMsg tests Update handles DataMsg
func TestAnalyze_UpdateWithDataMsg(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	entries := []ResourceEntry{
		{
			Type:       ResourceContainers,
			ID:         "abc123",
			Name:       "test-container",
			Selectable: true,
		},
	}

	msg := DataMsg{
		Entries:  entries,
		Warnings: []string{},
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.False(t, model.loading)
	assert.Len(t, model.entries, 1)
	assert.Equal(t, "test-container", model.entries[0].Name)
}

// TestAnalyze_UpdateWithWarnings tests Update handles warnings
func TestAnalyze_UpdateWithWarnings(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	msg := DataMsg{
		Entries:  []ResourceEntry{},
		Warnings: []string{"container fetch failed", "image fetch failed"},
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Len(t, model.warnings, 2)
	assert.Contains(t, model.warnings, "container fetch failed")
}

// TestAnalyze_UpdateWithQuitKey tests Update handles quit key
func TestAnalyze_UpdateWithQuitKey(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd := m.Update(msg)

	assert.NotNil(t, cmd)
}

// TestAnalyze_NavigationUp tests up key navigation
func TestAnalyze_NavigationUp(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	m.entries = []ResourceEntry{
		{Name: "item1", Selectable: true},
		{Name: "item2", Selectable: true},
		{Name: "item3", Selectable: true},
	}
	m.selected = 2
	m.loading = false

	msg := tea.KeyMsg{Type: tea.KeyUp}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, 1, model.selected)
}

// TestAnalyze_ViewRendersContent tests View renders something
func TestAnalyze_ViewRendersContent(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	m.entries = []ResourceEntry{
		{Name: "test-item", IsCategory: true},
		{Name: "item1", Selectable: true},
	}
	m.loading = false
	m.width = 60
	m.height = 30

	view := m.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Analyzer")
}

// TestAnalyze_ViewShowsLoading tests View renders loading message
func TestAnalyze_ViewShowsLoading(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	m.loading = true

	view := m.View()

	assert.Contains(t, view, "Loading")
}

// TestAnalyze_ViewShowsWarnings tests View renders warnings
func TestAnalyze_ViewShowsWarnings(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	m.entries = []ResourceEntry{}
	m.warnings = []string{"warning 1", "warning 2"}
	m.loading = false
	m.width = 60
	m.height = 30

	view := m.View()

	assert.Contains(t, view, "Warnings")
}

// TestAnalyze_ResourceTypeString tests ResourceType String() method
func TestAnalyze_ResourceTypeString(t *testing.T) {
	tests := []struct {
		rt       ResourceType
		expected string
	}{
		{ResourceAll, "Overview"},
		{ResourceContainers, "Containers"},
		{ResourceImages, "Images"},
		{ResourceVolumes, "Volumes"},
		{ResourceNetworks, "Networks"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.rt.String())
		})
	}
}

// TestAnalyze_StartContainer_Success tests successful container start
func TestAnalyze_StartContainer_Success(t *testing.T) {
	startCalled := false

	mock := &docker.MockDockerService{
		StartContainerFn: func(ctx context.Context, id string) error {
			startCalled = true
			assert.Equal(t, "abc123", id)
			return nil
		},
		ListContainersFn: func(ctx context.Context, all bool) ([]docker.ContainerInfo, error) {
			return []docker.ContainerInfo{
				{ID: "abc123", Name: "test-container", State: "running"},
			}, nil
		},
	}

	m := New(mock, Options{TypeFilter: "containers"})
	m.entries = []ResourceEntry{
		{
			Type:       ResourceContainers,
			ID:         "abc123",
			Name:       "test-container",
			Selectable: true,
		},
	}
	m.selected = 0
	m.loading = false

	// Call the start command
	cmd := m.startSelectedContainer()
	assert.NotNil(t, cmd)

	// Execute the command to verify it works
	msg := cmd()
	assert.NotNil(t, msg)
	assert.True(t, startCalled)
}

// TestAnalyze_StopContainer_Error tests successful container stop
func TestAnalyze_StopContainer_Error(t *testing.T) {
	stopCalled := false

	mock := &docker.MockDockerService{
		StopContainerFn: func(ctx context.Context, id string) error {
			stopCalled = true
			return nil
		},
		ListContainersFn: func(ctx context.Context, all bool) ([]docker.ContainerInfo, error) {
			return []docker.ContainerInfo{
				{ID: "abc123", Name: "test-container", State: "exited"},
			}, nil
		},
	}

	m := New(mock, Options{TypeFilter: "containers"})
	m.entries = []ResourceEntry{
		{
			Type:       ResourceContainers,
			ID:         "abc123",
			Name:       "test-container",
			Selectable: true,
		},
	}
	m.selected = 0
	m.loading = false
	m.warnings = []string{}

	// Call the stop command
	cmd := m.stopSelectedContainer()
	assert.NotNil(t, cmd)

	// Execute the command - it should call fetchResources which returns new data
	msg := cmd()
	// The command returns the result of fetchResources()()
	assert.NotNil(t, msg)
	assert.True(t, stopCalled)
}

// TestAnalyze_RestartContainer_Success tests successful container restart
func TestAnalyze_RestartContainer_Success(t *testing.T) {
	restartCalled := false

	mock := &docker.MockDockerService{
		RestartContainerFn: func(ctx context.Context, id string) error {
			restartCalled = true
			assert.Equal(t, "abc123", id)
			return nil
		},
		ListContainersFn: func(ctx context.Context, all bool) ([]docker.ContainerInfo, error) {
			return []docker.ContainerInfo{
				{ID: "abc123", Name: "test-container", State: "running"},
			}, nil
		},
	}

	m := New(mock, Options{TypeFilter: "containers"})
	m.entries = []ResourceEntry{
		{
			Type:       ResourceContainers,
			ID:         "abc123",
			Name:       "test-container",
			Selectable: true,
		},
	}
	m.selected = 0
	m.loading = false

	// Call the restart command
	cmd := m.restartSelectedContainer()
	assert.NotNil(t, cmd)

	// Execute the command to verify it works
	msg := cmd()
	assert.NotNil(t, msg)
	assert.True(t, restartCalled)
}

// TestAnalyze_CanOperateOnSelected tests the canOperateOnSelected helper
func TestAnalyze_CanOperateOnSelected(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	// No entries
	assert.False(t, m.canOperateOnSelected())

	// Add a non-selectable category
	m.entries = []ResourceEntry{
		{Type: ResourceContainers, Name: "Containers", IsCategory: true},
	}
	m.selected = 0
	assert.False(t, m.canOperateOnSelected())

	// Add a selectable container
	m.entries = append(m.entries, ResourceEntry{
		Type:       ResourceContainers,
		ID:         "abc123",
		Name:       "test",
		Selectable: true,
	})
	m.selected = 1
	assert.True(t, m.canOperateOnSelected())

	// Out of bounds
	m.selected = 10
	assert.False(t, m.canOperateOnSelected())
}

