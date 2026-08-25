package probe4

// Revision-4 additions to the kernel-identity helpers.
//
// Revision 4 has to answer two questions revisions 1-3 never asked:
//
//   - can an observer tell that the process holding `broker.lock` is STOPPED
//     rather than merely slow? That is the whole content of the honest
//     `starting-unverified` state in spec section 3.1;
//   - can an observer enumerate the same-uid processes that could plausibly be
//     that holder, so `runtime status` can REPORT candidates instead of
//     inferring one? Reporting is admissible; inferring is the defect the
//     third review named in F3.
//
// Both are answered from sysctl, never from a process describing itself.

import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Darwin `struct extern_proc` p_stat values. Only SZOMB was needed before
// revision 4; SSTOP is what makes a frozen lock holder reportable.
const (
	pStatSSLEEP int8 = 1
	pStatSRUN   = 2
	pStatSSTOP  = 4
	pStatSZOMB  = 5
)

// ProcStat returns the kernel p_stat for a pid. An error is a refusal input,
// never an absence.
func ProcStat(pid int) (int8, error) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return 0, fmt.Errorf("kern.proc.pid: %w", err)
	}
	return kp.Proc.P_stat, nil
}

// Stopped reports whether the kernel says the process is job-control stopped.
func Stopped(pid int) (bool, error) {
	st, err := ProcStat(pid)
	if err != nil {
		return false, err
	}
	return st == pStatSSTOP, nil
}

// UidProcs enumerates the pids owned by uid through kern.proc.uid. This is the
// enumeration `runtime status` needs to list candidate lock holders.
func UidProcs(uid uint32) ([]int, error) {
	raw, err := unix.SysctlRaw("kern.proc", 5 /* KERN_PROC_UID */, int(uid))
	if err != nil {
		return nil, fmt.Errorf("kern.proc.uid: %w", err)
	}
	sz := int(unsafe.Sizeof(unix.KinfoProc{}))
	if sz == 0 || len(raw)%sz != 0 {
		return nil, fmt.Errorf("kern.proc.uid: %d bytes is not a multiple of kinfo_proc %d", len(raw), sz)
	}
	var out []int
	for off := 0; off+sz <= len(raw); off += sz {
		kp := (*unix.KinfoProc)(unsafe.Pointer(&raw[off]))
		out = append(out, int(kp.Proc.P_pid))
	}
	return out, nil
}

// CandidateHolders is the REPORTING filter the specification permits: same uid,
// live, executing the observer's own binary inode, and carrying an argv that
// names this runtime key. It deliberately returns a SET. Section 10.2 refuses
// to signal even when the set has one element, because a loser frozen between
// the election and its own publication is indistinguishable from the winner.
func CandidateHolders(selfExeDev uint64, selfExeIno uint64, argvMatch func([]string) bool) ([]ProcIdentity, error) {
	uid := uint32(os.Geteuid())
	pids, err := UidProcs(uid)
	if err != nil {
		return nil, err
	}
	var out []ProcIdentity
	for _, pid := range pids {
		st, err := ProcStat(pid)
		if err != nil || st == pStatSZOMB {
			continue
		}
		id, err := Identify(pid)
		if err != nil {
			continue
		}
		if !argvMatch(id.Argv) {
			continue
		}
		var stt unix.Stat_t
		if err := unix.Stat(id.Exe, &stt); err != nil {
			continue
		}
		if uint64(stt.Dev) != uint64(selfExeDev) || uint64(stt.Ino) != selfExeIno {
			continue
		}
		out = append(out, id)
	}
	return out, nil
}

func waitStat(pid int, want int8, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if st, err := ProcStat(pid); err == nil && st == want {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	st, err := ProcStat(pid)
	return err == nil && st == want
}

func waitFile(path string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if exists(path) {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return exists(path)
}

// blockForever parks a helper process without a Go deadlock panic. `select {}`
// aborts the runtime, which would turn every held-lock helper into a zombie and
// silently break the probe it is holding the lock for.
func blockForever() {
	for {
		time.Sleep(time.Hour)
	}
}
