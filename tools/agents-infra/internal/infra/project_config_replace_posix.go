//go:build aix || android || darwin || dragonfly || freebsd || illumos || ios || linux || netbsd || openbsd || solaris

package infra

import "os"

// replaceProjectConfigFile relies on the POSIX atomic rename contract. Setup
// stages the source beside the destination so both paths are on one filesystem.
func replaceProjectConfigFile(source, destination string) error {
	return os.Rename(source, destination)
}
