package docker

import (
	"context"
	"io"
	"os"
	"os/signal"

	"github.com/docker/docker/api/types/container"

	"golang.org/x/term"
)

// DockerExecCommand implements tea.ExecCommand for interactive container exec sessions.
// It handles terminal raw mode, I/O streaming, and SIGWINCH resize forwarding.
type DockerExecCommand struct {
	api         DockerAPI
	containerID string
	shell       string
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
}

// NewDockerExecCommand creates a new exec command for the given container and shell.
func NewDockerExecCommand(api DockerAPI, containerID, shell string) *DockerExecCommand {
	return &DockerExecCommand{
		api:         api,
		containerID: containerID,
		shell:       shell,
	}
}

// SetStdin sets the stdin reader (called by tea.Exec).
func (d *DockerExecCommand) SetStdin(r io.Reader) {
	d.stdin = r
}

// SetStdout sets the stdout writer (called by tea.Exec).
func (d *DockerExecCommand) SetStdout(w io.Writer) {
	d.stdout = w
}

// SetStderr sets the stderr writer (called by tea.Exec).
func (d *DockerExecCommand) SetStderr(w io.Writer) {
	d.stderr = w
}

// Run executes the interactive shell session inside the container.
// It handles:
//   - Creating and attaching to an exec instance
//   - Setting the terminal to raw mode
//   - SIGWINCH resize forwarding
//   - Bidirectional I/O streaming
//   - Clean terminal restoration on exit
func (d *DockerExecCommand) Run() error {
	ctx := context.Background()

	// Create exec instance
	execConfig := container.ExecOptions{
		Cmd:          []string{d.shell},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	}

	createResp, err := d.api.ContainerExecCreate(ctx, d.containerID, execConfig)
	if err != nil {
		return err
	}

	// Attach to the exec instance
	attachResp, err := d.api.ContainerExecAttach(ctx, createResp.ID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		return err
	}
	defer attachResp.Close()

	// Set terminal to raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer func() {
		_ = term.Restore(fd, oldState)
	}()

	// Start SIGWINCH resize monitoring
	done := make(chan struct{})
	defer close(done)
	go monitorResize(ctx, d.api, createResp.ID, fd, done)

	// Do initial resize before starting I/O
	resizeExec(ctx, d.api, createResp.ID, fd)

	// Determine stdin/stdout to use
	stdin := d.stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := d.stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	// Start output copy goroutine
	outputDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(stdout, attachResp.Reader)
		close(outputDone)
	}()

	// Start input copy goroutine
	go func() {
		_, _ = io.Copy(attachResp.Conn, stdin)
		_ = attachResp.CloseWrite()
	}()

	// Wait for output to finish (exec process exited, reader got EOF)
	<-outputDone

	return nil
}

// monitorResize watches for terminal resize signals and propagates them to the exec session.
func monitorResize(ctx context.Context, api DockerAPI, execID string, fd int, done <-chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	startResizeListener(sigCh)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-sigCh:
			resizeExec(ctx, api, execID, fd)
		case <-done:
			return
		case <-ctx.Done():
			return
		}
	}
}

// resizeExec sends the current terminal size to the exec session (best-effort).
func resizeExec(ctx context.Context, api DockerAPI, execID string, fd int) {
	width, height, err := term.GetSize(fd)
	if err != nil {
		return // best-effort
	}
	_ = api.ContainerExecResize(ctx, execID, container.ResizeOptions{
		Height: uint(height),
		Width:  uint(width),
	})
}
