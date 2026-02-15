package analyze

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bsisduck/octo/internal/clipboard"
	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/tui/common"
	"github.com/bsisduck/octo/internal/ui/format"
	"github.com/bsisduck/octo/internal/ui/styles"
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
	Type            ResourceType
	ID              string
	Name            string
	Size            int64
	Status          string
	Created         time.Time
	Extra           string
	IsUnused        bool
	IsDangling      bool
	Selectable      bool
	IsCategory      bool
	CategoryIdx     int
	ComposeProject  string // Compose project name (empty if not part of a project)
	ComposeService  string // Compose service name
	IsProjectHeader bool   // True if this entry is a Compose project group header
	ProjectName     string // Project name for project header entries
	CPUPercent      float64
	MemUsage        uint64
	MemLimit        uint64
	MemPercent      float64
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
	if e.ComposeProject != "" {
		parts = append(parts, fmt.Sprintf("Compose Project: %s", e.ComposeProject))
	}
	if e.ComposeService != "" {
		parts = append(parts, fmt.Sprintf("Compose Service: %s", e.ComposeService))
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
	spinnerFrame      int
	// Filtering
	filtering       bool
	filterText      string
	filteredEntries []ResourceEntry
	// Logs view
	viewMode       int // 0=list, 1=logs
	logEntries     []docker.LogEntry
	logOffset      int
	logContainer   string
	logContainerID string
	logFollowing   bool
	logCancelFn    func()
	logFilterText  string
	logFiltering   bool
}

const (
	viewList = 0
	viewLogs = 1
)

// spinnerTick is a message for animating the loading spinner
type spinnerTick struct{}

var spinnerChars = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type DataMsg struct {
	Entries  []ResourceEntry
	Warnings []string
	Err      error
}

// LogDataMsg carries fetched log entries
type LogDataMsg struct {
	Entries []docker.LogEntry
	Err     error
}

// LogStreamMsg carries a single streamed log entry
type LogStreamMsg struct {
	Entry docker.LogEntry
}

// LogStreamErrMsg signals a stream error
type LogStreamErrMsg struct {
	Err error
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
	return tea.Batch(m.fetchResources(), m.tickSpinner())
}

func (m Model) tickSpinner() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTick{}
	})
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

				// Group by Compose project
				groups, ungrouped := docker.GroupByComposeProject(containers)

				// Add grouped containers
				for _, group := range groups {
					// Project header (selectable for project-level operations)
					entries = append(entries, ResourceEntry{
						Type:            ResourceContainers,
						Name:            fmt.Sprintf("[%s] (%d containers)", group.ProjectName, len(group.Containers)),
						IsProjectHeader: true,
						ProjectName:     group.ProjectName,
						Selectable:      true,
					})
					for _, c := range group.Containers {
						if m.showDangling && c.State == "running" {
							continue
						}
						serviceName := ""
						if c.Labels != nil {
							serviceName = c.Labels[docker.ComposeServiceLabel]
						}
						entries = append(entries, ResourceEntry{
							Type:           ResourceContainers,
							ID:             c.ID,
							Name:           c.Name,
							Size:           c.Size,
							Status:         c.Status,
							Created:        c.Created,
							Extra:          c.Image,
							IsUnused:       c.State != "running",
							Selectable:     true,
							ComposeProject: group.ProjectName,
							ComposeService: serviceName,
						})
					}
				}

				// Add ungrouped containers
				for _, c := range ungrouped {
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

		// Fetch metrics for running containers (cap at 20 to avoid excessive API calls)
		entries = enrichContainerMetrics(ctx, m.docker, entries)

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

// enrichContainerMetrics fetches CPU/memory stats for running containers.
// Caps at 20 running containers to avoid excessive API calls.
func enrichContainerMetrics(ctx context.Context, svc docker.DockerService, entries []ResourceEntry) []ResourceEntry {
	// Find running container indices
	type target struct {
		idx int
		id  string
	}
	var targets []target
	for i, e := range entries {
		if e.Type == ResourceContainers && !e.IsCategory && !e.IsProjectHeader && !e.IsUnused && e.ID != "" {
			targets = append(targets, target{idx: i, id: e.ID})
			if len(targets) >= 20 {
				break
			}
		}
	}
	if len(targets) == 0 {
		return entries
	}

	// Fetch stats in parallel
	type statsResult struct {
		idx     int
		metrics *docker.ContainerMetrics
	}
	results := make([]statsResult, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		wg.Add(1)
		go func(i int, t target) {
			defer wg.Done()
			metrics, err := svc.GetContainerStats(ctx, t.id)
			if err == nil {
				results[i] = statsResult{idx: t.idx, metrics: metrics}
			}
		}(i, t)
	}
	wg.Wait()

	for _, r := range results {
		if r.metrics != nil {
			entries[r.idx].CPUPercent = r.metrics.CPUPercent
			entries[r.idx].MemUsage = r.metrics.MemoryUsage
			entries[r.idx].MemLimit = r.metrics.MemoryLimit
			entries[r.idx].MemPercent = r.metrics.MemoryPercent
		}
	}
	return entries
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Logs view mode key handling
		if m.viewMode == viewLogs {
			return m.updateLogsView(msg)
		}

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

		// Filter mode key handling
		if m.filtering {
			switch msg.Type {
			case tea.KeyEscape:
				m.filtering = false
				m.filterText = ""
				m.filteredEntries = nil
				m.selected = 0
				m.offset = 0
				return m, nil
			case tea.KeyEnter:
				m.filtering = false
				return m, nil
			case tea.KeyBackspace:
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.applyFilter()
				}
				return m, nil
			case tea.KeyRunes:
				m.filterText += string(msg.Runes)
				m.applyFilter()
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.filterText != "" {
				m.filterText = ""
				m.filteredEntries = nil
				m.selected = 0
				m.offset = 0
				return m, nil
			}
			if m.filterType != ResourceAll {
				m.filterType = ResourceAll
				m.loading = true
				return m, m.fetchResources()
			}
			return m, tea.Quit
		case "/":
			m.filtering = true
			m.filterText = ""
			return m, nil
		case "l":
			if m.canOperateOnSelected() && m.selectedEntry().Type == ResourceContainers {
				entry := m.selectedEntry()
				m.viewMode = viewLogs
				m.logContainer = entry.Name
				m.logContainerID = entry.ID
				m.logEntries = nil
				m.logOffset = 0
				m.logFollowing = false
				m.logFilterText = ""
				m.logFiltering = false
				return m, m.fetchLogs(entry.ID, 200)
			}
		case "up", "k":
			m.moveSelection(-1)
		case "down", "j":
			m.moveSelection(1)
		case "enter", "right":
			visible := m.visibleEntries()
			if m.selected < len(visible) {
				entry := visible[m.selected]
				if entry.IsCategory {
					m.filterType = entry.Type
					m.filterText = ""
					m.filteredEntries = nil
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
			visible := m.visibleEntries()
			if m.selected < len(visible) {
				entry := visible[m.selected]
				if entry.Selectable && !entry.IsCategory {
					m.deleteTarget = &entry
					// Phase 1: Call DryRun to get confirmation info (and re-check state)
					return m, m.reCheckAndShowConfirmation()
				}
			}
		case "s":
			if m.canOperateOnSelected() {
				entry := m.selectedEntry()
				if entry.IsProjectHeader {
					return m, m.startComposeProject(entry.ProjectName)
				} else if entry.Type == ResourceContainers {
					return m, m.startSelectedContainer()
				}
			}
		case "t":
			if m.canOperateOnSelected() {
				entry := m.selectedEntry()
				if entry.IsProjectHeader {
					return m, m.stopComposeProject(entry.ProjectName)
				} else if entry.Type == ResourceContainers {
					return m, m.stopSelectedContainer()
				}
			}
		case "r":
			if m.canOperateOnSelected() {
				entry := m.selectedEntry()
				if entry.IsProjectHeader {
					return m, m.restartComposeProject(entry.ProjectName)
				} else if entry.Type == ResourceContainers {
					return m, m.restartSelectedContainer()
				}
			} else {
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
				visible := m.visibleEntries()
				headerLines := 3 // title + separator + blank
				if m.filtering || m.filterText != "" {
					headerLines += 2 // filter bar + blank
				}
				idx := msg.Y - headerLines + m.offset
				if idx >= 0 && idx < len(visible) {
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

	case LogDataMsg:
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Log fetch error: %v", msg.Err)
		} else {
			m.logEntries = msg.Entries
			m.logOffset = max(0, len(m.logEntries)-m.logViewportHeight())
		}

	case LogStreamMsg:
		m.logEntries = append(m.logEntries, msg.Entry)
		if len(m.logEntries) > 5000 {
			m.logEntries = m.logEntries[len(m.logEntries)-5000:]
		}
		if m.logFollowing {
			m.logOffset = max(0, len(m.logEntries)-m.logViewportHeight())
		}

	case LogStreamErrMsg:
		m.logFollowing = false
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Log stream error: %v", msg.Err)
		}

	case spinnerTick:
		if m.loading {
			m.spinnerFrame++
			return m, m.tickSpinner()
		}

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

// visibleEntries returns filtered entries if a filter is active, otherwise all entries.
func (m *Model) visibleEntries() []ResourceEntry {
	if m.filterText != "" && m.filteredEntries != nil {
		return m.filteredEntries
	}
	return m.entries
}

// applyFilter filters entries by the current filterText.
func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filteredEntries = nil
		m.selected = 0
		m.offset = 0
		return
	}

	query := strings.ToLower(m.filterText)
	var filtered []ResourceEntry
	for _, e := range m.entries {
		if e.IsCategory {
			// Include category headers if any child matches
			filtered = append(filtered, e)
			continue
		}
		if strings.Contains(strings.ToLower(e.Name), query) ||
			strings.Contains(strings.ToLower(e.ID), query) ||
			strings.Contains(strings.ToLower(e.Extra), query) ||
			strings.Contains(strings.ToLower(e.Status), query) {
			filtered = append(filtered, e)
		}
	}

	// Remove orphan category headers (categories with no children after them)
	var cleaned []ResourceEntry
	for i, e := range filtered {
		if e.IsCategory {
			// Check if next entry exists and is not a category
			if i+1 < len(filtered) && !filtered[i+1].IsCategory {
				cleaned = append(cleaned, e)
			}
		} else {
			cleaned = append(cleaned, e)
		}
	}

	m.filteredEntries = cleaned
	m.selected = 0
	m.offset = 0
}

func (m *Model) moveSelection(delta int) {
	visible := m.visibleEntries()
	newSelected := m.selected + delta
	if newSelected < 0 {
		newSelected = 0
	}
	if newSelected >= len(visible) {
		newSelected = len(visible) - 1
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
	visible := m.visibleEntries()
	return m.selected < len(visible) && visible[m.selected].Selectable && !visible[m.selected].IsCategory
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
	visible := m.visibleEntries()
	if m.selected < len(visible) {
		return visible[m.selected]
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

func (m Model) startComposeProject(projectName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutAction*3)
		defer cancel()
		_, err := m.docker.StartComposeProject(ctx, projectName)
		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("Failed to start project %s: %v", projectName, err))}
		}
		return m.fetchResources()()
	}
}

func (m Model) stopComposeProject(projectName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutAction*3)
		defer cancel()
		_, err := m.docker.StopComposeProject(ctx, projectName)
		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("Failed to stop project %s: %v", projectName, err))}
		}
		return m.fetchResources()()
	}
}

func (m Model) restartComposeProject(projectName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutAction*3)
		defer cancel()
		_, err := m.docker.RestartComposeProject(ctx, projectName)
		if err != nil {
			return DataMsg{Entries: m.entries, Warnings: append(m.warnings, fmt.Sprintf("Failed to restart project %s: %v", projectName, err))}
		}
		return m.fetchResources()()
	}
}

// fetchLogs fetches container logs asynchronously.
func (m Model) fetchLogs(containerID string, tail int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutLogs)
		defer cancel()
		entries, err := m.docker.GetContainerLogs(ctx, containerID, tail)
		return LogDataMsg{Entries: entries, Err: err}
	}
}

// startLogStream starts following log output.
func (m *Model) startLogStream() tea.Cmd {
	ctx := context.Background()
	logCh, errCh, cancel := m.docker.StreamContainerLogs(ctx, m.logContainerID)
	m.logCancelFn = cancel
	m.logFollowing = true

	return func() tea.Msg {
		// Read from both channels
		select {
		case entry, ok := <-logCh:
			if !ok {
				return LogStreamErrMsg{Err: nil}
			}
			return LogStreamMsg{Entry: entry}
		case err := <-errCh:
			return LogStreamErrMsg{Err: err}
		}
	}
}

// continueLogStream continues reading from the stream.
func (m Model) continueLogStream() tea.Cmd {
	if !m.logFollowing || m.logCancelFn == nil {
		return nil
	}
	ctx := context.Background()
	logCh, errCh, cancel := m.docker.StreamContainerLogs(ctx, m.logContainerID)
	// Cancel previous stream first
	if m.logCancelFn != nil {
		m.logCancelFn()
	}
	m.logCancelFn = cancel

	return func() tea.Msg {
		select {
		case entry, ok := <-logCh:
			if !ok {
				return LogStreamErrMsg{Err: nil}
			}
			return LogStreamMsg{Entry: entry}
		case err := <-errCh:
			return LogStreamErrMsg{Err: err}
		}
	}
}

// logViewportHeight returns how many log lines fit in the viewport.
func (m Model) logViewportHeight() int {
	h := m.height - 6 // header + footer
	if h < 5 {
		h = 5
	}
	return h
}

// updateLogsView handles key events in logs view mode.
func (m Model) updateLogsView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Log filter mode
	if m.logFiltering {
		switch msg.Type {
		case tea.KeyEscape:
			m.logFiltering = false
			m.logFilterText = ""
			return m, nil
		case tea.KeyEnter:
			m.logFiltering = false
			return m, nil
		case tea.KeyBackspace:
			if len(m.logFilterText) > 0 {
				m.logFilterText = m.logFilterText[:len(m.logFilterText)-1]
			}
			return m, nil
		case tea.KeyRunes:
			m.logFilterText += string(msg.Runes)
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "esc", "q":
		// Stop stream if following
		if m.logCancelFn != nil {
			m.logCancelFn()
			m.logCancelFn = nil
		}
		m.viewMode = viewList
		m.logFollowing = false
		m.logEntries = nil
		m.logFilterText = ""
		m.logFiltering = false
		return m, nil
	case "up", "k":
		if m.logOffset > 0 {
			m.logOffset--
		}
		m.logFollowing = false
	case "down", "j":
		maxOffset := max(0, len(m.visibleLogEntries())-m.logViewportHeight())
		if m.logOffset < maxOffset {
			m.logOffset++
		}
	case "G":
		m.logOffset = max(0, len(m.visibleLogEntries())-m.logViewportHeight())
	case "g":
		m.logOffset = 0
	case "f":
		if m.logFollowing {
			// Stop following
			if m.logCancelFn != nil {
				m.logCancelFn()
				m.logCancelFn = nil
			}
			m.logFollowing = false
		} else {
			return m, m.startLogStream()
		}
	case "/":
		m.logFiltering = true
		m.logFilterText = ""
	}
	return m, nil
}

// visibleLogEntries returns log entries filtered by logFilterText.
func (m Model) visibleLogEntries() []docker.LogEntry {
	if m.logFilterText == "" {
		return m.logEntries
	}
	query := strings.ToLower(m.logFilterText)
	var filtered []docker.LogEntry
	for _, e := range m.logEntries {
		if strings.Contains(strings.ToLower(e.Content), query) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	if m.loading {
		spinner := spinnerChars[m.spinnerFrame%len(spinnerChars)]
		return fmt.Sprintf("%s Loading Docker resources...", spinner)
	}

	// Logs view mode
	if m.viewMode == viewLogs {
		return m.renderLogsView()
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

	// Filter bar
	if m.filtering || m.filterText != "" {
		filterDisplay := "Filter: " + m.filterText
		if m.filtering {
			filterDisplay += "█"
		}
		b.WriteString(styles.Info.Render(filterDisplay))
		b.WriteString("\n\n")
	}

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

	visible := m.visibleEntries()

	// Render entries
	for i := m.offset; i < len(visible) && i < m.offset+viewport; i++ {
		entry := visible[i]
		var line string

		if entry.IsCategory {
			line = styles.Section.Render(fmt.Sprintf("► %s", entry.Name))
		} else if entry.IsProjectHeader {
			line = styles.Section.Render(fmt.Sprintf("  [compose] %s", entry.Name))
		} else {
			name := entry.Name
			maxNameLen := 30
			if entry.ComposeProject != "" {
				maxNameLen = 26 // shorter to account for indent
			}
			if len(name) > maxNameLen {
				name = name[:maxNameLen-3] + "..."
			}

			sizeStr := ""
			if entry.Size > 0 {
				sizeStr = format.Size(uint64(entry.Size))
			}

			statusStr := ""
			if entry.IsUnused || entry.IsDangling {
				statusStr = styles.Warning.Render(" (unused)")
			}

			// Metrics suffix for running containers
			metricsStr := ""
			if entry.Type == ResourceContainers && !entry.IsUnused && entry.CPUPercent > 0 {
				metricsStr = styles.Label.Render(fmt.Sprintf("  CPU: %.1f%%  MEM: %s/%s",
					entry.CPUPercent,
					format.Size(entry.MemUsage),
					format.Size(entry.MemLimit)))
			}

			// Align size to right
			sizeWidth := 10
			if len(sizeStr) < sizeWidth {
				sizeStr = strings.Repeat(" ", sizeWidth-len(sizeStr)) + sizeStr
			}

			if entry.ComposeProject != "" {
				// Indent compose-grouped containers with service label
				serviceLabel := ""
				if entry.ComposeService != "" {
					serviceLabel = fmt.Sprintf(" (%s)", entry.ComposeService)
				}
				line = fmt.Sprintf("    %-28s%s %s%s%s", name, serviceLabel, styles.Label.Render(sizeStr), statusStr, metricsStr)
			} else {
				line = fmt.Sprintf("%-32s %s%s%s", name, styles.Label.Render(sizeStr), statusStr, metricsStr)
			}
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
	b.WriteString(styles.Help.Render("↑↓/jk: navigate | /: filter | l: logs | s/t/r: start/stop/restart | x: shell | y: copy | d: delete | q: quit"))

	return b.String()
}

// renderLogsView renders the logs viewer.
func (m Model) renderLogsView() string {
	var b strings.Builder

	// Header
	followStr := ""
	if m.logFollowing {
		followStr = " [FOLLOWING]"
	}
	title := fmt.Sprintf("Logs: %s%s", m.logContainer, followStr)
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")

	// Filter bar for logs
	if m.logFiltering || m.logFilterText != "" {
		filterDisplay := "Filter: " + m.logFilterText
		if m.logFiltering {
			filterDisplay += "█"
		}
		b.WriteString(styles.Info.Render(filterDisplay))
		b.WriteString("\n")
	}

	logEntries := m.visibleLogEntries()
	viewport := m.logViewportHeight()

	if len(logEntries) == 0 {
		b.WriteString(styles.Info.Render("  No log entries"))
		b.WriteString("\n")
	} else {
		end := m.logOffset + viewport
		if end > len(logEntries) {
			end = len(logEntries)
		}
		start := m.logOffset
		if start < 0 {
			start = 0
		}

		for i := start; i < end; i++ {
			entry := logEntries[i]
			ts := entry.Timestamp.Format("2006-01-02 15:04:05")
			line := fmt.Sprintf("%s  %-6s  %s", ts, entry.Stream, entry.Content)
			if entry.Stream == "stderr" {
				line = styles.Error.Render(line)
			} else {
				line = styles.Normal.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Status
	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(styles.Info.Render("  " + m.statusMessage))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(strings.Repeat("─", 60))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render("↑↓/jk: scroll | g/G: top/bottom | f: follow | /: filter | esc: back"))

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
