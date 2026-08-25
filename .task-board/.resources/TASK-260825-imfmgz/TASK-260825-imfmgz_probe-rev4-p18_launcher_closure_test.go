package probe4

// P18 - the launcher's execution inputs, closed at the production entry point.
//
// Review RUN-260825-969723 finding F1, BLOCKING: revision 3's B7 spawned
// `runtime runtime-launch --runtime-key <hex>` and its B9 frame carried only
// schema, protocol version, runtime_key and launcher_pid - yet B10 required the
// launcher to apply `runtime.startup_timeout_seconds` and to execve
// `runtime.executable + runtime.argv`. None of those was supplied and a SHA-256
// cannot be inverted. The revision-3 probes hid the gap: their launcher took
// PROBE_TARGET / PROBE_TARGET_ARG / PROBE_AUTH_TIMEOUT_MS from its environment,
// a probe-only capability with no normative counterpart.
//
// Revision 4 closes it WITHOUT adding a data channel. The launcher is the same
// binary as the broker, so it re-runs the SAME full profile composition the
// broker ran, from `--profile-project DIR --profile NAME`, and derives the
// executable, argv, cwd and timeout from it. The recomputed `runtime_key` must
// equal `--runtime-key`. The frame therefore never says WHAT to run; it says
// only WHETHER, and to whom.
//
//   P18.A  Control - matching composition and a valid frame: the launcher
//          execve's in place and the kernel shows the target's exec path and
//          exact argv on the SAME pid.
//   P18.B  Divergent project: composition yields a different key. Refuse before
//          the frame is even read, and NEVER carry the target's exec path.
//   P18.C  Unreadable project: refuse. The frame in this variant is valid and
//          the mutant P18.C-mutant proves the assertion discriminates - a build
//          that falls back on a read failure DOES exec.
//   P18.D  Frame whose exec_plan_digest disagrees with the launcher's own
//          recomputation: refuse without exec. This is the record/frame
//          divergence case.
//   P18.E  Frame naming another pid, and no descriptor 3 at all: refuse.

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
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

const (
	exitLaunchUnauthorized      = 70
	exitLaunchIdentityMismatch  = 71
	exitLaunchIdentityUnresolvable = 72
	exitAuthorizationMismatch   = 73
	exitLaunchProtocolViolation = 74
)

func init() { helpers["p18_launcher"] = p18Launcher }

// ---- composition -----------------------------------------------------------

type p18Profile struct {
	Executable string
	Argv       []string
	StartupMS  int
}

// p18Compose is the stand-in for the real full profile composition: every
// `*.conf` in the project directory, byte-sorted, last definition of the named
// profile wins whole. The point of the probe is not the config format; it is
// that the launcher reaches the same values from the same DIRECTORY the broker
// used, and that a failure to read is fatal rather than a fallback.
func p18Compose(dir, profile string) (p18Profile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return p18Profile{}, fmt.Errorf("compose: %w", err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".conf") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var out p18Profile
	found := false
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return p18Profile{}, fmt.Errorf("compose: %w", err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || !strings.HasPrefix(line, profile+".") {
				continue
			}
			kv := strings.SplitN(strings.TrimPrefix(line, profile+"."), "=", 2)
			if len(kv) != 2 {
				return p18Profile{}, fmt.Errorf("compose: malformed line %q", line)
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
				return p18Profile{}, fmt.Errorf("compose: unknown field %q", kv[0])
			}
		}
	}
	if !found {
		return p18Profile{}, fmt.Errorf("compose: profile %q not defined", profile)
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

func p18Key(p p18Profile) string {
	h := sha256.New()
	rec(h, "executable", p.Executable)
	rec(h, "argc", fmt.Sprintf("%d", len(p.Argv)))
	for _, a := range p.Argv {
		rec(h, "argv", a)
	}
	rec(h, "startup_ms", fmt.Sprintf("%d", p.StartupMS))
	return hex.EncodeToString(h.Sum(nil))
}

// p18PlanDigest covers exactly the values the launcher will act on. The broker
// writes it into the durable record at B10 and repeats it in the B11 frame; the
// launcher recomputes it from its OWN composition and requires equality.
func p18PlanDigest(p p18Profile, cwd string) string {
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

type p18Frame struct {
	Schema      string `json:"schema"`
	Protocol    int    `json:"protocol_version"`
	RuntimeKey  string `json:"runtime_key"`
	LauncherPid int    `json:"launcher_pid"`
	PlanDigest  string `json:"exec_plan_digest"`
}

// ---- the launcher entry point ----------------------------------------------

func p18flag(name string) string {
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, "--"+name+"=") {
			return strings.TrimPrefix(a, "--"+name+"=")
		}
	}
	return ""
}

func p18Launcher() {
	mutant := os.Getenv("P4_MUTANT")

	// Gate 1 - descriptor 3 must be a FIFO.
	var st unix.Stat_t
	if err := unix.Fstat(3, &st); err != nil || st.Mode&unix.S_IFMT != unix.S_IFIFO {
		os.Exit(exitLaunchUnauthorized)
	}

	// Gate 2 - recompose from the project directory. This is the ONLY source of
	// executable, argv, cwd and timeout. It happens before the frame is read,
	// because the frame's read deadline is one of the values it produces.
	fellBack := false
	prof, err := p18Compose(p18flag("profile-project"), p18flag("profile"))
	if err != nil {
		if mutant == "compose_fallback" {
			// MUTANT: treat an unreadable project as satisfied and continue on
			// a last-known target, skipping every check that depended on the
			// read. This is the single behaviour P18.C must redden: "a failed
			// read is not an absence" has no teeth unless a build that treats
			// it as one is observably different.
			prof = p18Profile{Executable: os.Getenv("P4_FALLBACK_EXE"), Argv: []string{"7"}, StartupMS: 2000}
			fellBack = true
		} else {
			os.Exit(exitLaunchIdentityUnresolvable)
		}
	}
	if !fellBack && p18Key(prof) != p18flag("runtime-key") {
		os.Exit(exitLaunchIdentityMismatch)
	}

	// Gate 3 - read exactly one frame, bounded by the composed timeout.
	f := os.NewFile(3, "auth")
	_ = f.SetReadDeadline(time.Now().Add(time.Duration(prof.StartupMS) * time.Millisecond))
	buf := make([]byte, 65536)
	n, rerr := f.Read(buf)
	if rerr != nil || n == 0 {
		os.Exit(exitLaunchUnauthorized) // EOF: the broker died before authorizing
	}
	var fr p18Frame
	if json.Unmarshal(buf[:n], &fr) != nil {
		os.Exit(exitLaunchProtocolViolation)
	}

	// Gate 4 - the frame authorizes THIS process for THIS runtime and agrees
	// with the plan the launcher itself composed.
	cwd, _ := os.Getwd()
	if fr.LauncherPid != os.Getpid() || fr.RuntimeKey != p18flag("runtime-key") {
		os.Exit(exitAuthorizationMismatch)
	}
	if !fellBack && mutant != "skip_plan_digest" && fr.PlanDigest != p18PlanDigest(prof, cwd) {
		os.Exit(exitAuthorizationMismatch)
	}

	// Gate 5 - execve in place. No shell, no PATH lookup, no interpolation.
	argv := append([]string{prof.Executable}, prof.Argv...)
	_ = unix.Exec(prof.Executable, argv, os.Environ())
	os.Exit(exitLaunchUnauthorized)
}

// ---- harness ---------------------------------------------------------------

// realPath resolves /tmp -> /private/tmp so the parent's expected cwd matches
// the child's os.Getwd(). A cwd disagreement would make every plan digest
// diverge and the probe would prove nothing about the gate it is testing.
func realPath(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func p18Project(t *testing.T, argvTok string) (dir string, prof p18Profile) {
	t.Helper()
	dir = shortTempDir(t)
	body := "qwen.executable=/bin/sleep\nqwen.argv=" + argvTok + "\nqwen.startup_ms=3000\n"
	if err := os.WriteFile(filepath.Join(dir, "10-base.conf"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	p, err := p18Compose(dir, "qwen")
	if err != nil {
		t.Fatal(err)
	}
	return dir, p
}

type p18Run struct {
	cmd  *exec.Cmd
	pid  int
	w    *os.File
	seen chan string // exec paths observed on the launcher pid
}

func p18Start(t *testing.T, projDir, workDir, key string, env ...string) *p18Run {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe,
		"-p4-runtime-launch",
		"--runtime-key="+key,
		"--profile-project="+projDir,
		"--profile=qwen",
	)
	cmd.Env = append(os.Environ(), helperEnv+"=p18_launcher")
	cmd.Env = append(cmd.Env, env...)
	cmd.ExtraFiles = []*os.File{r}
	cmd.Dir = workDir
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	_ = r.Close()
	run := &p18Run{cmd: cmd, pid: cmd.Process.Pid, w: w, seen: make(chan string, 64)}

	// Poll the kernel for what this pid is executing. "the runtime is not
	// running now" is not "the runtime never ran"; only a poll that never sees
	// the target's exec path can say that.
	go func() {
		defer close(run.seen)
		for i := 0; i < 400; i++ {
			id, err := Identify(run.pid)
			if err != nil {
				return
			}
			select {
			case run.seen <- id.Exe:
			default:
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()
	t.Cleanup(func() {
		_ = w.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	return run
}

func (r *p18Run) authorize(t *testing.T, f p18Frame) {
	t.Helper()
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.w.Write(b); err != nil {
		t.Fatalf("authorize: %v", err)
	}
}

func (r *p18Run) waitExit(t *testing.T) int {
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
	case <-time.After(15 * time.Second):
		t.Fatal("launcher did not exit")
	}
	return -1
}

func (r *p18Run) everCarried(target string) bool {
	for e := range r.seen {
		if e == target {
			return true
		}
	}
	return false
}

func TestP18_LauncherInputClosure(t *testing.T) {
	t.Run("A_control_matching_composition_and_valid_frame_execs", func(t *testing.T) {
		dir, prof := p18Project(t, "31")
		key := p18Key(prof)
		run := p18Start(t, dir, dir, key)
		run.authorize(t, p18Frame{Schema: "agents-infra.pi.shared-runtime.auth.v1", Protocol: 1,
			RuntimeKey: key, LauncherPid: run.pid, PlanDigest: p18PlanDigest(prof, realPath(t, dir))})

		var id ProcIdentity
		ok := false
		for i := 0; i < 400; i++ {
			if got, err := Identify(run.pid); err == nil && got.Exe == "/bin/sleep" {
				id, ok = got, true
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if !ok {
			t.Fatal("launcher never became the target")
		}
		want := []string{"/bin/sleep", "31"}
		if len(id.Argv) != len(want) || id.Argv[0] != want[0] || id.Argv[1] != want[1] {
			t.Fatalf("argv = %v, want %v", id.Argv, want)
		}
		t.Logf("P18.A control: pid %d execve'd /bin/sleep argv=%v derived from composition alone - no target, argv or timeout crossed the pipe", run.pid, id.Argv)
	})

	t.Run("B_divergent_project_refuses_before_reading_the_frame", func(t *testing.T) {
		_, prof := p18Project(t, "31")
		key := p18Key(prof) // the broker's key, from the broker's composition

		// The launcher's project now composes a DIFFERENT argv token.
		other := shortTempDir(t)
		if err := os.WriteFile(filepath.Join(other, "10-base.conf"),
			[]byte("qwen.executable=/bin/sleep\nqwen.argv=32\nqwen.startup_ms=3000\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		run := p18Start(t, other, other, key)
		// A perfectly valid frame is offered anyway.
		run.authorize(t, p18Frame{Protocol: 1, RuntimeKey: key, LauncherPid: run.pid, PlanDigest: p18PlanDigest(prof, realPath(t, other))})

		if code := run.waitExit(t); code != exitLaunchIdentityMismatch {
			t.Fatalf("exit = %d, want runtime_launch_identity_mismatch (%d)", code, exitLaunchIdentityMismatch)
		}
		if run.everCarried("/bin/sleep") {
			t.Fatal("launcher carried the target exec path despite refusing")
		}
		t.Log("P18.B: divergent composition refused, and the pid NEVER carried /bin/sleep - it did not run briefly")
	})

	t.Run("C_unreadable_project_refuses_and_never_falls_back", func(t *testing.T) {
		dir, prof := p18Project(t, "33")
		key := p18Key(prof)
		gone := filepath.Join(dir, "does-not-exist")

		run := p18Start(t, gone, dir, key)
		run.authorize(t, p18Frame{Protocol: 1, RuntimeKey: key, LauncherPid: run.pid, PlanDigest: p18PlanDigest(prof, realPath(t, dir))})
		if code := run.waitExit(t); code != exitLaunchIdentityUnresolvable {
			t.Fatalf("exit = %d, want runtime_launch_identity_unresolvable (%d)", code, exitLaunchIdentityUnresolvable)
		}
		if run.everCarried("/bin/sleep") {
			t.Fatal("launcher exec'd despite an unresolvable composition")
		}
		t.Log("P18.C: unreadable project ⇒ refusal, no exec")

		// MUTANT control: a build that treats the read failure as permission to
		// use a last-known target DOES exec. Without this the assertion above
		// could be satisfied by a launcher that never execs at all.
		mut := p18Start(t, gone, dir, key, "P4_MUTANT=compose_fallback", "P4_FALLBACK_EXE=/bin/sleep")
		mut.authorize(t, p18Frame{Protocol: 1, RuntimeKey: key, LauncherPid: mut.pid, PlanDigest: p18PlanDigest(prof, realPath(t, dir))})
		carried := false
		for i := 0; i < 400 && !carried; i++ {
			if id, err := Identify(mut.pid); err == nil && id.Exe == "/bin/sleep" {
				carried = true
			}
			time.Sleep(5 * time.Millisecond)
		}
		if !carried {
			t.Fatal("mutant did not exec; P18.C would pass against a launcher that never execs")
		}
		t.Log("P18.C mutant: compose-failure fallback DOES exec /bin/sleep - the refusal above is a real distinction")
	})

	t.Run("D_frame_plan_digest_divergence_refuses", func(t *testing.T) {
		dir, prof := p18Project(t, "34")
		key := p18Key(prof)
		run := p18Start(t, dir, dir, key)
		bad := p18PlanDigest(p18Profile{Executable: prof.Executable, Argv: []string{"99"}, StartupMS: prof.StartupMS}, realPath(t, dir))
		run.authorize(t, p18Frame{Protocol: 1, RuntimeKey: key, LauncherPid: run.pid, PlanDigest: bad})
		if code := run.waitExit(t); code != exitAuthorizationMismatch {
			t.Fatalf("exit = %d, want runtime_authorization_mismatch (%d)", code, exitAuthorizationMismatch)
		}
		if run.everCarried("/bin/sleep") {
			t.Fatal("launcher exec'd on a divergent plan digest")
		}
		t.Log("P18.D: the record's plan and the launcher's own composition must agree, or nothing runs")
	})

	t.Run("E_foreign_pid_partial_frame_and_no_fifo", func(t *testing.T) {
		dir, prof := p18Project(t, "35")
		key := p18Key(prof)

		foreign := p18Start(t, dir, dir, key)
		foreign.authorize(t, p18Frame{Protocol: 1, RuntimeKey: key, LauncherPid: foreign.pid + 100000, PlanDigest: p18PlanDigest(prof, realPath(t, dir))})
		if code := foreign.waitExit(t); code != exitAuthorizationMismatch {
			t.Fatalf("foreign pid: exit = %d, want %d", code, exitAuthorizationMismatch)
		}
		if foreign.everCarried("/bin/sleep") {
			t.Fatal("exec'd on a frame naming another pid")
		}

		partial := p18Start(t, dir, dir, key)
		if _, err := partial.w.Write([]byte(`{"protocol_version":1,"runtime_`)); err != nil {
			t.Fatal(err)
		}
		if code := partial.waitExit(t); code != exitLaunchProtocolViolation {
			t.Fatalf("partial frame: exit = %d, want %d", code, exitLaunchProtocolViolation)
		}
		if partial.everCarried("/bin/sleep") {
			t.Fatal("exec'd on a truncated frame")
		}

		exe, _ := os.Executable()
		nofifo := exec.Command(exe, "-p4-runtime-launch", "--runtime-key="+key, "--profile-project="+dir, "--profile=qwen")
		nofifo.Env = append(os.Environ(), helperEnv+"=p18_launcher")
		out := nofifo.Run()
		code := 0
		if ee, ok := out.(*exec.ExitError); ok {
			code = ee.ExitCode()
		}
		if code != exitLaunchUnauthorized {
			t.Fatalf("no descriptor 3: exit = %d, want %d", code, exitLaunchUnauthorized)
		}
		t.Log("P18.E: foreign pid, truncated frame and absent FIFO all refuse without exec")
	})
}
