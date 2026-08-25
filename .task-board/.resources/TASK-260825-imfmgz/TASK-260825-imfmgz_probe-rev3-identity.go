package probe3

// Shared kernel-identity helpers. Every process fact in this probe suite comes
// from sysctl, never from the process describing itself.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// ProcIdentity is the kernel-sourced identity tuple the specification attests.
type ProcIdentity struct {
	Pid   int
	Uid   uint32
	Start unix.Timeval
	Exe   string
	Argv  []string
}

func (p ProcIdentity) StartKey() string {
	return fmt.Sprintf("%d.%06d", p.Start.Sec, p.Start.Usec)
}

// Identify reads uid + start time from kern.proc.pid and exec path + exact argv
// from kern.procargs2. A dead pid or a foreign-uid pid returns an error, which
// the specification treats as a refusal input, never as an absence.
func Identify(pid int) (ProcIdentity, error) {
	id := ProcIdentity{Pid: pid}
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return id, fmt.Errorf("kern.proc.pid: %w", err)
	}
	id.Uid = kp.Eproc.Ucred.Uid
	id.Start = kp.Proc.P_starttime

	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return id, fmt.Errorf("kern.procargs2: %w", err)
	}
	if len(raw) < 4 {
		return id, fmt.Errorf("kern.procargs2 short")
	}
	argc := int(binary.LittleEndian.Uint32(raw[:4]))
	rest := raw[4:]
	i := bytes.IndexByte(rest, 0)
	if i < 0 {
		return id, fmt.Errorf("kern.procargs2 no exec path")
	}
	id.Exe = string(rest[:i])
	rest = rest[i:]
	for len(rest) > 0 && rest[0] == 0 {
		rest = rest[1:]
	}
	for _, p := range bytes.Split(rest, []byte{0}) {
		if len(id.Argv) >= argc {
			break
		}
		id.Argv = append(id.Argv, string(p))
	}
	return id, nil
}

// Alive is the CORRECTED liveness predicate. `kern.proc.pid` succeeds for an
// exited-but-unreaped process, so "the call returned" is not "the process is
// running" - see P15, which was found because this very helper reported a
// SIGKILLed broker as live and broke P14.
func Alive(pid int) bool {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return false
	}
	return kp.Proc.P_stat != 5 // SZOMB
}

// InProcessTable is the naive predicate, kept only so probes can show the
// difference between it and Alive.
func InProcessTable(pid int) bool {
	_, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	return err == nil
}

// shortTempDir keeps AF_UNIX paths inside sun_path. t.TempDir() embeds the full
// subtest name and overflows the 103-byte limit documented in spec section 5.2.
func shortTempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("/tmp", "p3")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(d) })
	return d
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func waitGone(pid int, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if !Alive(pid) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return !Alive(pid)
}
