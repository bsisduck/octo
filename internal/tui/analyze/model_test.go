package analyze

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/tui/common"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// createAnalyzeModelWithEntries creates an analyze model with test entries
func createAnalyzeModelWithEntries() Model {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	entries := []ResourceEntry{
		{Type: ResourceContainers, Name: "Containers", IsCategory: true},
		{Type: ResourceContainers, ID: "c1", Name: "web-app", Selectable: true, Size: 1024},
		{Type: ResourceContainers, ID: "c2", Name: "db-server", Selectable: true, Size: 2048},
		{Type: ResourceImages, Name: "Images", IsCategory: true},
		{Type: ResourceImages, ID: "i1", Name: "nginx:latest", Selectable: true, Size: 50000},
		{Type: ResourceImages, ID: "i2", Name: "postgres:16", Selectable: true, Size: 80000},
	}

	msg := DataMsg{Entries: entries, Warnings: []string{}}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	model.height = 40 // Set a reasonable terminal height
	model.width = 80
	return model
}

// TestMouseClickSelectsEntry tests that left-clicking selects the correct entry in analyze view
func TestMouseClickSelectsEntry(t *testing.T) {
	// Header is 3 lines (title + separator + blank)
	// Click on second visible entry (index 1 in entries, at Y=3+1=4)
	tests := []struct {
		name        string
		clickY      int
		expectedIdx int
	}{
		{"click on first entry (category)", 3, 0},
		{"click on second entry (web-app)", 4, 1},
		{"click on third entry (db-server)", 5, 2},
		{"click on fourth entry (Images category)", 6, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createAnalyzeModelWithEntries()
			msg := tea.MouseMsg{
				Action: tea.MouseActionPress,
				Button: tea.MouseButtonLeft,
				X:      10,
				Y:      tt.clickY,
			}

			updated, _ := m.Update(msg)
			model := updated.(Model)

			assert.Equal(t, tt.expectedIdx, model.selected)
		})
	}
}

// TestMouseClickWithScrollOffset tests mouse click accounting for scroll offset
func TestMouseClickWithScrollOffset(t *testing.T) {
	mock := &docker.MockDockerService{}
	m := New(mock, Options{})

	// Create many entries to force scrolling
	var entries []ResourceEntry
	for i := 0; i < 30; i++ {
		entries = append(entries, ResourceEntry{
			Type:       ResourceContainers,
			ID:         fmt.Sprintf("c%d", i),
			Name:       fmt.Sprintf("container-%d", i),
			Selectable: true,
		})
	}

	msg := DataMsg{Entries: entries, Warnings: []string{}}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	model.height = 20 // Small viewport to force scrolling
	model.width = 80

	// Simulate scrolling down by setting offset manually
	model.offset = 10

	// Click at Y=3 (first visible line after header)
	// With offset=10, this should select entry at index 10
	clickMsg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      3, // header is 3 lines
	}

	updated2, _ := model.Update(clickMsg)
	model2 := updated2.(Model)

	// idx = Y(3) - headerLines(3) + offset(10) = 10
	assert.Equal(t, 10, model2.selected)

	// Click at Y=5 with same offset
	model.offset = 10
	clickMsg2 := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      5,
	}

	updated3, _ := model.Update(clickMsg2)
	model3 := updated3.(Model)

	// idx = Y(5) - headerLines(3) + offset(10) = 12
	assert.Equal(t, 12, model3.selected)
}

// TestMouseClickDuringConfirmIgnored tests that clicks during confirmation dialog are ignored
func TestMouseClickDuringConfirmIgnored(t *testing.T) {
	m := createAnalyzeModelWithEntries()
	m.deleteConfirm = true
	m.deleteTarget = &ResourceEntry{
		Type: ResourceContainers,
		ID:   "c1",
		Name: "web-app",
	}
	m.deleteConfirmInfo = &docker.ConfirmationInfo{
		Title:       "Delete container",
		Description: "This will remove the container",
	}
	originalSelected := m.selected

	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      5,
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, originalSelected, model.selected, "selection should not change during confirmation")
	assert.True(t, model.deleteConfirm, "confirmation state should remain")
}

// TestMouseClickOnCategoryEntry tests that clicking on a category entry selects it without crash
func TestMouseClickOnCategoryEntry(t *testing.T) {
	m := createAnalyzeModelWithEntries()

	// First entry is a category header at Y=3
	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      3, // header(3) + 0 = first entry (category)
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, 0, model.selected)
	assert.True(t, model.entries[model.selected].IsCategory)
}

// TestMouseClickOutOfBoundsAnalyze tests that out-of-bounds clicks are ignored in analyze
func TestMouseClickOutOfBoundsAnalyze(t *testing.T) {
	m := createAnalyzeModelWithEntries()
	originalSelected := m.selected

	tests := []struct {
		name string
		y    int
	}{
		{"click on header", 0},
		{"click on separator", 1},
		{"click below all entries", 3 + len(m.entries) + 5},
		{"negative index after offset calc", 2}, // Y=2 - 3 + 0 = -1
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := createAnalyzeModelWithEntries()
			msg := tea.MouseMsg{
				Action: tea.MouseActionPress,
				Button: tea.MouseButtonLeft,
				X:      10,
				Y:      tt.y,
			}

			updated, _ := m.Update(msg)
			model := updated.(Model)

			assert.Equal(t, originalSelected, model.selected, "selection should not change for out-of-bounds click")
		})
	}
}

// TestMouseRightClickAnalyzeIgnored tests that right-click is ignored in analyze view
func TestMouseRightClickAnalyzeIgnored(t *testing.T) {
	m := createAnalyzeModelWithEntries()
	originalSelected := m.selected

	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonRight,
		X:      10,
		Y:      4, // valid position for an entry
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, originalSelected, model.selected)
}

// TestMouseReleaseAnalyzeIgnored tests that mouse release is ignored in analyze view
func TestMouseReleaseAnalyzeIgnored(t *testing.T) {
	m := createAnalyzeModelWithEntries()
	originalSelected := m.selected

	msg := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      4,
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, originalSelected, model.selected)
}

// TestYKeyCopyTriggersClipboard tests that pressing y on a selected entry dispatches a clipboard command
func TestYKeyCopyTriggersClipboard(t *testing.T) {
	m := createAnalyzeModelWithEntries()
	// Select a non-category entry (index 1 = "web-app" container)
	m.selected = 1

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	_, cmd := m.Update(msg)

	require.NotNil(t, cmd, "y key on selectable entry should return a clipboard tea.Cmd")

	// Execute the Cmd and verify it returns a ClipboardMsg
	result := cmd()
	clipMsg, ok := result.(common.ClipboardMsg)
	require.True(t, ok, "clipboard command should return common.ClipboardMsg, got %T", result)
	// The copy itself may fail (no clipboard tool in test environment), but the message type is correct
	assert.NotEmpty(t, clipMsg.Text)
}

// TestYKeyOnCategoryIgnored tests that pressing y on a category entry does nothing
func TestYKeyOnCategoryIgnored(t *testing.T) {
	m := createAnalyzeModelWithEntries()
	// Select the category entry (index 0 = "Containers" category)
	m.selected = 0

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")}
	_, cmd := m.Update(msg)

	assert.Nil(t, cmd, "y key on category should not dispatch a clipboard command")
}

// TestClipboardMsgSuccess tests that a successful ClipboardMsg sets the status message
func TestClipboardMsgSuccess(t *testing.T) {
	m := createAnalyzeModelWithEntries()

	msg := common.ClipboardMsg{
		Success: true,
		Text:    "some copied text",
		Err:     nil,
	}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, "Copied to clipboard", model.statusMessage)
	assert.NotNil(t, cmd, "should return a tick command for clearing status")
}

// TestClipboardMsgFailure tests that a failed ClipboardMsg shows the error in status
func TestClipboardMsgFailure(t *testing.T) {
	m := createAnalyzeModelWithEntries()

	msg := common.ClipboardMsg{
		Success: false,
		Text:    "some text",
		Err:     errors.New("no clipboard tool found"),
	}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Contains(t, model.statusMessage, "no clipboard tool found")
	assert.NotNil(t, cmd, "should return a tick command for clearing status")
}

// TestClearStatusMsg tests that ClearStatusMsg clears the status message
func TestClearStatusMsg(t *testing.T) {
	m := createAnalyzeModelWithEntries()
	m.statusMessage = "Copied to clipboard"

	msg := common.ClearStatusMsg{}
	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.Empty(t, model.statusMessage)
}

// TestClipboardTextFormatting tests that ClipboardText produces human-readable output
func TestClipboardTextFormatting(t *testing.T) {
	tests := []struct {
		name     string
		entry    ResourceEntry
		contains []string
	}{
		{
			name: "container with all fields",
			entry: ResourceEntry{
				Type:   ResourceContainers,
				Name:   "web-app",
				ID:     "abc123def456",
				Size:   1048576, // 1 MB
				Status: "running",
				Extra:  "nginx:latest",
			},
			contains: []string{
				"Type: Containers",
				"Name: web-app",
				"ID: abc123def456",
				"Status: running",
				"Details: nginx:latest",
			},
		},
		{
			name: "image with size only",
			entry: ResourceEntry{
				Type: ResourceImages,
				Name: "nginx:latest",
				ID:   "sha256:abc",
				Size: 52428800, // 50 MB
			},
			contains: []string{
				"Type: Images",
				"Name: nginx:latest",
				"ID: sha256:abc",
			},
		},
		{
			name: "minimal entry",
			entry: ResourceEntry{
				Type: ResourceVolumes,
				Name: "my-volume",
			},
			contains: []string{
				"Type: Volumes",
				"Name: my-volume",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text := tt.entry.ClipboardText()
			for _, expected := range tt.contains {
				assert.Contains(t, text, expected)
			}
			// Verify it's NOT Go struct format
			assert.NotContains(t, text, "{")
			assert.NotContains(t, text, "}")
		})
	}
}
