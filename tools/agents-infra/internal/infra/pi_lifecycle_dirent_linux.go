//go:build linux

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
	offsetOffset := int(unsafe.Offsetof(unix.Dirent{}.Off))
	recordLengthOffset := int(unsafe.Offsetof(unix.Dirent{}.Reclen))
	nameOffset := int(unsafe.Offsetof(unix.Dirent{}.Name))
	var result []piLifecycleDirent
	for len(buffer) > 0 {
		if len(buffer) < nameOffset {
			return nil, false, errors.New("short Linux directory record")
		}
		if inoOffset+8 > len(buffer) || offsetOffset+8 > len(buffer) || recordLengthOffset+2 > len(buffer) {
			return nil, false, errors.New("short Linux directory record fields")
		}
		inode := binary.NativeEndian.Uint64(buffer[inoOffset : inoOffset+8])
		offset := int64(binary.NativeEndian.Uint64(buffer[offsetOffset : offsetOffset+8]))
		recordLength := int(binary.NativeEndian.Uint16(buffer[recordLengthOffset : recordLengthOffset+2]))
		if recordLength < nameOffset || recordLength > len(buffer) {
			return nil, false, errors.New("invalid Linux directory record length")
		}
		nameBytes := buffer[nameOffset:recordLength]
		if end := bytes.IndexByte(nameBytes, 0); end >= 0 {
			nameBytes = nameBytes[:end]
		}
		name := string(nameBytes)
		if inode != 0 && name != "." && name != ".." {
			result = append(result, piLifecycleDirent{Name: name, Cookie: offset})
		}
		buffer = buffer[recordLength:]
	}
	return result, false, nil
}
