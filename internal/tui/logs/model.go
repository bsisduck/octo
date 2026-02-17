package logs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/bsisduck/octo/internal/docker"
	"github.com/bsisduck/octo/internal/ui/styles"
)

// Messages for async operations

// InitialLogsMsg carries the result of the initial log fetch.
type InitialLogsMsg struct {
	Entries []docker.LogEntry
	Err     error
}

// StreamLogMsg carries a single streamed log entry.
type StreamLogMsg struct {
	Entry docker.LogEntry
}

// StreamErrMsg signals a stream error or end-of-stream.
type StreamErrMsg struct {
	Err error
}

// exportDoneMsg carries the result of the export operation.
type exportDoneMsg struct {
	path    string
	count   int
	err     error
}

// clearStatusMsg clears the status message after a timeout.
type clearStatusMsg struct{}

// Model is a Bubble Tea model for viewing container logs with follow,
// search, and export functionality, backed by a ring buffer.
type Model struct {
	docker        docker.DockerService
	containerID   string
	containerName string

	buffer    *RingBuffer
	viewLines []string // current visible lines (from buffer, possibly filtered)
	offset    int      // scroll position in viewLines
	width     int
	height    int

	following bool // auto-scroll to latest line

	filterText    string
	filtering     bool   // currently typing in filter input
	useRegex      bool   // regex vs text search toggle
	compiledRegex *regexp.Regexp

	err               error
	statusMessage     string
	truncationWarning string

	logCancelFn func() // cancel for active stream goroutine
}

// New creates a logs model for the given container.
func New(service docker.DockerService, containerID, containerName string) Model {
	return Model{
		docker:        service,
		containerID:   containerID,
		containerName: containerName,
		buffer:        NewRingBuffer(DefaultCapacity),
		following:     true,
	}
}

// Init starts the initial log fetch.
func (m Model) Init() tea.Cmd {
	return m.fetchInitialLogs()
}

// fetchInitialLogs fetches the last 500 lines and returns an InitialLogsMsg.
func (m Model) fetchInitialLogs() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), docker.TimeoutLogs)
		defer cancel()
		entries, err := m.docker.GetContainerLogs(ctx, m.containerID, 500)
		return InitialLogsMsg{Entries: entries, Err: err}
	}
}

// startStream begins following the container log stream.
func (m *Model) startStream() tea.Cmd {
	ctx := context.Background()
	logCh, errCh, cancel := m.docker.StreamContainerLogs(ctx, m.containerID)
	m.logCancelFn = cancel

	return func() tea.Msg {
		select {
		case entry, ok := <-logCh:
			if !ok {
				return StreamErrMsg{Err: nil}
			}
			return StreamLogMsg{Entry: entry}
		case err := <-errCh:
			return StreamErrMsg{Err: err}
		}
	}
}

// continueStream reads the next entry from the active stream.
func (m Model) continueStream() tea.Cmd {
	if m.logCancelFn == nil {
		return nil
	}
	ctx := context.Background()
	logCh, errCh, cancel := m.docker.StreamContainerLogs(ctx, m.containerID)
	m.logCancelFn = cancel

	return func() tea.Msg {
		select {
		case entry, ok := <-logCh:
			if !ok {
				return StreamErrMsg{Err: nil}
			}
			return StreamLogMsg{Entry: entry}
		case err := <-errCh:
			return StreamErrMsg{Err: err}
		}
	}
}

func formatLogLine(e docker.LogEntry) string {
	ts := e.Timestamp.Format("2006-01-02 15:04:05")
	return fmt.Sprintf("%s  %-6s  %s", ts, e.Stream, e.Content)
}

// refreshViewLines rebuilds viewLines from the buffer, applying filter if active.
func (m *Model) refreshViewLines() {
	lines := m.buffer.Lines()
	if lines == nil {
		m.viewLines = nil
		return
	}

	if m.filterText == "" {
		m.viewLines = lines
		return
	}

	if m.useRegex && m.compiledRegex != nil {
		var filtered []string
		for _, line := range lines {
			if m.compiledRegex.MatchString(line) {
				filtered = append(filtered, line)
			}
		}
		m.viewLines = filtered
		return
	}

	// Plain text search (case-insensitive)
	query := strings.ToLower(m.filterText)
	var filtered []string
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), query) {
			filtered = append(filtered, line)
		}
	}
	m.viewLines = filtered
}

// updateTruncationWarning updates the truncation warning based on dropped lines.
func (m *Model) updateTruncationWarning() {
	dropped := m.buffer.Dropped()
	if dropped > 0 {
		m.truncationWarning = fmt.Sprintf("Logs truncated: oldest %d lines dropped", dropped)
	} else {
		m.truncationWarning = ""
	}
}

// viewportHeight returns the number of log lines that fit in the viewport.
func (m Model) viewportHeight() int {
	h := m.height - 7 // header + truncation + filter + footer + padding
	if h < 5 {
		h = 5
	}
	return h
}

// scrollToBottom moves offset to show the latest lines.
func (m *Model) scrollToBottom() {
	maxOffset := len(m.viewLines) - m.viewportHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	m.offset = maxOffset
}

// Update handles messages and key events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case InitialLogsMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		for _, entry := range msg.Entries {
			m.buffer.Append(formatLogLine(entry))
		}
		m.refreshViewLines()
		if m.following {
			m.scrollToBottom()
		}
		// Start streaming
		return m, m.startStream()

	case StreamLogMsg:
		m.buffer.Append(formatLogLine(msg.Entry))
		m.refreshViewLines()
		m.updateTruncationWarning()
		if m.following {
			m.scrollToBottom()
		}
		// Continue reading stream
		return m, m.continueStream()

	case StreamErrMsg:
		if msg.Err != nil {
			m.statusMessage = fmt.Sprintf("Stream error: %v", msg.Err)
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		// Stream ended (channel closed)
		return m, nil

	case exportDoneMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Export error: %v", msg.err)
		} else {
			m.statusMessage = fmt.Sprintf("Exported %d lines to %s", msg.count, msg.path)
		}
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// handleKeyMsg dispatches key events based on current mode.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filtering {
		return m.handleFilterKey(msg)
	}
	return m.handleNormalKey(msg)
}

// handleFilterKey handles key events while in filter input mode.
func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.filtering = false
		// Compile regex if in regex mode
		if m.useRegex && m.filterText != "" {
			re, err := regexp.Compile(m.filterText)
			if err != nil {
				m.compiledRegex = nil
				m.statusMessage = fmt.Sprintf("Invalid regex: %v", err)
				return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
					return clearStatusMsg{}
				})
			}
			m.compiledRegex = re
		}
		m.refreshViewLines()
		if m.following {
			m.scrollToBottom()
		}
		return m, nil

	case tea.KeyEscape:
		m.filtering = false
		m.filterText = ""
		m.compiledRegex = nil
		m.refreshViewLines()
		m.offset = 0
		return m, nil

	case tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
		}
		return m, nil

	case tea.KeyRunes:
		m.filterText += string(msg.Runes)
		return m, nil
	}

	return m, nil
}

// handleNormalKey handles key events in normal (non-filter) mode.
func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.offset > 0 {
			m.offset--
		}
		m.following = false
		return m, nil

	case "down", "j":
		maxOffset := len(m.viewLines) - m.viewportHeight()
		if maxOffset < 0 {
			maxOffset = 0
		}
		if m.offset < maxOffset {
			m.offset++
		}
		// If scrolled to very bottom, re-enable following
		if m.offset >= maxOffset {
			m.following = true
		}
		return m, nil

	case "g":
		m.offset = 0
		m.following = false
		return m, nil

	case "G":
		m.scrollToBottom()
		m.following = true
		return m, nil

	case "f":
		m.following = !m.following
		if m.following {
			m.scrollToBottom()
		}
		return m, nil

	case "/":
		m.filtering = true
		m.filterText = ""
		m.useRegex = false
		m.compiledRegex = nil
		return m, nil

	case "ctrl+r":
		m.filtering = true
		m.filterText = ""
		m.useRegex = !m.useRegex
		m.compiledRegex = nil
		return m, nil

	case "e":
		return m, m.exportLogs()

	case "q", "esc":
		if m.logCancelFn != nil {
			m.logCancelFn()
			m.logCancelFn = nil
		}
		return m, tea.Quit
	}

	return m, nil
}

// exportLogs writes all buffered lines to ~/.octo/logs/{containerID}.log.
func (m Model) exportLogs() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return exportDoneMsg{err: fmt.Errorf("home dir: %w", err)}
		}

		dir := filepath.Join(home, ".octo", "logs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return exportDoneMsg{err: fmt.Errorf("create dir: %w", err)}
		}

		path := filepath.Join(dir, m.containerID+".log")
		lines := m.buffer.Lines()

		f, err := os.Create(path)
		if err != nil {
			return exportDoneMsg{err: fmt.Errorf("create file: %w", err)}
		}
		defer f.Close()

		for _, line := range lines {
			if _, err := fmt.Fprintln(f, line); err != nil {
				return exportDoneMsg{err: fmt.Errorf("write: %w", err)}
			}
		}

		return exportDoneMsg{path: path, count: len(lines)}
	}
}

// View renders the logs viewer.
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.", m.err)
	}

	var b strings.Builder

	// Header
	followStr := ""
	if m.following {
		followStr = " [FOLLOWING]"
	}
	title := fmt.Sprintf("Logs: %s%s", m.containerName, followStr)
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("\u2500", 60))
	b.WriteString("\n")

	// Truncation warning
	if m.truncationWarning != "" {
		b.WriteString(styles.Warning.Render("\u26a0 " + m.truncationWarning))
		b.WriteString("\n")
	}

	// Filter bar
	if m.filtering || m.filterText != "" {
		filterDisplay := "Filter: " + m.filterText
		if m.filtering {
			filterDisplay += "\u2588"
		}
		if m.useRegex {
			filterDisplay += " [regex]"
		}
		b.WriteString(styles.Info.Render(filterDisplay))
		b.WriteString("\n")
	}

	// Log lines viewport
	viewport := m.viewportHeight()

	if len(m.viewLines) == 0 {
		b.WriteString(styles.Info.Render("  No log entries"))
		b.WriteString("\n")
	} else {
		start := m.offset
		if start < 0 {
			start = 0
		}
		end := start + viewport
		if end > len(m.viewLines) {
			end = len(m.viewLines)
		}

		for i := start; i < end; i++ {
			line := m.viewLines[i]
			// Color stderr lines differently
			if strings.Contains(line, "stderr") {
				line = styles.Error.Render(line)
			} else {
				line = styles.Normal.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Status/error line
	if m.statusMessage != "" {
		b.WriteString("\n")
		b.WriteString(styles.Info.Render("  " + m.statusMessage))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(strings.Repeat("\u2500", 60))
	b.WriteString("\n")
	b.WriteString(styles.Help.Render(
		"\u2191\u2193/jk: scroll | g/G: top/bottom | f: follow | /: filter | ctrl+r: regex | e: export | q: back",
	))

	return b.String()
}
