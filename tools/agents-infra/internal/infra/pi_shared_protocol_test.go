//go:build !windows

package infra

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSharedRuntimeListenerBusyHasNominalBrokerExitCode(t *testing.T) {
	code, ok := SharedRuntimeExitCode(sharedRuntimeError("runtime_listener_occupied", errors.New("busy")))
	if !ok || code != sharedRuntimeExitListenerBusy {
		t.Fatalf("listener-busy exit=(%d,%t) want=(%d,true)", code, ok, sharedRuntimeExitListenerBusy)
	}
}

func TestSharedRuntimeQuarantineHasNominalBrokerExitCode(t *testing.T) {
	code, ok := SharedRuntimeExitCode(sharedRuntimeError("shared_runtime_quarantined", errors.New("quarantined")))
	if !ok || code != sharedRuntimeExitQuarantined {
		t.Fatalf("quarantine exit=(%d,%t) want=(%d,true)", code, ok, sharedRuntimeExitQuarantined)
	}
}

func TestSharedRuntimeDigestsExcludeSharingAndCoverTheExecutionPlan(t *testing.T) {
	profile := mustParsedPiProfile(t, false)
	profile.Runtime.Sharing = &PiRuntimeSharing{Mode: "shared", LingerSeconds: 0, MaxLeases: 1, MaxSegmentBytes: 1024, MaxSegments: 2, HeartbeatIntervalSeconds: 1, LeaseStaleSeconds: 2, BrokerStartTimeoutSeconds: 40}
	key, digest := SharedRuntimeKey(profile)

	peer := profile
	peer.Runtime.Sharing = &PiRuntimeSharing{Mode: "shared", LingerSeconds: 300, MaxLeases: 16, MaxSegmentBytes: 2048, MaxSegments: 4, HeartbeatIntervalSeconds: 15, LeaseStaleSeconds: 60, BrokerStartTimeoutSeconds: 240}
	peer.Source = "/another/project/config.toml"
	peerKey, peerDigest := SharedRuntimeKey(peer)
	if key != peerKey || digest != peerDigest {
		t.Fatal("sharing policy or project provenance changed runtime identity")
	}

	mutated := peer
	mutated.Runtime.Argv = append([]string(nil), peer.Runtime.Argv...)
	mutated.Runtime.Argv[0] = "different"
	mutatedKey, mutatedDigest := SharedRuntimeKey(mutated)
	if mutatedKey == key || mutatedDigest == digest {
		t.Fatal("runtime argv mutation did not change profile and runtime digests")
	}
	if SharedRuntimeExecPlanDigest(profile, "/runtime/cwd") == SharedRuntimeExecPlanDigest(profile, "/other/cwd") {
		t.Fatal("runtime cwd is absent from exec plan digest")
	}
	absentProfileDigest := SharedRuntimeProfileDigest(profile)
	absentExecDigest := SharedRuntimeExecPlanDigest(profile, "/runtime/cwd")
	cacheBudget := int64(6_442_450_944)
	profile.CacheBudgetBytes = &cacheBudget
	lowerProfileDigest := SharedRuntimeProfileDigest(profile)
	lowerExecDigest := SharedRuntimeExecPlanDigest(profile, "/runtime/cwd")
	if lowerProfileDigest == absentProfileDigest || lowerExecDigest == absentExecDigest {
		t.Fatal("cache budget presence is absent from a shared-runtime digest")
	}
	higher := int64(12_884_901_888)
	profile.CacheBudgetBytes = &higher
	if SharedRuntimeProfileDigest(profile) == lowerProfileDigest || SharedRuntimeExecPlanDigest(profile, "/runtime/cwd") == lowerExecDigest {
		t.Fatal("cache budget value is absent from a shared-runtime digest")
	}
}

func TestSharedRuntimePathsAreHashOnlyContainedAndBounded(t *testing.T) {
	cache, err := os.MkdirTemp("/tmp", "agents-infra-shared-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(cache) })
	key := strings.Repeat("a", 64)
	paths, err := ResolveSharedRuntimePaths(cache, key)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(paths.Root) != key || filepath.Base(paths.RendezvousDir) != key[:32] || filepath.Base(paths.RendezvousSocket) != "b.sock" {
		t.Fatalf("unexpected shared paths: %#v", paths)
	}
	if err := CreateSharedRuntimeTree(paths); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.Root, paths.LeasesDir, paths.RuntimeCWD, paths.RendezvousDir} {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Fatalf("managed directory %s: info=%v err=%v", path, info, err)
		}
	}
	if _, err := ResolveSharedRuntimePaths(cache, "profile-name"); err == nil {
		t.Fatal("non-hash runtime key admitted")
	}
	longBase, err := os.MkdirTemp("/tmp", "agents-infra-long-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(longBase) })
	longCache := filepath.Join(longBase, strings.Repeat("x", 110))
	if err := os.Mkdir(longCache, 0o700); err != nil {
		t.Fatal(err)
	}
	_, err = ResolveSharedRuntimePaths(longCache, key)
	var sharedErr *SharedRuntimeError
	if !errors.As(err, &sharedErr) || sharedErr.Code != "broker_rendezvous_path_too_long" {
		t.Fatalf("long rendezvous refusal=%v", err)
	}
}

func validSharedAuthorizationFrame() sharedRuntimeAuthorizationFrame {
	return sharedRuntimeAuthorizationFrame{
		Schema:          sharedRuntimeAuthSchema,
		ProtocolVersion: SharedRuntimeProtocolVersion,
		RuntimeKey:      strings.Repeat("b", 64),
		LauncherPID:     123,
		ExecPlanDigest:  strings.Repeat("c", 64),
	}
}

func rawSharedAuthorizationFrame(t *testing.T, frame sharedRuntimeAuthorizationFrame) []byte {
	t.Helper()
	var body bytes.Buffer
	if err := writeAuthorizationFrame(&body, frame); err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(body.Bytes())
}

func TestSharedAuthorizationDecoderClosesTheDecodedMemberMultiset(t *testing.T) {
	frame := validSharedAuthorizationFrame()
	valid := rawSharedAuthorizationFrame(t, frame)
	escaped := bytes.ReplaceAll(valid, []byte(`"schema"`), []byte(`"\u0073chema"`))
	tests := []struct {
		name   string
		raw    []byte
		ok     bool
		reason string
		field  string
	}{
		{name: "valid", raw: valid, ok: true},
		{name: "escaped valid member", raw: escaped, ok: true},
		{name: "unknown", raw: bytes.Replace(valid, []byte(`}`), []byte(`,"future_extension":true}`), 1), reason: "frame_unknown_field", field: "future_extension"},
		{name: "duplicate same value", raw: bytes.Replace(valid, []byte(`{`), []byte(`{"schema":"`+sharedRuntimeAuthSchema+`",`), 1), reason: "frame_duplicate_field", field: "schema"},
		{name: "plain escaped duplicate", raw: bytes.Replace(valid, []byte(`{`), []byte(`{"\u0073chema":"`+sharedRuntimeAuthSchema+`",`), 1), reason: "frame_duplicate_field", field: "schema"},
		{name: "missing", raw: removeJSONMember(t, valid, "runtime_key"), reason: "frame_missing_field", field: "runtime_key"},
		{name: "second object", raw: append(append([]byte(nil), valid...), []byte(` {}`)...), reason: "frame_not_single_object"},
		{name: "array", raw: []byte(`[]`), reason: "frame_not_single_object"},
		{name: "truncated", raw: valid[:len(valid)-1], reason: "frame_unparseable"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, evidence, err := decodeSharedRuntimeAuthorizationFrame(testCase.raw)
			if testCase.ok {
				if err != nil || !reflect.DeepEqual(got, frame) || evidence.Decision != "admit" {
					t.Fatalf("valid decode got=%#v evidence=%#v err=%v", got, evidence, err)
				}
				return
			}
			var sharedErr *SharedRuntimeError
			if !errors.As(err, &sharedErr) || sharedErr.Code != "protocol_violation" || sharedErr.Reason != testCase.reason || sharedErr.MismatchField != testCase.field {
				t.Fatalf("refusal=%v want reason=%s field=%q evidence=%#v", err, testCase.reason, testCase.field, evidence)
			}
		})
	}
}

func removeJSONMember(t *testing.T, raw []byte, name string) []byte {
	t.Helper()
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	delete(values, name)
	out, err := json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
