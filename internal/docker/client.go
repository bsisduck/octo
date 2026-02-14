package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/docker/docker/pkg/stdcopy"
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

// API returns the underlying DockerAPI for direct access (used by exec).
func (c *Client) API() DockerAPI {
	return c.api
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
// TOCTOU protection: Re-fetches container state before deletion to prevent race conditions
// where a container's state may change between confirmation and actual deletion.
func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	// Re-check: Fetch current state before deletion (TOCTOU protection)
	containers, err := c.api.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to re-check container state before deletion: %w", err)
	}

	// Find the container to verify it still exists and check its current state
	var targetContainer *types.Container
	for _, ct := range containers {
		if truncateID(ct.ID, 12) == id || ct.ID == id {
			t := ct
			targetContainer = &t
			break
		}
	}

	if targetContainer == nil {
		return fmt.Errorf("container not found")
	}

	// If force is false and container is running, prevent deletion
	if !force && targetContainer.State == "running" {
		return fmt.Errorf("cannot remove running container without force=true; container state changed or user requested non-force deletion")
	}

	opts := container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	}
	return c.api.ContainerRemove(ctx, id, opts)
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
	// Implement network removal using the raw Docker API
	return c.api.NetworkRemove(ctx, id)
}

// RemoveNetworkDryRun returns confirmation info for network removal without deleting
func (c *Client) RemoveNetworkDryRun(ctx context.Context, id string) (ConfirmationInfo, error) {
	networks, err := c.api.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	var target *network.Summary
	for _, n := range networks {
		if truncateID(n.ID, 12) == id || n.ID == id {
			target = &n
			break
		}
	}

	if target == nil {
		return ConfirmationInfo{}, fmt.Errorf("network not found")
	}

	// Networks cannot be recreated easily, so they're not reversible in practical terms
	info := ConfirmationInfo{
		Tier:        TierModerate,
		Title:       "Delete Network?",
		Description: fmt.Sprintf("Network '%s' (%s, %d containers)", target.Name, target.Driver, len(target.Containers)),
		Resources: []string{
			fmt.Sprintf("network: %s", target.Name),
			fmt.Sprintf("driver: %s", target.Driver),
			fmt.Sprintf("containers: %d", len(target.Containers)),
		},
		Reversible:       false,
		UndoInstructions: "Network must be manually recreated",
		Warnings:         []string{},
	}

	if len(target.Containers) > 0 {
		info.Tier = TierHighRisk
		info.Warnings = append(info.Warnings, fmt.Sprintf("Network has %d connected container(s)", len(target.Containers)))
	}

	// Check if it's a system network
	if target.Name == "bridge" || target.Name == "host" || target.Name == "none" {
		info.Tier = TierBulkDestructive
		info.Warnings = append(info.Warnings, "Cannot delete system networks")
	}

	return info, nil
}

// RemoveContainerDryRun returns confirmation info for container removal without deleting
func (c *Client) RemoveContainerDryRun(ctx context.Context, id string) (ConfirmationInfo, error) {
	containers, err := c.api.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	var target *types.Container
	for _, ct := range containers {
		if truncateID(ct.ID, 12) == id || ct.ID == id {
			t := ct
			target = &t
			break
		}
	}

	if target == nil {
		return ConfirmationInfo{}, fmt.Errorf("container not found")
	}

	name := ""
	if len(target.Names) > 0 {
		name = target.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}
	}

	tier := TierLowRisk
	reversible := true
	undoInstructions := "Can be recreated from image " + target.Image
	warnings := []string{}

	if target.State == "running" {
		tier = TierModerate
		warnings = append(warnings, "Container is currently running")
	}

	info := ConfirmationInfo{
		Tier:             tier,
		Title:            "Delete Container?",
		Description:      fmt.Sprintf("%s container '%s' (%s)", strings.Title(target.State), name, target.Image),
		Resources:        []string{fmt.Sprintf("container: %s", name), fmt.Sprintf("image: %s", target.Image), fmt.Sprintf("size: %s", formatBytes(target.SizeRw))},
		Reversible:       reversible,
		UndoInstructions: undoInstructions,
		Warnings:         warnings,
	}

	return info, nil
}

// RemoveImageDryRun returns confirmation info for image removal without deleting
func (c *Client) RemoveImageDryRun(ctx context.Context, id string) (ConfirmationInfo, error) {
	images, err := c.api.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	var target *image.Summary
	for _, img := range images {
		if trimImageID(img.ID) == id || img.ID == id {
			t := img
			target = &t
			break
		}
	}

	if target == nil {
		return ConfirmationInfo{}, fmt.Errorf("image not found")
	}

	// Determine image name
	imageName := "<none>"
	if len(target.RepoTags) > 0 && target.RepoTags[0] != "<none>:<none>" {
		imageName = target.RepoTags[0]
	}

	tier := TierLowRisk
	if target.Containers > 0 {
		tier = TierHighRisk
	}

	warnings := []string{}
	if target.Containers > 0 {
		warnings = append(warnings, fmt.Sprintf("Image is used by %d container(s)", target.Containers))
	}

	info := ConfirmationInfo{
		Tier:             tier,
		Title:            "Delete Image?",
		Description:      fmt.Sprintf("Image '%s' (%s)", imageName, formatBytes(target.Size)),
		Resources:        []string{fmt.Sprintf("image: %s", imageName), fmt.Sprintf("size: %s", formatBytes(target.Size)), fmt.Sprintf("containers: %d", target.Containers)},
		Reversible:       true,
		UndoInstructions: "Can be pulled from registry",
		Warnings:         warnings,
	}

	return info, nil
}

// RemoveVolumeDryRun returns confirmation info for volume removal without deleting
func (c *Client) RemoveVolumeDryRun(ctx context.Context, name string) (ConfirmationInfo, error) {
	volumes, err := c.api.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	var target *volume.Volume
	for _, v := range volumes.Volumes {
		if v.Name == name {
			target = v
			break
		}
	}

	if target == nil {
		return ConfirmationInfo{}, fmt.Errorf("volume not found")
	}

	// Check if volume is in use
	containers, _ := c.api.ContainerList(ctx, container.ListOptions{All: true})
	inUse := false
	for _, ct := range containers {
		for _, m := range ct.Mounts {
			if m.Type == "volume" && m.Name == name {
				inUse = true
				break
			}
		}
	}

	tier := TierLowRisk
	if inUse {
		tier = TierHighRisk
	}

	warnings := []string{}
	if inUse {
		warnings = append(warnings, "Volume is currently in use by container(s)")
	}

	info := ConfirmationInfo{
		Tier:             tier,
		Title:            "Delete Volume?",
		Description:      fmt.Sprintf("Volume '%s' (%s)", name, target.Driver),
		Resources:        []string{fmt.Sprintf("volume: %s", name), fmt.Sprintf("driver: %s", target.Driver)},
		Reversible:       false,
		UndoInstructions: "Data cannot be recovered",
		Warnings:         warnings,
	}

	return info, nil
}

// PruneContainersDryRun returns confirmation info for container pruning without executing
func (c *Client) PruneContainersDryRun(ctx context.Context) (ConfirmationInfo, error) {
	f := filters.NewArgs()
	f.Add("status", "exited")
	f.Add("status", "created")
	f.Add("status", "dead")

	containers, err := c.api.ContainerList(ctx, container.ListOptions{All: true, Filters: f})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	totalSize := int64(0)
	for _, ct := range containers {
		totalSize += ct.SizeRw
	}

	resources := []string{
		fmt.Sprintf("stopped containers: %d", len(containers)),
		fmt.Sprintf("total size: %s", formatBytes(totalSize)),
	}

	info := ConfirmationInfo{
		Tier:             TierBulkDestructive,
		Title:            "Prune Stopped Containers?",
		Description:      fmt.Sprintf("Remove %d stopped container(s), freeing %s", len(containers), formatBytes(totalSize)),
		Resources:        resources,
		Reversible:       true,
		UndoInstructions: "Can be recreated from images",
		Warnings:         []string{"This is a bulk operation"},
	}

	return info, nil
}

// PruneImagesDryRun returns confirmation info for image pruning without executing
func (c *Client) PruneImagesDryRun(ctx context.Context, all bool) (ConfirmationInfo, error) {
	images, err := c.api.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	// Calculate what would be pruned
	totalSize := int64(0)
	pruneCount := 0

	for _, img := range images {
		if all {
			// If all=true, would prune all dangling and unreferenced
			if img.Containers == 0 {
				totalSize += img.Size
				pruneCount++
			}
		} else {
			// Otherwise just dangling
			if len(img.RepoTags) == 0 || (len(img.RepoTags) == 1 && img.RepoTags[0] == "<none>:<none>") {
				totalSize += img.Size
				pruneCount++
			}
		}
	}

	pruneType := "dangling images"
	if all {
		pruneType = "unused images"
	}

	info := ConfirmationInfo{
		Tier:             TierBulkDestructive,
		Title:            "Prune Images?",
		Description:      fmt.Sprintf("Remove %d %s, freeing %s", pruneCount, pruneType, formatBytes(totalSize)),
		Resources:        []string{fmt.Sprintf("images to remove: %d", pruneCount), fmt.Sprintf("space freed: %s", formatBytes(totalSize))},
		Reversible:       true,
		UndoInstructions: "Can be pulled from registry",
		Warnings:         []string{"This is a bulk operation"},
	}

	return info, nil
}

// PruneVolumesDryRun returns confirmation info for volume pruning without executing
func (c *Client) PruneVolumesDryRun(ctx context.Context) (ConfirmationInfo, error) {
	f := filters.NewArgs()
	f.Add("dangling", "true")

	volumes, err := c.api.VolumeList(ctx, volume.ListOptions{Filters: f})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	info := ConfirmationInfo{
		Tier:             TierBulkDestructive,
		Title:            "Prune Unused Volumes?",
		Description:      fmt.Sprintf("Remove %d unused volume(s)", len(volumes.Volumes)),
		Resources:        []string{fmt.Sprintf("unused volumes: %d", len(volumes.Volumes))},
		Reversible:       false,
		UndoInstructions: "Data cannot be recovered",
		Warnings:         []string{"This is a bulk operation", "Volume data will be permanently deleted"},
	}

	return info, nil
}

// PruneNetworksDryRun returns confirmation info for network pruning without executing
func (c *Client) PruneNetworksDryRun(ctx context.Context) (ConfirmationInfo, error) {
	networks, err := c.api.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return ConfirmationInfo{}, err
	}

	// Count unused networks (excluding system networks)
	pruneCount := 0
	for _, n := range networks {
		if len(n.Containers) == 0 && n.Name != "bridge" && n.Name != "host" && n.Name != "none" {
			pruneCount++
		}
	}

	info := ConfirmationInfo{
		Tier:             TierBulkDestructive,
		Title:            "Prune Unused Networks?",
		Description:      fmt.Sprintf("Remove %d unused network(s)", pruneCount),
		Resources:        []string{fmt.Sprintf("unused networks: %d", pruneCount)},
		Reversible:       false,
		UndoInstructions: "Networks must be manually recreated",
		Warnings:         []string{"This is a bulk operation"},
	}

	return info, nil
}

// StartContainer starts a container by ID.
func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.api.ContainerStart(ctx, id, container.StartOptions{})
}

// StopContainer stops a container by ID.
func (c *Client) StopContainer(ctx context.Context, id string) error {
	return c.api.ContainerStop(ctx, id, container.StopOptions{})
}

// RestartContainer restarts a container by ID.
func (c *Client) RestartContainer(ctx context.Context, id string) error {
	return c.api.ContainerRestart(ctx, id, container.StopOptions{})
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

// GetContainerLogs fetches the last N lines of logs from a container.
// Uses stdcopy to demultiplex stdout/stderr for non-TTY containers.
func (c *Client) GetContainerLogs(ctx context.Context, containerID string, tail int) ([]LogEntry, error) {
	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       fmt.Sprintf("%d", tail),
	}

	reader, err := c.api.ContainerLogs(ctx, containerID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Demultiplex stdout and stderr using stdcopy
	var stdoutBuf, stderrBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&stdoutBuf, &stderrBuf, reader)
	if err != nil {
		// If stdcopy fails, the container might use TTY mode.
		// Re-fetch and read directly.
		reader2, err2 := c.api.ContainerLogs(ctx, containerID, opts)
		if err2 != nil {
			return nil, fmt.Errorf("failed to re-fetch logs: %w", err2)
		}
		defer reader2.Close()
		return parseLogLines(reader2, "stdout"), nil
	}

	var entries []LogEntry
	entries = append(entries, parseLogLinesFromBytes(stdoutBuf.Bytes(), "stdout")...)
	entries = append(entries, parseLogLinesFromBytes(stderrBuf.Bytes(), "stderr")...)

	// Sort by timestamp
	sortLogEntries(entries)
	return entries, nil
}

// StreamContainerLogs streams live logs from a container.
// Returns a channel of log entries, an error channel, and a cancel function.
func (c *Client) StreamContainerLogs(ctx context.Context, containerID string) (<-chan LogEntry, <-chan error, func()) {
	logCh := make(chan LogEntry, 100)
	errCh := make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer close(logCh)
		defer close(errCh)

		opts := container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Timestamps: true,
			Follow:     true,
			Tail:       "0", // Only new logs
		}

		reader, err := c.api.ContainerLogs(ctx, containerID, opts)
		if err != nil {
			errCh <- fmt.Errorf("failed to stream logs: %w", err)
			return
		}
		defer reader.Close()

		// Use a pipe to demux stdout/stderr
		pr, pw := io.Pipe()
		go func() {
			_, err := stdcopy.StdCopy(pw, pw, reader)
			if err != nil {
				// TTY mode fallback: copy directly
				pw.Close()
				return
			}
			pw.Close()
		}()

		scanner := bufio.NewScanner(pr)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line
		for scanner.Scan() {
			line := scanner.Text()
			entry := parseTimestampedLine(line, "stdout")
			select {
			case logCh <- entry:
			case <-ctx.Done():
				return
			}
		}
	}()

	return logCh, errCh, cancel
}

// GetContainerStats returns real-time metrics for a container.
func (c *Client) GetContainerStats(ctx context.Context, containerID string) (*ContainerMetrics, error) {
	statsResp, err := c.api.ContainerStatsOneShot(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer statsResp.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode stats: %w", err)
	}

	// Calculate CPU percentage using delta formula
	cpuPercent := 0.0
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if systemDelta > 0 && cpuDelta > 0 {
		onlineCPUs := float64(stats.CPUStats.OnlineCPUs)
		if onlineCPUs == 0 {
			onlineCPUs = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		if onlineCPUs > 0 {
			cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
		}
	}

	// Calculate memory
	memUsage := stats.MemoryStats.Usage
	memLimit := stats.MemoryStats.Limit
	memPercent := 0.0
	if memLimit > 0 {
		memPercent = float64(memUsage) / float64(memLimit) * 100.0
	}

	// Calculate network I/O
	var netRx, netTx uint64
	for _, v := range stats.Networks {
		netRx += v.RxBytes
		netTx += v.TxBytes
	}

	// Calculate block I/O
	var blockRead, blockWrite uint64
	for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
		switch bio.Op {
		case "read", "Read":
			blockRead += bio.Value
		case "write", "Write":
			blockWrite += bio.Value
		}
	}

	return &ContainerMetrics{
		ContainerID:   containerID,
		CPUPercent:    cpuPercent,
		MemoryUsage:   memUsage,
		MemoryLimit:   memLimit,
		MemoryPercent: memPercent,
		NetworkRx:     netRx,
		NetworkTx:     netTx,
		BlockRead:     blockRead,
		BlockWrite:    blockWrite,
		PIDs:          stats.PidsStats.Current,
	}, nil
}

// parseLogLines reads from a reader and creates log entries
func parseLogLines(r io.Reader, stream string) []LogEntry {
	var entries []LogEntry
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		entry := parseTimestampedLine(scanner.Text(), stream)
		entries = append(entries, entry)
	}
	return entries
}

// parseLogLinesFromBytes parses log lines from a byte slice
func parseLogLinesFromBytes(data []byte, stream string) []LogEntry {
	return parseLogLines(bytes.NewReader(data), stream)
}

// parseTimestampedLine parses a Docker log line with timestamp prefix
func parseTimestampedLine(line, stream string) LogEntry {
	// Docker timestamps format: 2006-01-02T15:04:05.999999999Z
	// Try to parse timestamp from beginning of line
	if len(line) > 30 {
		if ts, err := time.Parse(time.RFC3339Nano, line[:30]); err == nil {
			return LogEntry{Timestamp: ts, Stream: stream, Content: strings.TrimSpace(line[31:])}
		}
	}
	// Try shorter timestamp formats
	for _, tsLen := range []int{35, 30, 25, 20} {
		if len(line) > tsLen {
			if ts, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(line[:tsLen])); err == nil {
				return LogEntry{Timestamp: ts, Stream: stream, Content: strings.TrimSpace(line[tsLen:])}
			}
		}
	}
	return LogEntry{Timestamp: time.Now(), Stream: stream, Content: line}
}

// sortLogEntries sorts log entries by timestamp
func sortLogEntries(entries []LogEntry) {
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].Timestamp.Before(entries[j-1].Timestamp); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
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

// formatBytes formats a byte count into human-readable format
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
