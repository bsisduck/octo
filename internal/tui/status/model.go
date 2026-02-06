package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/ui/format"
	"github.com/bsisduck/octo/internal/ui/styles"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	client      docker.DockerService
	lastUpdated time.Time
	err         error
	width       int
	height      int
	cancelFetch context.CancelFunc

	// Cached data
	containers    []docker.ContainerInfo
	images        []docker.ImageInfo
	volumes       []docker.VolumeInfo
	diskUsage     *docker.DiskUsageInfo
	serverVersion string
	osInfo        string
	warnings      []string
}

type tickMsg time.Time

type DataMsg struct {
	Containers    []docker.ContainerInfo
	Images        []docker.ImageInfo
	Volumes       []docker.VolumeInfo
	DiskUsage     *docker.DiskUsageInfo
	ServerVersion string
	OsInfo        string
	Err           error
	Warnings      []string
}

// Exported for testing
type dataMsg = DataMsg

func New(client docker.DockerService, watch bool) Model {
	return Model{
		client: client,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.fetchData(), tickStatus())
}

func tickStatus() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) fetchData() tea.Cmd {
	// Cancel previous fetch if still running
	if m.cancelFetch != nil {
		m.cancelFetch()
	}

	ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutWatch)
	m.cancelFetch = cancel

	return func() tea.Msg {
		// No defer cancel() here because we store it in the struct
		// It will be called by the next fetchData or when the context times out

		var warnings []string

		containers, err := m.client.ListContainers(ctx, true)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("containers: %v", err))
		}
		images, err := m.client.ListImages(ctx, true)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("images: %v", err))
		}
		volumes, err := m.client.ListVolumes(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("volumes: %v", err))
		}
		diskUsage, err := m.client.GetDiskUsage(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("disk usage: %v", err))
		}

		info, err := m.client.GetServerInfo(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("server info: %v", err))
		}

		return DataMsg{
			Containers:    containers,
			Images:        images,
			Volumes:       volumes,
			DiskUsage:     diskUsage,
			ServerVersion: info.ServerVersion,
			OsInfo:        fmt.Sprintf("%s (%s)", info.OperatingSystem, info.Architecture),
			Warnings:      warnings,
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			if m.cancelFetch != nil {
				m.cancelFetch()
			}
			m.client.Close()
			return m, tea.Quit
		case "r":
			return m, m.fetchData()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, tea.Batch(m.fetchData(), tickStatus())
	case DataMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.containers = msg.Containers
			m.images = msg.Images
			m.volumes = msg.Volumes
			m.diskUsage = msg.DiskUsage
			m.serverVersion = msg.ServerVersion
			m.osInfo = msg.OsInfo
			m.warnings = msg.Warnings
			m.lastUpdated = time.Now()
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress 'q' to quit.", m.err)
	}

	var b strings.Builder

	// Header
	b.WriteString(styles.Title.Render("🐙 Octo Docker Status"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Server: %s | %s\n", m.serverVersion, m.osInfo))
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n\n")

	// Containers section
	running := 0
	stopped := 0
	for _, c := range m.containers {
		if c.State == "running" {
			running++
		} else {
			stopped++
		}
	}

	b.WriteString(styles.Section.Render("Containers"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Running:"), styles.Success.Render(fmt.Sprintf("%d", running))))
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Stopped:"), styles.Warning.Render(fmt.Sprintf("%d", stopped))))
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Total:"), styles.Value.Render(fmt.Sprintf("%d", len(m.containers)))))
	b.WriteString("\n")

	// Images section
	b.WriteString(styles.Section.Render("Images"))
	b.WriteString("\n")
	dangling := 0
	for _, img := range m.images {
		if img.Dangling {
			dangling++
		}
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Total:"), styles.Value.Render(fmt.Sprintf("%d", len(m.images)))))
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Dangling:"), styles.Warning.Render(fmt.Sprintf("%d", dangling))))
	if m.diskUsage != nil {
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Size:"), styles.Value.Render(format.Size(uint64(m.diskUsage.Images)))))
	}
	b.WriteString("\n")

	// Volumes section
	b.WriteString(styles.Section.Render("Volumes"))
	b.WriteString("\n")
	unused := 0
	for _, v := range m.volumes {
		if !v.InUse {
			unused++
		}
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Total:"), styles.Value.Render(fmt.Sprintf("%d", len(m.volumes)))))
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Unused:"), styles.Warning.Render(fmt.Sprintf("%d", unused))))
	if m.diskUsage != nil {
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Size:"), styles.Value.Render(format.Size(uint64(m.diskUsage.Volumes)))))
	}
	b.WriteString("\n")

	// Disk Usage
	if m.diskUsage != nil {
		b.WriteString(styles.Section.Render("Disk Usage"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Total:"), styles.Value.Render(format.Size(uint64(m.diskUsage.Total)))))
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Reclaimable:"), styles.Success.Render(format.Size(uint64(m.diskUsage.TotalReclaimable)))))
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Label.Render("Build Cache:"), styles.Value.Render(format.Size(uint64(m.diskUsage.BuildCache)))))
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

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render(
		fmt.Sprintf("Last updated: %s | Press 'r' to refresh, 'q' to quit", m.lastUpdated.Format("15:04:05"))))

	return b.String()
}
