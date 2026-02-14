package docker

import (
	"sort"
	"sync"
	"time"
)

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
	Labels  map[string]string // Container labels (includes Compose metadata)
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
	Timestamp time.Time
	Stream    string // "stdout" or "stderr"
	Content   string
}

// ContainerMetrics holds real-time metrics for a container
type ContainerMetrics struct {
	ContainerID   string
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	NetworkRx     uint64
	NetworkTx     uint64
	BlockRead     uint64
	BlockWrite    uint64
	PIDs          uint64
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
	Tier             SafetyTier
	Title            string   // "Delete Container?"
	Description      string   // "Stopped container 'web' (500MB)"
	Resources        []string // ["container: web", "size: 500MB"]
	Reversible       bool     // Can action be undone?
	UndoInstructions string   // "Can be recreated from image"
	Warnings         []string // Additional warnings
}
