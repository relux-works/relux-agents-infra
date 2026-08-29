package probe6

import "testing"

type p21Frame struct {
	Schema      string `json:"schema"`
	Protocol    int    `json:"protocol_version"`
	RuntimeKey  string `json:"runtime_key"`
	LauncherPid int    `json:"launcher_pid"`
	PlanDigest  string `json:"exec_plan_digest"`
}

func TestExactDecoderRejectsSameValueDuplicate(t *testing.T) {
	raw := []byte(`{"schema":"agents-infra.pi.shared-runtime.auth.v1","protocol_version":6,"protocol_version":6,"runtime_key":"key","launcher_pid":42,"exec_plan_digest":"digest"}`)
	_, fault := p21DecodeFrame(raw, "")
	if fault == nil || fault.reason != "frame_duplicate_field" || fault.field != "protocol_version" {
		t.Fatalf("got %#v", fault)
	}
	t.Logf("current probe decoder refuses the new attack: %s/%s", fault.reason, fault.field)
}

func TestExactDecoderRejectsAnotherUnknownName(t *testing.T) {
	raw := []byte(`{"schema":"agents-infra.pi.shared-runtime.auth.v1","protocol_version":6,"runtime_key":"key","launcher_pid":42,"exec_plan_digest":"digest","future_extension":"ignored"}`)
	_, fault := p21DecodeFrame(raw, "")
	if fault == nil || fault.reason != "frame_unknown_field" || fault.field != "future_extension" {
		t.Fatalf("got %#v", fault)
	}
	t.Logf("current probe decoder refuses the new attack: %s/%s", fault.reason, fault.field)
}
