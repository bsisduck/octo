package cmd

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/bsisduck/octo/internal/tui/menu"
)

// NewInteractiveMenu creates a new interactive menu
func NewInteractiveMenu() *InteractiveMenu {
	return &InteractiveMenu{}
}

// InteractiveMenu provides the TUI-based main menu
type InteractiveMenu struct {
	program *tea.Program
}

// Run starts the interactive menu and returns the chosen action.
// Returns an empty string if the user quit without selecting.
func (m *InteractiveMenu) Run() (string, error) {
	p := tea.NewProgram(menu.New(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.program = p
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}
	if mdl, ok := finalModel.(menu.Model); ok {
		return mdl.ChosenAction(), nil
	}
	return "", nil
}
