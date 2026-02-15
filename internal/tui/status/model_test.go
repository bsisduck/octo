package status

import (
	"context"
	"testing"
	"time"

	"github.com/bsisduck/octo/internal/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// TestStatus_NewCreatesModel tests New creates a model
func TestStatus_NewCreatesModel(t *testing.T) {
	mock := &docker.MockDockerService{}

	m := New(mock, false)

	assert.NotNil(t, m.client)
	assert.True(t, m.lastUpdated.IsZero())
}

// TestStatus_InitReturnsCmd tests Init returns a command
func TestStatus_InitReturnsCmd(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	cmd := m.Init()

	assert.NotNil(t, cmd)
}

// TestStatus_UpdateWithDataMsg tests Update handles DataMsg
func TestStatus_UpdateWithDataMsg(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	containers := []docker.ContainerInfo{
		{ID: "abc123", Name: "test-container", State: "running"},
	}
	images := []docker.ImageInfo{
		{ID: "xyz789", Repository: "nginx", Tag: "latest"},
	}
	diskUsage := &docker.DiskUsageInfo{
		Total:            1000000000,
		TotalReclaimable: 500000000,
	}

	msg := DataMsg{
		Containers:    containers,
		Images:        images,
		DiskUsage:     diskUsage,
		ServerVersion: "25.0.0",
		OsInfo:        "Docker Desktop",
		Warnings:      []string{},
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Len(t, model.containers, 1)
	assert.Len(t, model.images, 1)
	assert.NotNil(t, model.diskUsage)
	assert.Equal(t, "25.0.0", model.serverVersion)
	assert.False(t, model.lastUpdated.IsZero())
}

// TestStatus_UpdateWithWarnings tests Update handles warnings
func TestStatus_UpdateWithWarnings(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	msg := DataMsg{
		Containers: []docker.ContainerInfo{},
		Images:     []docker.ImageInfo{},
		DiskUsage:  &docker.DiskUsageInfo{},
		Warnings:   []string{"timeout on images", "partial data"},
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Len(t, model.warnings, 2)
	assert.Contains(t, model.warnings, "timeout on images")
}

// TestStatus_UpdateWithQuitKey tests quit key handling
func TestStatus_UpdateWithQuitKey(t *testing.T) {
	mock := &docker.MockDockerService{}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	m := New(mock, false)
	_, cmd := m.Update(msg)

	assert.NotNil(t, cmd)
}

// TestStatus_ViewRendersContent tests View renders content
func TestStatus_ViewRendersContent(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	m.containers = []docker.ContainerInfo{
		{ID: "abc123", Name: "test-container", State: "running"},
		{ID: "def456", Name: "stopped-container", State: "exited"},
	}
	m.images = []docker.ImageInfo{
		{ID: "img1", Repository: "nginx", Dangling: false},
		{ID: "img2", Repository: "", Dangling: true},
	}
	m.diskUsage = &docker.DiskUsageInfo{
		Total:            1000000000,
		TotalReclaimable: 500000000,
	}
	m.serverVersion = "25.0.0"
	m.osInfo = "Docker Desktop"
	m.width = 60
	m.height = 30

	view := m.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Status")
	assert.Contains(t, view, "Containers")
	assert.Contains(t, view, "Images")
}

// TestStatus_ViewShowsContainerStats tests View renders container stats
func TestStatus_ViewShowsContainerStats(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	m.containers = []docker.ContainerInfo{
		{ID: "abc123", Name: "running-app", State: "running"},
		{ID: "def456", Name: "stopped-app", State: "exited"},
	}
	m.width = 60
	m.height = 30

	view := m.View()

	assert.Contains(t, view, "Running")
	assert.Contains(t, view, "Stopped")
}

// TestStatus_ViewShowsServerVersion tests View shows server version
func TestStatus_ViewShowsServerVersion(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	m.serverVersion = "25.0.0"
	m.osInfo = "Linux"
	m.width = 60
	m.height = 30

	view := m.View()

	assert.Contains(t, view, "25.0.0")
	assert.Contains(t, view, "Linux")
}

// TestStatus_UpdateWithWindowSize tests window size handling
func TestStatus_UpdateWithWindowSize(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, 100, model.width)
	assert.Equal(t, 50, model.height)
}

// TestStatus_UpdateWithTick tests tick message handling
func TestStatus_UpdateWithTick(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	msg := tickMsg(time.Now())
	_, cmd := m.Update(msg)

	assert.NotNil(t, cmd)
}

// TestStatus_CancelsFetchOnQuit tests cancel is called on quit
func TestStatus_CancelsFetchOnQuit(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFetch = cancel

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, _ = m.Update(msg)

	assert.NotNil(t, ctx.Err())
}

// TestStatus_CountsRunningContainers tests container state counting
func TestStatus_CountsRunningContainers(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, false)

	m.containers = []docker.ContainerInfo{
		{State: "running"},
		{State: "running"},
		{State: "exited"},
		{State: "created"},
	}
	m.width = 60
	m.height = 30

	view := m.View()

	assert.Contains(t, view, "2")
}
