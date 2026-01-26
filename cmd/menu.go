package cmd

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

// InteractiveMenu provides the TUI-based main menu
type InteractiveMenu struct {
	program *tea.Program
}

// NewInteractiveMenu creates a new interactive menu
func NewInteractiveMenu() *InteractiveMenu {
	return &InteractiveMenu{}
}

// Run starts the interactive menu
func (m *InteractiveMenu) Run() error {
	p := tea.NewProgram(newMenuModel(), tea.WithAltScreen())
	m.program = p
	_, err := p.Run()
	return err
}

// Menu model
type menuModel struct {
	selected   int
	items      []menuItem
	width      int
	height     int
	err        error
	dockerOK   bool
	diskUsage  *DiskUsageInfo
	containers int
	running    int
	images     int
	volumes    int
}

type menuItem struct {
	title    string
	subtitle string
	action   string
}

type menuInitMsg struct {
	dockerOK   bool
	diskUsage  *DiskUsageInfo
	containers int
	running    int
	images     int
	volumes    int
	err        error
}

func newMenuModel() menuModel {
	return menuModel{
		selected: 0,
		items: []menuItem{
			{title: "Status", subtitle: "Monitor system health", action: "status"},
			{title: "Analyze", subtitle: "Explore resource usage", action: "analyze"},
			{title: "Cleanup", subtitle: "Smart cleanup with safety", action: "cleanup"},
			{title: "Prune", subtitle: "Deep cleanup all unused", action: "prune"},
			{title: "Diagnose", subtitle: "Check Docker health", action: "diagnose"},
		},
	}
}

func (m menuModel) Init() tea.Cmd {
	return func() tea.Msg {
		client, err := NewDockerClient()
		if err != nil {
			return menuInitMsg{dockerOK: false, err: err}
		}
		defer client.Close()

		diskUsage, _ := client.GetDiskUsage()
		containers, _ := client.ListContainers(true)
		images, _ := client.ListImages(true)
		volumes, _ := client.ListVolumes()

		running := 0
		for _, c := range containers {
			if c.State == "running" {
				running++
			}
		}

		return menuInitMsg{
			dockerOK:   true,
			diskUsage:  diskUsage,
			containers: len(containers),
			running:    running,
			images:     len(images),
			volumes:    len(volumes),
		}
	}
}

func (m menuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case "enter", " ":
			return m, m.executeAction()
		case "1":
			m.selected = 0
			return m, m.executeAction()
		case "2":
			m.selected = 1
			return m, m.executeAction()
		case "3":
			m.selected = 2
			return m, m.executeAction()
		case "4":
			m.selected = 3
			return m, m.executeAction()
		case "5":
			m.selected = 4
			return m, m.executeAction()
		case "v":
			// Version
			runVersion(nil, nil)
			return m, tea.Quit
		case "?", "h":
			// Help - just quit and show help
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case menuInitMsg:
		m.dockerOK = msg.dockerOK
		m.err = msg.err
		m.diskUsage = msg.diskUsage
		m.containers = msg.containers
		m.running = msg.running
		m.images = msg.images
		m.volumes = msg.volumes
	}

	return m, nil
}

func (m menuModel) executeAction() tea.Cmd {
	return func() tea.Msg {
		// Exit the TUI first, then command will be executed based on selection
		return tea.Quit()
	}
}

func (m menuModel) View() string {
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("69")).
		MarginBottom(1)

	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("69"))

	taglineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Bold(true).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))

	statLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(12)

	statValueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	var b strings.Builder

	// Logo
	logo := `
   ___       _
  / _ \  ___| |_ ___
 | | | |/ __| __/ _ \
 | |_| | (__| || (_) |
  \___/ \___|\__\___/`

	b.WriteString(logoStyle.Render(logo))
	b.WriteString("\n")
	b.WriteString(taglineStyle.Render("  Orchestrate your Docker containers like an octopus."))
	b.WriteString("\n\n")

	// Docker status
	if !m.dockerOK {
		b.WriteString(errorStyle.Render("  Docker: Not connected"))
		if m.err != nil {
			b.WriteString(errorStyle.Render(fmt.Sprintf(" (%v)", m.err)))
		}
		b.WriteString("\n\n")
	} else if m.diskUsage != nil {
		b.WriteString(titleStyle.Render("  Quick Stats"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s\n",
			statLabelStyle.Render("Containers:"),
			statValueStyle.Render(fmt.Sprintf("%d (%d running)", m.containers, m.running))))
		b.WriteString(fmt.Sprintf("  %s %s\n",
			statLabelStyle.Render("Images:"),
			statValueStyle.Render(fmt.Sprintf("%d", m.images))))
		b.WriteString(fmt.Sprintf("  %s %s\n",
			statLabelStyle.Render("Volumes:"),
			statValueStyle.Render(fmt.Sprintf("%d", m.volumes))))
		b.WriteString(fmt.Sprintf("  %s %s\n",
			statLabelStyle.Render("Disk Used:"),
			statValueStyle.Render(humanize.Bytes(uint64(m.diskUsage.Total)))))
		if m.diskUsage.TotalReclaimable > 0 {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				statLabelStyle.Render("Reclaimable:"),
				lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(
					humanize.Bytes(uint64(m.diskUsage.TotalReclaimable)))))
		}
		b.WriteString("\n")
	}

	// Menu items
	b.WriteString(titleStyle.Render("  Commands"))
	b.WriteString("\n")

	for i, item := range m.items {
		cursor := "  "
		style := normalStyle
		if i == m.selected {
			cursor = "▸ "
			style = selectedStyle
		}

		line := fmt.Sprintf("%s%d. %s", cursor, i+1, item.title)
		b.WriteString(style.Render(line))
		b.WriteString("  ")
		b.WriteString(subtitleStyle.Render(item.subtitle))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑↓/jk: navigate | Enter: select | v: version | q: quit"))
	b.WriteString("\n")

	// Note about using commands directly
	b.WriteString(infoStyle.Render("  Tip: Use 'octo status -w' for live monitoring"))
	b.WriteString("\n")

	return b.String()
}
