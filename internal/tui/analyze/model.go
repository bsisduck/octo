package analyze

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bsisduck/octo/internal/clipboard"
	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/tui/common"
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

// ClipboardText formats a human-readable string for clipboard copy.
func (e ResourceEntry) ClipboardText() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Type: %s", e.Type.String()))
	if e.Name != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", e.Name))
	}
	if e.ID != "" {
		parts = append(parts, fmt.Sprintf("ID: %s", e.ID))
	}
	if e.Size > 0 {
		parts = append(parts, fmt.Sprintf("Size: %s", format.Size(uint64(e.Size))))
	}
	if e.Status != "" {
		parts = append(parts, fmt.Sprintf("Status: %s", e.Status))
	}
	if e.Extra != "" {
		parts = append(parts, fmt.Sprintf("Details: %s", e.Extra))
	}
	return strings.Join(parts, "\n")
}

type Model struct {
	docker            docker.DockerService
	entries           []ResourceEntry
	selected          int
	offset            int
	width             int
	height            int
	err               error
	loading           bool
	filterType        ResourceType
	showDangling      bool
	deleteConfirm     bool
	deleteTarget      *ResourceEntry
	deleteConfirmInfo *docker.ConfirmationInfo
	warnings          []string
	statusMessage     string
}

type DataMsg struct {
	Entries  []ResourceEntry
	Warnings []string
	Err      error
}

// Exported for testing
type dataMsg = DataMsg

// ConfirmationMsg contains the result of a DryRun operation
type ConfirmationMsg struct {
	Info *docker.ConfirmationInfo
	Err  error
}

// Exported for testing
type confirmationMsg = ConfirmationMsg

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
					// Phase 1: Call DryRun to get confirmation info (and re-check state)
					return m, m.reCheckAndShowConfirmation()
				}
			}
		case "s":
			if m.canOperateOnSelected() && m.selectedEntry().Type == ResourceContainers {
				return m, m.startSelectedContainer()
			}
		case "t":
			if m.canOperateOnSelected() && m.selectedEntry().Type == ResourceContainers {
				return m, m.stopSelectedContainer()
			}
		case "r":
			if m.canOperateOnSelected() && m.selectedEntry().Type == ResourceContainers {
				return m, m.restartSelectedContainer()
			} else {
				// Only refresh if not operating on a container
				if m.canOperateOnSelected() {
					break
				}
				m.loading = true
				return m, m.fetchResources()
			}
		case "y":
			if m.canOperateOnSelected() {
				entry := m.selectedEntry()
				text := entry.ClipboardText()
				return m, func() tea.Msg {
					err := clipboard.Copy(text)
					return common.ClipboardMsg{
						Success: err == nil,
						Text:    text,
						Err:     err,
					}
				}
			}
		case "x":
			if m.canOperateOnSelected() && m.selectedEntry().Type == ResourceContainers {
				if m.canExecOnSelected() {
					entry := m.selectedEntry()
					api := m.docker.API()
					cmd := docker.NewDockerExecCommand(api, entry.ID, "/bin/sh")
					return m, tea.Exec(cmd, func(err error) tea.Msg {
						return common.ExecFinishedMsg{Err: err}
					})
				}
				m.statusMessage = "Cannot exec: container is not running"
				return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
					return common.ClearStatusMsg{}
				})
			}
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			if !m.deleteConfirm { // Don't process clicks during confirmation
				headerLines := 3 // title + separator + blank
				idx := msg.Y - headerLines + m.offset
				if idx >= 0 && idx < len(m.entries) {
					m.selected = idx
					// Adjust viewport to follow selection
					viewport := m.height - 12
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
			}
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

	case ConfirmationMsg:
		// Phase 1 completion: DryRun returned, show confirmation dialog
		if msg.Err != nil {
			m.warnings = append(m.warnings, fmt.Sprintf("Failed to prepare deletion: %v", msg.Err))
			m.deleteTarget = nil
			m.deleteConfirmInfo = nil
		} else {
			m.deleteConfirmInfo = msg.Info
			m.deleteConfirm = true
		}

	case common.ClipboardMsg:
		if msg.Success {
			m.statusMessage = "Copied to clipboard"
		} else {
			m.statusMessage = fmt.Sprintf("Clipboard: %v", msg.Err)
		}
		return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
			return common.ClearStatusMsg{}
		})

	case common.ClearStatusMsg:
		m.statusMessage = ""

	case common.ExecFinishedMsg:
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Shell exited with error: %v", msg.Err)
		} else {
			m.statusMessage = "Shell session ended"
		}
		// Refresh data -- container state may have changed during exec
		m.loading = true
		return m, tea.Batch(
			m.fetchResources(),
			tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return common.ClearStatusMsg{}
			}),
		)
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

// reCheckAndShowConfirmation is Phase 1: Call DryRun to get confirmation info (and re-check state)
func (m Model) reCheckAndShowConfirmation() tea.Cmd {
	return func() tea.Msg {
		if m.deleteTarget == nil {
			return ConfirmationMsg{Err: fmt.Errorf("no target selected")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutList)
		defer cancel()

		var info docker.ConfirmationInfo
		var err error

		switch m.deleteTarget.Type {
		case ResourceContainers:
			info, err = m.docker.RemoveContainerDryRun(ctx, m.deleteTarget.ID)
		case ResourceImages:
			info, err = m.docker.RemoveImageDryRun(ctx, m.deleteTarget.ID)
		case ResourceVolumes:
			info, err = m.docker.RemoveVolumeDryRun(ctx, m.deleteTarget.ID)
		case ResourceNetworks:
			info, err = m.docker.RemoveNetworkDryRun(ctx, m.deleteTarget.ID)
		default:
			err = fmt.Errorf("unsupported resource type for deletion: %v", m.deleteTarget.Type)
		}

		if err != nil {
			return ConfirmationMsg{Err: err}
		}

		return ConfirmationMsg{Info: &info}
	}
}

// deleteResource is Phase 2: Execute the actual deletion after user confirmation
// Re-checks state one more time before executing (TOCTOU protection)
func (m Model) deleteResource() tea.Cmd {
	return func() tea.Msg {
		if m.deleteTarget == nil {
			return DataMsg{Entries: m.entries, Warnings: m.warnings}
		}

		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutRemove)
		defer cancel()

		// Phase 2: Re-check state one final time before execution (TOCTOU protection)
		// Call DryRun again to verify state hasn't changed unexpectedly
		var currentInfo docker.ConfirmationInfo
		var err error

		switch m.deleteTarget.Type {
		case ResourceContainers:
			currentInfo, err = m.docker.RemoveContainerDryRun(ctx, m.deleteTarget.ID)
		case ResourceImages:
			currentInfo, err = m.docker.RemoveImageDryRun(ctx, m.deleteTarget.ID)
		case ResourceVolumes:
			currentInfo, err = m.docker.RemoveVolumeDryRun(ctx, m.deleteTarget.ID)
		case ResourceNetworks:
			currentInfo, err = m.docker.RemoveNetworkDryRun(ctx, m.deleteTarget.ID)
		default:
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("unsupported resource type: %v", m.deleteTarget.Type))}
		}

		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("State changed during confirmation (TOCTOU): %v", err))}
		}

		// Check if tier changed (indicates state change)
		if m.deleteConfirmInfo != nil && currentInfo.Tier != m.deleteConfirmInfo.Tier {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("WARNING: Resource state changed from %s to %s - operation aborted for safety", m.deleteConfirmInfo.Tier.String(), currentInfo.Tier.String()))}
		}

		// Now execute the actual deletion
		switch m.deleteTarget.Type {
		case ResourceContainers:
			err = m.docker.RemoveContainer(ctx, m.deleteTarget.ID, false)
		case ResourceImages:
			err = m.docker.RemoveImage(ctx, m.deleteTarget.ID, false)
		case ResourceVolumes:
			err = m.docker.RemoveVolume(ctx, m.deleteTarget.ID, false)
		case ResourceNetworks:
			err = m.docker.RemoveNetwork(ctx, m.deleteTarget.ID)
		}

		// Clear confirmation state
		m.deleteConfirm = false
		m.deleteTarget = nil
		m.deleteConfirmInfo = nil

		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, err.Error())}
		}

		// Refresh data
		return m.fetchResources()()
	}
}

// canOperateOnSelected checks if we can operate on the currently selected entry.
func (m *Model) canOperateOnSelected() bool {
	return m.selected < len(m.entries) && m.entries[m.selected].Selectable && !m.entries[m.selected].IsCategory
}

// canExecOnSelected checks if exec is possible on the selected container (must be running).
func (m *Model) canExecOnSelected() bool {
	if !m.canOperateOnSelected() {
		return false
	}
	entry := m.selectedEntry()
	return entry.Type == ResourceContainers && !entry.IsUnused
}

// selectedEntry returns the currently selected entry.
func (m *Model) selectedEntry() ResourceEntry {
	if m.selected < len(m.entries) {
		return m.entries[m.selected]
	}
	return ResourceEntry{}
}

func (m Model) startSelectedContainer() tea.Cmd {
	return func() tea.Msg {
		if !m.canOperateOnSelected() {
			return DataMsg{Entries: m.entries, Warnings: m.warnings}
		}

		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutAction)
		defer cancel()

		err := m.docker.StartContainer(ctx, m.selectedEntry().ID)

		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("Failed to start container: %v", err))}
		}

		// Refresh data to show updated state
		return m.fetchResources()()
	}
}

func (m Model) stopSelectedContainer() tea.Cmd {
	return func() tea.Msg {
		if !m.canOperateOnSelected() {
			return DataMsg{Entries: m.entries, Warnings: m.warnings}
		}

		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutAction)
		defer cancel()

		err := m.docker.StopContainer(ctx, m.selectedEntry().ID)

		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("Failed to stop container: %v", err))}
		}

		// Refresh data to show updated state
		return m.fetchResources()()
	}
}

func (m Model) restartSelectedContainer() tea.Cmd {
	return func() tea.Msg {
		if !m.canOperateOnSelected() {
			return DataMsg{Entries: m.entries, Warnings: m.warnings}
		}

		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutAction)
		defer cancel()

		err := m.docker.RestartContainer(ctx, m.selectedEntry().ID)

		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("Failed to restart container: %v", err))}
		}

		// Refresh data to show updated state
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

	// Delete confirmation dialog (detailed)
	if m.deleteConfirm && m.deleteTarget != nil && m.deleteConfirmInfo != nil {
		b.WriteString(m.renderConfirmationDialog(*m.deleteConfirmInfo))
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

	// Status message
	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(styles.Info.Render("  " + m.statusMessage))
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("↑↓/jk/click: navigate | Enter: drill down | x: shell | y: copy | d: delete | r: refresh | q: quit"))

	return b.String()
}

// renderConfirmationDialog renders a detailed confirmation dialog with safety tier colors
func (m Model) renderConfirmationDialog(info docker.ConfirmationInfo) string {
	var b strings.Builder

	// Tier-colored title
	tierStyle := styles.TierStyle(int(info.Tier))
	b.WriteString(tierStyle.Render(fmt.Sprintf("⚠ %s\n", info.Title)))

	// Description
	b.WriteString(styles.Info.Render(fmt.Sprintf("   %s\n", info.Description)))

	// Resources list
	if len(info.Resources) > 0 {
		b.WriteString(styles.Help.Render("   Resources:\n"))
		for _, r := range info.Resources {
			b.WriteString(styles.Help.Render(fmt.Sprintf("     • %s\n", r)))
		}
	}

	// Reversibility status
	if info.Reversible {
		b.WriteString(styles.Success.Render("   ✓ Reversible\n"))
		b.WriteString(styles.Help.Render(fmt.Sprintf("   %s\n", info.UndoInstructions)))
	} else {
		b.WriteString(tierStyle.Render("   ✗ NOT REVERSIBLE - Data will be permanently lost\n"))
		b.WriteString(styles.Help.Render(fmt.Sprintf("   %s\n", info.UndoInstructions)))
	}

	// Warnings
	if len(info.Warnings) > 0 {
		b.WriteString(styles.Warning.Render("   ⚠ Warnings:\n"))
		for _, w := range info.Warnings {
			b.WriteString(styles.Warning.Render(fmt.Sprintf("     • %s\n", w)))
		}
	}

	// Safety tier indicator
	b.WriteString(tierStyle.Render(fmt.Sprintf("   Safety Level: %s\n", info.Tier.String())))

	// Confirmation prompt
	b.WriteString("\n")
	b.WriteString(styles.DeleteConfirm.Render("   Confirm deletion? [y] Yes  [n] No"))

	return b.String()
}
