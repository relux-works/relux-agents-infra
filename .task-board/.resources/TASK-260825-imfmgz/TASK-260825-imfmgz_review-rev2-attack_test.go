package attack

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestLockGateBypassDuringPeerStartup(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "r2-lock-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	lockPath := filepath.Join(dir, "broker.lock")
	socketPath := filepath.Join(dir, "b.sock")

	holder, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if err := unix.Flock(int(holder.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	// This descriptor names the right inode but does not itself hold the lock.
	// It is the shape missing from producer probe P2: another open-file
	// description (the legitimate starting broker) holds the lock.
	unlockedFD3, err := os.OpenFile(lockPath, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer unlockedFD3.Close()

	portProbe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := portProbe.Addr().String()
	portProbe.Close()

	cmd := exec.Command(os.Args[0], "-test.run=TestLockGateAttackChild", "-test.v")
	cmd.Env = append(os.Environ(),
		"ATTACK_CHILD=lock-gate",
		"ATTACK_LOCK_PATH="+lockPath,
		"ATTACK_SOCKET_PATH="+socketPath,
		"ATTACK_ENDPOINT="+endpoint,
	)
	cmd.ExtraFiles = []*os.File{unlockedFD3}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ATTACK_RESULT=all_three_specified_refusals_bypassed") {
		t.Fatalf("bypass did not reproduce:\n%s", out)
	}
	t.Logf("fd3 was correct-but-unlocked while a peer held broker.lock; the second-descriptor gate passed, the runtime port preflight passed, and rendezvous bind passed")
}

func TestLockGateAttackChild(t *testing.T) {
	if os.Getenv("ATTACK_CHILD") != "lock-gate" {
		t.Skip("parent-driven")
	}
	lockPath := os.Getenv("ATTACK_LOCK_PATH")

	var got, want unix.Stat_t
	if err := unix.Fstat(3, &got); err != nil {
		t.Fatal(err)
	}
	if err := unix.Stat(lockPath, &want); err != nil {
		t.Fatal(err)
	}
	if got.Dev != want.Dev || got.Ino != want.Ino {
		t.Fatal("fd3 does not identify broker.lock")
	}
	second, err := os.OpenFile(lockPath, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if err := unix.Flock(int(second.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != unix.EWOULDBLOCK {
		t.Fatalf("lock gate did not pass: %v", err)
	}

	portProbe, err := net.Listen("tcp4", os.Getenv("ATTACK_ENDPOINT"))
	if err != nil {
		t.Fatalf("runtime port preflight refused: %v", err)
	}
	portProbe.Close()
	rv, err := net.Listen("unix", os.Getenv("ATTACK_SOCKET_PATH"))
	if err != nil {
		t.Fatalf("rendezvous bind refused: %v", err)
	}
	rv.Close()
	fmt.Println("ATTACK_RESULT=all_three_specified_refusals_bypassed")
}

func TestRuntimeOrphanBeforeStatePublish(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "broker.lock")
	statePath := filepath.Join(dir, "broker-state.json")

	cmd := exec.Command(os.Args[0], "-test.run=TestBrokerAttackChild", "-test.v")
	cmd.Env = append(os.Environ(),
		"ATTACK_CHILD=broker",
		"ATTACK_LOCK_PATH="+lockPath,
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	scanner := bufio.NewScanner(stdout)
	runtimePID := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "RUNTIME_PID=") {
			runtimePID, err = strconv.Atoi(strings.TrimPrefix(line, "RUNTIME_PID="))
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if runtimePID == 0 {
		t.Fatal("broker child did not report runtime pid")
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	defer unix.Kill(-runtimePID, unix.SIGKILL)

	if err := unix.Kill(runtimePID, 0); err != nil {
		t.Fatalf("runtime did not survive broker SIGKILL: %v", err)
	}
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("expected absent state before readiness publication, got %v", err)
	}
	newStarter, err := os.OpenFile(lockPath, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer newStarter.Close()
	if err := unix.Flock(int(newStarter.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatalf("broker death did not free starter lock: %v", err)
	}
	t.Logf("after broker SIGKILL: runtime pid %d is alive, broker.lock is acquirable, and broker-state.json is absent; section 6 reclamation therefore classifies the orphan as 'nothing to reclaim'", runtimePID)
}

func TestBrokerAttackChild(t *testing.T) {
	if os.Getenv("ATTACK_CHILD") != "broker" {
		t.Skip("parent-driven")
	}
	lockFile, err := os.OpenFile(os.Getenv("ATTACK_LOCK_PATH"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Flock(int(lockFile.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		t.Fatal(err)
	}

	runtime := exec.Command(os.Args[0], "-test.run=TestRuntimeAttackChild", "-test.v")
	runtime.Env = append(os.Environ(), "ATTACK_CHILD=runtime")
	runtime.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("RUNTIME_PID=%d\n", runtime.Process.Pid)
	_ = os.Stdout.Sync()
	// Model the normative sequence: readiness is awaited before state is
	// published. The parent test kills this broker in that window.
	time.Sleep(30 * time.Second)
}

func TestRuntimeAttackChild(t *testing.T) {
	if os.Getenv("ATTACK_CHILD") != "runtime" {
		t.Skip("parent-driven")
	}
	time.Sleep(30 * time.Second)
}
