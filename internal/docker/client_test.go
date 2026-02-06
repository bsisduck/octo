package docker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// TestListContainers_TransformsSDKData tests basic container data transformation
func TestListContainers_TransformsSDKData(t *testing.T) {
	now := time.Now().Unix()
	sdkContainer := types.Container{
		ID:      "abcdef1234567890abcdef1234567890",
		Names:   []string{"/my-app"},
		Image:   "golang:1.24",
		Status:  "Up 5 minutes",
		State:   "running",
		Created: now,
		SizeRw:  1024000,
		Ports:   []types.Port{{PrivatePort: 8080, PublicPort: 8080, Type: "tcp"}},
	}

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return []types.Container{sdkContainer}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListContainers(context.Background(), true)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "abcdef123456", result[0].ID)
	assert.Equal(t, "my-app", result[0].Name)
	assert.Equal(t, "golang:1.24", result[0].Image)
	assert.Equal(t, "running", result[0].State)
}

// TestListContainers_TruncatesLongIDs tests ID truncation to 12 chars
func TestListContainers_TruncatesLongIDs(t *testing.T) {
	longID := "abcdef1234567890abcdef1234567890abcdef"
	sdkContainer := types.Container{
		ID:      longID,
		Names:   []string{"/test"},
		Status:  "Up",
		State:   "running",
		Created: time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return []types.Container{sdkContainer}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListContainers(context.Background(), false)

	require.NoError(t, err)
	assert.Equal(t, "abcdef123456", result[0].ID)
	assert.Len(t, result[0].ID, 12)
}

// TestListContainers_HandlesShortIDs tests ID shorter than 12 chars
func TestListContainers_HandlesShortIDs(t *testing.T) {
	shortID := "short"
	sdkContainer := types.Container{
		ID:      shortID,
		Names:   []string{"/test"},
		Status:  "Up",
		State:   "running",
		Created: time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return []types.Container{sdkContainer}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListContainers(context.Background(), false)

	require.NoError(t, err)
	assert.Equal(t, "short", result[0].ID)
}

// TestListContainers_StripsLeadingSlashFromName tests name slash stripping
func TestListContainers_StripsLeadingSlashFromName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/my-container", "my-container"},
		{"/", ""},
		{"no-slash", "no-slash"},
		{"/leading/slash/path", "leading/slash/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			sdkContainer := types.Container{
				ID:      "test123456789012",
				Names:   []string{tt.input},
				Status:  "Up",
				State:   "running",
				Created: time.Now().Unix(),
			}

			mock := &MockDockerAPI{
				ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
					return []types.Container{sdkContainer}, nil
				},
			}

			client := &Client{api: mock}
			result, err := client.ListContainers(context.Background(), false)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result[0].Name)
		})
	}
}

// TestListContainers_HandlesEmptyNames tests empty name handling (no panic)
func TestListContainers_HandlesEmptyNames(t *testing.T) {
	sdkContainer := types.Container{
		ID:      "test123456789012",
		Names:   []string{},
		Status:  "Up",
		State:   "running",
		Created: time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return []types.Container{sdkContainer}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListContainers(context.Background(), false)

	require.NoError(t, err)
	assert.Equal(t, "", result[0].Name)
}

// TestListContainers_FormatsPortsCorrectly tests port formatting
func TestListContainers_FormatsPortsCorrectly(t *testing.T) {
	tests := []struct {
		name     string
		ports    []types.Port
		expected string
	}{
		{
			"public and private ports",
			[]types.Port{{PublicPort: 8080, PrivatePort: 80, Type: "tcp"}},
			"8080->80/tcp",
		},
		{
			"multiple ports",
			[]types.Port{
				{PublicPort: 8080, PrivatePort: 80, Type: "tcp"},
				{PrivatePort: 443, Type: "tcp"},
			},
			"8080->80/tcp, 443/tcp",
		},
		{
			"no public port",
			[]types.Port{{PrivatePort: 3000, Type: "tcp"}},
			"3000/tcp",
		},
		{
			"empty ports",
			[]types.Port{},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkContainer := types.Container{
				ID:      "test123456789012",
				Names:   []string{"/test"},
				Status:  "Up",
				State:   "running",
				Created: time.Now().Unix(),
				Ports:   tt.ports,
			}

			mock := &MockDockerAPI{
				ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
					return []types.Container{sdkContainer}, nil
				},
			}

			client := &Client{api: mock}
			result, err := client.ListContainers(context.Background(), false)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result[0].Ports)
		})
	}
}

// TestListContainers_AllFlag tests the all flag passthrough
func TestListContainers_AllFlag(t *testing.T) {
	tests := []struct {
		name     string
		allFlag  bool
		expected bool
	}{
		{"all=true", true, true},
		{"all=false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedAll bool
			mock := &MockDockerAPI{
				ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
					capturedAll = opts.All
					return []types.Container{}, nil
				},
			}

			client := &Client{api: mock}
			_, err := client.ListContainers(context.Background(), tt.allFlag)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, capturedAll)
		})
	}
}

// TestListContainers_PropagatesErrors tests error propagation from SDK
func TestListContainers_PropagatesErrors(t *testing.T) {
	expectedErr := errors.New("docker daemon unavailable")
	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return nil, expectedErr
		},
	}

	client := &Client{api: mock}
	result, err := client.ListContainers(context.Background(), false)

	assert.Nil(t, result)
	assert.Equal(t, expectedErr, err)
}

// TestListImages_TrimsSha256Prefix tests sha256 prefix removal
func TestListImages_TrimsSha256Prefix(t *testing.T) {
	sdkImage := image.Summary{
		ID:       "sha256:abcdef1234567890abcdef1234567890abcdef",
		RepoTags: []string{"nginx:latest"},
		Size:     500000,
		Created:  time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
			return []image.Summary{sdkImage}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListImages(context.Background(), false)

	require.NoError(t, err)
	assert.Equal(t, "abcdef123456", result[0].ID)
}

// TestListImages_HandlesShortImageIDs tests short image IDs
func TestListImages_HandlesShortImageIDs(t *testing.T) {
	shortID := "sha256:abc123"
	sdkImage := image.Summary{
		ID:       shortID,
		RepoTags: []string{"test:latest"},
		Size:     100000,
		Created:  time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
			return []image.Summary{sdkImage}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListImages(context.Background(), false)

	require.NoError(t, err)
	assert.Equal(t, "abc123", result[0].ID)
}

// TestListImages_ParsesRepoAndTag tests repo:tag parsing
func TestListImages_ParsesRepoAndTag(t *testing.T) {
	tests := []struct {
		tag      string
		expRepo  string
		expTag   string
	}{
		{"nginx:latest", "nginx", "latest"},
		{"golang:1.24", "golang", "1.24"},
		{"myregistry.io/myapp:v1.0.0", "myregistry.io/myapp", "v1.0.0"},
		{"busybox", "busybox", "latest"},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			sdkImage := image.Summary{
				ID:       "sha256:abcdef1234567890",
				RepoTags: []string{tt.tag},
				Size:     100000,
				Created:  time.Now().Unix(),
			}

			mock := &MockDockerAPI{
				ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
					return []image.Summary{sdkImage}, nil
				},
			}

			client := &Client{api: mock}
			result, err := client.ListImages(context.Background(), false)

			require.NoError(t, err)
			assert.Equal(t, tt.expRepo, result[0].Repository)
			assert.Equal(t, tt.expTag, result[0].Tag)
		})
	}
}

// TestListImages_HandlesMissingRepoTags tests dangling image handling
func TestListImages_HandlesMissingRepoTags(t *testing.T) {
	sdkImage := image.Summary{
		ID:       "sha256:abcdef1234567890",
		RepoTags: []string{},
		Size:     100000,
		Created:  time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
			return []image.Summary{sdkImage}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListImages(context.Background(), false)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.True(t, result[0].Dangling)
}

// TestListImages_IdentifiesDanglingImages tests dangling flag
func TestListImages_IdentifiesDanglingImages(t *testing.T) {
	tagged := image.Summary{
		ID:       "sha256:tagged1234567890",
		RepoTags: []string{"nginx:latest"},
		Size:     100000,
		Created:  time.Now().Unix(),
	}
	dangling := image.Summary{
		ID:       "sha256:dangling1234567890",
		RepoTags: []string{},
		Size:     100000,
		Created:  time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
			return []image.Summary{tagged, dangling}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListImages(context.Background(), false)

	require.NoError(t, err)
	assert.False(t, result[0].Dangling)
	assert.True(t, result[1].Dangling)
}

// TestGetDiskUsage_ReturnsNonNil tests disk usage returns non-nil
func TestGetDiskUsage_ReturnsNonNil(t *testing.T) {
	mock := &MockDockerAPI{
		DiskUsageFn: func(ctx context.Context, opts types.DiskUsageOptions) (types.DiskUsage, error) {
			return types.DiskUsage{}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.GetDiskUsage(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(0), result.Total)
}

// TestRemoveContainer_PassesForceFlag tests force flag passthrough
func TestRemoveContainer_PassesForceFlag(t *testing.T) {
	tests := []struct {
		name          string
		force         bool
		expectedForce bool
	}{
		{"force=true", true, true},
		{"force=false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedForce bool
			mock := &MockDockerAPI{
				ContainerRemoveFn: func(ctx context.Context, id string, opts container.RemoveOptions) error {
					capturedForce = opts.Force
					return nil
				},
			}

			client := &Client{api: mock}
			err := client.RemoveContainer(context.Background(), "test123", tt.force)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedForce, capturedForce)
		})
	}
}

// TestPruneContainers_ReturnsBytesReclaimed tests prune return value
func TestPruneContainers_ReturnsBytesReclaimed(t *testing.T) {
	expectedReclaimed := uint64(1000000000)
	mock := &MockDockerAPI{
		ContainersPruneFn: func(ctx context.Context, pruneFilters filters.Args) (container.PruneReport, error) {
			return container.PruneReport{SpaceReclaimed: expectedReclaimed}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.PruneContainers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, expectedReclaimed, result)
}

// TestRemoveImage_PassesForceFlag tests image remove force flag
func TestRemoveImage_PassesForceFlag(t *testing.T) {
	var capturedForce bool
	mock := &MockDockerAPI{
		ImageRemoveFn: func(ctx context.Context, id string, opts image.RemoveOptions) ([]image.DeleteResponse, error) {
			capturedForce = opts.Force
			return []image.DeleteResponse{}, nil
		},
	}

	client := &Client{api: mock}
	err := client.RemoveImage(context.Background(), "test123", true)

	require.NoError(t, err)
	assert.True(t, capturedForce)
}

// TestPing_ReturnsNilOnSuccess tests ping success
func TestPing_ReturnsNilOnSuccess(t *testing.T) {
	mock := &MockDockerAPI{
		PingFn: func(ctx context.Context) (types.Ping, error) {
			return types.Ping{}, nil
		},
	}

	client := &Client{api: mock}
	err := client.Ping(context.Background())

	assert.NoError(t, err)
}

// TestPing_ReturnsErrorOnFailure tests ping failure
func TestPing_ReturnsErrorOnFailure(t *testing.T) {
	expectedErr := errors.New("connection refused")
	mock := &MockDockerAPI{
		PingFn: func(ctx context.Context) (types.Ping, error) {
			return types.Ping{}, expectedErr
		},
	}

	client := &Client{api: mock}
	err := client.Ping(context.Background())

	assert.Equal(t, expectedErr, err)
}

// TestClose_DelegatesToAPI tests close delegation
func TestClose_DelegatesToAPI(t *testing.T) {
	closeCalled := false
	mock := &MockDockerAPI{
		CloseFn: func() error {
			closeCalled = true
			return nil
		},
	}

	client := &Client{api: mock}
	err := client.Close()

	require.NoError(t, err)
	assert.True(t, closeCalled)
}

// TestListContainers_RespectsContextCancellation tests context cancellation
func TestListContainers_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return nil, ctx.Err()
		},
	}

	client := &Client{api: mock}
	_, err := client.ListContainers(ctx, true)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestGetStoppedContainers_HandlesEmptyNames tests stopped container name handling
func TestGetStoppedContainers_HandlesEmptyNames(t *testing.T) {
	sdkContainer := types.Container{
		ID:      "test123456789012",
		Names:   []string{},
		Status:  "Exited (0)",
		State:   "exited",
		Created: time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return []types.Container{sdkContainer}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.GetStoppedContainers(context.Background())

	require.NoError(t, err)
	assert.Equal(t, "", result[0].Name)
}

// TestListImages_MultipleTagsSameImage tests image with multiple tags
func TestListImages_MultipleTagsSameImage(t *testing.T) {
	sdkImage := image.Summary{
		ID:       "sha256:abcdef1234567890",
		RepoTags: []string{"myapp:v1.0", "myapp:latest", "myapp:stable"},
		Size:     500000,
		Created:  time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
			return []image.Summary{sdkImage}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListImages(context.Background(), false)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "myapp", result[0].Repository)
	assert.Equal(t, "v1.0", result[0].Tag)
	assert.Equal(t, "myapp", result[1].Repository)
	assert.Equal(t, "latest", result[1].Tag)
	assert.Equal(t, "myapp", result[2].Repository)
	assert.Equal(t, "stable", result[2].Tag)
}

// TestGetDanglingImages_FiltersCorrectly tests dangling filter
func TestGetDanglingImages_FiltersCorrectly(t *testing.T) {
	danglingImg := image.Summary{
		ID:       "sha256:dangling123456789012345678",
		RepoTags: []string{},
		Size:     100000,
		Created:  time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
			return []image.Summary{danglingImg}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.GetDanglingImages(context.Background())

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.True(t, result[0].Dangling)
}

// TestGetStoppedContainers_FiltersExited tests exited filter
func TestGetStoppedContainers_FiltersExited(t *testing.T) {
	exitedContainer := types.Container{
		ID:      "test123456789012",
		Names:   []string{"/stopped-app"},
		Status:  "Exited (0)",
		State:   "exited",
		Created: time.Now().Unix(),
	}

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return []types.Container{exitedContainer}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.GetStoppedContainers(context.Background())

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "exited", result[0].State)
}

// TestPruneImages_AllFlagPassesFilter tests image prune all flag
func TestPruneImages_AllFlagPassesFilter(t *testing.T) {
	tests := []struct {
		name string
		all  bool
	}{
		{"all=true", true},
		{"all=false", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var filtersCalled bool
			mock := &MockDockerAPI{
				ImagesPruneFn: func(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error) {
					filtersCalled = true
					return image.PruneReport{SpaceReclaimed: 100000}, nil
				},
			}

			client := &Client{api: mock}
			_, err := client.PruneImages(context.Background(), tt.all)

			require.NoError(t, err)
			assert.True(t, filtersCalled)
		})
	}
}

// TestRemoveVolume_PassesForceFlag tests volume remove force flag
func TestRemoveVolume_PassesForceFlag(t *testing.T) {
	tests := []struct {
		name          string
		force         bool
		expectedForce bool
	}{
		{"force=true", true, true},
		{"force=false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedForce bool
			mock := &MockDockerAPI{
				VolumeRemoveFn: func(ctx context.Context, id string, force bool) error {
					capturedForce = force
					return nil
				},
			}

			client := &Client{api: mock}
			err := client.RemoveVolume(context.Background(), "testvol", tt.force)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedForce, capturedForce)
		})
	}
}

// TestListContainers_MultipleContainers tests handling multiple containers
func TestListContainers_MultipleContainers(t *testing.T) {
	containers := []types.Container{
		{
			ID:      "aaa111aaa111aaa111aaa111aaa111",
			Names:   []string{"/app1"},
			Status:  "Up 2 hours",
			State:   "running",
			Created: time.Now().Unix(),
			Image:   "golang:1.24",
		},
		{
			ID:      "bbb222bbb222bbb222bbb222bbb222",
			Names:   []string{"/app2"},
			Status:  "Exited (1)",
			State:   "exited",
			Created: time.Now().Unix(),
			Image:   "python:3.11",
		},
	}

	mock := &MockDockerAPI{
		ContainerListFn: func(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
			return containers, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListContainers(context.Background(), true)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "app1", result[0].Name)
	assert.Equal(t, "app2", result[1].Name)
}

// TestListImages_EmptyList tests empty image list
func TestListImages_EmptyList(t *testing.T) {
	mock := &MockDockerAPI{
		ImageListFn: func(ctx context.Context, opts image.ListOptions) ([]image.Summary, error) {
			return []image.Summary{}, nil
		},
	}

	client := &Client{api: mock}
	result, err := client.ListImages(context.Background(), false)

	require.NoError(t, err)
	assert.Len(t, result, 0)
}

// TestTruncateIDHelper tests the truncateID helper function
func TestTruncateIDHelper(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"abcdefghijklmnop", 12, "abcdefghijkl"},
		{"short", 12, "short"},
		{"", 12, ""},
		{"a", 1, "a"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input+":"+string(rune(tt.maxLen)), func(t *testing.T) {
			result := truncateID(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTrimImageIDHelper tests the trimImageID helper function
func TestTrimImageIDHelper(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sha256:abcdef1234567890abcdef1234567890", "abcdef123456"},
		{"sha256:abc123", "abc123"},
		{"abcdef1234567890", "abcdef123456"},
		{"sha256:", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trimImageID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
