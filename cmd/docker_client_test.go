package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestParseImageTag(t *testing.T) {
	tests := []struct {
		input    string
		wantRepo string
		wantTag  string
	}{
		{"nginx:latest", "nginx", "latest"},
		{"nginx", "nginx", "latest"},
		{"nginx:1.21", "nginx", "1.21"},
		{"registry.example.com/myapp:v1.0", "registry.example.com/myapp", "v1.0"},
		{"registry.example.com:5000/myapp:v1.0", "registry.example.com:5000/myapp", "v1.0"},
		{"ubuntu:20.04", "ubuntu", "20.04"},
		{"myimage", "myimage", "latest"},
		{"org/repo:tag", "org/repo", "tag"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotRepo, gotTag := parseImageTag(tt.input)
			if gotRepo != tt.wantRepo || gotTag != tt.wantTag {
				t.Errorf("parseImageTag(%q) = (%q, %q), want (%q, %q)",
					tt.input, gotRepo, gotTag, tt.wantRepo, tt.wantTag)
			}
		})
	}
}

func TestDetectDockerSocket(t *testing.T) {
	// This test just ensures the function doesn't panic
	socket := detectDockerSocket()
	// Socket should be empty or start with unix:// or npipe://
	if socket != "" && !strings.HasPrefix(socket, "unix://") && !strings.HasPrefix(socket, "npipe://") {
		t.Errorf("detectDockerSocket() = %q, want empty or valid socket path", socket)
	}
}

func TestContainerInfo(t *testing.T) {
	created := time.Now()
	info := ContainerInfo{
		ID:      "abc123",
		Name:    "test-container",
		Image:   "nginx:latest",
		Status:  "Up 2 hours",
		State:   "running",
		Created: created,
		Ports:   "80/tcp",
		Size:    1024,
	}

	if info.ID != "abc123" {
		t.Errorf("ContainerInfo.ID = %q, want %q", info.ID, "abc123")
	}
	if info.Name != "test-container" {
		t.Errorf("ContainerInfo.Name = %q, want %q", info.Name, "test-container")
	}
	if info.Image != "nginx:latest" {
		t.Errorf("ContainerInfo.Image = %q, want %q", info.Image, "nginx:latest")
	}
	if info.Status != "Up 2 hours" {
		t.Errorf("ContainerInfo.Status = %q, want %q", info.Status, "Up 2 hours")
	}
	if info.State != "running" {
		t.Errorf("ContainerInfo.State = %q, want %q", info.State, "running")
	}
	if info.Created != created {
		t.Errorf("ContainerInfo.Created = %v, want %v", info.Created, created)
	}
	if info.Ports != "80/tcp" {
		t.Errorf("ContainerInfo.Ports = %q, want %q", info.Ports, "80/tcp")
	}
	if info.Size != 1024 {
		t.Errorf("ContainerInfo.Size = %d, want %d", info.Size, 1024)
	}
}

func TestImageInfo(t *testing.T) {
	created := time.Now()
	info := ImageInfo{
		ID:         "sha256abc",
		Repository: "nginx",
		Tag:        "latest",
		Size:       52428800,
		Created:    created,
		Containers: 2,
		Dangling:   false,
	}

	if info.ID != "sha256abc" {
		t.Errorf("ImageInfo.ID = %q, want %q", info.ID, "sha256abc")
	}
	if info.Repository != "nginx" {
		t.Errorf("ImageInfo.Repository = %q, want %q", info.Repository, "nginx")
	}
	if info.Tag != "latest" {
		t.Errorf("ImageInfo.Tag = %q, want %q", info.Tag, "latest")
	}
	if info.Size != 52428800 {
		t.Errorf("ImageInfo.Size = %d, want %d", info.Size, 52428800)
	}
	if info.Created != created {
		t.Errorf("ImageInfo.Created = %v, want %v", info.Created, created)
	}
	if info.Containers != 2 {
		t.Errorf("ImageInfo.Containers = %d, want %d", info.Containers, 2)
	}
	if info.Dangling {
		t.Error("ImageInfo.Dangling = true, want false")
	}
}

func TestVolumeInfo(t *testing.T) {
	created := time.Now()
	labels := map[string]string{"env": "test"}
	info := VolumeInfo{
		Name:       "my-volume",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/my-volume/_data",
		Size:       1048576,
		Created:    created,
		Labels:     labels,
		InUse:      true,
	}

	if info.Name != "my-volume" {
		t.Errorf("VolumeInfo.Name = %q, want %q", info.Name, "my-volume")
	}
	if info.Driver != "local" {
		t.Errorf("VolumeInfo.Driver = %q, want %q", info.Driver, "local")
	}
	if info.Mountpoint != "/var/lib/docker/volumes/my-volume/_data" {
		t.Errorf("VolumeInfo.Mountpoint = %q, want %q", info.Mountpoint, "/var/lib/docker/volumes/my-volume/_data")
	}
	if info.Size != 1048576 {
		t.Errorf("VolumeInfo.Size = %d, want %d", info.Size, 1048576)
	}
	if info.Created != created {
		t.Errorf("VolumeInfo.Created = %v, want %v", info.Created, created)
	}
	if info.Labels["env"] != "test" {
		t.Errorf("VolumeInfo.Labels[\"env\"] = %q, want %q", info.Labels["env"], "test")
	}
	if !info.InUse {
		t.Error("VolumeInfo.InUse = false, want true")
	}
}

func TestNetworkInfo(t *testing.T) {
	info := NetworkInfo{
		ID:         "net123",
		Name:       "my-network",
		Driver:     "bridge",
		Scope:      "local",
		Internal:   false,
		Containers: 3,
	}

	if info.ID != "net123" {
		t.Errorf("NetworkInfo.ID = %q, want %q", info.ID, "net123")
	}
	if info.Name != "my-network" {
		t.Errorf("NetworkInfo.Name = %q, want %q", info.Name, "my-network")
	}
	if info.Driver != "bridge" {
		t.Errorf("NetworkInfo.Driver = %q, want %q", info.Driver, "bridge")
	}
	if info.Scope != "local" {
		t.Errorf("NetworkInfo.Scope = %q, want %q", info.Scope, "local")
	}
	if info.Internal {
		t.Error("NetworkInfo.Internal = true, want false")
	}
	if info.Containers != 3 {
		t.Errorf("NetworkInfo.Containers = %d, want %d", info.Containers, 3)
	}
}

func TestDiskUsageInfo(t *testing.T) {
	info := DiskUsageInfo{
		Images:           10737418240,
		Containers:       5368709120,
		Volumes:          2147483648,
		BuildCache:       1073741824,
		TotalReclaimable: 8589934592,
		Total:            19327352832,
	}

	if info.Images != 10737418240 {
		t.Errorf("DiskUsageInfo.Images = %d, want %d", info.Images, 10737418240)
	}
	if info.Containers != 5368709120 {
		t.Errorf("DiskUsageInfo.Containers = %d, want %d", info.Containers, 5368709120)
	}
	if info.Volumes != 2147483648 {
		t.Errorf("DiskUsageInfo.Volumes = %d, want %d", info.Volumes, 2147483648)
	}
	if info.BuildCache != 1073741824 {
		t.Errorf("DiskUsageInfo.BuildCache = %d, want %d", info.BuildCache, 1073741824)
	}
	if info.TotalReclaimable != 8589934592 {
		t.Errorf("DiskUsageInfo.TotalReclaimable = %d, want %d", info.TotalReclaimable, 8589934592)
	}
	if info.Total != 19327352832 {
		t.Errorf("DiskUsageInfo.Total = %d, want %d", info.Total, 19327352832)
	}
	if info.Total < info.TotalReclaimable {
		t.Errorf("Total (%d) should be >= TotalReclaimable (%d)", info.Total, info.TotalReclaimable)
	}
}
