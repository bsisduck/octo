package format

import (
	"github.com/acarl005/stripansi"
	"github.com/dustin/go-humanize"
)

// Size formats a byte count into human-readable format.
func Size(bytes uint64) string {
	return humanize.Bytes(bytes)
}

// StripANSI removes all ANSI escape sequences from text
func StripANSI(text string) string {
	return stripansi.Strip(text)
}

// FormatText applies ANSI stripping if noColor is true
func FormatText(text string, noColor bool) string {
	if noColor {
		return StripANSI(text)
	}
	return text
}
