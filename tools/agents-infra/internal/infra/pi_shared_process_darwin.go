//go:build darwin

package infra

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

const darwinProcessStateZombie = int8(5)

type sharedProcessObservation struct {
	PID       int
	PPID      int
	PGID      int
	SID       int
	UID       uint32
	StartTime ProcessStartTime
	PStat     int8
	ExecPath  string
	Argv      []string
}

func (p sharedProcessObservation) live() bool { return p.PStat != darwinProcessStateZombie }

func inspectSharedProcess(pid int) (sharedProcessObservation, error) {
	if pid <= 0 {
		return sharedProcessObservation{}, errors.New("pid must be positive")
	}
	kinfo, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return sharedProcessObservation{}, err
	}
	execPath, argv, err := readDarwinProcessArgs(pid)
	if err != nil {
		return sharedProcessObservation{}, err
	}
	sid, err := unix.Getsid(pid)
	if err != nil {
		return sharedProcessObservation{}, err
	}
	return sharedProcessObservation{
		PID:       pid,
		PPID:      int(kinfo.Eproc.Ppid),
		PGID:      int(kinfo.Eproc.Pgid),
		SID:       sid,
		UID:       kinfo.Eproc.Ucred.Uid,
		StartTime: ProcessStartTime{Seconds: kinfo.Proc.P_starttime.Sec, Microseconds: kinfo.Proc.P_starttime.Usec},
		PStat:     kinfo.Proc.P_stat,
		ExecPath:  execPath,
		Argv:      argv,
	}, nil
}

func inspectSharedProcessKernel(pid int) (sharedProcessObservation, error) {
	kinfo, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return sharedProcessObservation{}, err
	}
	sid, err := unix.Getsid(pid)
	if err != nil {
		return sharedProcessObservation{}, err
	}
	return sharedProcessObservation{
		PID:       pid,
		PPID:      int(kinfo.Eproc.Ppid),
		PGID:      int(kinfo.Eproc.Pgid),
		SID:       sid,
		UID:       kinfo.Eproc.Ucred.Uid,
		StartTime: ProcessStartTime{Seconds: kinfo.Proc.P_starttime.Sec, Microseconds: kinfo.Proc.P_starttime.Usec},
		PStat:     kinfo.Proc.P_stat,
	}, nil
}

func sharedProcessGone(err error) bool {
	return errors.Is(err, syscall.ESRCH) || errors.Is(err, syscall.EIO)
}

func readDarwinProcessArgs(pid int) (string, []string, error) {
	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return "", nil, err
	}
	if len(raw) < 4 {
		return "", nil, errors.New("kern.procargs2 response is truncated")
	}
	argc := int(binary.LittleEndian.Uint32(raw[:4]))
	if argc < 0 || argc > 1<<20 {
		return "", nil, errors.New("kern.procargs2 argc is invalid")
	}
	rest := raw[4:]
	pathEnd := bytes.IndexByte(rest, 0)
	if pathEnd <= 0 {
		return "", nil, errors.New("kern.procargs2 executable path is absent")
	}
	execPath := string(rest[:pathEnd])
	rest = rest[pathEnd+1:]
	for len(rest) > 0 && rest[0] == 0 {
		rest = rest[1:]
	}
	argv := make([]string, 0, argc)
	for len(argv) < argc {
		end := bytes.IndexByte(rest, 0)
		if end < 0 {
			return "", nil, errors.New("kern.procargs2 argv is truncated")
		}
		argv = append(argv, string(rest[:end]))
		rest = rest[end+1:]
	}
	return execPath, argv, nil
}

func resolvedExecutableIdentity(path string) (string, FileIdentity, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", FileIdentity{}, err
	}
	identity, err := fileIdentity(resolved)
	if err != nil {
		return "", FileIdentity{}, err
	}
	return resolved, identity, nil
}

func ownResolvedExecutableIdentity() (string, FileIdentity, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", FileIdentity{}, err
	}
	return resolvedExecutableIdentity(executable)
}

func sharedUnixPeerIdentity(connection *net.UnixConn) (uint32, int, error) {
	raw, err := connection.SyscallConn()
	if err != nil {
		return 0, 0, err
	}
	var uid uint32
	var pid int
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		credential, err := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
		if err != nil {
			controlErr = err
			return
		}
		peerPID, err := unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
		if err != nil {
			controlErr = err
			return
		}
		uid, pid = credential.Uid, peerPID
	}); err != nil {
		return 0, 0, err
	}
	if controlErr != nil {
		return 0, 0, controlErr
	}
	if pid <= 0 {
		return 0, 0, fmt.Errorf("invalid peer pid %d", pid)
	}
	return uid, pid, nil
}

func processExecIdentity(observation sharedProcessObservation) (FileIdentity, error) {
	_, identity, err := resolvedExecutableIdentity(observation.ExecPath)
	return identity, err
}
