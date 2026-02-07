package format

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFormatJSON(t *testing.T) {
	data := map[string]interface{}{
		"name": "test-container",
		"status": "running",
		"count": 42,
	}

	var buf bytes.Buffer
	err := FormatJSON(&buf, data)
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	// Verify JSON is valid
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("JSON output is invalid: %v", err)
	}

	if result["name"] != "test-container" {
		t.Errorf("Expected name='test-container', got %v", result["name"])
	}

	// Verify indentation (2 spaces)
	if !strings.Contains(buf.String(), "  ") {
		t.Error("JSON should be indented with 2 spaces")
	}
}

func TestFormatYAML(t *testing.T) {
	data := map[string]interface{}{
		"name": "test-volume",
		"size": 1024,
	}

	var buf bytes.Buffer
	err := FormatYAML(&buf, data)
	if err != nil {
		t.Fatalf("FormatYAML failed: %v", err)
	}

	// Verify YAML is valid
	var result map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("YAML output is invalid: %v", err)
	}

	if result["name"] != "test-volume" {
		t.Errorf("Expected name='test-volume', got %v", result["name"])
	}
}

func TestMarshalResponse(t *testing.T) {
	data := map[string]string{"key": "value"}
	response := MarshalResponse(data, true, "success message")

	if response["success"] != true {
		t.Error("Expected success=true")
	}
	if response["message"] != "success message" {
		t.Error("Expected message to match")
	}
	if response["data"] == nil {
		t.Error("Expected data field to be set")
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "color codes",
			input: "\x1b[31mRed\x1b[0m",
			want:  "Red",
		},
		{
			name:  "no codes",
			input: "plain text",
			want:  "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatText(t *testing.T) {
	input := "\x1b[31mRed\x1b[0m"

	// With noColor=false, should preserve codes
	got := FormatText(input, false)
	if got != input {
		t.Errorf("FormatText(noColor=false) should preserve codes")
	}

	// With noColor=true, should strip codes
	got = FormatText(input, true)
	if got != "Red" {
		t.Errorf("FormatText(noColor=true) = %q, want %q", got, "Red")
	}
}
