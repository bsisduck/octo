package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bsisduck/octo/internal/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze Docker resource usage",
	Long: `Analyze Docker resources with an interactive tree view:
- Explore containers, images, volumes, and networks
- View size breakdown and usage patterns
- Identify large or unused resources
- Navigate with arrow keys, delete with 'd'`,
	Run: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringP("type", "t", "", "Filter by type: containers, images, volumes, networks")
	analyzeCmd.Flags().BoolP("dangling", "d", false, "Show only dangling/unused resources")
}

func runAnalyze(cmd *cobra.Command, args []string) {
	resourceType, _ := cmd.Flags().GetString("type")
	dangling, _ := cmd.Flags().GetBool("dangling")

	p := tea.NewProgram(newAnalyzeModel(resourceType, dangling), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Resource types for the analyzer
type ResourceType int

const (
	ResourceAll ResourceType = iota
	ResourceContainers
	ResourceImages
	ResourceVolumes
	ResourceNetworks
)

func (r ResourceType) String() string {
	switch r {
	case ResourceContainers:
		return "Containers"
	case ResourceImages:
		return "Images"
	case ResourceVolumes:
		return "Volumes"
	case ResourceNetworks:
		return "Networks"
	default:
		return "Overview"
	}
}

// ResourceEntry represents a single resource item
type ResourceEntry struct {
	Type        ResourceType
	ID          string
	Name        string
	Size        int64
	Status      string
	Created     time.Time
	Extra       string
	IsUnused    bool
	IsDangling  bool
	Selectable  bool
	IsCategory  bool
	CategoryIdx int
}

type analyzeModel struct {
	entries       []ResourceEntry
	selected      int
	offset        int
	width         int
	height        int
	err           error
	loading       bool
	filterType    ResourceType
	showDangling  bool
	deleteConfirm bool
	deleteTarget  *ResourceEntry
}

type analyzeDataMsg struct {
	entries []ResourceEntry
	err     error
}

func newAnalyzeModel(typeFilter string, dangling bool) analyzeModel {
	var filterType ResourceType
	switch strings.ToLower(typeFilter) {
	case "containers", "container", "c":
		filterType = ResourceContainers
	case "images", "image", "i":
		filterType = ResourceImages
	case "volumes", "volume", "v":
		filterType = ResourceVolumes
	case "networks", "network", "n":
		filterType = ResourceNetworks
	default:
		filterType = ResourceAll
	}

	return analyzeModel{
		loading:      true,
		filterType:   filterType,
		showDangling: dangling,
	}
}

func (m analyzeModel) Init() tea.Cmd {
	return m.fetchResources()
}

func (m analyzeModel) fetchResources() tea.Cmd {
	return func() tea.Msg {
		client, err := docker.NewClient()
		if err != nil {
			return analyzeDataMsg{err: err}
		}
		defer client.Close()

		var entries []ResourceEntry

		// Fetch all resource types
		if m.filterType == ResourceAll || m.filterType == ResourceContainers {
			containers, err := client.ListContainers(context.Background(), true)
			if err == nil {
				if m.filterType == ResourceAll {
					entries = append(entries, ResourceEntry{
						Type:       ResourceContainers,
						Name:       "Containers",
						IsCategory: true,
					})
				}
				for _, c := range containers {
					if m.showDangling && c.State == "running" {
						continue
					}
					entries = append(entries, ResourceEntry{
						Type:       ResourceContainers,
						ID:         c.ID,
						Name:       c.Name,
						Size:       c.Size,
						Status:     c.Status,
						Created:    c.Created,
						Extra:      c.Image,
						IsUnused:   c.State != "running",
						Selectable: true,
					})
				}
			}
		}

		if m.filterType == ResourceAll || m.filterType == ResourceImages {
			images, err := client.ListImages(context.Background(), true)
			if err == nil {
				if m.filterType == ResourceAll {
					entries = append(entries, ResourceEntry{
						Type:       ResourceImages,
						Name:       "Images",
						IsCategory: true,
					})
				}
				// Sort images by size (largest first)
				sort.Slice(images, func(i, j int) bool {
					return images[i].Size > images[j].Size
				})
				for _, img := range images {
					if m.showDangling && !img.Dangling {
						continue
					}
					name := img.Repository
					if img.Tag != "" && img.Tag != "latest" {
						name = fmt.Sprintf("%s:%s", img.Repository, img.Tag)
					}
					if img.Dangling {
						name = "<none>"
					}
					entries = append(entries, ResourceEntry{
						Type:       ResourceImages,
						ID:         img.ID,
						Name:       name,
						Size:       img.Size,
						Created:    img.Created,
						Extra:      fmt.Sprintf("%d containers", img.Containers),
						IsDangling: img.Dangling,
						IsUnused:   img.Containers == 0,
						Selectable: true,
					})
				}
			}
		}

		if m.filterType == ResourceAll || m.filterType == ResourceVolumes {
			volumes, err := client.ListVolumes(context.Background())
			if err == nil {
				if m.filterType == ResourceAll {
					entries = append(entries, ResourceEntry{
						Type:       ResourceVolumes,
						Name:       "Volumes",
						IsCategory: true,
					})
				}
				for _, v := range volumes {
					if m.showDangling && v.InUse {
						continue
					}
					entries = append(entries, ResourceEntry{
						Type:       ResourceVolumes,
						ID:         v.Name,
						Name:       v.Name,
						Size:       v.Size,
						Created:    v.Created,
						Extra:      v.Driver,
						IsUnused:   !v.InUse,
						Selectable: true,
					})
				}
			}
		}

		if m.filterType == ResourceAll || m.filterType == ResourceNetworks {
			networks, err := client.ListNetworks(context.Background())
			if err == nil {
				if m.filterType == ResourceAll {
					entries = append(entries, ResourceEntry{
						Type:       ResourceNetworks,
						Name:       "Networks",
						IsCategory: true,
					})
				}
				for _, n := range networks {
					// Skip default networks
					if n.Name == "bridge" || n.Name == "host" || n.Name == "none" {
						continue
					}
					if m.showDangling && n.Containers > 0 {
						continue
					}
					entries = append(entries, ResourceEntry{
						Type:       ResourceNetworks,
						ID:         n.ID,
						Name:       n.Name,
						Extra:      fmt.Sprintf("%s (%d containers)", n.Driver, n.Containers),
						IsUnused:   n.Containers == 0,
						Selectable: true,
					})
				}
			}
		}

		return analyzeDataMsg{entries: entries}
	}
}

func (m analyzeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.deleteConfirm {
			switch msg.String() {
			case "y", "Y", "enter":
				m.deleteConfirm = false
				return m, m.deleteResource()
			case "n", "N", "esc", "q":
				m.deleteConfirm = false
				m.deleteTarget = nil
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.filterType != ResourceAll {
				m.filterType = ResourceAll
				m.loading = true
				return m, m.fetchResources()
			}
			return m, tea.Quit
		case "up", "k":
			m.moveSelection(-1)
		case "down", "j":
			m.moveSelection(1)
		case "enter", "l", "right":
			if m.selected < len(m.entries) {
				entry := m.entries[m.selected]
				if entry.IsCategory {
					m.filterType = entry.Type
					m.loading = true
					return m, m.fetchResources()
				}
			}
		case "h", "left":
			if m.filterType != ResourceAll {
				m.filterType = ResourceAll
				m.loading = true
				return m, m.fetchResources()
			}
		case "d", "delete", "backspace":
			if m.selected < len(m.entries) {
				entry := m.entries[m.selected]
				if entry.Selectable && !entry.IsCategory {
					m.deleteTarget = &entry
					m.deleteConfirm = true
				}
			}
		case "r":
			m.loading = true
			return m, m.fetchResources()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case analyzeDataMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.entries = msg.entries
			m.selected = 0
			m.offset = 0
			// Skip to first selectable item
			for i, e := range m.entries {
				if e.Selectable || e.IsCategory {
					m.selected = i
					break
				}
			}
		}
	}

	return m, nil
}

func (m *analyzeModel) moveSelection(delta int) {
	newSelected := m.selected + delta
	if newSelected < 0 {
		newSelected = 0
	}
	if newSelected >= len(m.entries) {
		newSelected = len(m.entries) - 1
	}
	m.selected = newSelected

	// Adjust viewport
	viewport := m.height - 8
	if viewport < 5 {
		viewport = 5
	}
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+viewport {
		m.offset = m.selected - viewport + 1
	}
}

func (m analyzeModel) deleteResource() tea.Cmd {
	return func() tea.Msg {
		if m.deleteTarget == nil {
			return analyzeDataMsg{entries: m.entries}
		}

		client, err := docker.NewClient()
		if err != nil {
			return analyzeDataMsg{err: err}
		}
		defer client.Close()

		if IsDryRun() {
			// In dry-run mode, just refresh without deleting
			return m.fetchResources()()
		}

		switch m.deleteTarget.Type {
		case ResourceContainers:
			err = client.RemoveContainer(context.Background(), m.deleteTarget.ID, true)
		case ResourceImages:
			err = client.RemoveImage(context.Background(), m.deleteTarget.ID, true)
		case ResourceVolumes:
			err = client.RemoveVolume(context.Background(), m.deleteTarget.ID, false)
		}

		if err != nil {
			return analyzeDataMsg{err: err}
		}

		// Refresh data
		return m.fetchResources()()
	}
}

func (m analyzeModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	if m.loading {
		return "Loading Docker resources..."
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	categoryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).PaddingLeft(1)
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("255"))
	normalStyle := lipgloss.NewStyle().PaddingLeft(3)
	unusedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sizeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(10).Align(lipgloss.Right)
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var b strings.Builder

	// Header
	title := fmt.Sprintf("🐙 Octo Analyzer - %s", m.filterType.String())
	if m.showDangling {
		title += " (unused only)"
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	// Delete confirmation
	if m.deleteConfirm && m.deleteTarget != nil {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(
			fmt.Sprintf("Delete %s '%s'? (y/n)", m.deleteTarget.Type.String(), m.deleteTarget.Name)))
		b.WriteString("\n\n")
	}

	// Calculate viewport
	viewport := m.height - 10
	if viewport < 5 {
		viewport = 5
	}

	// Render entries
	for i := m.offset; i < len(m.entries) && i < m.offset+viewport; i++ {
		entry := m.entries[i]
		var line string

		if entry.IsCategory {
			line = categoryStyle.Render(fmt.Sprintf("► %s", entry.Name))
		} else {
			name := entry.Name
			if len(name) > 30 {
				name = name[:27] + "..."
			}

			sizeStr := ""
			if entry.Size > 0 {
				sizeStr = humanize.Bytes(uint64(entry.Size))
			}

			status := ""
			if entry.IsUnused || entry.IsDangling {
				status = unusedStyle.Render(" (unused)")
			}

			line = fmt.Sprintf("%-32s %s%s", name, sizeStyle.Render(sizeStr), status)
			line = normalStyle.Render(line)
		}

		if i == m.selected {
			line = selectedStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓/jk: navigate | Enter: drill down | d: delete | r: refresh | q: quit"))

	return b.String()
}
