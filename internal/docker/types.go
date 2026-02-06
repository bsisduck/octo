package docker

import "time"

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
