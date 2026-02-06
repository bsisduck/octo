package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
)

// Compile-time interface check
var _ DockerService = (*MockDockerService)(nil)

// MockDockerService is a hand-rolled mock implementation of DockerService.
// Use it in tests by setting the function fields to return specific values.
type MockDockerService struct {
	PingFn                 func(ctx context.Context) error
	CloseFn                func() error
	GetServerInfoFn        func(ctx context.Context) (system.Info, error)
	ListContainersFn       func(ctx context.Context, all bool) ([]ContainerInfo, error)
	ListImagesFn           func(ctx context.Context, all bool) ([]ImageInfo, error)
	ListVolumesFn          func(ctx context.Context) ([]VolumeInfo, error)
	ListNetworksFn         func(ctx context.Context) ([]NetworkInfo, error)
	GetDanglingImagesFn    func(ctx context.Context) ([]ImageInfo, error)
	GetStoppedContainersFn func(ctx context.Context) ([]ContainerInfo, error)
	GetUnusedVolumesFn     func(ctx context.Context) ([]VolumeInfo, error)
	GetDiskUsageFn         func(ctx context.Context) (*DiskUsageInfo, error)
	RemoveContainerFn      func(ctx context.Context, id string, force bool) error
	RemoveImageFn          func(ctx context.Context, id string, force bool) error
	RemoveVolumeFn         func(ctx context.Context, name string, force bool) error
	RemoveNetworkFn        func(ctx context.Context, id string) error
	PruneContainersFn      func(ctx context.Context) (uint64, error)
	PruneImagesFn          func(ctx context.Context, all bool) (uint64, error)
	PruneVolumesFn         func(ctx context.Context) (uint64, error)
	PruneNetworksFn        func(ctx context.Context) error
	PruneBuildCacheFn      func(ctx context.Context, all bool) (uint64, error)
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

// Compile-time interface check
var _ DockerAPI = (*MockDockerAPI)(nil)

// MockDockerAPI is a hand-rolled mock implementation of DockerAPI.
// Use it in tests to verify Client's data transformation logic.
type MockDockerAPI struct {
	PingFn            func(ctx context.Context) (types.Ping, error)
	InfoFn            func(ctx context.Context) (system.Info, error)
	CloseFn           func() error
	ContainerListFn   func(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ImageListFn       func(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	VolumeListFn      func(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error)
	NetworkListFn     func(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
	DiskUsageFn       func(ctx context.Context, options types.DiskUsageOptions) (types.DiskUsage, error)
	ContainerRemoveFn func(ctx context.Context, containerID string, options container.RemoveOptions) error
	ImageRemoveFn     func(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	VolumeRemoveFn    func(ctx context.Context, volumeID string, force bool) error
	ContainersPruneFn func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error)
	ImagesPruneFn     func(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error)
	VolumesPruneFn    func(ctx context.Context, pruneFilters filters.Args) (volume.PruneReport, error)
	NetworksPruneFn   func(ctx context.Context, pruneFilters filters.Args) (network.PruneReport, error)
	BuildCachePruneFn func(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error)
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
