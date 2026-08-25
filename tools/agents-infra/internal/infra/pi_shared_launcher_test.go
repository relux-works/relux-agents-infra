//go:build darwin

package infra

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const sharedLauncherTargetEvent = "shared-launcher-target-entered"

type sharedLauncherFixture struct {
	project      string
	home         string
	cache        string
	target       string
	marker       string
	resolved     sharedResolvedProfile
	evidencePath string
}

func newSharedLauncherFixture(t *testing.T) sharedLauncherFixture {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	project := filepath.Join(root, "p")
	home := filepath.Join(root, "h")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	target := buildSharedLauncherTarget(t, root)
	marker := filepath.Join(root, "exec.marker")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	body := validPiProfileWithArgv(t, "profile", target, port, []string{"serve", "--model", "Model", "--marker", marker}, 2)
	body += `
[agents.pi.profiles.profile.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
broker_start_timeout_seconds = 35
`
	writePiProjectConfig(t, project, body)
	resolved, err := resolveSharedProfile(project, home, cache, "profile")
	if err != nil {
		t.Fatal(err)
	}
	if err := CreateSharedRuntimeTree(resolved.Paths); err != nil {
		t.Fatal(err)
	}
	return sharedLauncherFixture{
		project: project, home: home, cache: cache, target: target, marker: marker,
		resolved: resolved, evidencePath: filepath.Join(root, "auth-evidence.json"),
	}
}

func buildSharedLauncherTarget(t *testing.T, root string) string {
	t.Helper()
	source := filepath.Join(root, "launcher-target.go")
	target := filepath.Join(root, "launcher-target")
	body := `package main
import ("fmt"; "os"; "time")
func main() {
  for i, arg := range os.Args { if arg == "--marker" && i+1 < len(os.Args) { _ = os.WriteFile(os.Args[i+1], []byte("exec"), 0600) } }
  _, _ = fmt.Fprintln(os.Stdout, "shared-launcher-target-entered")
  time.Sleep(10*time.Second)
}
`
	if err := os.WriteFile(source, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", target, source)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build launcher target: %v\n%s", err, output)
	}
	return target
}

type sharedLauncherRun struct {
	command    *exec.Cmd
	writeEnd   *os.File
	stderr     *bytes.Buffer
	target     string
	sawTarget  bool
	donePoll   chan struct{}
	targetHit  chan struct{}
	outputDone chan struct{}
}

func startSharedLauncherRun(t *testing.T, fixture sharedLauncherFixture, withAuthorizationDescriptor bool) *sharedLauncherRun {
	return startSharedLauncherRunWithEnv(t, fixture, withAuthorizationDescriptor)
}

func startSharedLauncherRunWithEnv(t *testing.T, fixture sharedLauncherFixture, withAuthorizationDescriptor bool, extraEnv ...string) *sharedLauncherRun {
	t.Helper()
	var readEnd *os.File
	var writeEnd *os.File
	if withAuthorizationDescriptor {
		var err error
		readEnd, writeEnd, err = os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
	}
	return startSharedLauncherRunWithFiles(t, fixture, fixture.resolved.RuntimeKey, readEnd, writeEnd, extraEnv...)
}

func startSharedLauncherRunWithFiles(t *testing.T, fixture sharedLauncherFixture, runtimeKey string, readEnd, writeEnd *os.File, extraEnv ...string) *sharedLauncherRun {
	t.Helper()
	args := []string{"runtime", "runtime-launch", "--runtime-key", runtimeKey, "--profile-project", fixture.project, "--profile", "profile"}
	command := exec.Command(os.Args[0], args...)
	command.Dir = fixture.resolved.Paths.RuntimeCWD
	command.Env = append(os.Environ(), "HOME="+fixture.home, sharedAuthEvidenceEnv+"="+fixture.evidencePath)
	command.Env = append(command.Env, extraEnv...)
	stderr := new(bytes.Buffer)
	command.Stderr = stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if readEnd != nil {
		command.ExtraFiles = []*os.File{readEnd}
	}
	if err := command.Start(); err != nil {
		if readEnd != nil {
			_ = readEnd.Close()
		}
		if writeEnd != nil {
			_ = writeEnd.Close()
		}
		t.Fatal(err)
	}
	if readEnd != nil {
		_ = readEnd.Close()
	}
	run := &sharedLauncherRun{
		command: command, writeEnd: writeEnd, stderr: stderr, target: fixture.target,
		donePoll: make(chan struct{}), targetHit: make(chan struct{}), outputDone: make(chan struct{}),
	}
	go func() {
		defer close(run.outputDone)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if scanner.Text() == sharedLauncherTargetEvent {
				close(run.targetHit)
				return
			}
		}
	}()
	go func() {
		defer close(run.donePoll)
		for {
			observation, err := inspectSharedProcess(command.Process.Pid)
			if err != nil || !observation.live() {
				return
			}
			if observation.ExecPath == fixture.target {
				run.sawTarget = true
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	return run
}

func (run *sharedLauncherRun) authorize(t *testing.T, raw []byte) {
	t.Helper()
	if err := run.sendAuthorization(raw); err != nil {
		t.Fatal(err)
	}
}

func (run *sharedLauncherRun) sendAuthorization(raw []byte) error {
	if run.writeEnd == nil {
		return errors.New("launcher has no authorization descriptor")
	}
	_, writeErr := run.writeEnd.Write(append(append([]byte(nil), raw...), '\n'))
	closeErr := run.writeEnd.Close()
	run.writeEnd = nil
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func sharedLauncherSocketPair(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	return os.NewFile(uintptr(fds[0]), "authorization-reader"), os.NewFile(uintptr(fds[1]), "authorization-writer")
}

func (run *sharedLauncherRun) wait(t *testing.T, timeout time.Duration) error {
	t.Helper()
	result := make(chan error, 1)
	go func() { result <- run.command.Wait() }()
	select {
	case err := <-result:
		<-run.donePoll
		return err
	case <-time.After(timeout):
		_ = run.command.Process.Kill()
		err := <-result
		<-run.donePoll
		return fmt.Errorf("launcher wait timed out: %w", err)
	}
}

func (run *sharedLauncherRun) carriedTarget() bool {
	return run.sawTarget
}

func (run *sharedLauncherRun) requireTargetEvent(t *testing.T, failure string) {
	t.Helper()
	select {
	case <-run.targetHit:
		return
	case <-run.outputDone:
		select {
		case <-run.targetHit:
			return
		default:
		}
		waitErr := run.wait(t, 3*time.Second)
		t.Fatalf("%s: target output closed before event: wait=%v stderr=%q", failure, waitErr, run.stderr.String())
	}
}

func (run *sharedLauncherRun) requireLauncherPIDExecve(t *testing.T, failure string) {
	t.Helper()
	run.requireTargetEvent(t, failure)
	observation, err := inspectSharedProcess(run.command.Process.Pid)
	if err != nil {
		_ = run.command.Process.Kill()
		waitErr := run.wait(t, 3*time.Second)
		t.Fatalf("%s: inspect launcher pid %d after target event: %v (wait=%v)", failure, run.command.Process.Pid, err, waitErr)
	}
	if !observation.live() || observation.ExecPath != run.target {
		_ = run.command.Process.Kill()
		waitErr := run.wait(t, 3*time.Second)
		t.Fatalf(
			"%s: target event came from a process other than the execve'd launcher: launcher_pid=%d live=%t exec_path=%q target=%q wait=%v",
			failure, run.command.Process.Pid, observation.live(), observation.ExecPath, run.target, waitErr,
		)
	}
}

func validSharedLauncherFrame(fixture sharedLauncherFixture, pid int) sharedRuntimeAuthorizationFrame {
	return sharedRuntimeAuthorizationFrame{
		Schema: sharedRuntimeAuthSchema, ProtocolVersion: SharedRuntimeProtocolVersion,
		RuntimeKey: fixture.resolved.RuntimeKey, LauncherPID: pid,
		ExecPlanDigest: SharedRuntimeExecPlanDigest(fixture.resolved.Profile, fixture.resolved.Paths.RuntimeCWD),
	}
}

func rawSharedLauncherFrame(t *testing.T, frame sharedRuntimeAuthorizationFrame) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := writeAuthorizationFrame(&buffer, frame); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte{'\n'})
}

func sharedRuntimeErrorFromOutput(t *testing.T, output string) SharedRuntimeError {
	t.Helper()
	line := strings.TrimSpace(output)
	var payload SharedRuntimeError
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("decode shared runtime error %q: %v", line, err)
	}
	return payload
}

func TestSharedRuntimeLauncherRefusesAbsentAuthorizationDescriptorWithReason(t *testing.T) {
	fixture := newSharedLauncherFixture(t)
	run := startSharedLauncherRun(t, fixture, false)
	if err := run.wait(t, 3*time.Second); err == nil {
		t.Fatal("launcher without descriptor 3 succeeded")
	}
	refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
	if refusal.Code != "runtime_launch_unauthorized" || refusal.Reason != "no_authorization_descriptor" || run.carriedTarget() {
		t.Fatalf("refusal=%#v carried_target=%t", refusal, run.carriedTarget())
	}
}

func TestSharedRuntimeLauncherUnauthorizedReasonsAreDistinctAndBounded(t *testing.T) {
	fixture := newSharedLauncherFixture(t)
	t.Run("broker died before authorizing", func(t *testing.T) {
		run := startSharedLauncherRun(t, fixture, true)
		if err := run.writeEnd.Close(); err != nil {
			t.Fatal(err)
		}
		run.writeEnd = nil
		if err := run.wait(t, 3*time.Second); err == nil {
			t.Fatal("launcher succeeded after authorization EOF")
		}
		refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
		if refusal.Code != "runtime_launch_unauthorized" || refusal.Reason != "broker_died_before_authorizing" || run.carriedTarget() {
			t.Fatalf("refusal=%#v carried_target=%t", refusal, run.carriedTarget())
		}
	})

	t.Run("authorization deadline", func(t *testing.T) {
		run := startSharedLauncherRun(t, fixture, true)
		started := time.Now()
		if err := run.wait(t, 4*time.Second); err == nil {
			t.Fatal("launcher succeeded without an authorization frame")
		}
		elapsed := time.Since(started)
		if run.writeEnd != nil {
			_ = run.writeEnd.Close()
			run.writeEnd = nil
		}
		refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
		if refusal.Code != "runtime_launch_unauthorized" || refusal.Reason != "authorization_deadline" || elapsed < 1500*time.Millisecond || elapsed > 3500*time.Millisecond || run.carriedTarget() {
			t.Fatalf("refusal=%#v elapsed=%s carried_target=%t", refusal, elapsed, run.carriedTarget())
		}
	})

	t.Run("deadline unavailable injection", func(t *testing.T) {
		run := startSharedLauncherRunWithEnv(t, fixture, true, sharedSetNonblockFail+"=1")
		if err := run.wait(t, 3*time.Second); err == nil {
			t.Fatal("launcher ignored set-nonblock failure")
		}
		if run.writeEnd != nil {
			_ = run.writeEnd.Close()
			run.writeEnd = nil
		}
		refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
		if refusal.Code != "runtime_launch_unauthorized" || refusal.Reason != "deadline_unavailable" || run.carriedTarget() {
			t.Fatalf("refusal=%#v carried_target=%t", refusal, run.carriedTarget())
		}
	})
}

func TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry(t *testing.T) {
	fixture := newSharedLauncherFixture(t)
	tests := []struct {
		name  string
		field string
		alter func(*sharedRuntimeAuthorizationFrame)
	}{
		{name: "exact protocol version control"},
		{name: "schema", field: "schema", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.Schema = "future" }},
		{name: "schema empty", field: "schema", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.Schema = "" }},
		{name: "future protocol version", field: "protocol_version", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.ProtocolVersion = SharedRuntimeProtocolVersion + 1 }},
		{name: "past protocol version", field: "protocol_version", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.ProtocolVersion = SharedRuntimeProtocolVersion - 1 }},
		{name: "launcher pid", field: "launcher_pid", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.LauncherPID++ }},
		{name: "launcher pid zero", field: "launcher_pid", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.LauncherPID = 0 }},
		{name: "runtime key", field: "runtime_key", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.RuntimeKey = strings.Repeat("f", 64) }},
		{name: "runtime key empty", field: "runtime_key", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.RuntimeKey = "" }},
		{name: "exec plan digest", field: "exec_plan_digest", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.ExecPlanDigest = strings.Repeat("e", 64) }},
		{name: "exec plan digest empty", field: "exec_plan_digest", alter: func(frame *sharedRuntimeAuthorizationFrame) { frame.ExecPlanDigest = "" }},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_ = os.Remove(fixture.marker)
			_ = os.Remove(fixture.evidencePath)
			run := startSharedLauncherRun(t, fixture, true)
			frame := validSharedLauncherFrame(fixture, run.command.Process.Pid)
			if testCase.alter != nil {
				testCase.alter(&frame)
			}
			run.authorize(t, rawSharedLauncherFrame(t, frame))
			if testCase.field == "" {
				run.requireLauncherPIDExecve(t, "exact-version authorization never reached execve on the launcher pid")
				_ = run.command.Process.Kill()
				_ = run.wait(t, 3*time.Second)
				var evidence sharedAuthDecodeEvidence
				data, err := os.ReadFile(fixture.evidencePath)
				if err != nil || json.Unmarshal(data, &evidence) != nil {
					t.Fatalf("read launcher evidence: %v data=%q", err, data)
				}
				want := append([]string(nil), sharedRuntimeAuthFields[:]...)
				if !reflect.DeepEqual(evidence.DecodedKeys, want) || !sameStringSet(evidence.ComparedFields, want) || !reflect.DeepEqual(evidence.ConstantFieldSet, want) || evidence.DecisionCallSite != "decodeSharedRuntimeAuthorizationFrame" {
					t.Fatalf("production wiring evidence=%#v want fields=%q", evidence, want)
				}
				return
			}
			if err := run.wait(t, 3*time.Second); err == nil {
				t.Fatal("divergent authorization reached successful exit")
			}
			refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
			if refusal.Code != "runtime_authorization_mismatch" || refusal.MismatchField != testCase.field || run.carriedTarget() {
				t.Fatalf("refusal=%#v field=%q carried_target=%t", refusal, testCase.field, run.carriedTarget())
			}
			if _, err := os.Stat(fixture.marker); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("refused launcher executed target: %v", err)
			}
		})
	}
}

func TestSharedRuntimeLauncherRejectsAuthorizationChannelGuardBypassesAtProductionEntry(t *testing.T) {
	fixture := newSharedLauncherFixture(t)
	tests := []struct {
		name   string
		attack func(*testing.T)
	}{
		{
			name: "descriptor three is a socket rather than a fifo",
			attack: func(t *testing.T) {
				reader, writer := sharedLauncherSocketPair(t)
				run := startSharedLauncherRunWithFiles(t, fixture, fixture.resolved.RuntimeKey, reader, writer)
				_ = run.sendAuthorization(rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid)))
				requireSharedLauncherRefusal(t, fixture, run, "runtime_launch_unauthorized", "no_authorization_descriptor")
			},
		},
		{
			name: "runtime key argument differs only after shared prefix",
			attack: func(t *testing.T) {
				wrongKey := fixture.resolved.RuntimeKey
				last := byte('0')
				if wrongKey[len(wrongKey)-1] == last {
					last = '1'
				}
				wrongKey = wrongKey[:len(wrongKey)-1] + string(last)
				run := startSharedLauncherRunWithEnvAndRuntimeKey(t, fixture, wrongKey)
				frame := validSharedLauncherFrame(fixture, run.command.Process.Pid)
				frame.RuntimeKey = wrongKey
				_ = run.sendAuthorization(rawSharedLauncherFrame(t, frame))
				requireSharedLauncherRefusal(t, fixture, run, "runtime_launch_identity_mismatch", "")
			},
		},
		{
			name: "content follows the authorization frame",
			attack: func(t *testing.T) {
				run := startSharedLauncherRun(t, fixture, true)
				raw := rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid))
				raw = append(raw, []byte("\n{\"second\":true}")...)
				run.authorize(t, raw)
				requireSharedLauncherRefusal(t, fixture, run, "protocol_violation", "frame_unparseable")
			},
		},
		{
			name: "authorization frame is one byte over the bound",
			attack: func(t *testing.T) {
				run := startSharedLauncherRun(t, fixture, true)
				raw := rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid))
				padding := bytes.Repeat([]byte{' '}, sharedRuntimeMaxFrameBytes-len(raw)+1)
				raw = append(append(append([]byte(nil), raw[:len(raw)-1]...), padding...), '}')
				if len(raw) != sharedRuntimeMaxFrameBytes+1 {
					t.Fatalf("oversize witness=%d want=%d", len(raw), sharedRuntimeMaxFrameBytes+1)
				}
				_ = run.sendAuthorization(raw)
				requireSharedLauncherRefusal(t, fixture, run, "protocol_violation", "frame_unparseable")
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			requireSharedLauncherValidControl(t, fixture)
			_ = os.Remove(fixture.marker)
			testCase.attack(t)
		})
	}
}

func startSharedLauncherRunWithEnvAndRuntimeKey(t *testing.T, fixture sharedLauncherFixture, runtimeKey string, extraEnv ...string) *sharedLauncherRun {
	t.Helper()
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	return startSharedLauncherRunWithFiles(t, fixture, runtimeKey, readEnd, writeEnd, extraEnv...)
}

func requireSharedLauncherValidControl(t *testing.T, fixture sharedLauncherFixture) {
	t.Helper()
	_ = os.Remove(fixture.marker)
	run := startSharedLauncherRun(t, fixture, true)
	run.authorize(t, rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid)))
	run.requireTargetEvent(t, "plain valid authorization did not reach the production launcher target")
	_ = run.command.Process.Kill()
	_ = run.wait(t, 3*time.Second)
}

func requireSharedLauncherRefusal(t *testing.T, fixture sharedLauncherFixture, run *sharedLauncherRun, code, reason string) {
	t.Helper()
	waitErr := run.wait(t, 3*time.Second)
	_, markerErr := os.Stat(fixture.marker)
	if run.carriedTarget() || markerErr == nil {
		t.Fatalf("launcher admitted guard bypass code=%q reason=%q wait=%v carried_target=%t", code, reason, waitErr, run.carriedTarget())
	}
	if waitErr == nil {
		t.Fatalf("launcher admitted guard bypass code=%q reason=%q", code, reason)
	}
	refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
	if refusal.Code != code || refusal.Reason != reason || run.carriedTarget() {
		t.Fatalf("refusal=%#v want code=%q reason=%q carried_target=%t", refusal, code, reason, run.carriedTarget())
	}
	if !errors.Is(markerErr, os.ErrNotExist) {
		t.Fatalf("refused launcher marker state: %v", markerErr)
	}
}

func TestSharedRuntimeLauncherShapeGateRejectsRawUnknownAndDuplicateMembers(t *testing.T) {
	fixture := newSharedLauncherFixture(t)
	tests := []struct {
		name   string
		reason string
		field  string
		build  func([]byte) []byte
	}{
		{name: "unknown", reason: "frame_unknown_field", field: "future_extension", build: func(valid []byte) []byte {
			return bytes.Replace(valid, []byte{'{'}, []byte(`{"future_extension":true,`), 1)
		}},
		{name: "duplicate same valid value", reason: "frame_duplicate_field", field: "protocol_version", build: func(valid []byte) []byte {
			return bytes.Replace(valid, []byte{'{'}, []byte(fmt.Sprintf(`{"protocol_version":%d,`, SharedRuntimeProtocolVersion)), 1)
		}},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			_ = os.Remove(fixture.marker)
			run := startSharedLauncherRun(t, fixture, true)
			valid := rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid))
			run.authorize(t, testCase.build(valid))
			if err := run.wait(t, 3*time.Second); err == nil {
				t.Fatal("invalid raw frame succeeded")
			}
			refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
			if refusal.Code != "protocol_violation" || refusal.Reason != testCase.reason || refusal.MismatchField != testCase.field || run.carriedTarget() {
				t.Fatalf("refusal=%#v carried_target=%t", refusal, run.carriedTarget())
			}
		})
	}
}

func TestSharedRuntimeEveryShapeMutantAdmitsPlainValidFrameAtProductionEntry(t *testing.T) {
	fixture := newSharedLauncherFixture(t)
	for _, mutant := range authShapeMutants {
		mutant := mutant
		t.Run(mutant.name, func(t *testing.T) {
			_ = os.Remove(fixture.marker)
			run := startSharedLauncherRunWithEnv(t, fixture, true, sharedShapeMutantEnv+"="+mutant.name)
			run.authorize(t, rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid)))
			run.requireTargetEvent(t, "shape mutant refused the plain valid frame at production entry")
			_ = run.command.Process.Kill()
			_ = run.wait(t, 3*time.Second)
		})
	}
}

func TestSharedRuntimeShapeMutantsDriveProductionLauncherGate(t *testing.T) {
	fixture := newSharedLauncherFixture(t)

	t.Run("unknown ignored admits the delivered unknown member", func(t *testing.T) {
		_ = os.Remove(fixture.marker)
		run := startSharedLauncherRunWithEnv(t, fixture, true, sharedShapeMutantEnv+"=unknown_ignored")
		valid := rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid))
		forged := bytes.Replace(valid, []byte{'{'}, []byte(`{"future_extension":true,`), 1)
		run.authorize(t, forged)
		run.requireTargetEvent(t, "unknown_ignored was not installed at the production launcher gate")
		_ = run.command.Process.Kill()
		_ = run.wait(t, 3*time.Second)
	})

	t.Run("wire membership over-refuses an escaped valid frame", func(t *testing.T) {
		_ = os.Remove(fixture.marker)
		run := startSharedLauncherRunWithEnv(t, fixture, true, sharedShapeMutantEnv+"=unknown_by_wire_form")
		frame := validSharedLauncherFrame(fixture, run.command.Process.Pid)
		raw := authBuildObject([]authGeneratedMember{
			authEscapedMember("schema", authQuoted(frame.Schema)),
			authEscapedMember("protocol_version", fmt.Sprintf("%d", frame.ProtocolVersion)),
			authEscapedMember("runtime_key", authQuoted(frame.RuntimeKey)),
			authEscapedMember("launcher_pid", fmt.Sprintf("%d", frame.LauncherPID)),
			authEscapedMember("exec_plan_digest", authQuoted(frame.ExecPlanDigest)),
		})
		run.authorize(t, raw)
		if err := run.wait(t, 3*time.Second); err == nil {
			t.Fatal("wire-form mutant admitted the escaped valid frame")
		}
		refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
		if refusal.Code != "protocol_violation" || refusal.Reason != "frame_unknown_field" || refusal.MismatchField != "schema" {
			t.Fatalf("wire-form refusal=%#v", refusal)
		}
		if _, err := os.Stat(fixture.marker); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("wire-form mutant executed target: %v", err)
		}
	})

	t.Run("wire keyed duplicate admits decoded duplicate", func(t *testing.T) {
		_ = os.Remove(fixture.marker)
		run := startSharedLauncherRunWithEnv(t, fixture, true, sharedShapeMutantEnv+"=dup_keyed_on_wire_form")
		frame := validSharedLauncherFrame(fixture, run.command.Process.Pid)
		raw := authBuildObject([]authGeneratedMember{
			authMember("schema", authQuoted(frame.Schema)),
			authEscapedMember("schema", authQuoted(frame.Schema)),
			authMember("protocol_version", fmt.Sprintf("%d", frame.ProtocolVersion)),
			authMember("runtime_key", authQuoted(frame.RuntimeKey)),
			authMember("launcher_pid", fmt.Sprintf("%d", frame.LauncherPID)),
			authMember("exec_plan_digest", authQuoted(frame.ExecPlanDigest)),
		})
		run.authorize(t, raw)
		run.requireTargetEvent(t, "wire-keyed duplicate mutant was not installed at the production launcher gate")
		_ = run.command.Process.Kill()
		_ = run.wait(t, 3*time.Second)
	})
}

func TestSharedRuntimeRejectAllProbeReddensAtProductionEntry(t *testing.T) {
	fixture := newSharedLauncherFixture(t)
	_ = os.Remove(fixture.marker)
	run := startSharedLauncherRunWithEnv(t, fixture, true, sharedShapeMutantEnv+"=reject_all_probe")
	run.authorize(t, rawSharedLauncherFrame(t, validSharedLauncherFrame(fixture, run.command.Process.Pid)))
	if err := run.wait(t, 3*time.Second); err == nil {
		t.Fatal("reject-all probe admitted the plain valid frame")
	}
	refusal := sharedRuntimeErrorFromOutput(t, run.stderr.String())
	if refusal.Code != "protocol_violation" || refusal.Reason != "frame_unknown_field" || refusal.MismatchField != "schema" {
		t.Fatalf("reject-all production refusal=%#v", refusal)
	}
	if _, err := os.Stat(fixture.marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reject-all probe executed target: %v", err)
	}
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := map[string]int{}
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}
