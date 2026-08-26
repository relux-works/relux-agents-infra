//go:build !windows

package infra

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const (
	SharedRuntimeProtocolVersion  = 6
	sharedRuntimeAuthSchema       = "agents-infra.pi.shared-runtime.auth.v1"
	sharedRuntimeProfileSchema    = "agents-infra.pi.shared-runtime.profile.v1"
	sharedRuntimeKeySchema        = "agents-infra.pi.shared-runtime.v1"
	sharedRuntimeExitElectionLost = 75
	sharedRuntimeExitListenerBusy = 76
	sharedRuntimeMaxFrameBytes    = 65536
)

var sharedRuntimeAuthFields = [...]string{
	"schema",
	"protocol_version",
	"runtime_key",
	"launcher_pid",
	"exec_plan_digest",
}

type SharedRuntimeError struct {
	Code          string         `json:"code"`
	Reason        string         `json:"reason,omitempty"`
	MismatchField string         `json:"mismatch_field,omitempty"`
	Details       map[string]any `json:"details,omitempty"`
	Err           error          `json:"-"`
}

func (e *SharedRuntimeError) Error() string {
	payload := struct {
		Code          string         `json:"code"`
		Reason        string         `json:"reason,omitempty"`
		MismatchField string         `json:"mismatch_field,omitempty"`
		Details       map[string]any `json:"details,omitempty"`
		Message       string         `json:"message,omitempty"`
	}{Code: e.Code, Reason: e.Reason, MismatchField: e.MismatchField, Details: e.Details}
	if e.Err != nil {
		payload.Message = e.Err.Error()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return e.Code
	}
	return string(data)
}

func (e *SharedRuntimeError) Unwrap() error { return e.Err }

func sharedRuntimeError(code string, err error) error {
	return &SharedRuntimeError{Code: code, Err: err}
}

func sharedRuntimeReason(code, reason string, err error) error {
	return &SharedRuntimeError{Code: code, Reason: reason, Err: err}
}

func sharedRuntimeMismatch(code, field string) error {
	return &SharedRuntimeError{Code: code, MismatchField: field, Err: fmt.Errorf("%s differs", field)}
}

func SharedRuntimeExitCode(err error) (int, bool) {
	var shared *SharedRuntimeError
	if !errors.As(err, &shared) {
		return 0, false
	}
	if shared.Code == "broker_election_lost" {
		return sharedRuntimeExitElectionLost, true
	}
	if shared.Code == "runtime_listener_occupied" {
		return sharedRuntimeExitListenerBusy, true
	}
	return 1, true
}

type sharedDigestBuilder struct {
	hash []byte
}

func (b *sharedDigestBuilder) add(name string, value []byte) {
	b.hash = append(b.hash, name...)
	b.hash = append(b.hash, 0)
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(value)))
	b.hash = append(b.hash, length[:]...)
	b.hash = append(b.hash, value...)
	b.hash = append(b.hash, 0x1e)
}

func (b *sharedDigestBuilder) addString(name, value string) { b.add(name, []byte(value)) }
func (b *sharedDigestBuilder) sum() string {
	sum := sha256.Sum256(b.hash)
	return hex.EncodeToString(sum[:])
}

func SharedRuntimeProfileDigest(profile PiProfile) string {
	var digest sharedDigestBuilder
	digest.addString("schema", sharedRuntimeProfileSchema)
	digest.addString("provider", profile.Provider)
	digest.addString("model", profile.Model)
	digest.addString("base_url", profile.BaseURL)
	digest.addString("api", profile.API)
	digest.addString("readiness_path", profile.Runtime.ReadinessPath)
	digest.addString("reasoning", strconv.FormatBool(profile.Reasoning))
	digest.addString("thinking", profile.Thinking)
	digest.addString("context_window", strconv.Itoa(profile.ContextWindow))
	digest.addString("max_tokens", strconv.Itoa(profile.MaxTokens))
	for i, value := range profile.Input {
		digest.addString(fmt.Sprintf("input[%d]", i), value)
	}
	compat := map[string]string{}
	if value := profile.Compat.SupportsDeveloperRole; value != nil {
		compat["supports_developer_role"] = strconv.FormatBool(*value)
	}
	if value := profile.Compat.SupportsReasoningEffort; value != nil {
		compat["supports_reasoning_effort"] = strconv.FormatBool(*value)
	}
	if value := profile.Compat.SupportsUsageStreaming; value != nil {
		compat["supports_usage_in_streaming"] = strconv.FormatBool(*value)
	}
	if value := profile.Compat.SupportsFinishReason; value != nil {
		compat["supports_finish_reason"] = strconv.FormatBool(*value)
	}
	if value := profile.Compat.MaxTokensField; value != nil {
		compat["max_tokens_field"] = *value
	}
	if value := profile.Compat.ThinkingFormat; value != nil {
		compat["thinking_format"] = *value
	}
	compatNames := make([]string, 0, len(compat))
	for name := range compat {
		compatNames = append(compatNames, name)
	}
	sort.Strings(compatNames)
	for _, name := range compatNames {
		digest.addString("compat."+name, compat[name])
	}
	digest.addString("runtime.executable", profile.Runtime.Executable)
	digest.addString("runtime.argc", strconv.Itoa(len(profile.Runtime.Argv)))
	for i, value := range profile.Runtime.Argv {
		digest.addString(fmt.Sprintf("runtime.argv[%d]", i), value)
	}
	digest.addString("runtime.startup_timeout_seconds", strconv.Itoa(profile.Runtime.StartupTimeoutSeconds))
	digest.addString("runtime.shutdown_timeout_seconds", strconv.Itoa(profile.Runtime.ShutdownTimeoutSeconds))
	if dflash := profile.Runtime.DFlash; dflash != nil {
		digest.addString("runtime.dflash.target_model", dflash.TargetModel)
		digest.addString("runtime.dflash.draft_model", dflash.DraftModel)
		for i, value := range dflash.TargetArgv {
			digest.addString(fmt.Sprintf("runtime.dflash.target_argv[%d]", i), value)
		}
		for i, value := range dflash.DraftArgv {
			digest.addString(fmt.Sprintf("runtime.dflash.draft_argv[%d]", i), value)
		}
	}
	return digest.sum()
}

func SharedRuntimeKey(profile PiProfile) (string, string) {
	profileDigest := SharedRuntimeProfileDigest(profile)
	sum := sha256.Sum256([]byte(sharedRuntimeKeySchema + "\x00" + profile.BaseURL + "\x00" + profileDigest))
	return hex.EncodeToString(sum[:]), profileDigest
}

func SharedRuntimeExecPlanDigest(profile PiProfile, cwd string) string {
	var digest sharedDigestBuilder
	digest.addString("runtime.executable", profile.Runtime.Executable)
	digest.addString("runtime.argc", strconv.Itoa(len(profile.Runtime.Argv)))
	for i, value := range profile.Runtime.Argv {
		digest.addString(fmt.Sprintf("runtime.argv[%d]", i), value)
	}
	digest.addString("runtime.cwd", cwd)
	digest.addString("runtime.startup_timeout_seconds", strconv.Itoa(profile.Runtime.StartupTimeoutSeconds))
	return digest.sum()
}

type SharedRuntimePaths struct {
	CanonicalCacheRoot string `json:"canonical_cache_root"`
	RuntimeKey         string `json:"runtime_key"`
	Root               string `json:"root"`
	BrokerLock         string `json:"broker_lock"`
	BrokerState        string `json:"broker_state"`
	BrokerLog          string `json:"broker_log"`
	RuntimeLog         string `json:"runtime_log"`
	LeasesDir          string `json:"leases_dir"`
	RuntimeCWD         string `json:"runtime_cwd"`
	RendezvousDir      string `json:"rendezvous_dir"`
	RendezvousSocket   string `json:"rendezvous_socket"`
}

func ResolveSharedRuntimePaths(cacheRoot, runtimeKey string) (SharedRuntimePaths, error) {
	if !piStateKeyPattern.MatchString(runtimeKey) {
		return SharedRuntimePaths{}, sharedRuntimeError("shared_runtime_state_path_invalid", errors.New("runtime key is not lowercase SHA-256 hex"))
	}
	if cacheRoot == "" {
		var err error
		cacheRoot, err = os.UserCacheDir()
		if err != nil {
			return SharedRuntimePaths{}, sharedRuntimeError("shared_runtime_state_path_invalid", err)
		}
	}
	abs, err := filepath.Abs(cacheRoot)
	if err != nil {
		return SharedRuntimePaths{}, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	canonical, err := filepath.EvalSymlinks(filepath.Clean(abs))
	if err != nil {
		return SharedRuntimePaths{}, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		if err == nil {
			err = errors.New("cache root is not a directory")
		}
		return SharedRuntimePaths{}, sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	root := filepath.Join(canonical, "agents-infra", "pi-runtimes", runtimeKey)
	rv := filepath.Join(canonical, "agents-infra", "pi-rv", runtimeKey[:32])
	paths := SharedRuntimePaths{
		CanonicalCacheRoot: canonical,
		RuntimeKey:         runtimeKey,
		Root:               root,
		BrokerLock:         filepath.Join(root, "broker.lock"),
		BrokerState:        filepath.Join(root, "broker-state.json"),
		BrokerLog:          filepath.Join(root, "broker.log"),
		RuntimeLog:         filepath.Join(root, "runtime.log"),
		LeasesDir:          filepath.Join(root, "leases"),
		RuntimeCWD:         filepath.Join(root, "cwd"),
		RendezvousDir:      rv,
		RendezvousSocket:   filepath.Join(rv, "b.sock"),
	}
	for _, path := range []string{root, rv} {
		rel, relErr := filepath.Rel(canonical, path)
		if relErr != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
			return SharedRuntimePaths{}, sharedRuntimeError("shared_runtime_state_path_invalid", errors.New("managed runtime path escaped cache root"))
		}
	}
	if len(paths.RendezvousSocket) > 103 {
		return SharedRuntimePaths{}, sharedRuntimeError("broker_rendezvous_path_too_long", fmt.Errorf("rendezvous path is %d bytes", len(paths.RendezvousSocket)))
	}
	return paths, nil
}

func CreateSharedRuntimeTree(paths SharedRuntimePaths) error {
	rootFD, err := unix.Open(paths.CanonicalCacheRoot, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	defer unix.Close(rootFD)
	create := func(components ...string) error {
		fd := rootFD
		for _, component := range components {
			next, openErr := openOrCreateDirAt(fd, component)
			if fd != rootFD {
				unix.Close(fd)
			}
			if openErr != nil {
				return openErr
			}
			fd = next
		}
		if fd != rootFD {
			unix.Close(fd)
		}
		return nil
	}
	if err := create("agents-infra", "pi-runtimes", paths.RuntimeKey, "leases"); err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	if err := create("agents-infra", "pi-runtimes", paths.RuntimeKey, "cwd"); err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	if err := create("agents-infra", "pi-rv", paths.RuntimeKey[:32]); err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	return nil
}

type ProcessStartTime struct {
	Seconds      int64 `json:"seconds"`
	Microseconds int32 `json:"microseconds"`
}

type FileIdentity struct {
	Dev  uint64      `json:"dev"`
	Ino  uint64      `json:"ino"`
	Size int64       `json:"size"`
	Mode os.FileMode `json:"mode"`
}

type ProcessExecShape struct {
	ExecPath string   `json:"exec_path"`
	Argv     []string `json:"argv"`
}

type SharedRuntimeProcessRecord struct {
	PID                int              `json:"pid"`
	PGID               int              `json:"pgid"`
	StartTime          ProcessStartTime `json:"start_time"`
	UID                uint32           `json:"uid"`
	PreExec            ProcessExecShape `json:"pre_exec"`
	PostExec           ProcessExecShape `json:"post_exec"`
	CWD                string           `json:"cwd"`
	Endpoint           string           `json:"endpoint"`
	ExecutableIdentity FileIdentity     `json:"executable_identity"`
	ExecPlanDigest     string           `json:"exec_plan_digest"`
	Stage              string           `json:"stage"`
}

type SharedBrokerIdentity struct {
	PID                int              `json:"pid"`
	PGID               int              `json:"pgid"`
	SID                int              `json:"sid"`
	StartTime          ProcessStartTime `json:"start_time"`
	UID                uint32           `json:"uid"`
	ExecPath           string           `json:"exec_path"`
	Argv               []string         `json:"argv"`
	ExecutableIdentity FileIdentity     `json:"executable_identity"`
}

type SharedBrokerRecord struct {
	Stage             string                      `json:"stage"`
	State             string                      `json:"state"`
	ProtocolVersion   int                         `json:"protocol_version"`
	Broker            SharedBrokerIdentity        `json:"broker"`
	RuntimeKeyClaimed string                      `json:"runtime_key_claimed"`
	RuntimeKey        string                      `json:"runtime_key,omitempty"`
	ProfileDigest     string                      `json:"profile_digest,omitempty"`
	Endpoint          string                      `json:"endpoint,omitempty"`
	Sharing           *PiRuntimeSharing           `json:"sharing,omitempty"`
	Runtime           *SharedRuntimeProcessRecord `json:"runtime,omitempty"`
	ReadyAt           *time.Time                  `json:"ready_at,omitempty"`
}

type SharedLeaseRecord struct {
	LeaseID         string           `json:"lease_id"`
	RunID           string           `json:"run_id,omitempty"`
	ClientKey       string           `json:"client_key"`
	ClientPID       int              `json:"client_pid"`
	ClientStartTime ProcessStartTime `json:"client_start_time"`
	RuntimeKey      string           `json:"runtime_key"`
	ProfileDigest   string           `json:"profile_digest"`
	Endpoint        string           `json:"endpoint"`
	RuntimePID      int              `json:"runtime_pid"`
	GrantedAt       time.Time        `json:"granted_at"`
	LastHeartbeatAt time.Time        `json:"last_heartbeat_at"`
}

func readSharedBrokerRecord(path string) (*SharedBrokerRecord, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, sharedRuntimeError("shared_runtime_state_unreadable", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var record SharedBrokerRecord
	if err := decoder.Decode(&record); err != nil {
		return nil, false, sharedRuntimeError("shared_runtime_state_unreadable", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, false, sharedRuntimeError("shared_runtime_state_unreadable", err)
	}
	return &record, true, nil
}

func writeSharedJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	dirFD, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	defer unix.Close(dirFD)
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	temp := "." + base + ".tmp-" + strconv.Itoa(os.Getpid()) + "-" + hex.EncodeToString(nonce[:])
	fd, err := unix.Openat(dirFD, temp, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	file := os.NewFile(uintptr(fd), temp)
	writeErr := error(nil)
	if n, err := file.Write(data); err != nil || n != len(data) {
		if err == nil {
			err = io.ErrShortWrite
		}
		writeErr = err
	}
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if closeErr := file.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = unix.Unlinkat(dirFD, temp, 0)
		return sharedRuntimeError("shared_runtime_state_path_invalid", writeErr)
	}
	if err := unix.Renameat(dirFD, temp, dirFD, base); err != nil {
		_ = unix.Unlinkat(dirFD, temp, 0)
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	if err := unix.Fsync(dirFD); err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	return nil
}

func removeSharedLeaseMirrors(directory string) error {
	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return sharedRuntimeError("shared_runtime_state_path_invalid", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || filepath.Ext(entry.Name()) != ".json" {
			return sharedRuntimeError("shared_runtime_state_path_invalid", fmt.Errorf("unexpected lease mirror entry %q", entry.Name()))
		}
		if err := os.Remove(filepath.Join(directory, entry.Name())); err != nil {
			return sharedRuntimeError("shared_runtime_state_path_invalid", err)
		}
	}
	return nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("content follows JSON object")
		}
		return err
	}
	return nil
}

type sharedRuntimeAuthorizationFrame struct {
	Schema          string `json:"schema"`
	ProtocolVersion int    `json:"protocol_version"`
	RuntimeKey      string `json:"runtime_key"`
	LauncherPID     int    `json:"launcher_pid"`
	ExecPlanDigest  string `json:"exec_plan_digest"`
}

type sharedAuthDecodeEvidence struct {
	DecodedKeys      []string `json:"decoded_keys"`
	ComparedFields   []string `json:"compared_fields,omitempty"`
	Decision         string   `json:"decision"`
	DecisionCallSite string   `json:"decision_call_site"`
	ConstantFieldSet []string `json:"constant_field_set"`
}

type sharedAuthShapeVerdict struct {
	Accepted      bool
	Reason        string
	MismatchField string
}

type sharedAuthFrameMember struct {
	WireName    string
	DecodedName string
	Value       json.RawMessage
}

type sharedAuthShapeInput struct {
	Members          []sharedAuthFrameMember
	StructuralReason string
	StructuralErr    error
	CompleteObject   bool
}

var sharedAuthorizationShapeDecision = sharedAuthCompiledShapeDecision

func decodeSharedRuntimeAuthorizationFrame(data []byte) (sharedRuntimeAuthorizationFrame, sharedAuthDecodeEvidence, error) {
	frame, members, completeObject, structuralReason, structuralErr := tokenizeSharedAuthorizationFrame(data)
	keys := sharedAuthDecodedMemberNames(members)
	evidence := sharedAuthDecodeEvidence{
		DecodedKeys:      append([]string(nil), keys...),
		DecisionCallSite: "decodeSharedRuntimeAuthorizationFrame",
		ConstantFieldSet: append([]string(nil), sharedRuntimeAuthFields[:]...),
	}
	verdict := sharedAuthorizationShapeDecision(sharedAuthShapeInput{
		Members:          append([]sharedAuthFrameMember(nil), members...),
		StructuralReason: structuralReason,
		StructuralErr:    structuralErr,
		CompleteObject:   completeObject,
	})
	if !verdict.Accepted {
		evidence.Decision = "refuse"
		var decisionErr error = errors.New("authorization frame shape differs from the selected decision")
		if structuralErr != nil && verdict.Reason == structuralReason {
			decisionErr = structuralErr
		}
		return frame, evidence, &SharedRuntimeError{Code: "protocol_violation", Reason: verdict.Reason, MismatchField: verdict.MismatchField, Err: decisionErr}
	}
	evidence.Decision = "admit"
	return frame, evidence, nil
}

func sharedAuthCompiledShapeDecision(input sharedAuthShapeInput) sharedAuthShapeVerdict {
	if input.StructuralErr != nil {
		return sharedAuthShapeVerdict{Reason: input.StructuralReason}
	}
	return sharedAuthMultisetVerdict(sharedAuthDecodedMemberNames(input.Members))
}

func sharedAuthDecodedMemberNames(members []sharedAuthFrameMember) []string {
	result := make([]string, 0, len(members))
	for _, member := range members {
		result = append(result, member.DecodedName)
	}
	return result
}

func tokenizeSharedAuthorizationFrame(data []byte) (sharedRuntimeAuthorizationFrame, []sharedAuthFrameMember, bool, string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return sharedRuntimeAuthorizationFrame{}, nil, false, "frame_unparseable", err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return sharedRuntimeAuthorizationFrame{}, nil, false, "frame_not_single_object", errors.New("authorization frame is not a JSON object")
	}
	members := []sharedAuthFrameMember{}
	values := map[string]json.RawMessage{}
	for decoder.More() {
		wireStart := decoder.InputOffset()
		keyToken, err := decoder.Token()
		if err != nil {
			return sharedRuntimeAuthorizationFrame{}, members, false, "frame_unparseable", err
		}
		key, ok := keyToken.(string)
		if !ok {
			return sharedRuntimeAuthorizationFrame{}, members, false, "frame_unparseable", errors.New("object member name is not a string")
		}
		member := sharedAuthFrameMember{DecodedName: key}
		wireName, err := sharedAuthorizationMemberWireName(data, wireStart, decoder.InputOffset())
		if err != nil {
			members = append(members, member)
			return sharedRuntimeAuthorizationFrame{}, members, false, "frame_unparseable", err
		}
		member.WireName = wireName
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			members = append(members, member)
			return sharedRuntimeAuthorizationFrame{}, members, false, "frame_unparseable", err
		}
		member.Value = append(json.RawMessage(nil), raw...)
		members = append(members, member)
		values[key] = append(json.RawMessage(nil), raw...)
	}
	closing, err := decoder.Token()
	if err != nil {
		return sharedRuntimeAuthorizationFrame{}, members, false, "frame_unparseable", err
	}
	if closing != json.Delim('}') {
		return sharedRuntimeAuthorizationFrame{}, members, false, "frame_unparseable", errors.New("authorization object did not close")
	}
	var frame sharedRuntimeAuthorizationFrame
	decode := func(name string, target any) error {
		raw, ok := values[name]
		if !ok {
			return nil
		}
		return json.Unmarshal(raw, target)
	}
	if err := decode("schema", &frame.Schema); err != nil {
		return frame, members, true, "frame_unparseable", err
	}
	if err := decode("protocol_version", &frame.ProtocolVersion); err != nil {
		return frame, members, true, "frame_unparseable", err
	}
	if err := decode("runtime_key", &frame.RuntimeKey); err != nil {
		return frame, members, true, "frame_unparseable", err
	}
	if err := decode("launcher_pid", &frame.LauncherPID); err != nil {
		return frame, members, true, "frame_unparseable", err
	}
	if err := decode("exec_plan_digest", &frame.ExecPlanDigest); err != nil {
		return frame, members, true, "frame_unparseable", err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return frame, members, true, "frame_not_single_object", errors.New("content follows authorization object")
	}
	return frame, members, true, "", nil
}

func sharedAuthorizationMemberWireName(data []byte, start, end int64) (string, error) {
	if start < 0 || end < start || end > int64(len(data)) {
		return "", errors.New("authorization member offset is out of bounds")
	}
	wire := bytes.TrimSpace(data[start:end])
	if len(wire) > 0 && wire[0] == ',' {
		wire = bytes.TrimSpace(wire[1:])
	}
	var decoded string
	if len(wire) == 0 || json.Unmarshal(wire, &decoded) != nil {
		return "", errors.New("authorization member wire name is not one JSON string")
	}
	return string(wire), nil
}

func sharedAuthMultisetVerdict(keys []string) sharedAuthShapeVerdict {
	counts := make(map[string]int, len(keys))
	for _, key := range keys {
		counts[key]++
	}
	want := make(map[string]int, len(sharedRuntimeAuthFields))
	for _, key := range sharedRuntimeAuthFields {
		want[key] = 1
	}
	equal := len(counts) == len(want)
	if equal {
		for key, count := range want {
			if counts[key] != count {
				equal = false
				break
			}
		}
	}
	if equal && len(keys) == len(sharedRuntimeAuthFields) {
		return sharedAuthShapeVerdict{Accepted: true}
	}
	for _, key := range keys {
		if _, ok := want[key]; !ok {
			return sharedAuthShapeVerdict{Reason: "frame_unknown_field", MismatchField: key}
		}
	}
	seen := map[string]bool{}
	for _, key := range keys {
		if seen[key] {
			return sharedAuthShapeVerdict{Reason: "frame_duplicate_field", MismatchField: key}
		}
		seen[key] = true
	}
	for _, key := range sharedRuntimeAuthFields {
		if counts[key] == 0 {
			return sharedAuthShapeVerdict{Reason: "frame_missing_field", MismatchField: key}
		}
	}
	return sharedAuthShapeVerdict{Reason: "frame_unparseable"}
}

func writeAuthorizationFrame(writer io.Writer, frame sharedRuntimeAuthorizationFrame) error {
	// This writer is deliberately sourced from the same compiled field order the
	// decoder compares its decoded multiset against.
	var body bytes.Buffer
	body.WriteByte('{')
	values := []any{frame.Schema, frame.ProtocolVersion, frame.RuntimeKey, frame.LauncherPID, frame.ExecPlanDigest}
	for i, name := range sharedRuntimeAuthFields {
		if i > 0 {
			body.WriteByte(',')
		}
		encodedName, _ := json.Marshal(name)
		encodedValue, err := json.Marshal(values[i])
		if err != nil {
			return err
		}
		body.Write(encodedName)
		body.WriteByte(':')
		body.Write(encodedValue)
	}
	body.WriteString("}\n")
	_, err := writer.Write(body.Bytes())
	return err
}

func fileIdentity(path string) (FileIdentity, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileIdentity{}, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return FileIdentity{}, errors.New("stat identity unavailable")
	}
	return FileIdentity{Dev: uint64(stat.Dev), Ino: uint64(stat.Ino), Size: info.Size(), Mode: info.Mode()}, nil
}
