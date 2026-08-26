//go:build !windows && !darwin && !linux

package infra

import (
	"io"
	"os"
)

func piTerminalFD(reader io.Reader) (int, bool) {
	file, ok := reader.(*os.File)
	if !ok {
		return 0, false
	}
	info, err := file.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return 0, false
	}
	return int(file.Fd()), true
}
