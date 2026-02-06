package menu

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
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
		key         string
		expectedIdx int
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

