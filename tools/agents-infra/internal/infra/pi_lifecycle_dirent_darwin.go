//go:build darwin

package infra

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unsafe"

	"golang.org/x/sys/unix"
)

func readPiLifecycleDirentBatch(fd int, bufferSize int) ([]piLifecycleDirent, bool, error) {
	buffer := make([]byte, bufferSize)
	n, err := unix.ReadDirent(fd, buffer)
	if err != nil {
		return nil, false, err
	}
	if n == 0 {
		return nil, true, nil
	}
	buffer = buffer[:n]
	inoOffset := int(unsafe.Offsetof(unix.Dirent{}.Ino))
	seekOffset := int(unsafe.Offsetof(unix.Dirent{}.Seekoff))
	recordLengthOffset := int(unsafe.Offsetof(unix.Dirent{}.Reclen))
	nameOffset := int(unsafe.Offsetof(unix.Dirent{}.Name))
	var result []piLifecycleDirent
	for len(buffer) > 0 {
		if len(buffer) < nameOffset {
			return nil, false, errors.New("short Darwin directory record")
		}
		if inoOffset+8 > len(buffer) || seekOffset+8 > len(buffer) || recordLengthOffset+2 > len(buffer) {
			return nil, false, errors.New("short Darwin directory record fields")
		}
		inode := binary.NativeEndian.Uint64(buffer[inoOffset : inoOffset+8])
		seekOffsetValue := binary.NativeEndian.Uint64(buffer[seekOffset : seekOffset+8])
		recordLength := int(binary.NativeEndian.Uint16(buffer[recordLengthOffset : recordLengthOffset+2]))
		if recordLength < nameOffset || recordLength > len(buffer) {
			return nil, false, errors.New("invalid Darwin directory record length")
		}
		nameBytes := buffer[nameOffset:recordLength]
		if end := bytes.IndexByte(nameBytes, 0); end >= 0 {
			nameBytes = nameBytes[:end]
		}
		name := string(nameBytes)
		if inode != 0 && name != "." && name != ".." {
			result = append(result, piLifecycleDirent{Name: name, Cookie: int64(seekOffsetValue)})
		}
		buffer = buffer[recordLength:]
	}
	return result, false, nil
}
