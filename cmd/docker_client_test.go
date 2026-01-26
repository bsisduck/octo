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
	info := ContainerInfo{
		ID:      "abc123",
		Name:    "test-container",
		Image:   "nginx:latest",
		Status:  "Up 2 hours",
		State:   "running",
		Created: time.Now(),
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
}

func TestImageInfo(t *testing.T) {
	info := ImageInfo{
		ID:         "sha256abc",
		Repository: "nginx",
		Tag:        "latest",
		Size:       52428800,
		Created:    time.Now(),
		Containers: 2,
		Dangling:   false,
	}

	if info.Repository != "nginx" {
		t.Errorf("ImageInfo.Repository = %q, want %q", info.Repository, "nginx")
	}
	if info.Dangling {
		t.Error("ImageInfo.Dangling = true, want false")
	}
}

func TestVolumeInfo(t *testing.T) {
	info := VolumeInfo{
		Name:       "my-volume",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/my-volume/_data",
		Size:       1048576,
		Created:    time.Now(),
		Labels:     map[string]string{"env": "test"},
		InUse:      true,
	}

	if info.Name != "my-volume" {
		t.Errorf("VolumeInfo.Name = %q, want %q", info.Name, "my-volume")
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

	if info.Name != "my-network" {
		t.Errorf("NetworkInfo.Name = %q, want %q", info.Name, "my-network")
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

	if info.Total < info.TotalReclaimable {
		t.Errorf("Total (%d) should be >= TotalReclaimable (%d)", info.Total, info.TotalReclaimable)
	}
}
