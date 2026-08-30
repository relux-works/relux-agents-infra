//go:build darwin

package infra

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/relux-works/skill-agents-management/pkg/inferenceengine"
	"github.com/relux-works/skill-agents-management/pkg/plugin"
	"github.com/relux-works/skill-agents-management/pkg/vendorplugin"
)

// Reviewer findings F1 and F2 (CR-TASK-260830-y6infr-1 revision 1), and F2 of
// CR-TASK-260830-y6infr-2 revision 2: the concrete SanitizedEngineObservationReader
// must perform a real bounded read against agents-infra's own Process-B broker,
// not a self-minted fixture, and the production consumer's Process-A launch must
// be exercised alongside a real (fake-backed) Process-B lifecycle rather than a
// marker file or a self-minted observation reader substituted right before the
// production call. This test drives the real Registry/BuildLaunch/observation
// path with the real SharedRuntimeSanitizedEngineObservationReader bound to a
// genuinely attested (test-built, non-live-model) broker whose sanitized facts
// permit BuildLaunch, then runs BuildAndRunPiTurn with a fake Process A while an
// independent real Process-A-shaped shared-runtime lease is held concurrently
// with the pre-existing Process-B peer lease, proving Process-A activity and its
// own lease acquire/release lifecycle never disturb the independently-held peer
// lease or the runtime.
func TestSharedRuntimeEngineObservationReaderReadsRealBrokerAndProcessASpawnsNeverTouchProcessB(t *testing.T) {
	testRoot, err := os.MkdirTemp("/tmp", "x")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(testRoot) })
	project := filepath.Join(testRoot, "project")
	home := filepath.Join(testRoot, "home")
	cache := filepath.Join(home, "Library", "Caches")
	for _, directory := range []string{project, cache} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	runtimeExecutable := buildSharedFakeRuntime(t, testRoot)
	// The weight-artifact fact requires an absolute --model path; the readiness
	// check only requires the runtime's served id to equal the configured
	// `model` field, so both are set to the same absolute-path-shaped string.
	absoluteModelPath := filepath.Join(testRoot, "fixtures", "weights.safetensors")
	body := validPiProfileWithArgv(t, sharedTestProfileName, runtimeExecutable, port, []string{"serve", "--model", absoluteModelPath}, 8)
	body = strings.Replace(body, `model = "Model"`, fmt.Sprintf("model = %q", absoluteModelPath), 1)
	body += fmt.Sprintf(`
[agents.pi.profiles.%[1]q.runtime.sharing]
mode = "shared"
linger_seconds = 0
max_leases = 4
max_segment_bytes = 1048576
max_segments = 7
heartbeat_interval_seconds = 1
lease_stale_seconds = 5
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 40
resource_pressure_mode = "provider"

[agents.pi.profiles.%[1]q.runtime.sharing.resource_pressure]
observation_path = "/agents-infra/resources"
observation_timeout_milliseconds = 2000
pressure_threshold_bytes = 10_000_000_000
recovery_threshold_bytes = 8_000_000_000
eviction_grace_seconds = 0
pressure_action = "refuse-new-drain-idle"
unknown_action = "refuse-new"
busy_action = "observe"
`, sharedTestProfileName)
	writePiProjectConfig(t, project, body)

	resolved, err := resolveSharedProfile(project, home, cache, sharedTestProfileName)
	if err != nil {
		t.Fatal(err)
	}

	// Establish a real (fake-backed) Process B: a real broker process holding
	// a real lease against a real, independently launched runtime child.
	helper := startSharedLeaseHelper(t, project, home, "RUN-observation")
	t.Cleanup(func() {
		if helper.command.ProcessState == nil {
			_ = helper.command.Process.Kill()
			_ = helper.command.Wait()
		}
	})

	preStatus, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
	if err != nil {
		t.Fatal(err)
	}
	if preStatus.Broker.State != "serving" || preStatus.Runtime == nil || preStatus.Runtime.PID != helper.info.RuntimePID || len(preStatus.Leases) != 1 {
		t.Fatalf("Process B was not attested before the reader ran: %#v", preStatus)
	}

	// The real reader performs its own bounded read of the same attested
	// broker. The fake runtime now serves a genuine provider resource
	// observation, so a real (not fabricated) success is the correct outcome;
	// the reader must reach the live broker to produce it, and BuildLaunch
	// below must consume this exact reader rather than a self-minted one.
	reader := NewSharedRuntimeSanitizedEngineObservationReader(project, home, cache, resolved.Profile)
	engine := plugin.Ref{ID: "mlx", Kind: inferenceengine.Kind}
	directObservation, err := reader.ReadSanitizedEngineObservation(context.Background(), vendorplugin.EngineObservationQuery{
		Engine: engine, Runtime: vendorplugin.RuntimeID(sharedTestProfileName), Model: vendorplugin.ModelID("local-test-model"), Profile: sharedTestProfileName,
	})
	if err != nil {
		t.Fatalf("observation reader refused a genuinely measured broker-backed observation: %v", err)
	}
	if directObservation.Contract != SanitizedEngineObservationContract || len(directObservation.Facts) == 0 {
		t.Fatalf("observation reader returned an incomplete real observation: %#v", directObservation)
	}

	// Establish a second, independent, real shared-runtime lease against the
	// same broker, representing the lease a real (non-fake) Process A would
	// acquire through RunPi's shared-mode path. This is a genuine lease
	// acquired through the production client, not a fabricated marker.
	processARunID := "RUN-process-a"
	processAState, err := ResolvePiClientStatePaths(cache, resolved.Project, resolved.ProfileName, processARunID)
	if err != nil {
		t.Fatal(err)
	}
	if err := CreatePiStateTree(processAState); err != nil {
		t.Fatal(err)
	}
	processALock, err := AcquirePiProfileLock(processAState)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = processALock.Close() })
	processALease, err := acquireSharedRuntimeLease(resolved, processAState, processARunID, os.Environ(), nil, nil)
	if err != nil {
		t.Fatalf("Process-A-shaped shared-runtime lease acquisition failed: %v", err)
	}
	processALeaseMonitor := processALease.monitor()
	t.Cleanup(func() { processALease.close() })

	duringStatus, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
	if err != nil {
		t.Fatal(err)
	}
	if duringStatus.Broker.State != "serving" || duringStatus.Runtime == nil || duringStatus.Runtime.PID != helper.info.RuntimePID || len(duringStatus.Leases) != 2 {
		t.Fatalf("Process-A shared-runtime lease was not visible alongside the independently-held Process-B peer lease: %#v", duringStatus)
	}

	// Drive the real production consumer (fake Process A) concurrently, using
	// the exact production graph built from the real reader above. This must
	// never reach, signal, lease, or release the Process B (nor the
	// independent Process-A-shaped lease) this test is holding.
	resolvedTarget := ResolvedCanonicalTarget{
		Target:  ProjectTarget{Name: "qwen-infra", Vendor: "qwen", Environment: "pi", Model: "local-test-model", Profile: &resolved.ProfileName},
		Profile: &resolved.Profile,
	}
	graph, err := BuildPiPluginGraph(project, resolvedTarget, &fakeManagementStatusReader{}, reader)
	if err != nil {
		t.Fatalf("BuildPiPluginGraph: %v", err)
	}

	dir, env := fakeProcessA(t, "printf '%s' '"+okTurnDocument+"'\nexit 0\n")
	request := graph.SpawnRequest([]byte("prompt"), dir, env)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := BuildAndRunPiTurn(ctx, graph.Registry, request, OSProcessATurnRunner{})
	if err != nil || result.FinalText != "accepted" {
		t.Fatalf("BuildAndRunPiTurn = %#v, %v", result, err)
	}

	// Also drive a cancelled turn through the same real production consumer
	// while Process B and the independent Process-A-shaped lease are still
	// held, then assert both survived untouched: broker still serving, same
	// runtime PID, same lease count.
	cancelDir, cancelEnv := fakeProcessA(t, "sleep 30\n")
	cancelRequest := graph.SpawnRequest([]byte("prompt"), cancelDir, cancelEnv)
	cancelCtx, cancelNow := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancelNow()
	}()
	if _, err := BuildAndRunPiTurn(cancelCtx, graph.Registry, cancelRequest, OSProcessATurnRunner{}); err == nil {
		t.Fatal("cancelled turn reported success")
	}

	postRunStatus, err := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
	if err != nil {
		t.Fatal(err)
	}
	if postRunStatus.Broker.State != "serving" || postRunStatus.Runtime == nil || postRunStatus.Runtime.PID != helper.info.RuntimePID || len(postRunStatus.Leases) != 2 {
		t.Fatalf("Process B or the independent Process-A-shaped lease was disturbed by Process-A activity: before=%#v after=%#v", duringStatus, postRunStatus)
	}

	// Release the Process-A-shaped lease and prove only it goes away: the
	// Process-B peer lease and runtime remain exactly as before. Closing races
	// the client's own connection teardown against its monitor goroutine (the
	// same client-initiated-close race the existing subprocess lease helper
	// avoids by never inspecting its monitor channel after a deliberate
	// release), so the release is confirmed through broker status instead.
	processALease.close()
	<-processALeaseMonitor
	waitForSharedTest(t, 8*time.Second, func() bool {
		status, statusErr := SharedRuntimeStatusReport(SharedRuntimeOperatorOptions{ProjectDir: project, HomeDir: home, CacheRoot: cache, Profile: sharedTestProfileName})
		return statusErr == nil && status.Broker.State == "serving" && status.Runtime != nil && status.Runtime.PID == helper.info.RuntimePID && len(status.Leases) == 1
	}, "releasing the Process-A-shaped lease did not leave only the Process-B peer lease")

	helper.release(t)
}
