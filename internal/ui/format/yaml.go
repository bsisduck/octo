package format

import (
	"io"

	"gopkg.in/yaml.v3"
)

// FormatYAML marshals data to YAML
func FormatYAML(w io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2) // 2-space indentation
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(data)
}
