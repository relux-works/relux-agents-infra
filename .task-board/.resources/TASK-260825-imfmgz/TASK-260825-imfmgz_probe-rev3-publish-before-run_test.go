package probe3

// Revision-3 probes P12-P14: publish-before-run, reclamation, and the operator
// force-stop path.
//
// P12 is deliberately a NEGATIVE first. It runs revision 2's ordering and
// requires the reviewer's forbidden state to appear: a live runtime, a released
// lock, and no durable record. Only then does it run revision 3's ordering and
// require that state to be unreachable. A fix that is not preceded by a
// reproduction of the defect is not evidence.

import (
	"encoding/json"
	"fmt"
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

// runtimeBlock is the durable identity of the runtime process. It is written
// while that process is still an unauthorized launcher, which is why it carries
// BOTH admissible identity shapes: the pre-exec launcher and the post-exec
// runtime. P10 proved pid, pgid and start time survive the exec between them.
type runtimeBlock struct {
	Pid          int      `json:"pid"`
	Pgid         int      `json:"pgid"`
	Start        string   `json:"start"`
	Uid          uint32   `json:"uid"`
	PreExecExe   string   `json:"pre_exec_exe"`
	PreExecArgv  []string `json:"pre_exec_argv"`
	PostExecExe  string   `json:"post_exec_exe"`
	PostExecArgv []string `json:"post_exec_argv"`
}

type ownerRecord struct {
	Protocol    int           `json:"protocol"`
	State       string        `json:"state"`
	BrokerPid   int           `json:"broker_pid"`
	BrokerStart string        `json:"broker_start"`
	BrokerExe   string        `json:"broker_exe"`
	Runtime     *runtimeBlock `json:"runtime"`
}

func readRecord(path string) (*ownerRecord, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r ownerRecord
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("malformed: %w", err)
	}
	return &r, nil
}

func writeRecord(path string, r *ownerRecord) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// lockFree reports whether broker.lock can be acquired, i.e. whether any broker
// is alive. It takes and immediately releases the lock, so it is only ever used
// by the test as an observer, never by a client.
func lockFree(path string) bool {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return false
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return false
	}
	_ = unix.Flock(int(f.Fd()), unix.LOCK_UN)
	return true
}

func waitMarker(t *testing.T, path string, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if exists(path) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("marker %s never appeared", filepath.Base(path))
}

type brokerEnv struct {
	dir        string
	lock       string
	record     string
	launcherID string
	markers    map[string]string
	cmd        *exec.Cmd
}

func startBroker(t *testing.T, order string, stallPublishMS, stallAuthMS int) *brokerEnv {
	t.Helper()
	dir := shortTempDir(t)
	be := &brokerEnv{
		dir:        dir,
		lock:       filepath.Join(dir, "broker.lock"),
		record:     filepath.Join(dir, "broker-state.json"),
		launcherID: filepath.Join(dir, "launcher.pid"),
		markers: map[string]string{
			"forked":     filepath.Join(dir, "m.forked"),
			"published":  filepath.Join(dir, "m.published"),
			"authorized": filepath.Join(dir, "m.authorized"),
		},
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestBrokerChild")
	cmd.Env = append(os.Environ(),
		"PROBE_CHILD=broker",
		"PROBE_ORDER="+order,
		"PROBE_LOCK="+be.lock,
		"PROBE_RECORD="+be.record,
		"PROBE_LAUNCHER_PID="+be.launcherID,
		"PROBE_MARKER_DIR="+dir,
		"PROBE_STALL_PUBLISH_MS="+strconv.Itoa(stallPublishMS),
		"PROBE_STALL_AUTH_MS="+strconv.Itoa(stallAuthMS),
		"PROBE_TARGET=/bin/sleep",
		"PROBE_TARGET_ARG=41",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if logf, lerr := os.Create(filepath.Join(dir, "broker.log")); lerr == nil {
		cmd.Stdout, cmd.Stderr = logf, logf
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	be.cmd = cmd
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		if pid := be.launcherPid(); pid > 0 {
			_ = unix.Kill(-pid, unix.SIGKILL)
			_ = unix.Kill(pid, unix.SIGKILL)
		}
	})
	return be
}

func (be *brokerEnv) launcherPid() int {
	b, err := os.ReadFile(be.launcherID)
	if err != nil {
		return -1
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return -1
	}
	return pid
}

func (be *brokerEnv) sigkill(t *testing.T) {
	t.Helper()
	pid := be.cmd.Process.Pid
	if err := be.cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	_, _ = be.cmd.Process.Wait()
	deadline := time.Now().Add(3 * time.Second)
	for Alive(pid) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if Alive(pid) {
		t.Fatalf("broker pid=%d survived SIGKILL", pid)
	}
}

// ---------------------------------------------------------------------------
// P12.A - revision 2's ordering reproduces the reviewer's forbidden state
// ---------------------------------------------------------------------------

func TestP12Rev2OrderingLeavesUnrecordedRuntime(t *testing.T) {
	be := startBroker(t, "old", 4000, 0)
	waitMarker(t, be.markers["authorized"], 5*time.Second)

	pid := be.launcherPid()
	if pid <= 0 {
		t.Fatal("no runtime pid recorded by the probe harness")
	}
	// Wait until the runtime really is the runtime.
	deadline := time.Now().Add(3 * time.Second)
	var id ProcIdentity
	var err error
	for time.Now().Before(deadline) {
		id, err = Identify(pid)
		if err == nil && id.Exe == "/bin/sleep" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if id.Exe != "/bin/sleep" {
		t.Fatalf("runtime never started under revision 2 ordering: %v", err)
	}

	be.sigkill(t)

	runtimeAlive := Alive(pid)
	_, recErr := readRecord(be.record)
	recordAbsent := os.IsNotExist(recErr)
	free := lockFree(be.lock)

	if !(runtimeAlive && recordAbsent && free) {
		t.Fatalf("P12.A FAIL: could not reproduce the reviewer's F1 state "+
			"(runtimeAlive=%v recordAbsent=%v lockFree=%v)", runtimeAlive, recordAbsent, free)
	}
	t.Logf("P12.A OK (defect reproduced): revision 2 ordering + broker SIGKILL => runtime pid=%d "+
		"ALIVE, broker-state.json ABSENT, broker.lock FREE. The next starter reads the absent "+
		"record as \"nothing to reclaim\" and is authorized to spawn a second runtime.", pid)

	// And the reclaimer confirms the consequence: it has nothing to act on.
	killed, reason := reclaim(t, be.record)
	if killed {
		t.Fatalf("P12.A FAIL: reclamation somehow acted on an absent record")
	}
	t.Logf("P12.A OK: reclamation verdict on the absent record is %q while pid=%d is still "+
		"serving - absent evidence treated as satisfied", reason, pid)
	_ = unix.Kill(pid, unix.SIGKILL)
}

// ---------------------------------------------------------------------------
// P12.B/C/D - revision 3 ordering makes that state unreachable
// ---------------------------------------------------------------------------

// P12.B: broker killed after forking the launcher but BEFORE publishing.
func TestP12Rev3KillBeforePublish(t *testing.T) {
	be := startBroker(t, "new", 4000, 4000)
	waitMarker(t, be.markers["forked"], 5*time.Second)

	pid := be.launcherPid()
	if pid <= 0 {
		t.Fatal("launcher pid not reported")
	}
	id, err := Identify(pid)
	if err != nil {
		t.Fatalf("launcher not identifiable: %v", err)
	}
	if id.Exe == "/bin/sleep" {
		t.Fatalf("P12.B FAIL: launcher exec'd the runtime before authorization")
	}
	if exists(be.markers["published"]) {
		t.Fatal("test raced past the publish point; increase the stall")
	}

	be.sigkill(t)

	// The launcher must die on EOF, never having become the runtime.
	gone := waitGone(pid, 5*time.Second)
	if !gone {
		after, _ := Identify(pid)
		t.Fatalf("P12.B FAIL: launcher pid=%d survived its broker (exe=%q)", pid, after.Exe)
	}
	rec, rerr := readRecord(be.record)
	if rerr != nil {
		t.Fatalf("P12.B FAIL: owner record unreadable: %v", rerr)
	}
	if rec.Runtime != nil {
		t.Fatalf("P12.B FAIL: a runtime block exists though nothing was published: %+v", rec.Runtime)
	}
	if rec.BrokerPid != be.cmd.Process.Pid {
		t.Fatalf("P12.B FAIL: owner record names broker %d, want %d", rec.BrokerPid, be.cmd.Process.Pid)
	}
	killed, reason := reclaim(t, be.record)
	if killed || reason != "no-runtime-block" {
		t.Fatalf("P12.B FAIL: reclamation verdict killed=%v %q, want no-runtime-block", killed, reason)
	}
	if !lockFree(be.lock) {
		t.Fatalf("P12.B FAIL: broker.lock still held after the broker died")
	}
	t.Logf("P12.B OK: broker SIGKILLed between fork and publish => launcher pid=%d exited "+
		"unauthorized, NO runtime ever existed. The owner record is present and affirmative "+
		"(broker_pid=%d, runtime block absent), reclamation verdict %q, lock free. Nothing is "+
		"inferred from an absence.", pid, rec.BrokerPid, reason)
}

// P12.C: broker killed AFTER publishing but BEFORE authorizing.
func TestP12Rev3KillAfterPublishBeforeAuthorize(t *testing.T) {
	be := startBroker(t, "new", 150, 5000)
	waitMarker(t, be.markers["published"], 5*time.Second)
	if exists(be.markers["authorized"]) {
		t.Fatal("test raced past the authorization point; increase the stall")
	}

	pid := be.launcherPid()
	rec, err := readRecord(be.record)
	if err != nil {
		t.Fatalf("P12.C FAIL: record unreadable: %v", err)
	}
	if rec.Runtime == nil || rec.Runtime.Pid != pid {
		t.Fatalf("P12.C FAIL: published record does not name the launcher pid=%d: %+v", pid, rec.Runtime)
	}
	t.Logf("P12.C setup: record published naming pid=%d start=%s BEFORE the process was "+
		"authorized to become the runtime", rec.Runtime.Pid, rec.Runtime.Start)

	be.sigkill(t)

	if !waitGone(pid, 5*time.Second) {
		after, _ := Identify(pid)
		t.Fatalf("P12.C FAIL: launcher pid=%d survived (exe=%q)", pid, after.Exe)
	}
	// A reclaimer arriving now finds a record naming a dead pid: stale, delete,
	// nothing to kill - and, critically, it cannot conclude "nothing existed"
	// from an absence, because the record is present and affirmative.
	killed, reason := reclaim(t, be.record)
	if killed {
		t.Fatalf("P12.C FAIL: reclamation killed a process that had already exited")
	}
	if reason != "stale-record-process-gone" {
		t.Fatalf("P12.C FAIL: reclamation verdict %q, want stale-record-process-gone", reason)
	}
	t.Logf("P12.C OK: broker SIGKILLed between publish and authorize => launcher exited "+
		"unauthorized, record present and affirmative, reclamation verdict %q, no runtime "+
		"ever bound anything", reason)
}

// P12.D: broker killed AFTER authorizing - the record must identify and the
// reclaimer must kill the real orphan.
func TestP12Rev3KillAfterAuthorizeIsReclaimable(t *testing.T) {
	be := startBroker(t, "new", 100, 100)
	waitMarker(t, be.markers["authorized"], 5*time.Second)

	pid := be.launcherPid()
	deadline := time.Now().Add(3 * time.Second)
	var id ProcIdentity
	for time.Now().Before(deadline) {
		id, _ = Identify(pid)
		if id.Exe == "/bin/sleep" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if id.Exe != "/bin/sleep" {
		t.Fatalf("P12.D FAIL: authorized launcher never became the runtime (exe=%q)", id.Exe)
	}

	rec, err := readRecord(be.record)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Runtime.Start != id.StartKey() {
		t.Fatalf("P12.D FAIL: the start time published before exec (%s) does not match the "+
			"runtime after exec (%s)", rec.Runtime.Start, id.StartKey())
	}
	t.Logf("P12.D setup: record published pre-exec still identifies the post-exec runtime "+
		"(pid=%d start=%s exe=%q)", pid, id.StartKey(), id.Exe)

	be.sigkill(t)

	if !Alive(pid) {
		t.Fatal("P12.D FAIL: the orphan runtime is not alive; nothing to reclaim")
	}
	killed, reason := reclaim(t, be.record)
	if !killed {
		t.Fatalf("P12.D FAIL: reclamation refused a genuine orphan: %s", reason)
	}
	if !waitGone(pid, 5*time.Second) {
		t.Fatalf("P12.D FAIL: orphan pid=%d survived reclamation", pid)
	}
	t.Logf("P12.D OK: broker SIGKILLed after authorization => orphan pid=%d identified from the "+
		"durable record by pid+start time+argv shape, its process group killed, verdict %q",
		pid, reason)
}

// ---------------------------------------------------------------------------
// P13 - reclamation refuses to act on evidence it cannot verify
// ---------------------------------------------------------------------------

// reclaim models the specification's reclamation subroutine. It returns
// (killed, verdict). It never acts on a bare pid.
func reclaim(t *testing.T, recordPath string) (bool, string) {
	t.Helper()
	rec, err := readRecord(recordPath)
	if os.IsNotExist(err) {
		return false, "no-record"
	}
	if err != nil {
		return false, "unreadable-record" // never an absence, never authorizes a start
	}
	if rec.Runtime == nil {
		return false, "no-runtime-block"
	}
	rt := rec.Runtime
	id, err := Identify(rt.Pid)
	if err != nil {
		return false, "stale-record-process-gone"
	}
	if id.Uid != rt.Uid {
		return false, "refused-uid-mismatch"
	}
	if id.StartKey() != rt.Start {
		return false, "refused-start-time-mismatch"
	}
	pre := id.Exe == rt.PreExecExe && strings.Join(id.Argv, "\x00") == strings.Join(rt.PreExecArgv, "\x00")
	post := id.Exe == rt.PostExecExe && strings.Join(id.Argv, "\x00") == strings.Join(rt.PostExecArgv, "\x00")
	if !pre && !post {
		return false, "refused-identity-unrecognized"
	}
	_ = unix.Kill(-rt.Pgid, unix.SIGTERM)
	if waitGone(rt.Pid, 2*time.Second) {
		if pre {
			return true, "killed-pre-exec-launcher"
		}
		return true, "killed-post-exec-runtime"
	}
	_ = unix.Kill(-rt.Pgid, unix.SIGKILL)
	if waitGone(rt.Pid, 2*time.Second) {
		return true, "killed-post-exec-runtime-sigkill"
	}
	return false, "refused-shutdown-timeout"
}

func TestP13ReclamationRefusesUnverifiedEvidence(t *testing.T) {
	be := startBroker(t, "new", 100, 100)
	waitMarker(t, be.markers["authorized"], 5*time.Second)
	pid := be.launcherPid()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if id, _ := Identify(pid); id.Exe == "/bin/sleep" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	be.sigkill(t)
	if !Alive(pid) {
		t.Fatal("setup: orphan not alive")
	}

	rec, err := readRecord(be.record)
	if err != nil {
		t.Fatal(err)
	}

	// A: recycled pid - same pid, different start time. Must refuse.
	mutated := *rec
	rtA := *rec.Runtime
	rtA.Start = "1.000001"
	mutated.Runtime = &rtA
	pathA := filepath.Join(be.dir, "rec-recycled.json")
	if err := writeRecord(pathA, &mutated); err != nil {
		t.Fatal(err)
	}
	if killed, reason := reclaim(t, pathA); killed || reason != "refused-start-time-mismatch" {
		t.Fatalf("P13.A FAIL: killed=%v reason=%q, want refusal on start-time mismatch", killed, reason)
	}
	if !Alive(pid) {
		t.Fatalf("P13.A FAIL: an unverified pid was killed")
	}
	t.Logf("P13.A OK: pid alive but start time differs => refused-start-time-mismatch, pid=%d untouched", pid)

	// B: identity neither pre-exec nor post-exec. Must refuse, not guess.
	mutated2 := *rec
	rtB := *rec.Runtime
	rtB.PostExecArgv = []string{"/bin/sleep", "999999"}
	rtB.PreExecArgv = []string{"/nonexistent"}
	rtB.PreExecExe = "/nonexistent"
	mutated2.Runtime = &rtB
	pathB := filepath.Join(be.dir, "rec-argv.json")
	if err := writeRecord(pathB, &mutated2); err != nil {
		t.Fatal(err)
	}
	if killed, reason := reclaim(t, pathB); killed || reason != "refused-identity-unrecognized" {
		t.Fatalf("P13.B FAIL: killed=%v reason=%q, want refusal on unrecognized identity", killed, reason)
	}
	if !Alive(pid) {
		t.Fatalf("P13.B FAIL: a pid with unrecognized argv was killed")
	}
	t.Logf("P13.B OK: pid+start match but argv is neither admissible shape => "+
		"refused-identity-unrecognized, pid=%d untouched", pid)

	// C: malformed record must not degrade to "nothing to reclaim".
	pathC := filepath.Join(be.dir, "rec-malformed.json")
	if err := os.WriteFile(pathC, []byte(`{"protocol":1,"runtime":{`), 0o600); err != nil {
		t.Fatal(err)
	}
	if killed, reason := reclaim(t, pathC); killed || reason != "unreadable-record" {
		t.Fatalf("P13.C FAIL: killed=%v reason=%q, want unreadable-record", killed, reason)
	}
	t.Logf("P13.C OK: malformed record => unreadable-record, distinct from no-record; a failed " +
		"read never becomes an absence")

	// D: the discriminating control - the true record still reclaims.
	if killed, reason := reclaim(t, be.record); !killed {
		t.Fatalf("P13.D FAIL: the unmutated record failed to reclaim: %s", reason)
	}
	t.Logf("P13.D OK: control - the unmutated record reclaims pid=%d, so the refusals above "+
		"discriminate rather than always refusing", pid)
}

// ---------------------------------------------------------------------------
// P14 - operator force-stop of a live broker whose socket is gone
// ---------------------------------------------------------------------------

// forceStop models `agents-infra runtime stop --force` on the unreachable-broker
// path: no socket to connect to, a live broker holding the lock.
func forceStop(t *testing.T, recordPath, lockPath string) (bool, string) {
	t.Helper()
	rec, err := readRecord(recordPath)
	if os.IsNotExist(err) {
		if lockFree(lockPath) {
			return false, "absent-nothing-to-stop"
		}
		return false, "refused-owner-unidentifiable"
	}
	if err != nil {
		return false, "unreadable-record"
	}
	id, idErr := Identify(rec.BrokerPid)
	if idErr != nil {
		return false, "stale-record-broker-gone"
	}
	if id.StartKey() != rec.BrokerStart {
		return false, "refused-broker-start-time-mismatch"
	}
	if id.Exe != rec.BrokerExe {
		return false, "refused-broker-executable-mismatch"
	}
	_ = unix.Kill(rec.BrokerPid, unix.SIGTERM)
	if !waitGone(rec.BrokerPid, 2*time.Second) {
		_ = unix.Kill(rec.BrokerPid, unix.SIGKILL)
		if !waitGone(rec.BrokerPid, 2*time.Second) {
			return false, "refused-broker-shutdown-timeout"
		}
	}
	return true, "stopped-broker"
}

func TestP14ForceStopLiveBrokerWithoutSocket(t *testing.T) {
	be := startBroker(t, "new", 100, 100)
	waitMarker(t, be.markers["authorized"], 5*time.Second)
	runtimePid := be.launcherPid()
	brokerPid := be.cmd.Process.Pid

	if lockFree(be.lock) {
		t.Fatal("setup: broker does not hold the lock")
	}

	// A: a record whose broker start time does not match must never be signalled.
	rec, err := readRecord(be.record)
	if err != nil {
		t.Fatal(err)
	}
	bad := *rec
	bad.BrokerStart = "1.000001"
	pathBad := filepath.Join(be.dir, "rec-badbroker.json")
	if err := writeRecord(pathBad, &bad); err != nil {
		t.Fatal(err)
	}
	if stopped, reason := forceStop(t, pathBad, be.lock); stopped || reason != "refused-broker-start-time-mismatch" {
		t.Fatalf("P14.A FAIL: stopped=%v reason=%q", stopped, reason)
	}
	if !Alive(brokerPid) {
		t.Fatalf("P14.A FAIL: an unverified broker pid was signalled")
	}
	t.Logf("P14.A OK: unverifiable owner record => refused-broker-start-time-mismatch, live "+
		"broker pid=%d untouched", brokerPid)

	// B: the real record identifies the live broker and stops it, and the
	// runtime is then reclaimable from the same record.
	stopped, reason := forceStop(t, be.record, be.lock)
	if !stopped {
		t.Fatalf("P14.B FAIL: could not stop a live identified broker: %s", reason)
	}
	if Alive(brokerPid) {
		t.Fatalf("P14.B FAIL: broker pid=%d survived", brokerPid)
	}
	if !lockFree(be.lock) {
		t.Fatalf("P14.B FAIL: broker.lock still held after the broker exited")
	}
	killed, rreason := reclaim(t, be.record)
	if !killed {
		t.Fatalf("P14.B FAIL: runtime not reclaimable after force-stop: %s", rreason)
	}
	if !waitGone(runtimePid, 5*time.Second) {
		t.Fatalf("P14.B FAIL: runtime pid=%d survived", runtimePid)
	}
	t.Logf("P14.B OK: live broker pid=%d verified from the owner record and stopped, lock "+
		"released, runtime pid=%d reclaimed (%s) - the socket was never needed",
		brokerPid, runtimePid, rreason)

	// C: with everything gone, force-stop is a clean no-op rather than a guess.
	_ = os.Remove(be.record)
	if stopped, reason := forceStop(t, be.record, be.lock); stopped || reason != "absent-nothing-to-stop" {
		t.Fatalf("P14.C FAIL: stopped=%v reason=%q", stopped, reason)
	}
	t.Logf("P14.C OK: no record and a free lock => absent-nothing-to-stop")
}

// TestBrokerChild models the broker's startup ordering. PROBE_ORDER=old is
// revision 2 (spawn the runtime, then publish); PROBE_ORDER=new is revision 3
// (create an unauthorized launcher, publish, then authorize).
func TestBrokerChild(t *testing.T) {
	if os.Getenv("PROBE_CHILD") != "broker" {
		t.Skip("child entry point")
	}
	markerDir := os.Getenv("PROBE_MARKER_DIR")
	mark := func(name string) {
		_ = os.WriteFile(filepath.Join(markerDir, "m."+name), []byte("1"), 0o600)
	}
	fail := func(code int, args ...any) {
		fmt.Println(args...)
		os.Exit(code)
	}

	lock, err := os.OpenFile(os.Getenv("PROBE_LOCK"), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		fail(1, "lock open:", err)
	}
	if err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		fail(exitElectionLost, "election_lost:", err)
	}

	recordPath := os.Getenv("PROBE_RECORD")
	order := os.Getenv("PROBE_ORDER")
	stallPublish, _ := strconv.Atoi(os.Getenv("PROBE_STALL_PUBLISH_MS"))
	stallAuth, _ := strconv.Atoi(os.Getenv("PROBE_STALL_AUTH_MS"))
	target := os.Getenv("PROBE_TARGET")
	targetArg := os.Getenv("PROBE_TARGET_ARG")

	self, _ := Identify(os.Getpid())
	owner := &ownerRecord{
		Protocol:    1,
		State:       "starting",
		BrokerPid:   os.Getpid(),
		BrokerStart: self.StartKey(),
		BrokerExe:   self.Exe,
	}
	if order == "new" {
		// The owner record exists before anything else, so a live broker is
		// always identifiable by an operator even before a runtime exists.
		if err := writeRecord(recordPath, owner); err != nil {
			fail(1, "owner publish:", err)
		}
	}

	authR, authW, err := os.Pipe()
	if err != nil {
		fail(1, "pipe:", err)
	}
	launcher := exec.Command(os.Args[0], "-test.run=TestLauncherChild")
	launcher.Env = append(os.Environ(),
		"PROBE_CHILD=launcher",
		"PROBE_TARGET="+target,
		"PROBE_TARGET_ARG="+targetArg,
		"PROBE_AUTH_TIMEOUT_MS=60000",
		"PROBE_READY=",
	)
	launcher.ExtraFiles = []*os.File{authR}
	launcher.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := launcher.Start(); err != nil {
		fail(1, "launcher start:", err)
	}
	authR.Close()
	lpid := launcher.Process.Pid
	_ = os.WriteFile(os.Getenv("PROBE_LAUNCHER_PID"), []byte(strconv.Itoa(lpid)), 0o600)
	mark("forked")

	publish := func() {
		id, err := Identify(lpid)
		if err != nil {
			fail(1, "launcher identify:", err)
		}
		pgid, err := unix.Getpgid(lpid)
		if err != nil {
			fail(1, "getpgid:", err)
		}
		owner.State = "starting"
		owner.Runtime = &runtimeBlock{
			Pid:          lpid,
			Pgid:         pgid,
			Start:        id.StartKey(),
			Uid:          id.Uid,
			PreExecExe:   id.Exe,
			PreExecArgv:  id.Argv,
			PostExecExe:  target,
			PostExecArgv: []string{target, targetArg},
		}
		if err := writeRecord(recordPath, owner); err != nil {
			fail(1, "runtime publish:", err)
		}
		mark("published")
	}
	authorize := func() {
		if _, err := fmt.Fprintf(authW, "AUTH pid=%d\n", lpid); err != nil {
			fail(1, "authorize:", err)
		}
		mark("authorized")
	}

	switch order {
	case "old":
		// Revision 2: the runtime is created and running before any durable
		// record of it exists. The stall is the crash window.
		authorize()
		time.Sleep(time.Duration(stallPublish) * time.Millisecond)
		publish()
	case "new":
		// Revision 3: publish first, authorize second. The stalls are the same
		// crash windows, and neither of them can leave a runtime behind.
		time.Sleep(time.Duration(stallPublish) * time.Millisecond)
		publish()
		time.Sleep(time.Duration(stallAuth) * time.Millisecond)
		authorize()
	default:
		fail(1, "unknown order", order)
	}

	// Serve until killed. A bare select{} would trip Go's deadlock detector and
	// make the broker exit on its own, which P15 shows is indistinguishable from a
	// live broker if liveness is read from kern.proc.pid alone.
	for {
		time.Sleep(time.Second)
	}
}
