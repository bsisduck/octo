package menu

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dustin/go-humanize"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/ui/styles"
)

type Model struct {
	selected     int
	chosenAction string
	items        []item
	width        int
	height       int
	headerHeight int
	err          error
	dockerOK     bool
	diskUsage    *docker.DiskUsageInfo
	containers   int
	running      int
	images       int
	volumes      int
}

type item struct {
	title    string
	subtitle string
	action   string
}

type InitMsg struct {
	DockerOK   bool
	DiskUsage  *docker.DiskUsageInfo
	Containers int
	Running    int
	Images     int
	Volumes    int
	Err        error
}

// New creates a new interactive menu model
func New() Model {
	return Model{
		selected: 0,
		items: []item{
			{title: "Status", subtitle: "Monitor system health", action: "status"},
			{title: "Analyze", subtitle: "Explore resource usage", action: "analyze"},
			{title: "Cleanup", subtitle: "Smart cleanup with safety", action: "cleanup"},
			{title: "Prune", subtitle: "Deep cleanup all unused", action: "prune"},
			{title: "Diagnose", subtitle: "Check Docker health", action: "diagnose"},
		},
	}
}

// ChosenAction returns the action selected by the user
func (m Model) ChosenAction() string {
	return m.chosenAction
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		client, err := docker.NewClient()
		if err != nil {
			return InitMsg{DockerOK: false, Err: err}
		}
		defer func() { _ = client.Close() }()

		ctx := context.Background()

		diskUsage, _ := client.GetDiskUsage(ctx)
		containers, _ := client.ListContainers(ctx, true)
		images, _ := client.ListImages(ctx, true)
		volumes, _ := client.ListVolumes(ctx)

		running := 0
		for _, c := range containers {
			if c.State == "running" {
				running++
			}
		}

		return InitMsg{
			DockerOK:   true,
			DiskUsage:  diskUsage,
			Containers: len(containers),
			Running:    running,
			Images:     len(images),
			Volumes:    len(volumes),
		}
	}
}

func (m Model) computeHeaderHeight() int {
	h := 0
	h += 6 // logo (5 lines of ASCII art + leading newline = 6 rendered lines)
	h += 1 // tagline
	h += 2 // two blank lines (\n after logo, \n\n after tagline)

	if !m.dockerOK {
		h += 2 // error line + blank
	} else if m.diskUsage != nil {
		h += 1 // "Quick Stats" title
		h += 4 // containers, images, volumes, disk used
		if m.diskUsage.TotalReclaimable > 0 {
			h += 1 // reclaimable line
		}
		h += 1 // blank line after stats
	}

	h += 1 // "Commands" title
	return h
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			idx := msg.Y - m.headerHeight
			if idx >= 0 && idx < len(m.items) {
				m.selected = idx
				m.chosenAction = m.items[idx].action
				return m, tea.Quit
			}
		}

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
			if m.selected >= 0 && m.selected < len(m.items) {
				m.chosenAction = m.items[m.selected].action
			}
			return m, tea.Quit
		case "1":
			m.selected = 0
			m.chosenAction = m.items[0].action
			return m, tea.Quit
		case "2":
			m.selected = 1
			m.chosenAction = m.items[1].action
			return m, tea.Quit
		case "3":
			m.selected = 2
			m.chosenAction = m.items[2].action
			return m, tea.Quit
		case "4":
			m.selected = 3
			m.chosenAction = m.items[3].action
			return m, tea.Quit
		case "5":
			m.selected = 4
			m.chosenAction = m.items[4].action
			return m, tea.Quit
		case "v":
			m.chosenAction = "version"
			return m, tea.Quit
		case "?", "h":
			// Help - just quit
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case InitMsg:
		m.dockerOK = msg.DockerOK
		m.err = msg.Err
		m.diskUsage = msg.DiskUsage
		m.containers = msg.Containers
		m.running = msg.Running
		m.images = msg.Images
		m.volumes = msg.Volumes
		m.headerHeight = m.computeHeaderHeight()
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	// Logo
	logo := `
   ___       _
  / _ \  ___| |_ ___
 | | | |/ __| __/ _ \
 | |_| | (__| || (_) |
  \___/ \___|\__\___/`

	b.WriteString(styles.Logo.Render(logo))
	b.WriteString("\n")
	b.WriteString(styles.Tagline.Render("  Orchestrate your Docker containers like an octopus."))
	b.WriteString("\n\n")

	// Docker status
	if !m.dockerOK {
		b.WriteString(styles.Error.Render("  Docker: Not connected"))
		if m.err != nil {
			b.WriteString(styles.Error.Render(fmt.Sprintf(" (%v)", m.err)))
		}
		b.WriteString("\n\n")
	} else if m.diskUsage != nil {
		b.WriteString(styles.TitleWithMargin.Render("  Quick Stats"))
		b.WriteString("\n")
		fmt.Fprintf(&b, "  %s %s\n",
			styles.StatLabel.Render("Containers:"),
			styles.StatValue.Render(fmt.Sprintf("%d (%d running)", m.containers, m.running)))
		fmt.Fprintf(&b, "  %s %s\n",
			styles.StatLabel.Render("Images:"),
			styles.StatValue.Render(fmt.Sprintf("%d", m.images)))
		fmt.Fprintf(&b, "  %s %s\n",
			styles.StatLabel.Render("Volumes:"),
			styles.StatValue.Render(fmt.Sprintf("%d", m.volumes)))
		fmt.Fprintf(&b, "  %s %s\n",
			styles.StatLabel.Render("Disk Used:"),
			styles.StatValue.Render(humanize.Bytes(uint64(m.diskUsage.Total))))
		if m.diskUsage.TotalReclaimable > 0 {
			fmt.Fprintf(&b, "  %s %s\n",
				styles.StatLabel.Render("Reclaimable:"),
				styles.Warning.Render(
					humanize.Bytes(uint64(m.diskUsage.TotalReclaimable))))
		}
		b.WriteString("\n")
	}

	// Menu items
	b.WriteString(styles.TitleWithMargin.Render("  Commands"))
	b.WriteString("\n")

	for i, item := range m.items {
		cursor := "  "
		style := styles.Normal
		if i == m.selected {
			cursor = "▸ "
			style = styles.Selected
		}

		line := fmt.Sprintf("%s%d. %s", cursor, i+1, item.title)
		b.WriteString(style.Render(line))
		b.WriteString("  ")
		b.WriteString(styles.Subtitle.Render(item.subtitle))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("  ↑↓/jk/click: navigate | Enter/1-5: select | v: version | q: quit"))
	b.WriteString("\n")

	// Note about using commands directly
	b.WriteString(styles.Info.Render("  Tip: Use 'octo status -w' for live monitoring"))
	b.WriteString("\n")

	return b.String()
}
