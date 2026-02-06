package docker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

// Compile-time interface check
var _ DockerService = (*Client)(nil)

// Client wraps the Docker SDK client with helper methods.
// It uses the DockerAPI interface internally for testability.
type Client struct {
	api DockerAPI
}

// NewClient creates a new Docker client with automatic socket detection.
// It returns a Client implementing the DockerService interface.
func NewClient() (*Client, error) {
	// Try environment variable first
	host := os.Getenv("DOCKER_HOST")

	// Platform-specific socket detection
	if host == "" {
		host = detectDockerSocket()
	}

	opts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	if host != "" {
		opts = append(opts, client.WithHost(host))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx := context.Background()

	// Test connection with ping
	_, err = cli.Ping(ctx)
	if err != nil {
		cli.Close()
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	return &Client{api: cli}, nil
}

// Ping returns nil if the Docker daemon is reachable.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.api.Ping(ctx)
	return err
}

// Close closes the Docker client connection.
func (c *Client) Close() error {
	return c.api.Close()
}

// GetServerInfo returns Docker daemon information.
func (c *Client) GetServerInfo(ctx context.Context) (system.Info, error) {
	return c.api.Info(ctx)
}

// ListContainers returns all containers (running and stopped).
func (c *Client) ListContainers(ctx context.Context, all bool) ([]ContainerInfo, error) {
	containers, err := c.api.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, len(containers))
	for i, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		ports := formatPorts(c.Ports)

		result[i] = ContainerInfo{
			ID:      truncateID(c.ID, 12),
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Created: time.Unix(c.Created, 0),
			Ports:   ports,
			Size:    c.SizeRw,
		}
	}

	return result, nil
}

// ListImages returns all images.
func (c *Client) ListImages(ctx context.Context, all bool) ([]ImageInfo, error) {
	images, err := c.api.ImageList(ctx, image.ListOptions{All: all})
	if err != nil {
		return nil, err
	}

	result := make([]ImageInfo, 0, len(images))
	for _, img := range images {
		// Handle images with multiple tags
		if len(img.RepoTags) == 0 {
			result = append(result, ImageInfo{
				ID:       trimImageID(img.ID),
				Size:     img.Size,
				Created:  time.Unix(img.Created, 0),
				Dangling: true,
			})
		} else {
			for _, tag := range img.RepoTags {
				repo, tagName := parseImageTag(tag)
				result = append(result, ImageInfo{
					ID:         trimImageID(img.ID),
					Repository: repo,
					Tag:        tagName,
					Size:       img.Size,
					Created:    time.Unix(img.Created, 0),
					Containers: int(img.Containers),
					Dangling:   false,
				})
			}
		}
	}

	return result, nil
}

// ListVolumes returns all volumes.
func (c *Client) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	volumes, err := c.api.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get containers to check volume usage
	containers, _ := c.api.ContainerList(ctx, container.ListOptions{All: true})
	usedVolumes := make(map[string]bool)
	for _, ct := range containers {
		for _, m := range ct.Mounts {
			if m.Type == "volume" {
				usedVolumes[m.Name] = true
			}
		}
	}

	result := make([]VolumeInfo, len(volumes.Volumes))
	for i, v := range volumes.Volumes {
		created, _ := time.Parse(time.RFC3339, v.CreatedAt)
		result[i] = VolumeInfo{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Created:    created,
			Labels:     v.Labels,
			InUse:      usedVolumes[v.Name],
		}
	}

	return result, nil
}

// ListNetworks returns all networks.
func (c *Client) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	networks, err := c.api.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInfo, len(networks))
	for i, n := range networks {
		result[i] = NetworkInfo{
			ID:         truncateID(n.ID, 12),
			Name:       n.Name,
			Driver:     n.Driver,
			Scope:      n.Scope,
			Internal:   n.Internal,
			Containers: len(n.Containers),
		}
	}

	return result, nil
}

// GetDiskUsage returns Docker disk usage information.
func (c *Client) GetDiskUsage(ctx context.Context) (*DiskUsageInfo, error) {
	du, err := c.api.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return nil, err
	}

	info := &DiskUsageInfo{}

	// Calculate image sizes
	for _, img := range du.Images {
		info.Images += img.Size
		if img.Containers == 0 {
			info.TotalReclaimable += img.Size
		}
	}

	// Calculate container sizes
	for _, ct := range du.Containers {
		info.Containers += ct.SizeRw
		if ct.State != "running" {
			info.TotalReclaimable += ct.SizeRw
		}
	}

	// Calculate volume sizes
	for _, v := range du.Volumes {
		info.Volumes += v.UsageData.Size
	}

	// Calculate build cache
	for _, bc := range du.BuildCache {
		info.BuildCache += bc.Size
		if !bc.InUse {
			info.TotalReclaimable += bc.Size
		}
	}

	info.Total = info.Images + info.Containers + info.Volumes + info.BuildCache

	return info, nil
}

// GetDanglingImages returns images with no tags (dangling).
func (c *Client) GetDanglingImages(ctx context.Context) ([]ImageInfo, error) {
	f := filters.NewArgs()
	f.Add("dangling", "true")

	images, err := c.api.ImageList(ctx, image.ListOptions{Filters: f})
	if err != nil {
		return nil, err
	}

	result := make([]ImageInfo, len(images))
	for i, img := range images {
		result[i] = ImageInfo{
			ID:       trimImageID(img.ID),
			Size:     img.Size,
			Created:  time.Unix(img.Created, 0),
			Dangling: true,
		}
	}

	return result, nil
}

// GetStoppedContainers returns containers that are not running.
func (c *Client) GetStoppedContainers(ctx context.Context) ([]ContainerInfo, error) {
	f := filters.NewArgs()
	f.Add("status", "exited")
	f.Add("status", "created")
	f.Add("status", "dead")

	containers, err := c.api.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, len(containers))
	for i, ct := range containers {
		name := ""
		if len(ct.Names) > 0 {
			name = ct.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		result[i] = ContainerInfo{
			ID:      truncateID(ct.ID, 12),
			Name:    name,
			Image:   ct.Image,
			Status:  ct.Status,
			State:   ct.State,
			Created: time.Unix(ct.Created, 0),
			Size:    ct.SizeRw,
		}
	}

	return result, nil
}

// GetUnusedVolumes returns volumes not attached to any container.
func (c *Client) GetUnusedVolumes(ctx context.Context) ([]VolumeInfo, error) {
	f := filters.NewArgs()
	f.Add("dangling", "true")

	volumes, err := c.api.VolumeList(ctx, volume.ListOptions{Filters: f})
	if err != nil {
		return nil, err
	}

	result := make([]VolumeInfo, len(volumes.Volumes))
	for i, v := range volumes.Volumes {
		created, _ := time.Parse(time.RFC3339, v.CreatedAt)
		result[i] = VolumeInfo{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Created:    created,
			Labels:     v.Labels,
			InUse:      false,
		}
	}

	return result, nil
}

// RemoveContainer removes a container by ID.
func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	return c.api.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
}

// RemoveImage removes an image by ID.
func (c *Client) RemoveImage(ctx context.Context, id string, force bool) error {
	_, err := c.api.ImageRemove(ctx, id, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	return err
}

// RemoveVolume removes a volume by name.
func (c *Client) RemoveVolume(ctx context.Context, name string, force bool) error {
	return c.api.VolumeRemove(ctx, name, force)
}

// RemoveNetwork removes a network by ID.
func (c *Client) RemoveNetwork(ctx context.Context, id string) error {
	return fmt.Errorf("not implemented")
}

// PruneContainers removes all stopped containers.
func (c *Client) PruneContainers(ctx context.Context) (uint64, error) {
	report, err := c.api.ContainersPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneImages removes all dangling images.
func (c *Client) PruneImages(ctx context.Context, all bool) (uint64, error) {
	f := filters.NewArgs()
	if all {
		f.Add("dangling", "false")
	}
	report, err := c.api.ImagesPrune(ctx, f)
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneVolumes removes all unused volumes.
func (c *Client) PruneVolumes(ctx context.Context) (uint64, error) {
	report, err := c.api.VolumesPrune(ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneNetworks removes all unused networks.
func (c *Client) PruneNetworks(ctx context.Context) error {
	_, err := c.api.NetworksPrune(ctx, filters.Args{})
	return err
}

// PruneBuildCache removes build cache.
func (c *Client) PruneBuildCache(ctx context.Context, all bool) (uint64, error) {
	report, err := c.api.BuildCachePrune(ctx, types.BuildCachePruneOptions{All: all})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// Helper functions

// truncateID returns the first maxLen characters of an ID string.
// If the string is shorter than maxLen, returns the full string.
func truncateID(id string, maxLen int) string {
	if len(id) > maxLen {
		return id[:maxLen]
	}
	return id
}

// trimImageID strips the "sha256:" prefix from a Docker image ID
// and returns up to 12 characters.
func trimImageID(id string) string {
	const prefix = "sha256:"
	if strings.HasPrefix(id, prefix) {
		id = id[len(prefix):]
	}
	return truncateID(id, 12)
}

// formatPorts formats a slice of Port mappings into a human-readable string.
func formatPorts(ports []types.Port) string {
	if len(ports) == 0 {
		return ""
	}

	result := ""
	for i, p := range ports {
		if i > 0 {
			result += ", "
		}
		if p.PublicPort != 0 {
			result += fmt.Sprintf("%d->%d/%s", p.PublicPort, p.PrivatePort, p.Type)
		} else {
			result += fmt.Sprintf("%d/%s", p.PrivatePort, p.Type)
		}
	}
	return result
}

// parseImageTag splits a Docker image tag into repository and tag components.
func parseImageTag(tag string) (repo, tagName string) {
	for i := len(tag) - 1; i >= 0; i-- {
		if tag[i] == ':' {
			return tag[:i], tag[i+1:]
		}
		if tag[i] == '/' {
			break
		}
	}
	return tag, "latest"
}
