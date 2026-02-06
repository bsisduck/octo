package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bsisduck/octo/internal/docker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Docker system status and resource usage",
	Long: `Display real-time Docker system status including:
- Running and stopped containers
- Image count and total size
- Volume usage
- Network status
- System resource consumption`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolP("watch", "w", false, "Continuously update status")
}

func runStatus(cmd *cobra.Command, args []string) error {
	watch, _ := cmd.Flags().GetBool("watch")

	if watch {
		// Launch TUI for continuous monitoring
		p := tea.NewProgram(newStatusModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("running status: %w", err)
		}
	} else {
		// One-shot status display
		return printStatusOnce()
	}
	return nil
}

func printStatusOnce() error {
	client, err := docker.NewClient()
	if err != nil {
		return fmt.Errorf("error connecting to Docker: %w", err)
	}
	defer client.Close()

	ctx := context.Background()

	info, err := client.GetServerInfo(ctx)
	if err != nil {
		return fmt.Errorf("error getting Docker info: %w", err)
	}

	diskUsage, err := client.GetDiskUsage(ctx)
	if err != nil {
		return fmt.Errorf("error getting disk usage: %w", err)
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	highlightStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	fmt.Println()
	fmt.Println(titleStyle.Render("Docker System Status"))
	fmt.Println(strings.Repeat("─", 40))

	// Server info
	fmt.Printf("%s %s\n", labelStyle.Render("Server Version:"), valueStyle.Render(info.ServerVersion))
	fmt.Printf("%s %s (%s)\n", labelStyle.Render("OS/Arch:"), valueStyle.Render(info.OperatingSystem), info.Architecture)
	fmt.Println()

	// Containers
	fmt.Println(titleStyle.Render("Containers"))
	fmt.Printf("  %s %s  ", labelStyle.Render("Running:"), highlightStyle.Render(fmt.Sprintf("%d", info.ContainersRunning)))
	fmt.Printf("%s %s  ", labelStyle.Render("Paused:"), valueStyle.Render(fmt.Sprintf("%d", info.ContainersPaused)))
	fmt.Printf("%s %s\n", labelStyle.Render("Stopped:"), warnStyle.Render(fmt.Sprintf("%d", info.ContainersStopped)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(fmt.Sprintf("%d", info.Containers)))
	fmt.Println()

	// Images
	fmt.Println(titleStyle.Render("Images"))
	fmt.Printf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(fmt.Sprintf("%d", info.Images)))
	fmt.Printf("  %s %s\n", labelStyle.Render("Size:"), valueStyle.Render(humanize.Bytes(uint64(diskUsage.Images))))
	fmt.Println()

	// Volumes
	fmt.Println(titleStyle.Render("Volumes"))
	volumes, _ := client.ListVolumes(ctx)
	unusedVolumes, _ := client.GetUnusedVolumes(ctx)
	fmt.Printf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(fmt.Sprintf("%d", len(volumes))))
	fmt.Printf("  %s %s\n", labelStyle.Render("Unused:"), warnStyle.Render(fmt.Sprintf("%d", len(unusedVolumes))))
	fmt.Printf("  %s %s\n", labelStyle.Render("Size:"), valueStyle.Render(humanize.Bytes(uint64(diskUsage.Volumes))))
	fmt.Println()

	// Disk Usage Summary
	fmt.Println(titleStyle.Render("Disk Usage"))
	fmt.Printf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(humanize.Bytes(uint64(diskUsage.Total))))
	fmt.Printf("  %s %s\n", labelStyle.Render("Reclaimable:"), highlightStyle.Render(humanize.Bytes(uint64(diskUsage.TotalReclaimable))))
	fmt.Printf("  %s %s\n", labelStyle.Render("Build Cache:"), valueStyle.Render(humanize.Bytes(uint64(diskUsage.BuildCache))))
	fmt.Println()
	return nil
}

// TUI Model for continuous status monitoring
type statusModel struct {
	client      docker.DockerService
	lastUpdated time.Time
	err         error
	width       int
	height      int

	// Cached data
	containers    []docker.ContainerInfo
	images        []docker.ImageInfo
	volumes       []docker.VolumeInfo
	diskUsage     *docker.DiskUsageInfo
	serverVersion string
	osInfo        string
}

type statusTickMsg time.Time
type statusDataMsg struct {
	containers    []docker.ContainerInfo
	images        []docker.ImageInfo
	volumes       []docker.VolumeInfo
	diskUsage     *docker.DiskUsageInfo
	serverVersion string
	osInfo        string
	err           error
}

func newStatusModel() statusModel {
	client, err := docker.NewClient()
	return statusModel{
		client: client,
		err:    err,
	}
}

func (m statusModel) Init() tea.Cmd {
	if m.err != nil {
		return nil
	}
	return tea.Batch(m.fetchData(), tickStatus())
}

func tickStatus() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return statusTickMsg(t)
	})
}

func (m statusModel) fetchData() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return statusDataMsg{err: fmt.Errorf("no Docker client")}
		}

		ctx := context.Background()

		containers, _ := m.client.ListContainers(ctx, true)
		images, _ := m.client.ListImages(ctx, true)
		volumes, _ := m.client.ListVolumes(ctx)
		diskUsage, _ := m.client.GetDiskUsage(ctx)

		info, _ := m.client.GetServerInfo(ctx)

		return statusDataMsg{
			containers:    containers,
			images:        images,
			volumes:       volumes,
			diskUsage:     diskUsage,
			serverVersion: info.ServerVersion,
			osInfo:        fmt.Sprintf("%s (%s)", info.OperatingSystem, info.Architecture),
		}
	}
}

func (m statusModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			if m.client != nil {
				m.client.Close()
			}
			return m, tea.Quit
		case "r":
			return m, m.fetchData()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case statusTickMsg:
		return m, tea.Batch(m.fetchData(), tickStatus())
	case statusDataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.containers = msg.containers
			m.images = msg.images
			m.volumes = msg.volumes
			m.diskUsage = msg.diskUsage
			m.serverVersion = msg.serverVersion
			m.osInfo = msg.osInfo
			m.lastUpdated = time.Now()
		}
	}
	return m, nil
}

func (m statusModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\nPress 'q' to quit.", m.err)
	}

	// Styles
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69")).MarginBottom(1)
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Width(14)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	runningStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	stoppedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render("🐙 Octo Docker Status"))
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

	b.WriteString(sectionStyle.Render("Containers"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Running:"), runningStyle.Render(fmt.Sprintf("%d", running))))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Stopped:"), stoppedStyle.Render(fmt.Sprintf("%d", stopped))))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(fmt.Sprintf("%d", len(m.containers)))))
	b.WriteString("\n")

	// Images section
	b.WriteString(sectionStyle.Render("Images"))
	b.WriteString("\n")
	dangling := 0
	for _, img := range m.images {
		if img.Dangling {
			dangling++
		}
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(fmt.Sprintf("%d", len(m.images)))))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Dangling:"), stoppedStyle.Render(fmt.Sprintf("%d", dangling))))
	if m.diskUsage != nil {
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Size:"), valueStyle.Render(humanize.Bytes(uint64(m.diskUsage.Images)))))
	}
	b.WriteString("\n")

	// Volumes section
	b.WriteString(sectionStyle.Render("Volumes"))
	b.WriteString("\n")
	unused := 0
	for _, v := range m.volumes {
		if !v.InUse {
			unused++
		}
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(fmt.Sprintf("%d", len(m.volumes)))))
	b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Unused:"), stoppedStyle.Render(fmt.Sprintf("%d", unused))))
	if m.diskUsage != nil {
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Size:"), valueStyle.Render(humanize.Bytes(uint64(m.diskUsage.Volumes)))))
	}
	b.WriteString("\n")

	// Disk Usage
	if m.diskUsage != nil {
		b.WriteString(sectionStyle.Render("Disk Usage"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Total:"), valueStyle.Render(humanize.Bytes(uint64(m.diskUsage.Total)))))
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Reclaimable:"), runningStyle.Render(humanize.Bytes(uint64(m.diskUsage.TotalReclaimable)))))
		b.WriteString(fmt.Sprintf("  %s %s\n", labelStyle.Render("Build Cache:"), valueStyle.Render(humanize.Bytes(uint64(m.diskUsage.BuildCache)))))
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		fmt.Sprintf("Last updated: %s | Press 'r' to refresh, 'q' to quit", m.lastUpdated.Format("15:04:05"))))

	return b.String()
}
