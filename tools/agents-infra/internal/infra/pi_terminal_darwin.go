//go:build darwin

package infra

import (
	"io"
	"os"

	"golang.org/x/sys/unix"
)

func piTerminalFD(reader io.Reader) (int, bool) {
	file, ok := reader.(*os.File)
	if !ok {
		return 0, false
	}
	fd := int(file.Fd())
	if _, err := unix.IoctlGetTermios(fd, unix.TIOCGETA); err != nil {
		return 0, false
	}
	return fd, true
}
