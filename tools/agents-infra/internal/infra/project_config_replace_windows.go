//go:build windows

package infra

import "github.com/natefinch/atomic"

// replaceProjectConfigFile uses atomic.ReplaceFile's documented all-or-nothing
// Windows implementation, backed by MoveFileExW with REPLACE_EXISTING and
// WRITE_THROUGH. The source is staged beside the destination by the caller.
func replaceProjectConfigFile(source, destination string) error {
	return atomic.ReplaceFile(source, destination)
}
