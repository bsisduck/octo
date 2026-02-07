package format

import (
	"encoding/json"
	"io"
)

// FormatJSON marshals data to JSON with pretty-printing
func FormatJSON(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")  // 2-space indentation
	return encoder.Encode(data)
}

// MarshalResponse wraps data in common response structure
func MarshalResponse(data interface{}, success bool, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": success,
		"message": message,
		"data":    data,
	}
}
