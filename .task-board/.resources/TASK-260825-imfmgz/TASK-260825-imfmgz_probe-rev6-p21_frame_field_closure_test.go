package probe6

// P21 - every field the authorization frame carries is a gate, and each one is
// proved by varying exactly that field.
//
// Review RUN-260825-668303 finding F1, BLOCKING: specification section 6.2 B11
// transmits `schema` and `protocol_version`, B12 step 4 requires protocol-version
// equality before `execve`, and section 11 promises
// `runtime_authorization_mismatch` for another protocol version. The revision-4
// P18 launcher parsed `Protocol` and never compared it, and no P18 case supplied
// a wrong version, so a green P18 could not support the claim. The reviewer minted
// a frame with the correct pid, runtime key and plan digest but
// `protocol_version = 999` and reached `execve` on the same pid.
//
// The defect was in the probe's launcher, not in the specification's rule - but a
// rule with no discriminating evidence is a rule an implementation may omit while
// satisfying the entire named suite, which is exactly what the reviewer showed.
//
// P21 closes it by making the launcher compare every field it reads and by
// proving each comparison separately:
//
//   P21.A  Control - a wholly valid frame execve's, and the kernel shows the
//          composed exec path and exact argv on the SAME pid.
//   P21.B  protocol_version = 999 refuses, names the field, and never carries the
//          target. Two mutants reproduce the defeat: `version_unchecked` is the
//          revision-4 probe's own defect verbatim, and `version_ge1` is the
//          NARROWED comparison - a range test rather than equality - which the
//          delete-only mutant would not have caught.
//   P21.C  A foreign `schema` refuses and names the field; `schema_unchecked`
//          reaches execve. This is revision 5's answer to review item 4: `schema`
//          is an equality gate, not a decorative field.
//   P21.D  The independent-variation table. Each of the five frame fields is
//          varied ALONE, with the other four correct, and must refuse naming that
//          field; the all-correct row in the same table must exec, so no row is
//          satisfied by a launcher that always refuses. Mutant `unnamed_field`
//          proves the field-naming assertion discriminates.
//   P21.E  The three `runtime_launch_unauthorized` reasons are distinguishable:
//          absent descriptor 3, a descriptor that is not a FIFO, a broker that
//          died before writing, and a broker that never wrote. Mutant
//          `collapse_fd3` deletes the descriptor gate and shows the absent-FIFO
//          case then reports the EOF reason - which is what made the revision-4
//          P18.E claim "absent FIFO refuses" unearned, since it asserted only an
//          exit code the EOF branch also produces.

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const (
	exitLaunchUnauthorized         = 70
	exitLaunchIdentityMismatch     = 71
	exitLaunchIdentityUnresolvable = 72
	exitAuthorizationMismatch      = 73
	exitLaunchProtocolViolation    = 74
)

// The two frame constants. Both are compiled into the binary; neither is read
// from configuration, because a frame field an attacker could also configure
// would gate nothing.
const (
	authSchema   = "agents-infra.pi.shared-runtime.auth.v1"
	authProtocol = 6
)

func init() { helpers["p21_launcher"] = p21Launcher }

// ---- composition (unchanged in shape from P18; the launcher's only source of
// executable, argv, cwd and timeout) ---------------------------------------

type p21Profile struct {
	Executable string
	Argv       []string
	StartupMS  int
}

func p21Compose(dir, profile string) (p21Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return p21Profile{}, fmt.Errorf("compose: %w", err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".conf") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var out p21Profile
	found := false
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return p21Profile{}, fmt.Errorf("compose: %w", err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, profile+".") {
				continue
			}
			kv := strings.SplitN(strings.TrimPrefix(line, profile+"."), "=", 2)
			if len(kv) != 2 {
				return p21Profile{}, fmt.Errorf("compose: malformed line %q", line)
			}
			found = true
			switch kv[0] {
			case "executable":
				out.Executable = kv[1]
			case "argv":
				out.Argv = nil
				if kv[1] != "" {
					out.Argv = strings.Split(kv[1], ",")
				}
			case "startup_ms":
				var v int
				fmt.Sscanf(kv[1], "%d", &v)
				out.StartupMS = v
			default:
				return p21Profile{}, fmt.Errorf("compose: unknown field %q", kv[0])
			}
		}
	}
	if !found {
		return p21Profile{}, fmt.Errorf("compose: profile %q not defined", profile)
	}
	return out, nil
}

func rec(h interface{ Write([]byte) (int, error) }, name, val string) {
	var l [8]byte
	binary.BigEndian.PutUint64(l[:], uint64(len(val)))
	h.Write([]byte(name))
	h.Write([]byte{0})
	h.Write(l[:])
	h.Write([]byte(val))
	h.Write([]byte{0x1E})
}

func p21Key(p p21Profile) string {
	h := sha256.New()
	rec(h, "executable", p.Executable)
	rec(h, "argc", fmt.Sprintf("%d", len(p.Argv)))
	for _, a := range p.Argv {
		rec(h, "argv", a)
	}
	rec(h, "startup_ms", fmt.Sprintf("%d", p.StartupMS))
	return hex.EncodeToString(h.Sum(nil))
}

func p21PlanDigest(p p21Profile, cwd string) string {
	h := sha256.New()
	rec(h, "executable", p.Executable)
	rec(h, "argc", fmt.Sprintf("%d", len(p.Argv)))
	for _, a := range p.Argv {
		rec(h, "argv", a)
	}
	rec(h, "cwd", cwd)
	rec(h, "startup_ms", fmt.Sprintf("%d", p.StartupMS))
	return hex.EncodeToString(h.Sum(nil))
}

type p21Frame struct {
	Schema      string `json:"schema"`
	Protocol    int    `json:"protocol_version"`
	RuntimeKey  string `json:"runtime_key"`
	LauncherPid int    `json:"launcher_pid"`
	PlanDigest  string `json:"exec_plan_digest"`
}

// refusal is the typed refusal the launcher emits on stderr, which B9 routes to
// the runtime log. `Field` is populated only for runtime_authorization_mismatch
// and is what lets a test say WHICH comparison refused rather than only that the
// process exited with a code four different branches share.
type refusal struct {
	Code   string `json:"code"`
	Reason string `json:"reason,omitempty"`
	Field  string `json:"mismatch_field,omitempty"`
}

func refuse(code, reason, field string, exit int) {
	b, _ := json.Marshal(refusal{Code: code, Reason: reason, Field: field})
	os.Stderr.Write(append(b, '\n'))
	os.Exit(exit)
}

// ---- the launcher entry point ----------------------------------------------

func p21flag(name string) string {
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--"+name+"=") {
			return strings.TrimPrefix(a, "--"+name+"=")
		}
	}
	return ""
}

func p21Launcher() {
	mutant := os.Getenv("P5_MUTANT")

	// Gate 1 - descriptor 3 must exist and must be a FIFO. Its refusal reason is
	// distinct from the gate-3 reasons even though the code is shared, because
	// "nobody authorized me" and "my broker died" are different facts about the
	// world and a test that cannot tell them apart proves neither.
	if mutant != "collapse_fd3" {
		var st unix.Stat_t
		if err := unix.Fstat(3, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFIFO {
			refuse("runtime_launch_unauthorized", "no_authorization_descriptor", "", exitLaunchUnauthorized)
		}
	}

	// Gate 2 - recompose. The only source of executable, argv, cwd and timeout,
	// and it runs before the read because the read's deadline is one of its
	// outputs.
	prof, err := p21Compose(p21flag("profile-project"), p21flag("profile"))
	if err != nil {
		refuse("runtime_launch_identity_unresolvable", "composition_failed", "", exitLaunchIdentityUnresolvable)
	}
	if p21Key(prof) != p21flag("runtime-key") {
		refuse("runtime_launch_identity_mismatch", "recomposed_key_differs", "", exitLaunchIdentityMismatch)
	}

	// Gate 3 - exactly one frame, bounded by the composed timeout.
	//
	// An inherited descriptor arrives in BLOCKING mode, and Go only registers a
	// nonblocking descriptor with its poller - so `SetReadDeadline` on the
	// descriptor as received returns ErrNoDeadline and the read is unbounded.
	// Revision 4's launcher ignored that error, which is why no revision-4 case
	// could reach the deadline branch at all. A launcher that cannot arm its
	// bound must refuse rather than read forever.
	// `collapse_fd3` and `deadline_ignored` are the revision-4 launcher verbatim:
	// the descriptor arrives blocking, `SetReadDeadline` therefore fails, and
	// revision 4 discarded that error - so its read had no bound at all and no
	// revision-4 case could reach the deadline branch.
	lax := mutant == "collapse_fd3" || mutant == "deadline_ignored"
	// P5_INJECT is fault INJECTION at the arming seam, not a mutant: gate 1 has
	// already proved descriptor 3 is a FIFO and a FIFO is pollable, so no input
	// reaches this branch. It is specified and exercised anyway because the
	// alternative to refusing is waiting forever, and "this cannot fail" is the
	// assumption revision 4 made about this exact call.
	if os.Getenv("P5_INJECT") == "arming_failure" {
		refuse("runtime_launch_unauthorized", "deadline_unavailable", "", exitLaunchUnauthorized)
	}
	if !lax {
		if err := unix.SetNonblock(3, true); err != nil {
			refuse("runtime_launch_unauthorized", "deadline_unavailable", "", exitLaunchUnauthorized)
		}
	}
	f := os.NewFile(3, "auth")
	deadline := time.Now().Add(time.Duration(prof.StartupMS) * time.Millisecond)
	if err := f.SetReadDeadline(deadline); err != nil && !lax {
		refuse("runtime_launch_unauthorized", "deadline_unavailable", "", exitLaunchUnauthorized)
	}
	buf := make([]byte, 65536)
	n, rerr := f.Read(buf)
	switch {
	case rerr != nil && os.IsTimeout(rerr):
		refuse("runtime_launch_unauthorized", "authorization_deadline", "", exitLaunchUnauthorized)
	case rerr != nil || n == 0:
		refuse("runtime_launch_unauthorized", "broker_died_before_authorizing", "", exitLaunchUnauthorized)
	}
	// Gate 3b - the frame's SHAPE, before any value is compared.
	//
	// Review RUN-260825-a8a4ef finding F1: revision 5 decoded with an ordinary
	// json.Unmarshal into a five-field struct and then compared those five struct
	// members. Go silently discards unknown object members and resolves duplicate
	// keys last-wins, so the five equality rows were all green while the frame
	// carried data the launcher never compared. The reviewer minted a frame with
	// an unknown sixth member, and a frame with protocol_version 999 followed by
	// the valid 5, and both reached execve.
	//
	// A comparison cannot close a field set. Only the decoder can, and only by
	// looking at the key multiset the bytes actually carry rather than at the
	// struct the implementation already knows about.
	fr, shape := p21DecodeFrame(buf[:n], mutant)
	if keylog := os.Getenv("P6_KEYLOG"); keylog != "" {
		// Section 12.4's comparison-set obligation, discharged on the decoded key
		// multiset rather than on the struct members: what the launcher SAW, not
		// what it happened to have a field for. Written BEFORE the verdict is
		// acted on, so a refusing run leaves the same evidence a passing one does.
		b, _ := json.Marshal(shapeEvidence{Keys: p21LastKeys, Compared: p21ComparedFields()})
		_ = os.WriteFile(keylog, b, 0o600)
	}
	if shape != nil {
		refuse("protocol_violation", shape.reason, shape.field, exitLaunchProtocolViolation)
	}

	// Gate 4 - every field of the frame is compared, in this order. There is no
	// field the launcher reads and ignores: that rule, not the individual
	// comparisons, is what stops this class of defect from returning.
	cwd, _ := os.Getwd()
	named := func(field string) string {
		if mutant == "unnamed_field" {
			return ""
		}
		return field
	}
	if mutant != "schema_unchecked" && fr.Schema != authSchema {
		refuse("runtime_authorization_mismatch", "frame_field_differs", named("schema"), exitAuthorizationMismatch)
	}
	switch mutant {
	case "version_unchecked":
		// MUTANT: the revision-4 probe's own defect - parse the field, never
		// compare it. This is what the reviewer's attack exploited.
	case "version_ge1":
		// MUTANT: NARROWED rather than deleted. A range test looks like a check
		// and admits every version above the floor, including 999.
		if fr.Protocol < 1 {
			refuse("runtime_authorization_mismatch", "frame_field_differs", named("protocol_version"), exitAuthorizationMismatch)
		}
	default:
		if fr.Protocol != authProtocol {
			refuse("runtime_authorization_mismatch", "frame_field_differs", named("protocol_version"), exitAuthorizationMismatch)
		}
	}
	if fr.LauncherPid != os.Getpid() {
		refuse("runtime_authorization_mismatch", "frame_field_differs", named("launcher_pid"), exitAuthorizationMismatch)
	}
	if fr.RuntimeKey != p21flag("runtime-key") {
		refuse("runtime_authorization_mismatch", "frame_field_differs", named("runtime_key"), exitAuthorizationMismatch)
	}
	if fr.PlanDigest != p21PlanDigest(prof, cwd) {
		refuse("runtime_authorization_mismatch", "frame_field_differs", named("exec_plan_digest"), exitAuthorizationMismatch)
	}

	// Gate 5 - execve in place.
	argv := append([]string{prof.Executable}, prof.Argv...)
	_ = unix.Exec(prof.Executable, argv, os.Environ())
	refuse("runtime_launch_unauthorized", "exec_failed", "", exitLaunchUnauthorized)
}

// ---- harness ---------------------------------------------------------------

func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func p21Project(t *testing.T, argvTok string, startupMS int) (dir string, prof p21Profile) {
	t.Helper()
	dir = shortTempDir(t)
	body := fmt.Sprintf("qwen.executable=/bin/sleep\nqwen.argv=%s\nqwen.startup_ms=%d\n", argvTok, startupMS)
	if err := os.WriteFile(filepath.Join(dir, "10-base.conf"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := p21Compose(dir, "qwen")
	if err != nil {
		t.Fatal(err)
	}
	return dir, p
}

type p21Run struct {
	cmd    *exec.Cmd
	pid    int
	w      *os.File
	errLog string

	mu   sync.Mutex
	seen map[string]bool
	done chan struct{}
}

type p21Opts struct {
	projDir string
	workDir string
	key     string
	env     []string
	fd3     *os.File // nil ⇒ a pipe is created; use noFD3 to pass none at all
	noFD3   bool
}

func p21Start(t *testing.T, o p21Opts) *p21Run {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	errPath := filepath.Join(shortTempDir(t), "launcher.err")
	errFile, err := os.Create(errPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe,
		"-p5-runtime-launch",
		"--runtime-key="+o.key,
		"--profile-project="+o.projDir,
		"--profile=qwen",
	)
	cmd.Env = append(os.Environ(), helperEnv+"=p21_launcher")
	cmd.Env = append(cmd.Env, o.env...)
	cmd.Dir = o.workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = errFile

	var w *os.File
	switch {
	case o.noFD3:
		// nothing on descriptor 3 at all
	case o.fd3 != nil:
		cmd.ExtraFiles = []*os.File{o.fd3}
	default:
		r, pw, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		cmd.ExtraFiles = []*os.File{r}
		w = pw
		defer r.Close()
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = errFile.Close()

	run := &p21Run{cmd: cmd, pid: cmd.Process.Pid, w: w, errLog: errPath,
		seen: map[string]bool{}, done: make(chan struct{})}

	// Poll the kernel for every exec path this pid ever carried. "the runtime is
	// not running now" is not "the runtime never ran"; only a poll that never
	// observes the target can say that. A set rather than a bounded channel,
	// because a full channel would silently drop the one observation that
	// matters.
	go func() {
		defer close(run.done)
		for i := 0; i < 600; i++ {
			id, err := Identify(run.pid)
			if err != nil {
				return
			}
			run.mu.Lock()
			run.seen[id.Exe] = true
			run.mu.Unlock()
			time.Sleep(5 * time.Millisecond)
		}
	}()
	t.Cleanup(func() {
		if run.w != nil {
			_ = run.w.Close()
		}
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		<-run.done
	})
	return run
}

func (r *p21Run) authorize(t *testing.T, f p21Frame) {
	t.Helper()
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.w.Write(b); err != nil {
		t.Fatalf("authorize: %v", err)
	}
}

func (r *p21Run) waitExit(t *testing.T) int {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- r.cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			return 0
		}
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		t.Fatal(err)
	case <-time.After(20 * time.Second):
		t.Fatal("launcher did not exit")
	}
	return -1
}

// refusal reads the typed refusal the launcher wrote to its log.
func (r *p21Run) refusal(t *testing.T) refusal {
	t.Helper()
	b, err := os.ReadFile(r.errLog)
	if err != nil {
		t.Fatalf("refusal log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	var out refusal
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &out); err != nil {
		t.Fatalf("refusal log %q: %v", string(b), err)
	}
	return out
}

// everCarried waits for the poller to finish and answers whether the pid was
// ever observed executing target.
func (r *p21Run) everCarried(target string) bool {
	<-r.done
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.seen[target]
}

// becameTarget waits until the pid is observed executing target.
func (r *p21Run) becameTarget(target string) (ProcIdentity, bool) {
	for i := 0; i < 600; i++ {
		if got, err := Identify(r.pid); err == nil && got.Exe == target {
			return got, true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return ProcIdentity{}, false
}

func validFrame(prof p21Profile, key string, pid int, cwd string) p21Frame {
	return p21Frame{
		Schema:      authSchema,
		Protocol:    authProtocol,
		RuntimeKey:  key,
		LauncherPid: pid,
		PlanDigest:  p21PlanDigest(prof, cwd),
	}
}

func TestP21_FrameFieldClosure(t *testing.T) {
	t.Run("A_control_valid_frame_execs", func(t *testing.T) {
		dir, prof := p21Project(t, "51", 5000)
		key := p21Key(prof)
		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		run.authorize(t, validFrame(prof, key, run.pid, realPath(t, dir)))

		id, ok := run.becameTarget("/bin/sleep")
		if !ok {
			t.Fatal("launcher never became the target")
		}
		want := []string{"/bin/sleep", "51"}
		if len(id.Argv) != 2 || id.Argv[0] != want[0] || id.Argv[1] != want[1] {
			t.Fatalf("argv = %v, want %v", id.Argv, want)
		}
		t.Logf("P21.A control: pid %d execve'd %v under a wholly valid frame", run.pid, id.Argv)
	})

	t.Run("B_wrong_protocol_version_refuses", func(t *testing.T) {
		// The reviewer's exact attack shape: correct pid, correct runtime key,
		// correct plan digest, incompatible protocol version.
		dir, prof := p21Project(t, "52", 5000)
		key := p21Key(prof)

		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		f := validFrame(prof, key, run.pid, realPath(t, dir))
		f.Protocol = 999
		run.authorize(t, f)

		if code := run.waitExit(t); code != exitAuthorizationMismatch {
			t.Fatalf("exit = %d, want runtime_authorization_mismatch (%d)", code, exitAuthorizationMismatch)
		}
		if got := run.refusal(t); got.Code != "runtime_authorization_mismatch" || got.Field != "protocol_version" {
			t.Fatalf("refusal = %+v, want code=runtime_authorization_mismatch field=protocol_version", got)
		}
		if run.everCarried("/bin/sleep") {
			t.Fatal("launcher carried the target exec path despite refusing")
		}
		t.Log("P21.B: protocol_version=999 refused by name, and the pid NEVER carried /bin/sleep")

		// MUTANT 1 - delete the comparison. This is the revision-4 P18 launcher
		// verbatim, and it reproduces the reviewer's DEFEAT.
		m1 := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: []string{"P5_MUTANT=version_unchecked"}})
		f1 := validFrame(prof, key, m1.pid, realPath(t, dir))
		f1.Protocol = 999
		m1.authorize(t, f1)
		if _, ok := m1.becameTarget("/bin/sleep"); !ok {
			t.Fatal("version_unchecked mutant did not exec; P21.B would pass against a launcher that never runs anything")
		}
		t.Log("P21.B mutant version_unchecked: protocol_version=999 REACHES execve - the revision-4 defect reproduced")

		// MUTANT 2 - NARROW the comparison to a range instead of deleting it. A
		// delete-only mutant proves the gate exists; this one proves the gate is
		// equality, which is the class the bump rule in section 9 depends on.
		m2 := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: []string{"P5_MUTANT=version_ge1"}})
		f2 := validFrame(prof, key, m2.pid, realPath(t, dir))
		f2.Protocol = 999
		m2.authorize(t, f2)
		if _, ok := m2.becameTarget("/bin/sleep"); !ok {
			t.Fatal("version_ge1 mutant did not exec; the narrowed comparison is not distinguished from equality")
		}
		t.Log("P21.B mutant version_ge1: a >=1 range test admits 999 - equality is load-bearing, not the presence of a check")

		// A version BELOW the floor still refuses under the narrowed mutant, so
		// the mutant is a narrowing rather than a deletion.
		m3 := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: []string{"P5_MUTANT=version_ge1"}})
		f3 := validFrame(prof, key, m3.pid, realPath(t, dir))
		f3.Protocol = 0
		m3.authorize(t, f3)
		if code := m3.waitExit(t); code != exitAuthorizationMismatch {
			t.Fatalf("version_ge1 with version 0: exit = %d, want %d", code, exitAuthorizationMismatch)
		}
		t.Log("P21.B: the version_ge1 mutant still refuses version 0, so it narrows the gate rather than removing it")
	})

	t.Run("C_wrong_schema_refuses", func(t *testing.T) {
		dir, prof := p21Project(t, "53", 5000)
		key := p21Key(prof)

		run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		f := validFrame(prof, key, run.pid, realPath(t, dir))
		f.Schema = "agents-infra.pi.shared-runtime.auth.v0"
		run.authorize(t, f)
		if code := run.waitExit(t); code != exitAuthorizationMismatch {
			t.Fatalf("exit = %d, want %d", code, exitAuthorizationMismatch)
		}
		if got := run.refusal(t); got.Field != "schema" {
			t.Fatalf("refusal = %+v, want field=schema", got)
		}
		if run.everCarried("/bin/sleep") {
			t.Fatal("launcher exec'd on a foreign schema")
		}
		t.Log("P21.C: a foreign schema refuses by name - review item 4 answered by binding, not by dropping the field")

		mut := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: []string{"P5_MUTANT=schema_unchecked"}})
		fm := validFrame(prof, key, mut.pid, realPath(t, dir))
		fm.Schema = "agents-infra.pi.shared-runtime.auth.v0"
		mut.authorize(t, fm)
		if _, ok := mut.becameTarget("/bin/sleep"); !ok {
			t.Fatal("schema_unchecked mutant did not exec; P21.C proves nothing")
		}
		t.Log("P21.C mutant schema_unchecked: a foreign schema REACHES execve")
	})

	t.Run("D_independent_field_variation_table", func(t *testing.T) {
		dir, prof := p21Project(t, "54", 5000)
		key := p21Key(prof)
		cwd := realPath(t, dir)
		otherDigest := p21PlanDigest(p21Profile{Executable: prof.Executable, Argv: []string{"99"}, StartupMS: prof.StartupMS}, cwd)

		rows := []struct {
			field string
			bend  func(*p21Frame, int)
		}{
			{"schema", func(f *p21Frame, _ int) { f.Schema = "not.the.schema.v1" }},
			{"protocol_version", func(f *p21Frame, _ int) { f.Protocol = authProtocol + 1 }},
			{"launcher_pid", func(f *p21Frame, pid int) { f.LauncherPid = pid + 100000 }},
			{"runtime_key", func(f *p21Frame, _ int) { f.RuntimeKey = strings.Repeat("0", 64) }},
			{"exec_plan_digest", func(f *p21Frame, _ int) { f.PlanDigest = otherDigest }},
		}
		for _, row := range rows {
			row := row
			t.Run(row.field, func(t *testing.T) {
				run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
				f := validFrame(prof, key, run.pid, cwd)
				row.bend(&f, run.pid)
				run.authorize(t, f)
				if code := run.waitExit(t); code != exitAuthorizationMismatch {
					t.Fatalf("exit = %d, want %d", code, exitAuthorizationMismatch)
				}
				if got := run.refusal(t); got.Field != row.field {
					t.Fatalf("refusal = %+v, want field=%s", got, row.field)
				}
				if run.everCarried("/bin/sleep") {
					t.Fatalf("launcher exec'd with a wrong %s", row.field)
				}
			})
		}

		// The discriminating row: the same table with nothing bent must exec, so
		// none of the five rows above is satisfied by a launcher that refuses
		// everything.
		t.Run("all_correct_control", func(t *testing.T) {
			run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
			run.authorize(t, validFrame(prof, key, run.pid, cwd))
			if _, ok := run.becameTarget("/bin/sleep"); !ok {
				t.Fatal("all-correct control did not exec; the five refusals above discriminate nothing")
			}
		})

		// MUTANT - refuse without naming the field. The exit code alone is shared
		// by all five rows, so a launcher that refuses anonymously would satisfy
		// an exit-code-only assertion for every row including the wrong ones.
		t.Run("mutant_unnamed_field", func(t *testing.T) {
			run := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: []string{"P5_MUTANT=unnamed_field"}})
			f := validFrame(prof, key, run.pid, cwd)
			f.Protocol = 999
			run.authorize(t, f)
			if code := run.waitExit(t); code != exitAuthorizationMismatch {
				t.Fatalf("exit = %d, want %d", code, exitAuthorizationMismatch)
			}
			if got := run.refusal(t); got.Field != "" {
				t.Fatalf("refusal = %+v, want an unnamed field under the mutant", got)
			}
			t.Log("P21.D mutant unnamed_field: the exit code survives, the field name does not - which is why the rows assert the name")
		})
	})

	t.Run("E_unauthorized_reasons_are_distinguishable", func(t *testing.T) {
		dir, prof := p21Project(t, "55", 800)
		key := p21Key(prof)

		// (i) no descriptor 3 at all
		none := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, noFD3: true})
		if code := none.waitExit(t); code != exitLaunchUnauthorized {
			t.Fatalf("absent fd3: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		if got := none.refusal(t); got.Reason != "no_authorization_descriptor" {
			t.Fatalf("absent fd3: refusal = %+v", got)
		}

		// (ii) descriptor 3 open on a regular file
		reg, err := os.Create(filepath.Join(shortTempDir(t), "notafifo"))
		if err != nil {
			t.Fatal(err)
		}
		file := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, fd3: reg})
		_ = reg.Close()
		if code := file.waitExit(t); code != exitLaunchUnauthorized {
			t.Fatalf("regular file fd3: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		if got := file.refusal(t); got.Reason != "no_authorization_descriptor" {
			t.Fatalf("regular file fd3: refusal = %+v", got)
		}

		// (iii) the broker dies before writing: the write end closes, EOF
		eof := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		_ = eof.w.Close()
		eof.w = nil
		if code := eof.waitExit(t); code != exitLaunchUnauthorized {
			t.Fatalf("EOF: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		if got := eof.refusal(t); got.Reason != "broker_died_before_authorizing" {
			t.Fatalf("EOF: refusal = %+v", got)
		}

		// (iv) the broker holds the write end and never writes: deadline
		slow := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		if code := slow.waitExit(t); code != exitLaunchUnauthorized {
			t.Fatalf("deadline: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		if got := slow.refusal(t); got.Reason != "authorization_deadline" {
			t.Fatalf("deadline: refusal = %+v", got)
		}

		// (v) a truncated frame is a protocol violation, not an authorization
		// question, and is never retried.
		trunc := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		if _, err := trunc.w.Write([]byte(`{"protocol_version":5,"runtime_`)); err != nil {
			t.Fatal(err)
		}
		if code := trunc.waitExit(t); code != exitLaunchProtocolViolation {
			t.Fatalf("truncated: exit = %d, want %d", code, exitLaunchProtocolViolation)
		}

		for _, r := range []*p21Run{none, file, eof, slow, trunc} {
			if r.everCarried("/bin/sleep") {
				t.Fatal("a refusing launcher carried the target exec path")
			}
		}
		t.Log("P21.E: four unauthorized shapes, three distinct reasons, one protocol violation - none of them exec")

		// MUTANT - delete the descriptor gate. The absent-fd3 case then falls
		// through to the read and reports the EOF reason, which is exactly why
		// revision 4's exit-code-only assertion did not earn its claim.
		mut := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, noFD3: true, env: []string{"P5_MUTANT=collapse_fd3"}})
		if code := mut.waitExit(t); code != exitLaunchUnauthorized {
			t.Fatalf("collapse_fd3: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		got := mut.refusal(t)
		if got.Reason != "broker_died_before_authorizing" {
			t.Fatalf("collapse_fd3 mutant reported %q, want the EOF reason - the mutant must reproduce revision 4, where an absent descriptor was indistinguishable from a dead broker", got.Reason)
		}
		t.Logf("P21.E mutant collapse_fd3: absent descriptor reports %q - the SAME exit code revision 4 asserted, attributing the refusal to a broker that never existed", got.Reason)
	})

	// P21.F is a self-found finding of revision 5, and it is a DELIBERATE
	// reproduction before it is a fix. Specification B12 step 3 says the launcher
	// reads one frame "with a deadline of the composed
	// runtime.startup_timeout_seconds". A descriptor inherited through
	// ExtraFiles arrives in BLOCKING mode; Go registers only a nonblocking
	// descriptor with its poller, so SetReadDeadline on the descriptor as
	// received fails and the read is unbounded. Revision 4's launcher discarded
	// that error, which is why no revision-4 case ever reached its own deadline
	// branch - the branch was unreachable, not merely untested. An implementation
	// that follows revision 4 literally waits forever for a broker that will
	// never write.
	t.Run("F_the_read_bound_must_be_armed_not_assumed", func(t *testing.T) {
		dir, prof := p21Project(t, "56", 700)
		key := p21Key(prof)

		// Production: the bound is armed, so a silent broker produces a deadline
		// refusal well inside the observation window.
		bounded := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key})
		start := time.Now()
		if code := bounded.waitExit(t); code != exitLaunchUnauthorized {
			t.Fatalf("bounded: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		if got := bounded.refusal(t); got.Reason != "authorization_deadline" {
			t.Fatalf("bounded: refusal = %+v, want authorization_deadline", got)
		}
		elapsed := time.Since(start)
		if elapsed > 5*time.Second {
			t.Fatalf("bounded launcher took %s for a 700ms bound", elapsed)
		}
		t.Logf("P21.F: the armed bound fired after %s and refused authorization_deadline", elapsed)

		// MUTANT deadline_ignored - revision 4's launcher. The same silent broker
		// leaves it blocked past several multiples of its own composed bound.
		mut := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: []string{"P5_MUTANT=deadline_ignored"}})
		done := make(chan int, 1)
		go func() {
			st, err := mut.cmd.Process.Wait()
			if err != nil {
				done <- -1
				return
			}
			done <- st.ExitCode()
		}()
		select {
		case code := <-done:
			t.Fatalf("deadline_ignored mutant exited with %d; the unbounded read did not reproduce", code)
		case <-time.After(4 * time.Second):
		}
		t.Log("P21.F mutant deadline_ignored: still blocked after 4s on a 700ms bound - revision 4's read had no bound at all")

		// The arming-failure branch, reached by INJECTION rather than by an
		// input. Recorded as an injection in the evidence so no reader mistakes
		// it for an input-driven case.
		inj := p21Start(t, p21Opts{projDir: dir, workDir: dir, key: key, env: []string{"P5_INJECT=arming_failure"}})
		if code := inj.waitExit(t); code != exitLaunchUnauthorized {
			t.Fatalf("arming failure: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		if got := inj.refusal(t); got.Reason != "deadline_unavailable" {
			t.Fatalf("arming failure: refusal = %+v, want deadline_unavailable", got)
		}
		if inj.everCarried("/bin/sleep") {
			t.Fatal("a launcher that could not arm its bound exec'd anyway")
		}
		t.Log("P21.F injection arming_failure: refuses deadline_unavailable without execve - a launcher that cannot bound its wait does not wait")
	})
}
