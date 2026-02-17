package docker

import (
	"sort"
	"sync"
	"time"
)

// ContainerInfo holds container details for display
type ContainerInfo struct {
	ID      string            `json:"containerId" yaml:"containerId"`
	Name    string            `json:"containerName" yaml:"containerName"`
	Image   string            `json:"containerImage" yaml:"containerImage"`
	Status  string            `json:"containerStatus" yaml:"containerStatus"`
	State   string            `json:"containerState" yaml:"containerState"`
	Created time.Time         `json:"containerCreated" yaml:"containerCreated"`
	Ports   string            `json:"containerPorts" yaml:"containerPorts"`
	Size    int64             `json:"containerSize" yaml:"containerSize"`
	Labels  map[string]string `json:"containerLabels" yaml:"containerLabels"` // Container labels (includes Compose metadata)
}

// ImageInfo holds image details for display
type ImageInfo struct {
	ID         string    `json:"imageId" yaml:"imageId"`
	Repository string    `json:"imageRepository" yaml:"imageRepository"`
	Tag        string    `json:"imageTag" yaml:"imageTag"`
	Size       int64     `json:"imageSize" yaml:"imageSize"`
	Created    time.Time `json:"imageCreated" yaml:"imageCreated"`
	Containers int       `json:"imageContainers" yaml:"imageContainers"`
	Dangling   bool      `json:"imageDangling" yaml:"imageDangling"`
}

// VolumeInfo holds volume details for display
type VolumeInfo struct {
	Name       string            `json:"volumeName" yaml:"volumeName"`
	Driver     string            `json:"volumeDriver" yaml:"volumeDriver"`
	Mountpoint string            `json:"volumeMountpoint" yaml:"volumeMountpoint"`
	Size       int64             `json:"volumeSize" yaml:"volumeSize"`
	Created    time.Time         `json:"volumeCreated" yaml:"volumeCreated"`
	Labels     map[string]string `json:"volumeLabels" yaml:"volumeLabels"`
	InUse      bool              `json:"volumeInUse" yaml:"volumeInUse"`
}

// NetworkInfo holds network details for display
type NetworkInfo struct {
	ID         string `json:"networkId" yaml:"networkId"`
	Name       string `json:"networkName" yaml:"networkName"`
	Driver     string `json:"networkDriver" yaml:"networkDriver"`
	Scope      string `json:"networkScope" yaml:"networkScope"`
	Internal   bool   `json:"networkInternal" yaml:"networkInternal"`
	Containers int    `json:"networkContainers" yaml:"networkContainers"`
}

// DiskUsageInfo holds Docker disk usage summary
type DiskUsageInfo struct {
	Images           int64 `json:"diskImages" yaml:"diskImages"`
	Containers       int64 `json:"diskContainers" yaml:"diskContainers"`
	Volumes          int64 `json:"diskVolumes" yaml:"diskVolumes"`
	BuildCache       int64 `json:"diskBuildCache" yaml:"diskBuildCache"`
	TotalReclaimable int64 `json:"diskTotalReclaimable" yaml:"diskTotalReclaimable"`
	Total            int64 `json:"diskTotal" yaml:"diskTotal"`
}

// SafetyTier represents danger level of destructive operation
type SafetyTier int

const (
	TierInformational SafetyTier = iota
	TierLowRisk
	TierModerate
	TierHighRisk
	TierBulkDestructive
)

func (t SafetyTier) String() string {
	switch t {
	case TierInformational:
		return "Informational"
	case TierLowRisk:
		return "Low Risk"
	case TierModerate:
		return "Moderate"
	case TierHighRisk:
		return "High Risk"
	case TierBulkDestructive:
		return "Bulk Destructive"
	default:
		return "Unknown"
	}
}

// LogEntry represents a single log line from a container
type LogEntry struct {
	Timestamp time.Time `json:"logTimestamp" yaml:"logTimestamp"`
	Stream    string    `json:"logStream" yaml:"logStream"`    // "stdout" or "stderr"
	Content   string    `json:"logContent" yaml:"logContent"`
}

// ContainerMetrics holds real-time metrics for a container
type ContainerMetrics struct {
	ContainerID   string  `json:"metricsContainerId" yaml:"metricsContainerId"`
	CPUPercent    float64 `json:"metricsCpuPercent" yaml:"metricsCpuPercent"`
	MemoryUsage   uint64  `json:"metricsMemoryUsage" yaml:"metricsMemoryUsage"`
	MemoryLimit   uint64  `json:"metricsMemoryLimit" yaml:"metricsMemoryLimit"`
	MemoryPercent float64 `json:"metricsMemoryPercent" yaml:"metricsMemoryPercent"`
	NetworkRx     uint64  `json:"metricsNetworkRx" yaml:"metricsNetworkRx"`
	NetworkTx     uint64  `json:"metricsNetworkTx" yaml:"metricsNetworkTx"`
	BlockRead     uint64  `json:"metricsBlockRead" yaml:"metricsBlockRead"`
	BlockWrite    uint64  `json:"metricsBlockWrite" yaml:"metricsBlockWrite"`
	PIDs          uint64  `json:"metricsPids" yaml:"metricsPids"`
}

// DiskUsageCache caches DiskUsage API results with TTL
type DiskUsageCache struct {
	mu        sync.Mutex
	data      *DiskUsageInfo
	fetchedAt time.Time
	ttl       time.Duration
}

// NewDiskUsageCache creates a cache with the given TTL
func NewDiskUsageCache(ttl time.Duration) *DiskUsageCache {
	return &DiskUsageCache{ttl: ttl}
}

// Get returns cached data if still valid, or nil
func (c *DiskUsageCache) Get() *DiskUsageInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data != nil && time.Since(c.fetchedAt) < c.ttl {
		return c.data
	}
	return nil
}

// Set stores data in the cache
func (c *DiskUsageCache) Set(data *DiskUsageInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
	c.fetchedAt = time.Now()
}

// ExecOptions configures an interactive exec session
type ExecOptions struct {
	ContainerID string
	Shell       string // "/bin/sh" or "/bin/bash"
}

// Docker Compose label constants
const (
	ComposeProjectLabel = "com.docker.compose.project"
	ComposeServiceLabel = "com.docker.compose.service"
)

// ComposeGroup groups containers belonging to the same Docker Compose project
type ComposeGroup struct {
	ProjectName string
	Containers  []ContainerInfo
}

// GroupByComposeProject splits containers into Compose groups and ungrouped.
// Containers without Compose labels are returned as ungrouped.
// Groups are sorted alphabetically by project name.
func GroupByComposeProject(containers []ContainerInfo) (groups []ComposeGroup, ungrouped []ContainerInfo) {
	projectMap := make(map[string][]ContainerInfo)
	for _, c := range containers {
		if c.Labels == nil {
			ungrouped = append(ungrouped, c)
			continue
		}
		project, ok := c.Labels[ComposeProjectLabel]
		if !ok || project == "" {
			ungrouped = append(ungrouped, c)
			continue
		}
		projectMap[project] = append(projectMap[project], c)
	}

	for name, cts := range projectMap {
		groups = append(groups, ComposeGroup{
			ProjectName: name,
			Containers:  cts,
		})
	}
	// Sort groups alphabetically
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ProjectName < groups[j].ProjectName
	})
	return groups, ungrouped
}

// ConfirmationInfo holds confirmation details for destructive operation
type ConfirmationInfo struct {
	Tier             SafetyTier `json:"confirmationTier" yaml:"confirmationTier"`
	Title            string     `json:"confirmationTitle" yaml:"confirmationTitle"`            // "Delete Container?"
	Description      string     `json:"confirmationDescription" yaml:"confirmationDescription"` // "Stopped container 'web' (500MB)"
	Resources        []string   `json:"confirmationResources" yaml:"confirmationResources"`     // ["container: web", "size: 500MB"]
	Reversible       bool       `json:"confirmationReversible" yaml:"confirmationReversible"`   // Can action be undone?
	UndoInstructions string     `json:"confirmationUndoInstructions" yaml:"confirmationUndoInstructions"` // "Can be recreated from image"
	Warnings         []string   `json:"confirmationWarnings" yaml:"confirmationWarnings"`       // Additional warnings
}
