package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

func procIdentity(pid int) (uid uint32, start unix.Timeval, exe string, argv []string, err error) {
	kp, err := unix.SysctlKinfoProc("kern.proc.pid", pid)
	if err != nil {
		return 0, unix.Timeval{}, "", nil, fmt.Errorf("kinfo: %w", err)
	}
	uid = kp.Eproc.Ucred.Uid
	start = unix.Timeval{Sec: kp.Proc.P_starttime.Sec, Usec: kp.Proc.P_starttime.Usec}
	raw, err := unix.SysctlRaw("kern.procargs2", pid)
	if err != nil {
		return uid, start, "", nil, fmt.Errorf("procargs2: %w", err)
	}
	if len(raw) < 4 {
		return uid, start, "", nil, fmt.Errorf("procargs2 short")
	}
	argc := int(binary.LittleEndian.Uint32(raw[:4]))
	rest := raw[4:]
	i := bytes.IndexByte(rest, 0)
	if i < 0 {
		return uid, start, "", nil, fmt.Errorf("no exec path")
	}
	exe = string(rest[:i])
	rest = rest[i:]
	for len(rest) > 0 && rest[0] == 0 {
		rest = rest[1:]
	}
	parts := bytes.Split(rest, []byte{0})
	for _, p := range parts {
		if len(argv) >= argc {
			break
		}
		argv = append(argv, string(p))
	}
	return uid, start, exe, argv, nil
}

func main() {
	self := os.Getpid()
	uid, start, exe, argv, err := procIdentity(self)
	fmt.Printf("SELF pid=%d uid=%d start=%d.%06d exe=%q argv=%q err=%v\n", self, uid, start.Sec, start.Usec, exe, argv, err)

	// child with distinct argv
	cmd := exec.Command("/bin/sleep", "97")
	if err := cmd.Start(); err != nil {
		fmt.Println("CHILD_START_ERR", err)
		return
	}
	cpid := cmd.Process.Pid
	cuid, cstart, cexe, cargv, cerr := procIdentity(cpid)
	fmt.Printf("CHILD pid=%d uid=%d start=%d.%06d exe=%q argv=%q err=%v\n", cpid, cuid, cstart.Sec, cstart.Usec, cexe, cargv, cerr)
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
	if _, _, _, _, err := procIdentity(cpid); err != nil {
		fmt.Printf("DEAD_PID_ERR ok=%v err=%v\n", err != nil, err)
	} else {
		fmt.Println("DEAD_PID_STILL_READABLE")
	}

	// pid 1 (different uid) readability
	if u, _, e, a, err := procIdentity(1); err == nil {
		fmt.Printf("PID1 uid=%d exe=%q argvlen=%d\n", u, e, len(a))
	} else {
		fmt.Println("PID1_ERR", err)
	}

	// UDS peer credentials
	dir, _ := os.MkdirTemp("", "probe")
	sock := dir + "/s"
	ln, err := net.Listen("unix", sock)
	if err != nil {
		fmt.Println("UDS_LISTEN_ERR", err)
		return
	}
	done := make(chan struct{})
	go func() {
		c, err := ln.Accept()
		if err != nil {
			fmt.Println("ACCEPT_ERR", err)
			close(done)
			return
		}
		uc := c.(*net.UnixConn)
		raw, _ := uc.SyscallConn()
		_ = raw.Control(func(fd uintptr) {
			xu, e1 := unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
			p, e2 := unix.GetsockoptInt(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERPID)
			uidv := uint32(0)
			if xu != nil {
				uidv = xu.Uid
			}
			fmt.Printf("PEER uid=%d err=%v pid=%d piderr=%v selfpid=%d\n", uidv, e1, p, e2, os.Getpid())
		})
		c.Close()
		close(done)
	}()
	c, err := net.Dial("unix", sock)
	if err != nil {
		fmt.Println("UDS_DIAL_ERR", err)
	} else {
		<-done
		c.Close()
	}
	ln.Close()

	// sun_path limit
	long := dir + "/" + strings.Repeat("a", 120) + ".sock"
	if _, err := net.Listen("unix", long); err != nil {
		fmt.Printf("LONG_PATH_REFUSED len=%d err=%v\n", len(long), err)
	} else {
		fmt.Printf("LONG_PATH_ACCEPTED len=%d\n", len(long))
	}
	fmt.Printf("CACHE_DIR=%s\n", must(os.UserCacheDir()))
}

func must(s string, err error) string {
	if err != nil {
		return "ERR:" + err.Error()
	}
	return s
}
