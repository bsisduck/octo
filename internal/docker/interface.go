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

// DockerAPI interface wraps the raw Docker SDK client.
// The Docker SDK's *client.Client already satisfies this interface implicitly.
type DockerAPI interface {
	Ping(ctx context.Context) (types.Ping, error)
	Info(ctx context.Context) (system.Info, error)
	Close() error
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error)
	NetworkList(ctx context.Context, options network.ListOptions) ([]network.Summary, error)
	DiskUsage(ctx context.Context, options types.DiskUsageOptions) (types.DiskUsage, error)
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error)
	VolumeRemove(ctx context.Context, volumeID string, force bool) error
	NetworkRemove(ctx context.Context, networkID string) error
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error
	ContainersPrune(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error)
	ImagesPrune(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error)
	VolumesPrune(ctx context.Context, pruneFilters filters.Args) (volume.PruneReport, error)
	NetworksPrune(ctx context.Context, pruneFilters filters.Args) (network.PruneReport, error)
	BuildCachePrune(ctx context.Context, opts types.BuildCachePruneOptions) (*types.BuildCachePruneReport, error)
}

// DockerService interface provides domain-level Docker operations.
// All methods accept context.Context as the first parameter for proper cancellation/timeouts.
type DockerService interface {
	Ping(ctx context.Context) error
	Close() error
	GetServerInfo(ctx context.Context) (system.Info, error)
	ListContainers(ctx context.Context, all bool) ([]ContainerInfo, error)
	ListImages(ctx context.Context, all bool) ([]ImageInfo, error)
	ListVolumes(ctx context.Context) ([]VolumeInfo, error)
	ListNetworks(ctx context.Context) ([]NetworkInfo, error)
	GetDanglingImages(ctx context.Context) ([]ImageInfo, error)
	GetStoppedContainers(ctx context.Context) ([]ContainerInfo, error)
	GetUnusedVolumes(ctx context.Context) ([]VolumeInfo, error)
	GetDiskUsage(ctx context.Context) (*DiskUsageInfo, error)
	RemoveContainer(ctx context.Context, id string, force bool) error
	RemoveImage(ctx context.Context, id string, force bool) error
	RemoveVolume(ctx context.Context, name string, force bool) error
	RemoveNetwork(ctx context.Context, id string) error
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string) error
	RestartContainer(ctx context.Context, id string) error
	PruneContainers(ctx context.Context) (uint64, error)
	PruneImages(ctx context.Context, all bool) (uint64, error)
	PruneVolumes(ctx context.Context) (uint64, error)
	PruneNetworks(ctx context.Context) error
	PruneBuildCache(ctx context.Context, all bool) (uint64, error)
	// DryRun methods return what WOULD be deleted without actually deleting
	RemoveContainerDryRun(ctx context.Context, id string) (ConfirmationInfo, error)
	RemoveImageDryRun(ctx context.Context, id string) (ConfirmationInfo, error)
	RemoveVolumeDryRun(ctx context.Context, name string) (ConfirmationInfo, error)
	RemoveNetworkDryRun(ctx context.Context, id string) (ConfirmationInfo, error)
	PruneContainersDryRun(ctx context.Context) (ConfirmationInfo, error)
	PruneImagesDryRun(ctx context.Context, all bool) (ConfirmationInfo, error)
	PruneVolumesDryRun(ctx context.Context) (ConfirmationInfo, error)
	PruneNetworksDryRun(ctx context.Context) (ConfirmationInfo, error)
}
