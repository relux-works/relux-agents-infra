package main

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestFlockSurvivesParentExitViaInheritedFD(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/broker.lock"
	// stage 1: this test process acts as the "starter": opens+flocks, hands FD to a detached child, closes its own copy.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/bin/sleep", "4")
	cmd.ExtraFiles = []*os.File{f}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	f.Close() // starter drops its copy; child's inherited descriptor shares the open file description
	time.Sleep(300 * time.Millisecond)

	probe, err := os.OpenFile(path, os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	err = syscall.Flock(int(probe.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		t.Fatalf("lock was released when the starter closed its descriptor; handoff does not hold")
	}
	t.Logf("held-by-detached-child: flock refused as expected: %v", err)

	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
	time.Sleep(300 * time.Millisecond)
	if err := syscall.Flock(int(probe.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("lock not released after holder death: %v", err)
	}
	t.Log("holder death released the lock")
}
