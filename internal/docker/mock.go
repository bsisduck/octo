package docker

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
)

var testTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// TestTime returns the time used in mock data for test assertions.
func TestTime() time.Time { return testTime }

// Compile-time interface check
var _ DockerService = (*MockDockerService)(nil)

// MockDockerService is a hand-rolled mock implementation of DockerService.
// Use it in tests by setting the function fields to return specific values.
type MockDockerService struct {
	PingFn                  func(ctx context.Context) error
	CloseFn                 func() error
	GetServerInfoFn         func(ctx context.Context) (system.Info, error)
	ListContainersFn        func(ctx context.Context, all bool) ([]ContainerInfo, error)
	ListImagesFn            func(ctx context.Context, all bool) ([]ImageInfo, error)
	ListVolumesFn           func(ctx context.Context) ([]VolumeInfo, error)
	ListNetworksFn          func(ctx context.Context) ([]NetworkInfo, error)
	GetDanglingImagesFn     func(ctx context.Context) ([]ImageInfo, error)
	GetStoppedContainersFn  func(ctx context.Context) ([]ContainerInfo, error)
	GetUnusedVolumesFn      func(ctx context.Context) ([]VolumeInfo, error)
	GetDiskUsageFn          func(ctx context.Context) (*DiskUsageInfo, error)
	RemoveContainerFn       func(ctx context.Context, id string, force bool) error
	RemoveImageFn           func(ctx context.Context, id string, force bool) error
	RemoveVolumeFn          func(ctx context.Context, name string, force bool) error
	RemoveNetworkFn         func(ctx context.Context, id string) error
	StartContainerFn        func(ctx context.Context, id string) error
	StopContainerFn         func(ctx context.Context, id string) error
	RestartContainerFn      func(ctx context.Context, id string) error
	PruneContainersFn       func(ctx context.Context) (uint64, error)
	PruneImagesFn           func(ctx context.Context, all bool) (uint64, error)
	PruneVolumesFn          func(ctx context.Context) (uint64, error)
	PruneNetworksFn         func(ctx context.Context) error
	PruneBuildCacheFn       func(ctx context.Context, all bool) (uint64, error)
	RemoveContainerDryRunFn func(ctx context.Context, id string) (ConfirmationInfo, error)
	RemoveImageDryRunFn     func(ctx context.Context, id string) (ConfirmationInfo, error)
	RemoveVolumeDryRunFn    func(ctx context.Context, name string) (ConfirmationInfo, error)
	RemoveNetworkDryRunFn   func(ctx context.Context, id string) (ConfirmationInfo, error)
	PruneContainersDryRunFn func(ctx context.Context) (ConfirmationInfo, error)
	PruneImagesDryRunFn     func(ctx context.Context, all bool) (ConfirmationInfo, error)
	PruneVolumesDryRunFn    func(ctx context.Context) (ConfirmationInfo, error)
	PruneNetworksDryRunFn   func(ctx context.Context) (ConfirmationInfo, error)
	GetContainerLogsFn      func(ctx context.Context, containerID string, tail int) ([]LogEntry, error)
	StreamContainerLogsFn   func(ctx context.Context, containerID string) (<-chan LogEntry, <-chan error, func())
	GetContainerStatsFn     func(ctx context.Context, containerID string) (*ContainerMetrics, error)
	StartComposeProjectFn   func(ctx context.Context, projectName string) (int, error)
	StopComposeProjectFn    func(ctx context.Context, projectName string) (int, error)
	RestartComposeProjectFn func(ctx context.Context, projectName string) (int, error)
	APIFn                   func() DockerAPI
}

func (m *MockDockerService) Ping(ctx context.Context) error {
	if m.PingFn != nil {
		return m.PingFn(ctx)
	}
	return nil
}

func (m *MockDockerService) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

func (m *MockDockerService) GetServerInfo(ctx context.Context) (system.Info, error) {
	if m.GetServerInfoFn != nil {
		return m.GetServerInfoFn(ctx)
	}
	return system.Info{}, nil
}

func (m *MockDockerService) ListContainers(ctx context.Context, all bool) ([]ContainerInfo, error) {
	if m.ListContainersFn != nil {
		return m.ListContainersFn(ctx, all)
	}
	return nil, nil
}

func (m *MockDockerService) ListImages(ctx context.Context, all bool) ([]ImageInfo, error) {
	if m.ListImagesFn != nil {
		return m.ListImagesFn(ctx, all)
	}
	return nil, nil
}

func (m *MockDockerService) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	if m.ListVolumesFn != nil {
		return m.ListVolumesFn(ctx)
	}
	return nil, nil
}

func (m *MockDockerService) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	if m.ListNetworksFn != nil {
		return m.ListNetworksFn(ctx)
	}
	return nil, nil
}

func (m *MockDockerService) GetDanglingImages(ctx context.Context) ([]ImageInfo, error) {
	if m.GetDanglingImagesFn != nil {
		return m.GetDanglingImagesFn(ctx)
	}
	return nil, nil
}

func (m *MockDockerService) GetStoppedContainers(ctx context.Context) ([]ContainerInfo, error) {
	if m.GetStoppedContainersFn != nil {
		return m.GetStoppedContainersFn(ctx)
	}
	return nil, nil
}

func (m *MockDockerService) GetUnusedVolumes(ctx context.Context) ([]VolumeInfo, error) {
	if m.GetUnusedVolumesFn != nil {
		return m.GetUnusedVolumesFn(ctx)
	}
	return nil, nil
}

func (m *MockDockerService) GetDiskUsage(ctx context.Context) (*DiskUsageInfo, error) {
	if m.GetDiskUsageFn != nil {
		return m.GetDiskUsageFn(ctx)
	}
	return &DiskUsageInfo{}, nil
}

func (m *MockDockerService) RemoveContainer(ctx context.Context, id string, force bool) error {
	if m.RemoveContainerFn != nil {
		return m.RemoveContainerFn(ctx, id, force)
	}
	return nil
}

func (m *MockDockerService) RemoveImage(ctx context.Context, id string, force bool) error {
	if m.RemoveImageFn != nil {
		return m.RemoveImageFn(ctx, id, force)
	}
	return nil
}

func (m *MockDockerService) RemoveVolume(ctx context.Context, name string, force bool) error {
	if m.RemoveVolumeFn != nil {
		return m.RemoveVolumeFn(ctx, name, force)
	}
	return nil
}

func (m *MockDockerService) RemoveNetwork(ctx context.Context, id string) error {
	if m.RemoveNetworkFn != nil {
		return m.RemoveNetworkFn(ctx, id)
	}
	return nil
}

func (m *MockDockerService) StartContainer(ctx context.Context, id string) error {
	if m.StartContainerFn != nil {
		return m.StartContainerFn(ctx, id)
	}
	return nil
}

func (m *MockDockerService) StopContainer(ctx context.Context, id string) error {
	if m.StopContainerFn != nil {
		return m.StopContainerFn(ctx, id)
	}
	return nil
}

func (m *MockDockerService) RestartContainer(ctx context.Context, id string) error {
	if m.RestartContainerFn != nil {
		return m.RestartContainerFn(ctx, id)
	}
	return nil
}

func (m *MockDockerService) PruneContainers(ctx context.Context) (uint64, error) {
	if m.PruneContainersFn != nil {
		return m.PruneContainersFn(ctx)
	}
	return 0, nil
}

func (m *MockDockerService) PruneImages(ctx context.Context, all bool) (uint64, error) {
	if m.PruneImagesFn != nil {
		return m.PruneImagesFn(ctx, all)
	}
	return 0, nil
}

func (m *MockDockerService) PruneVolumes(ctx context.Context) (uint64, error) {
	if m.PruneVolumesFn != nil {
		return m.PruneVolumesFn(ctx)
	}
	return 0, nil
}

func (m *MockDockerService) PruneNetworks(ctx context.Context) error {
	if m.PruneNetworksFn != nil {
		return m.PruneNetworksFn(ctx)
	}
	return nil
}

func (m *MockDockerService) PruneBuildCache(ctx context.Context, all bool) (uint64, error) {
	if m.PruneBuildCacheFn != nil {
		return m.PruneBuildCacheFn(ctx, all)
	}
	return 0, nil
}

func (m *MockDockerService) RemoveContainerDryRun(ctx context.Context, id string) (ConfirmationInfo, error) {
	if m.RemoveContainerDryRunFn != nil {
		return m.RemoveContainerDryRunFn(ctx, id)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) RemoveImageDryRun(ctx context.Context, id string) (ConfirmationInfo, error) {
	if m.RemoveImageDryRunFn != nil {
		return m.RemoveImageDryRunFn(ctx, id)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) RemoveVolumeDryRun(ctx context.Context, name string) (ConfirmationInfo, error) {
	if m.RemoveVolumeDryRunFn != nil {
		return m.RemoveVolumeDryRunFn(ctx, name)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) RemoveNetworkDryRun(ctx context.Context, id string) (ConfirmationInfo, error) {
	if m.RemoveNetworkDryRunFn != nil {
		return m.RemoveNetworkDryRunFn(ctx, id)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) PruneContainersDryRun(ctx context.Context) (ConfirmationInfo, error) {
	if m.PruneContainersDryRunFn != nil {
		return m.PruneContainersDryRunFn(ctx)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) PruneImagesDryRun(ctx context.Context, all bool) (ConfirmationInfo, error) {
	if m.PruneImagesDryRunFn != nil {
		return m.PruneImagesDryRunFn(ctx, all)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) PruneVolumesDryRun(ctx context.Context) (ConfirmationInfo, error) {
	if m.PruneVolumesDryRunFn != nil {
		return m.PruneVolumesDryRunFn(ctx)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) PruneNetworksDryRun(ctx context.Context) (ConfirmationInfo, error) {
	if m.PruneNetworksDryRunFn != nil {
		return m.PruneNetworksDryRunFn(ctx)
	}
	return ConfirmationInfo{}, nil
}

func (m *MockDockerService) GetContainerLogs(ctx context.Context, containerID string, tail int) ([]LogEntry, error) {
	if m.GetContainerLogsFn != nil {
		return m.GetContainerLogsFn(ctx, containerID, tail)
	}
	return []LogEntry{
		{Timestamp: testTime, Stream: "stdout", Content: "test log line 1"},
		{Timestamp: testTime, Stream: "stdout", Content: "test log line 2"},
		{Timestamp: testTime, Stream: "stderr", Content: "test error line"},
	}, nil
}

func (m *MockDockerService) StreamContainerLogs(ctx context.Context, containerID string) (<-chan LogEntry, <-chan error, func()) {
	if m.StreamContainerLogsFn != nil {
		return m.StreamContainerLogsFn(ctx, containerID)
	}
	logCh := make(chan LogEntry, 2)
	errCh := make(chan error)
	logCh <- LogEntry{Timestamp: testTime, Stream: "stdout", Content: "stream line 1"}
	logCh <- LogEntry{Timestamp: testTime, Stream: "stdout", Content: "stream line 2"}
	close(logCh)
	return logCh, errCh, func() {}
}

func (m *MockDockerService) GetContainerStats(ctx context.Context, containerID string) (*ContainerMetrics, error) {
	if m.GetContainerStatsFn != nil {
		return m.GetContainerStatsFn(ctx, containerID)
	}
	return &ContainerMetrics{
		ContainerID:   containerID,
		CPUPercent:    25.5,
		MemoryUsage:   1024 * 1024 * 100,
		MemoryLimit:   1024 * 1024 * 512,
		MemoryPercent: 19.53,
	}, nil
}

func (m *MockDockerService) StartComposeProject(ctx context.Context, projectName string) (int, error) {
	if m.StartComposeProjectFn != nil {
		return m.StartComposeProjectFn(ctx, projectName)
	}
	return 0, nil
}

func (m *MockDockerService) StopComposeProject(ctx context.Context, projectName string) (int, error) {
	if m.StopComposeProjectFn != nil {
		return m.StopComposeProjectFn(ctx, projectName)
	}
	return 0, nil
}

func (m *MockDockerService) RestartComposeProject(ctx context.Context, projectName string) (int, error) {
	if m.RestartComposeProjectFn != nil {
		return m.RestartComposeProjectFn(ctx, projectName)
	}
	return 0, nil
}

func (m *MockDockerService) API() DockerAPI {
	if m.APIFn != nil {
		return m.APIFn()
	}
	return &MockDockerAPI{}
}

// Compile-time interface check
var _ DockerAPI = (*MockDockerAPI)(nil)

// MockDockerAPI is a hand-rolled mock implementation of DockerAPI.
// Use it in tests to verify Client's data transformation logic.
type MockDockerAPI struct {
	PingFn                  func(ctx context.Context) (types.Ping, error)
	InfoFn                  func(ctx context.Context) (system.Info, error)
	CloseFn                 func() error
	ContainerListFn         func(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ImageListFn             func(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	VolumeListFn            func(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error)
	NetworkListFn           func(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
	DiskUsageFn             func(ctx context.Context, options types.DiskUsageOptions) (types.DiskUsage, error)
	ContainerRemoveFn       func(ctx context.Context, containerID string, options container.RemoveOptions) error
	ImageRemoveFn           func(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	VolumeRemoveFn          func(ctx context.Context, volumeID string, force bool) error
	NetworkRemoveFn         func(ctx context.Context, networkID string) error
	ContainerStartFn        func(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStopFn         func(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRestartFn      func(ctx context.Context, containerID string, options container.StopOptions) error
	ContainersPruneFn       func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error)
	ImagesPruneFn           func(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error)
	VolumesPruneFn          func(ctx context.Context, pruneFilters filters.Args) (volume.PruneReport, error)
	NetworksPruneFn         func(ctx context.Context, pruneFilters filters.Args) (network.PruneReport, error)
	BuildCachePruneFn       func(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error)
	ContainerLogsFn         func(ctx context.Context, ctr string, options container.LogsOptions) (io.ReadCloser, error)
	ContainerStatsOneShotFn func(ctx context.Context, containerID string) (container.StatsResponseReader, error)
	ContainerExecCreateFn   func(ctx context.Context, containerID string, options container.ExecOptions) (types.IDResponse, error)
	ContainerExecAttachFn   func(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error)
	ContainerExecResizeFn   func(ctx context.Context, execID string, options container.ResizeOptions) error
	ContainerExecInspectFn  func(ctx context.Context, execID string) (container.ExecInspect, error)
	ContainerExecStartFn    func(ctx context.Context, execID string, config container.ExecStartOptions) error
}

func (m *MockDockerAPI) Ping(ctx context.Context) (types.Ping, error) {
	if m.PingFn != nil {
		return m.PingFn(ctx)
	}
	return types.Ping{}, nil
}

func (m *MockDockerAPI) Info(ctx context.Context) (system.Info, error) {
	if m.InfoFn != nil {
		return m.InfoFn(ctx)
	}
	return system.Info{}, nil
}

func (m *MockDockerAPI) Close() error {
	if m.CloseFn != nil {
		return m.CloseFn()
	}
	return nil
}

func (m *MockDockerAPI) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	if m.ContainerListFn != nil {
		return m.ContainerListFn(ctx, options)
	}
	return nil, nil
}

func (m *MockDockerAPI) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	if m.ImageListFn != nil {
		return m.ImageListFn(ctx, options)
	}
	return nil, nil
}

func (m *MockDockerAPI) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	if m.VolumeListFn != nil {
		return m.VolumeListFn(ctx, options)
	}
	return volume.ListResponse{}, nil
}

func (m *MockDockerAPI) NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error) {
	if m.NetworkListFn != nil {
		return m.NetworkListFn(ctx, options)
	}
	return nil, nil
}

func (m *MockDockerAPI) DiskUsage(ctx context.Context, options types.DiskUsageOptions) (types.DiskUsage, error) {
	if m.DiskUsageFn != nil {
		return m.DiskUsageFn(ctx, options)
	}
	return types.DiskUsage{}, nil
}

func (m *MockDockerAPI) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	if m.ContainerRemoveFn != nil {
		return m.ContainerRemoveFn(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerAPI) ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	if m.ImageRemoveFn != nil {
		return m.ImageRemoveFn(ctx, imageID, options)
	}
	return nil, nil
}

func (m *MockDockerAPI) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	if m.VolumeRemoveFn != nil {
		return m.VolumeRemoveFn(ctx, volumeID, force)
	}
	return nil
}

func (m *MockDockerAPI) NetworkRemove(ctx context.Context, networkID string) error {
	if m.NetworkRemoveFn != nil {
		return m.NetworkRemoveFn(ctx, networkID)
	}
	return nil
}

func (m *MockDockerAPI) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	if m.ContainerStartFn != nil {
		return m.ContainerStartFn(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerAPI) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	if m.ContainerStopFn != nil {
		return m.ContainerStopFn(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerAPI) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	if m.ContainerRestartFn != nil {
		return m.ContainerRestartFn(ctx, containerID, options)
	}
	return nil
}

func (m *MockDockerAPI) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error) {
	if m.ContainersPruneFn != nil {
		return m.ContainersPruneFn(ctx, pruneFilters)
	}
	return container.PruneReport{}, nil
}

func (m *MockDockerAPI) ImagesPrune(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error) {
	if m.ImagesPruneFn != nil {
		return m.ImagesPruneFn(ctx, pruneFilters)
	}
	return image.PruneReport{}, nil
}

func (m *MockDockerAPI) VolumesPrune(ctx context.Context, pruneFilters filters.Args) (volume.PruneReport, error) {
	if m.VolumesPruneFn != nil {
		return m.VolumesPruneFn(ctx, pruneFilters)
	}
	return volume.PruneReport{}, nil
}

func (m *MockDockerAPI) NetworksPrune(ctx context.Context, pruneFilters filters.Args) (network.PruneReport, error) {
	if m.NetworksPruneFn != nil {
		return m.NetworksPruneFn(ctx, pruneFilters)
	}
	return network.PruneReport{}, nil
}

func (m *MockDockerAPI) BuildCachePrune(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error) {
	if m.BuildCachePruneFn != nil {
		return m.BuildCachePruneFn(ctx, opts)
	}
	return &types.BuildCachePruneReport{}, nil
}

func (m *MockDockerAPI) ContainerLogs(ctx context.Context, ctr string, options container.LogsOptions) (io.ReadCloser, error) {
	if m.ContainerLogsFn != nil {
		return m.ContainerLogsFn(ctx, ctr, options)
	}
	return io.NopCloser(io.LimitReader(nil, 0)), nil
}

func (m *MockDockerAPI) ContainerStatsOneShot(ctx context.Context, containerID string) (container.StatsResponseReader, error) {
	if m.ContainerStatsOneShotFn != nil {
		return m.ContainerStatsOneShotFn(ctx, containerID)
	}
	return container.StatsResponseReader{Body: io.NopCloser(io.LimitReader(nil, 0))}, nil
}

func (m *MockDockerAPI) ContainerExecCreate(ctx context.Context, containerID string, options container.ExecOptions) (types.IDResponse, error) {
	if m.ContainerExecCreateFn != nil {
		return m.ContainerExecCreateFn(ctx, containerID, options)
	}
	return types.IDResponse{ID: "mock-exec-id"}, nil
}

func (m *MockDockerAPI) ContainerExecAttach(ctx context.Context, execID string, config container.ExecAttachOptions) (types.HijackedResponse, error) {
	if m.ContainerExecAttachFn != nil {
		return m.ContainerExecAttachFn(ctx, execID, config)
	}
	return types.HijackedResponse{}, nil
}

func (m *MockDockerAPI) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	if m.ContainerExecResizeFn != nil {
		return m.ContainerExecResizeFn(ctx, execID, options)
	}
	return nil
}

func (m *MockDockerAPI) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	if m.ContainerExecInspectFn != nil {
		return m.ContainerExecInspectFn(ctx, execID)
	}
	return container.ExecInspect{}, nil
}

func (m *MockDockerAPI) ContainerExecStart(ctx context.Context, execID string, config container.ExecStartOptions) error {
	if m.ContainerExecStartFn != nil {
		return m.ContainerExecStartFn(ctx, execID, config)
	}
	return nil
}
