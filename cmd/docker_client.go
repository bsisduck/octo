package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// DockerClient wraps the Docker SDK client with helper methods
type DockerClient struct {
	Client *client.Client
	ctx    context.Context
}

// ContainerInfo holds container details for display
type ContainerInfo struct {
	ID      string
	Name    string
	Image   string
	Status  string
	State   string
	Created time.Time
	Ports   string
	Size    int64
}

// ImageInfo holds image details for display
type ImageInfo struct {
	ID         string
	Repository string
	Tag        string
	Size       int64
	Created    time.Time
	Containers int
	Dangling   bool
}

// VolumeInfo holds volume details for display
type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	Size       int64
	Created    time.Time
	Labels     map[string]string
	InUse      bool
}

// NetworkInfo holds network details for display
type NetworkInfo struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	Containers int
}

// DiskUsageInfo holds Docker disk usage summary
type DiskUsageInfo struct {
	Images           int64
	Containers       int64
	Volumes          int64
	BuildCache       int64
	TotalReclaimable int64
	Total            int64
}

// NewDockerClient creates a new Docker client with automatic socket detection
func NewDockerClient() (*DockerClient, error) {
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

	return &DockerClient{
		Client: cli,
		ctx:    ctx,
	}, nil
}

// detectDockerSocket returns the Docker socket path based on platform
func detectDockerSocket() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: Check Docker Desktop locations
		paths := []string{
			filepath.Join(os.Getenv("HOME"), ".docker/run/docker.sock"),
			filepath.Join(os.Getenv("HOME"), "Library/Containers/com.docker.docker/Data/docker.sock"),
			"/var/run/docker.sock",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return "unix://" + p
			}
		}
	case "linux":
		// Linux: Standard locations
		paths := []string{
			"/var/run/docker.sock",
			"/run/docker.sock",
			filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "docker.sock"), // Rootless
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return "unix://" + p
			}
		}
	case "windows":
		return "npipe:////./pipe/docker_engine"
	}
	return ""
}

// Close closes the Docker client connection
func (dc *DockerClient) Close() error {
	return dc.Client.Close()
}

// GetServerInfo returns Docker daemon information
func (dc *DockerClient) GetServerInfo() (system.Info, error) {
	return dc.Client.Info(dc.ctx)
}

// ListContainers returns all containers (running and stopped)
func (dc *DockerClient) ListContainers(all bool) ([]ContainerInfo, error) {
	containers, err := dc.Client.ContainerList(dc.ctx, container.ListOptions{All: all})
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
			ID:      c.ID[:12],
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

// ListImages returns all images
func (dc *DockerClient) ListImages(all bool) ([]ImageInfo, error) {
	images, err := dc.Client.ImageList(dc.ctx, image.ListOptions{All: all})
	if err != nil {
		return nil, err
	}

	result := make([]ImageInfo, 0, len(images))
	for _, img := range images {
		// Handle images with multiple tags
		if len(img.RepoTags) == 0 {
			result = append(result, ImageInfo{
				ID:       img.ID[7:19], // Remove "sha256:" prefix
				Size:     img.Size,
				Created:  time.Unix(img.Created, 0),
				Dangling: true,
			})
		} else {
			for _, tag := range img.RepoTags {
				repo, tagName := parseImageTag(tag)
				result = append(result, ImageInfo{
					ID:         img.ID[7:19],
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

// ListVolumes returns all volumes
func (dc *DockerClient) ListVolumes() ([]VolumeInfo, error) {
	volumes, err := dc.Client.VolumeList(dc.ctx, volume.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get containers to check volume usage
	containers, _ := dc.Client.ContainerList(dc.ctx, container.ListOptions{All: true})
	usedVolumes := make(map[string]bool)
	for _, c := range containers {
		for _, m := range c.Mounts {
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

// ListNetworks returns all networks
func (dc *DockerClient) ListNetworks() ([]NetworkInfo, error) {
	networks, err := dc.Client.NetworkList(dc.ctx, network.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := make([]NetworkInfo, len(networks))
	for i, n := range networks {
		result[i] = NetworkInfo{
			ID:         n.ID[:12],
			Name:       n.Name,
			Driver:     n.Driver,
			Scope:      n.Scope,
			Internal:   n.Internal,
			Containers: len(n.Containers),
		}
	}

	return result, nil
}

// GetDiskUsage returns Docker disk usage information
func (dc *DockerClient) GetDiskUsage() (*DiskUsageInfo, error) {
	du, err := dc.Client.DiskUsage(dc.ctx, types.DiskUsageOptions{})
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
	for _, c := range du.Containers {
		info.Containers += c.SizeRw
		if c.State != "running" {
			info.TotalReclaimable += c.SizeRw
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

// GetDanglingImages returns images with no tags (dangling)
func (dc *DockerClient) GetDanglingImages() ([]ImageInfo, error) {
	f := filters.NewArgs()
	f.Add("dangling", "true")

	images, err := dc.Client.ImageList(dc.ctx, image.ListOptions{Filters: f})
	if err != nil {
		return nil, err
	}

	result := make([]ImageInfo, len(images))
	for i, img := range images {
		result[i] = ImageInfo{
			ID:       img.ID[7:19],
			Size:     img.Size,
			Created:  time.Unix(img.Created, 0),
			Dangling: true,
		}
	}

	return result, nil
}

// GetStoppedContainers returns containers that are not running
func (dc *DockerClient) GetStoppedContainers() ([]ContainerInfo, error) {
	f := filters.NewArgs()
	f.Add("status", "exited")
	f.Add("status", "created")
	f.Add("status", "dead")

	containers, err := dc.Client.ContainerList(dc.ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, len(containers))
	for i, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0][1:]
		}
		result[i] = ContainerInfo{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			Status:  c.Status,
			State:   c.State,
			Created: time.Unix(c.Created, 0),
			Size:    c.SizeRw,
		}
	}

	return result, nil
}

// GetUnusedVolumes returns volumes not attached to any container
func (dc *DockerClient) GetUnusedVolumes() ([]VolumeInfo, error) {
	f := filters.NewArgs()
	f.Add("dangling", "true")

	volumes, err := dc.Client.VolumeList(dc.ctx, volume.ListOptions{Filters: f})
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

// RemoveContainer removes a container by ID
func (dc *DockerClient) RemoveContainer(id string, force bool) error {
	return dc.Client.ContainerRemove(dc.ctx, id, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
}

// RemoveImage removes an image by ID
func (dc *DockerClient) RemoveImage(id string, force bool) error {
	_, err := dc.Client.ImageRemove(dc.ctx, id, image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	})
	return err
}

// RemoveVolume removes a volume by name
func (dc *DockerClient) RemoveVolume(name string, force bool) error {
	return dc.Client.VolumeRemove(dc.ctx, name, force)
}

// PruneContainers removes all stopped containers
func (dc *DockerClient) PruneContainers() (uint64, error) {
	report, err := dc.Client.ContainersPrune(dc.ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneImages removes all dangling images
func (dc *DockerClient) PruneImages(all bool) (uint64, error) {
	f := filters.NewArgs()
	if all {
		f.Add("dangling", "false")
	}
	report, err := dc.Client.ImagesPrune(dc.ctx, f)
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneVolumes removes all unused volumes
func (dc *DockerClient) PruneVolumes() (uint64, error) {
	report, err := dc.Client.VolumesPrune(dc.ctx, filters.Args{})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// PruneNetworks removes all unused networks
func (dc *DockerClient) PruneNetworks() error {
	_, err := dc.Client.NetworksPrune(dc.ctx, filters.Args{})
	return err
}

// PruneBuildCache removes build cache
func (dc *DockerClient) PruneBuildCache(all bool) (uint64, error) {
	report, err := dc.Client.BuildCachePrune(dc.ctx, types.BuildCachePruneOptions{All: all})
	if err != nil {
		return 0, err
	}
	return report.SpaceReclaimed, nil
}

// Helper functions

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
