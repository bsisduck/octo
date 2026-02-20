package menu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/bsisduck/octo/internal/docker"
)

// TestMenu_New creates a new menu model
func TestMenu_New(t *testing.T) {
	m := New()

	assert.Equal(t, 0, m.selected)
	assert.Empty(t, m.chosenAction)
	assert.Len(t, m.items, 5)
	assert.Equal(t, "Status", m.items[0].title)
	assert.Equal(t, "Analyze", m.items[1].title)
}

// TestMenu_ChosenAction returns the selected action
func TestMenu_ChosenAction(t *testing.T) {
	m := New()
	assert.Empty(t, m.ChosenAction())

	m.chosenAction = "status"
	assert.Equal(t, "status", m.ChosenAction())
}

// TestMenu_EnterSelectsCurrentItem tests enter key selects action
func TestMenu_EnterSelectsCurrentItem(t *testing.T) {
	m := New()

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, "status", model.ChosenAction())
	assert.NotNil(t, cmd)
}

// TestMenu_NumberKeySelectsItem tests number keys select items
func TestMenu_NumberKeySelectsItem(t *testing.T) {
	tests := []struct {
		key            string
		expectedIdx    int
		expectedAction string
	}{
		{"1", 0, "status"},
		{"2", 1, "analyze"},
		{"3", 2, "cleanup"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			m := New()
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			updated, _ := m.Update(msg)
			model := updated.(Model)

			assert.Equal(t, tt.expectedIdx, model.selected)
			assert.Equal(t, tt.expectedAction, model.ChosenAction())
		})
	}
}

// TestMenu_UpDownNavigate tests arrow key navigation
func TestMenu_UpDownNavigate(t *testing.T) {
	m := New()

	msg := tea.KeyMsg{Type: tea.KeyDown}
	updated, _ := m.Update(msg)
	model := updated.(Model)
	assert.Equal(t, 1, model.selected)

	msg = tea.KeyMsg{Type: tea.KeyUp}
	updated, _ = updated.Update(msg)
	model = updated.(Model)
	assert.Equal(t, 0, model.selected)
}

// TestMenu_ViewRendersAllItems tests view renders all menu items
func TestMenu_ViewRendersAllItems(t *testing.T) {
	m := New()
	m.width = 50
	m.height = 30

	view := m.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Status")
	assert.Contains(t, view, "Analyze")
	assert.Contains(t, view, "Orchestrate")
}

// TestMenu_InitMsgUpdatesState tests InitMsg updates model state
func TestMenu_InitMsgUpdatesState(t *testing.T) {
	m := New()

	msg := InitMsg{
		DockerOK:   true,
		Containers: 5,
		Running:    3,
		Images:     10,
		Volumes:    2,
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	assert.True(t, model.dockerOK)
	assert.Equal(t, 5, model.containers)
	assert.Equal(t, 3, model.running)
}

// initMenuWithDocker creates a menu model and sends InitMsg with Docker connected and disk usage
func initMenuWithDocker() Model {
	m := New()
	msg := InitMsg{
		DockerOK:   true,
		DiskUsage:  &docker.DiskUsageInfo{Total: 1024, TotalReclaimable: 512},
		Containers: 3,
		Running:    1,
		Images:     5,
		Volumes:    2,
	}
	updated, _ := m.Update(msg)
	return updated.(Model)
}

// TestMouseClickSelectsMenuItem tests that left-clicking on a menu item selects and executes it
func TestMouseClickSelectsMenuItem(t *testing.T) {
	m := initMenuWithDocker()

	// headerHeight should be computed after InitMsg
	assert.Greater(t, m.headerHeight, 0)

	tests := []struct {
		name           string
		itemIdx        int
		expectedAction string
	}{
		{"first item (Status)", 0, "status"},
		{"second item (Analyze)", 1, "analyze"},
		{"third item (Cleanup)", 2, "cleanup"},
		{"fourth item (Prune)", 3, "prune"},
		{"fifth item (Diagnose)", 4, "diagnose"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := initMenuWithDocker()
			msg := tea.MouseMsg{
				Action: tea.MouseActionPress,
				Button: tea.MouseButtonLeft,
				X:      10,
				Y:      m.headerHeight + tt.itemIdx,
			}

			updated, cmd := m.Update(msg)
			model := updated.(Model)

			assert.Equal(t, tt.itemIdx, model.selected)
			assert.Equal(t, tt.expectedAction, model.ChosenAction())
			assert.NotNil(t, cmd, "should return tea.Quit command")
		})
	}
}

// TestMouseClickOutOfBoundsIgnored tests that clicking outside menu items is ignored
func TestMouseClickOutOfBoundsIgnored(t *testing.T) {
	m := initMenuWithDocker()

	tests := []struct {
		name string
		y    int
	}{
		{"click on header area", m.headerHeight - 2},
		{"click on line zero", 0},
		{"click below all items", m.headerHeight + len(m.items)},
		{"click far below", m.headerHeight + 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := initMenuWithDocker()
			msg := tea.MouseMsg{
				Action: tea.MouseActionPress,
				Button: tea.MouseButtonLeft,
				X:      10,
				Y:      tt.y,
			}

			updated, cmd := m.Update(msg)
			model := updated.(Model)

			assert.Equal(t, 0, model.selected, "selection should not change")
			assert.Empty(t, model.ChosenAction(), "no action should be chosen")
			assert.Nil(t, cmd, "no quit command should be returned")
		})
	}
}

// TestMouseClickRightButtonIgnored tests that right-click does not select
func TestMouseClickRightButtonIgnored(t *testing.T) {
	m := initMenuWithDocker()

	msg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonRight,
		X:      10,
		Y:      m.headerHeight + 1,
	}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, 0, model.selected)
	assert.Empty(t, model.ChosenAction())
	assert.Nil(t, cmd)
}

// TestMouseReleaseIgnored tests that mouse release does not trigger selection
func TestMouseReleaseIgnored(t *testing.T) {
	m := initMenuWithDocker()

	msg := tea.MouseMsg{
		Action: tea.MouseActionRelease,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      m.headerHeight + 0,
	}

	updated, cmd := m.Update(msg)
	model := updated.(Model)

	assert.Equal(t, 0, model.selected)
	assert.Empty(t, model.ChosenAction())
	assert.Nil(t, cmd)
}

// TestMouseClickHeaderHeightWithoutDiskUsage tests header computation when no disk usage
func TestMouseClickHeaderHeightWithoutDiskUsage(t *testing.T) {
	m := New()

	// InitMsg with docker OK but no disk usage
	msg := InitMsg{
		DockerOK:   true,
		DiskUsage:  nil,
		Containers: 0,
		Running:    0,
		Images:     0,
		Volumes:    0,
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	// Without disk usage or error, header is shorter
	// logo(6) + tagline(1) + blanks(2) + commands title(1) = 10
	assert.Equal(t, 10, model.headerHeight)

	// Click on first item should still work
	clickMsg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      model.headerHeight + 0,
	}

	updated2, cmd := model.Update(clickMsg)
	model2 := updated2.(Model)
	assert.Equal(t, "status", model2.ChosenAction())
	assert.NotNil(t, cmd)
}

// TestMouseClickHeaderHeightDockerNotConnected tests header computation when Docker is not connected
func TestMouseClickHeaderHeightDockerNotConnected(t *testing.T) {
	m := New()

	msg := InitMsg{
		DockerOK: false,
		Err:      nil,
	}

	updated, _ := m.Update(msg)
	model := updated.(Model)

	// With docker not OK: logo(6) + tagline(1) + blanks(2) + error(2) + commands title(1) = 12
	assert.Equal(t, 12, model.headerHeight)

	// Click on first item
	clickMsg := tea.MouseMsg{
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
		X:      10,
		Y:      model.headerHeight + 0,
	}

	updated2, cmd := model.Update(clickMsg)
	model2 := updated2.(Model)
	assert.Equal(t, "status", model2.ChosenAction())
	assert.NotNil(t, cmd)
}
