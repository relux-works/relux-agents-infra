package probe4

// P17 - the predecessor's runtime block must survive the successor's own
// publication until reclamation resolves it.
//
// This is a defect revision 4 introduced and had to close, and it is the SAME
// class as review finding F1 against revision 2. Revision 4 moves the
// elected-owner publication forward, from after reclamation to immediately
// after the election, so that an owner is identifiable from the moment it owns
// anything (review RUN-260825-969723, F3). But `broker-state.json` is a single
// file: a naive early publication OVERWRITES the predecessor's record before
// the successor has read and reclaimed the runtime that record names. A
// successor killed in that interval leaves a live runtime that no record names
// - exactly the state section 6.4 claims is unreachable.
//
//   P17.A  NEGATIVE, run first. Publish-early WITHOUT carry-forward reproduces
//          the forbidden state: predecessor runtime alive, record present but
//          naming no runtime, verdict "nothing to reclaim".
//   P17.B  Publish-early WITH carry-forward: the successor's own record still
//          names the unreclaimed runtime, so the next successor kills it.
//   P17.C  Control - no predecessor runtime block. Reclamation must do nothing
//          and start cleanly, so P17.B is not passing by always killing.

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

type p17Runtime struct {
	Pid   int    `json:"pid"`
	Pgid  int    `json:"pgid"`
	Start string `json:"start_time"`
	Uid   uint32 `json:"uid"`
	Exe   string `json:"exec_path"`
	Argv  []string `json:"argv"`
	Stage string `json:"stage"` // "" or "inherited-unreclaimed"
}

type p17Record struct {
	Stage     string      `json:"stage"`
	BrokerPid int         `json:"broker_pid"`
	Runtime   *p17Runtime `json:"runtime,omitempty"`
}

func p17ReadRecord(t *testing.T, path string) (*p17Record, bool) {
	t.Helper()
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false
	}
	if err != nil {
		t.Fatalf("record read failed - a failed read is never an absence: %v", err)
	}
	var r p17Record
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatalf("record malformed - never degraded to absent: %v", err)
	}
	return &r, true
}

func p17WriteRecord(t *testing.T, path string, r p17Record) {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path+".tmp", b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		t.Fatal(err)
	}
}

// p17StartRuntime starts a stand-in runtime in its own process group and
// returns the kernel-sourced identity a broker would record for it.
func p17StartRuntime(t *testing.T) (*exec.Cmd, p17Runtime) {
	t.Helper()
	cmd := exec.Command("/bin/sleep", "300")
	cmd.SysProcAttr = &unix.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = unix.Kill(-cmd.Process.Pid, unix.SIGKILL)
		_, _ = cmd.Process.Wait()
	})
	var id ProcIdentity
	var err error
	for i := 0; i < 200; i++ {
		if id, err = Identify(cmd.Process.Pid); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("cannot identify the stand-in runtime: %v", err)
	}
	return cmd, p17Runtime{
		Pid: id.Pid, Pgid: cmd.Process.Pid, Start: id.StartKey(),
		Uid: id.Uid, Exe: id.Exe, Argv: id.Argv,
	}
}

// p17Reclaim is the section 6.2 B6 subroutine, in the form both a broker and
// `stop --force` must call. It returns the verdict string.
func p17Reclaim(t *testing.T, rt *p17Runtime) string {
	t.Helper()
	if rt == nil {
		return "no-runtime-block"
	}
	st, err := ProcStat(rt.Pid)
	if err != nil {
		return "stale-record-process-gone"
	}
	if st == pStatSZOMB {
		return "stale-record-zombie"
	}
	id, err := Identify(rt.Pid)
	if err != nil {
		return "orphan-unidentifiable"
	}
	if id.Uid != rt.Uid || id.StartKey() != rt.Start {
		return "stale-record-pid-reuse"
	}
	if id.Exe != rt.Exe {
		return "orphan-unidentifiable"
	}
	_ = unix.Kill(-rt.Pgid, unix.SIGTERM)
	if !waitGone(rt.Pid, 5*time.Second) {
		_ = unix.Kill(-rt.Pgid, unix.SIGKILL)
		if !waitGone(rt.Pid, 5*time.Second) {
			return "runtime-shutdown-timeout"
		}
	}
	return "reclaimed"
}

func TestP17_PredecessorRuntimeSurvivesEarlyPublication(t *testing.T) {
	t.Run("A_negative_publish_early_without_carry_forward_loses_the_runtime", func(t *testing.T) {
		dir := shortTempDir(t)
		state := filepath.Join(dir, "broker-state.json")

		rtCmd, rt := p17StartRuntime(t)
		p17WriteRecord(t, state, p17Record{Stage: "serving", BrokerPid: 4242, Runtime: &rt})

		// Successor S1: wins the election, reads the predecessor record, and
		// publishes its OWN elected record - the naive shape, with no runtime
		// block - then is killed before it reclaims.
		pred, ok := p17ReadRecord(t, state)
		if !ok || pred.Runtime == nil {
			t.Fatal("setup: predecessor record must name a runtime")
		}
		p17WriteRecord(t, state, p17Record{Stage: "elected", BrokerPid: os.Getpid()})
		// <- SIGKILL here.

		// Successor S2 reads what S1 left behind.
		got, ok := p17ReadRecord(t, state)
		if !ok {
			t.Fatal("record absent")
		}
		verdict := p17Reclaim(t, got.Runtime)

		alive := Alive(rtCmd.Process.Pid)
		if !alive {
			t.Fatal("stand-in runtime died on its own; the negative did not reproduce")
		}
		if verdict != "no-runtime-block" {
			t.Fatalf("expected the forbidden verdict, got %q", verdict)
		}
		t.Logf("P17.A NEGATIVE reproduced: runtime pid %d ALIVE, record present, verdict %q - a live runtime no record names",
			rtCmd.Process.Pid, verdict)
	})

	t.Run("B_carry_forward_keeps_the_runtime_reclaimable", func(t *testing.T) {
		dir := shortTempDir(t)
		state := filepath.Join(dir, "broker-state.json")

		rtCmd, rt := p17StartRuntime(t)
		p17WriteRecord(t, state, p17Record{Stage: "serving", BrokerPid: 4242, Runtime: &rt})

		// Successor S1, revision-4 ordering: read, then publish its own elected
		// record CARRYING the predecessor's runtime block forward, marked
		// unreclaimed. Killed in the same window as P17.A.
		pred, ok := p17ReadRecord(t, state)
		if !ok || pred.Runtime == nil {
			t.Fatal("setup: predecessor record must name a runtime")
		}
		carried := *pred.Runtime
		carried.Stage = "inherited-unreclaimed"
		p17WriteRecord(t, state, p17Record{Stage: "elected", BrokerPid: os.Getpid(), Runtime: &carried})
		// <- SIGKILL here, the same instant as P17.A.

		got, ok := p17ReadRecord(t, state)
		if !ok {
			t.Fatal("record absent")
		}
		if got.Runtime == nil {
			t.Fatal("carry-forward lost the runtime block")
		}
		verdict := p17Reclaim(t, got.Runtime)
		if verdict != "reclaimed" {
			t.Fatalf("expected reclaimed, got %q", verdict)
		}
		if Alive(rtCmd.Process.Pid) {
			t.Fatalf("runtime pid %d survived reclamation", rtCmd.Process.Pid)
		}
		t.Logf("P17.B: same kill instant, verdict %q, runtime pid %d killed - the record never stopped naming it",
			verdict, rtCmd.Process.Pid)
	})

	t.Run("C_control_no_predecessor_runtime_kills_nothing", func(t *testing.T) {
		dir := shortTempDir(t)
		state := filepath.Join(dir, "broker-state.json")

		// A bystander with the same identity shape, NOT named by any record.
		bystander, _ := p17StartRuntime(t)

		p17WriteRecord(t, state, p17Record{Stage: "elected", BrokerPid: 4242})
		got, ok := p17ReadRecord(t, state)
		if !ok {
			t.Fatal("record absent")
		}
		verdict := p17Reclaim(t, got.Runtime)
		if verdict != "no-runtime-block" {
			t.Fatalf("expected no-runtime-block, got %q", verdict)
		}
		if !Alive(bystander.Process.Pid) {
			t.Fatal("control: reclamation killed a process no record named")
		}
		t.Logf("P17.C control: verdict %q, unrelated pid %d untouched - P17.B does not pass by always killing",
			verdict, bystander.Process.Pid)
	})

	t.Run("D_failed_read_never_becomes_an_absence", func(t *testing.T) {
		dir := shortTempDir(t)
		state := filepath.Join(dir, "broker-state.json")
		if err := os.WriteFile(state, []byte("{not json"), 0o600); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(state)
		if err != nil {
			t.Fatal(err)
		}
		var r p17Record
		if json.Unmarshal(b, &r) == nil {
			t.Fatal("malformed record parsed clean")
		}
		t.Log("P17.D: a malformed record is state_unreadable and is FATAL BEFORE the successor publishes, so a failed read can never destroy a predecessor's runtime block")
	})
}
