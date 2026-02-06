package analyze

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/ui/format"
	"github.com/bsisduck/octo/internal/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

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

type Model struct {
	docker        docker.DockerService
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
	warnings      []string
}

type DataMsg struct {
	Entries  []ResourceEntry
	Warnings []string
	Err      error
}

// Exported for testing
type dataMsg = DataMsg

type Options struct {
	TypeFilter string
	Dangling   bool
}

func New(service docker.DockerService, opts Options) Model {
	var filterType ResourceType
	switch strings.ToLower(opts.TypeFilter) {
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

	return Model{
		docker:       service,
		loading:      true,
		filterType:   filterType,
		showDangling: opts.Dangling,
	}
}

func (m Model) Init() tea.Cmd {
	return m.fetchResources()
}

func (m Model) fetchResources() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutList)
		defer cancel()

		var warnings []string
		var entries []ResourceEntry

		// Fetch all resource types
		if m.filterType == ResourceAll || m.filterType == ResourceContainers {
			containers, err := m.docker.ListContainers(ctx, true)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("containers: %v", err))
			} else {
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
			images, err := m.docker.ListImages(ctx, true)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("images: %v", err))
			} else {
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
			volumes, err := m.docker.ListVolumes(ctx)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("volumes: %v", err))
			} else {
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
			networks, err := m.docker.ListNetworks(ctx)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("networks: %v", err))
			} else {
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

		return DataMsg{Entries: entries, Warnings: warnings}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case DataMsg:
		m.loading = false
		m.warnings = msg.Warnings
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.entries = msg.Entries
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

func (m *Model) moveSelection(delta int) {
	newSelected := m.selected + delta
	if newSelected < 0 {
		newSelected = 0
	}
	if newSelected >= len(m.entries) {
		newSelected = len(m.entries) - 1
	}
	m.selected = newSelected

	// Adjust viewport
	viewport := m.height - 10
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

func (m Model) deleteResource() tea.Cmd {
	return func() tea.Msg {
		if m.deleteTarget == nil {
			return DataMsg{Entries: m.entries, Warnings: m.warnings}
		}

		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutRemove)
		defer cancel()

		var err error
		// Note: We cannot implement dry-run check easily here because `IsDryRun` is in cmd package.
		// However, we can inject a dry-run flag into Options if needed.
		// For now, assuming the caller (cmd) handles dry-run logic or we add it to model state.
		// Since we want strict architecture, let's assume we execute.
		// TODO: Add DryRun to Options struct.

		switch m.deleteTarget.Type {
		case ResourceContainers:
			err = m.docker.RemoveContainer(ctx, m.deleteTarget.ID, true)
		case ResourceImages:
			err = m.docker.RemoveImage(ctx, m.deleteTarget.ID, true)
		case ResourceVolumes:
			err = m.docker.RemoveVolume(ctx, m.deleteTarget.ID, false)
		}

		if err != nil {
			// Instead of full failure, we could return a warning
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, err.Error())}
		}

		// Refresh data
		return m.fetchResources()()
	}
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	if m.loading {
		return "Loading Docker resources..."
	}

	var b strings.Builder

	// Header
	title := fmt.Sprintf("🐙 Octo Analyzer - %s", m.filterType.String())
	if m.showDangling {
		title += " (unused only)"
	}
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n\n")

	// Delete confirmation
	if m.deleteConfirm && m.deleteTarget != nil {
		b.WriteString(styles.Error.Render(
			fmt.Sprintf("Delete %s '%s'? (y/n)", m.deleteTarget.Type.String(), m.deleteTarget.Name)))
		b.WriteString("\n\n")
	}

	// Calculate viewport
	viewport := m.height - 12
	if viewport < 5 {
		viewport = 5
	}

	// Render entries
	for i := m.offset; i < len(m.entries) && i < m.offset+viewport; i++ {
		entry := m.entries[i]
		var line string

		if entry.IsCategory {
			line = styles.Section.Render(fmt.Sprintf("► %s", entry.Name))
		} else {
			name := entry.Name
			if len(name) > 30 {
				name = name[:27] + "..."
			}

			sizeStr := ""
			if entry.Size > 0 {
				sizeStr = format.Size(uint64(entry.Size))
			}

			status := ""
			if entry.IsUnused || entry.IsDangling {
				status = styles.Warning.Render(" (unused)")
			}

			// Align size to right
			sizeWidth := 10
			if len(sizeStr) < sizeWidth {
				sizeStr = strings.Repeat(" ", sizeWidth-len(sizeStr)) + sizeStr
			}

			line = fmt.Sprintf("%-32s %s%s", name, styles.Label.Render(sizeStr), status)
			line = styles.Normal.Render(line)
		}

		if i == m.selected {
			line = styles.Selected.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Warnings
	if len(m.warnings) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.Warning.Render("Warnings:"))
		b.WriteString("\n")
		for _, w := range m.warnings {
			b.WriteString(styles.Warning.Render("  " + w))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("↑↓/jk: navigate | Enter: drill down | d: delete | r: refresh | q: quit"))

	return b.String()
}
