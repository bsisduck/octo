package docker

import (
	"testing"
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
