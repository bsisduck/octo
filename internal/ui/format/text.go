package format

import "github.com/dustin/go-humanize"

// Size formats a byte count into human-readable format.
func Size(bytes uint64) string {
	return humanize.Bytes(bytes)
}
