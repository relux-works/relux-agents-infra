//go:build !aix && !android && !darwin && !dragonfly && !freebsd && !illumos && !ios && !linux && !netbsd && !openbsd && !solaris && !windows

package infra

import (
	"fmt"
	"runtime"
)

// Unsupported targets fail closed rather than silently using a replacement
// primitive without a documented atomicity guarantee.
func replaceProjectConfigFile(_, _ string) error {
	return fmt.Errorf("atomic project config replacement is unsupported on %s", runtime.GOOS)
}
