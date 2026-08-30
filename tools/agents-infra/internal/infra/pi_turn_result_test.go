package infra

import (
	"errors"
	"os"
	"strings"
	"testing"

	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
)

const validPiTurnJSONL = `{"type":"session","version":3,"id":"fixture","timestamp":"2026-08-30T00:00:00Z","cwd":"/repo"}
{"type":"agent_start"}
{"type":"turn_start"}
{"type":"message_start","message":{"role":"assistant","content":[]}}
{"type":"message_update","usage":{},"assistantMessageEvent":{"type":"text_delta","contentIndex":0,"delta":"accepted"}}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"accepted"}]}}
{"type":"turn_end","message":{"role":"assistant","content":[]},"toolResults":[]}
{"type":"agent_end","messages":[]}
`

func TestPinnedPiGrammarDrivesSchemaOneTranslation(t *testing.T) {
	manifest, err := os.ReadFile("pi-v0.84.2-darwin-arm64-tree-manifest.txt")
	if err != nil {
		t.Fatal(err)
	}
	excerpt, err := os.ReadFile("testdata/pi-v0.84.2-json-grammar-excerpt.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(manifest), "9c324127fac36eadc781c5222937f3a2b938a5fd671976aab020b27d7c1362a7  ./docs/json.md") ||
		!strings.Contains(string(excerpt), "message_end` contains the final authoritative message") {
		t.Fatal("content-addressed Pi v0.84.2 grammar evidence drifted")
	}
	text, toolFailed, err := parsePiTurnJSONL([]byte(validPiTurnJSONL), false)
	if err != nil || toolFailed || text != "accepted" {
		t.Fatalf("parsePiTurnJSONL = %q, %v, %v", text, toolFailed, err)
	}
}

func TestPiTurnTranslatorRefusesNarrowedProtocolAttacks(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"wrong session version", strings.Replace(validPiTurnJSONL, `"version":3`, `"version":2`, 1)},
		{"duplicate type", strings.Replace(validPiTurnJSONL, `{"type":"agent_start"}`, `{"type":"agent_start","type":"agent_start"}`, 1)},
		{"unknown event", strings.Replace(validPiTurnJSONL, `"turn_start"`, `"invented"`, 1)},
		{"missing agent end", strings.Replace(validPiTurnJSONL, "{\"type\":\"agent_end\",\"messages\":[]}\n", "", 1)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := parsePiTurnJSONL([]byte(test.raw), false); err == nil {
				t.Fatal("protocol attack was admitted")
			}
		})
	}
	if _, _, err := parsePiTurnJSONL([]byte(validPiTurnJSONL), true); err == nil {
		t.Fatal("bounded-capture truncation was admitted")
	}
}

// Reviewer finding F3: the translator must require the final authoritative
// assistant message and must reject a tool lifecycle event outside the
// active agent_start/agent_end window, even when the rest of the stream is
// otherwise well-formed.
func TestPiTurnTranslatorRefusesMissingAuthoritativeMessageAndPostAgentTools(t *testing.T) {
	missingAssistantMessage := `{"type":"session","version":3,"id":"fixture","timestamp":"2026-08-30T00:00:00Z","cwd":"/repo"}
{"type":"agent_start"}
{"type":"turn_start"}
{"type":"turn_end","message":{"role":"assistant","content":[]},"toolResults":[]}
{"type":"agent_end","messages":[]}
`
	if _, _, err := parsePiTurnJSONL([]byte(missingAssistantMessage), false); err == nil {
		t.Fatal("missing authoritative assistant message was admitted as success")
	}

	postAgentTool := validPiTurnJSONL + `{"type":"tool_execution_start","toolCallId":"t1","toolName":"read","args":{}}
{"type":"tool_execution_end","toolCallId":"t1","toolName":"read","result":{},"isError":false}
`
	if _, _, err := parsePiTurnJSONL([]byte(postAgentTool), false); err == nil {
		t.Fatal("post-agent_end tool lifecycle was admitted as success")
	}

	postAgentToolUpdate := validPiTurnJSONL + `{"type":"tool_execution_update","toolCallId":"t1"}
`
	if _, _, err := parsePiTurnJSONL([]byte(postAgentToolUpdate), false); err == nil {
		t.Fatal("post-agent_end tool update was admitted as success")
	}
}

// Reviewer finding F1 (CR-TASK-260830-y6infr-2 revision 2): the translator
// tracked only the outer agent lifecycle plus "some assistant message_end
// exists". It did not track whether a turn was open, whether a message was
// started, or whether a second turn_start arrived while the first turn was
// still open, so a schema-1 success could be minted from an incomplete or
// contradictory raw Pi event stream. Every case here drives the real
// parsePiTurnJSONL production entry point directly with a single removed,
// duplicated, or reordered lifecycle event against the otherwise-valid fixture.
func TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations(t *testing.T) {
	removeLine := func(source, line string) string {
		replaced := strings.Replace(source, line, "", 1)
		if replaced == source {
			t.Fatalf("fixture line %q not found for removal", line)
		}
		return replaced
	}
	firstTurnWithoutAgentEnd := removeLine(validPiTurnJSONL, "{\"type\":\"agent_end\",\"messages\":[]}\n")
	tests := []struct {
		name string
		raw  string
	}{
		{
			// Reviewer attack 1: missing turn_start. message_start/message_end
			// must never be admitted without an open turn.
			name: "missing turn_start",
			raw:  removeLine(validPiTurnJSONL, "{\"type\":\"turn_start\"}\n"),
		},
		{
			// Reviewer attack 2: missing message_start. message_end must never
			// be admitted without a preceding message_start.
			name: "missing message_start",
			raw:  removeLine(validPiTurnJSONL, "{\"type\":\"message_start\",\"message\":{\"role\":\"assistant\",\"content\":[]}}\n"),
		},
		{
			// Reviewer attack 3: duplicate turn_start. A second turn_start
			// while the first turn is still open must never be admitted.
			name: "duplicate turn_start",
			raw:  strings.Replace(validPiTurnJSONL, "{\"type\":\"turn_start\"}\n", "{\"type\":\"turn_start\"}\n{\"type\":\"turn_start\"}\n", 1),
		},
		{
			// turn_end must never be admitted without an open turn: a second,
			// duplicate turn_end right after the first (already-closed) one
			// must be refused even though the rest of the stream is valid.
			name: "turn_end without turn_start",
			raw: strings.Replace(validPiTurnJSONL,
				"{\"type\":\"turn_end\",\"message\":{\"role\":\"assistant\",\"content\":[]},\"toolResults\":[]}\n",
				"{\"type\":\"turn_end\",\"message\":{\"role\":\"assistant\",\"content\":[]},\"toolResults\":[]}\n{\"type\":\"turn_end\",\"message\":{\"role\":\"assistant\",\"content\":[]},\"toolResults\":[]}\n", 1),
		},
		{
			// A second message_start while the first message is still open
			// must never be admitted.
			name: "duplicate message_start",
			raw: strings.Replace(validPiTurnJSONL,
				"{\"type\":\"message_start\",\"message\":{\"role\":\"assistant\",\"content\":[]}}\n",
				"{\"type\":\"message_start\",\"message\":{\"role\":\"assistant\",\"content\":[]}}\n{\"type\":\"message_start\",\"message\":{\"role\":\"assistant\",\"content\":[]}}\n", 1),
		},
		{
			// turn_end must never be admitted while the message it should have
			// closed first is still open. The first turn completes normally
			// inside the same still-open agent lifecycle (so the authoritative
			// -assistant-message requirement is already satisfied); the second
			// turn's message is left open when its turn_end fires.
			name: "turn_end while message open",
			raw: firstTurnWithoutAgentEnd + `{"type":"turn_start"}
{"type":"message_start","message":{"role":"assistant","content":[]}}
{"type":"message_update","usage":{},"assistantMessageEvent":{"type":"text_delta","contentIndex":0,"delta":"accepted"}}
{"type":"turn_end","message":{"role":"assistant","content":[]},"toolResults":[]}
{"type":"agent_end","messages":[]}
`,
		},
		{
			// agent_end must never be admitted while a turn is still open. The
			// first turn completes normally; the second turn's message closes
			// properly, but its turn is never closed before agent_end fires.
			name: "agent_end while turn open",
			raw: firstTurnWithoutAgentEnd + `{"type":"turn_start"}
{"type":"message_start","message":{"role":"assistant","content":[]}}
{"type":"message_update","usage":{},"assistantMessageEvent":{"type":"text_delta","contentIndex":0,"delta":"accepted"}}
{"type":"message_end","message":{"role":"assistant","content":[{"type":"text","text":"accepted"}]}}
{"type":"agent_end","messages":[]}
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := parsePiTurnJSONL([]byte(test.raw), false); err == nil {
				t.Fatalf("turn/message lifecycle attack %q was admitted as success", test.name)
			}
		})
	}
}

func TestPiTurnTranslatorClassPrecedence(t *testing.T) {
	code, _ := classifyPiTurn(errors.New("child failed"), PiRunReport{Managed: true, CleanupConfirmed: false}, []byte(validPiTurnJSONL), false)
	if code != managementpi.TurnCodeCleanupFailed {
		t.Fatalf("cleanup precedence = %q", code)
	}
	toolFailure := strings.Replace(validPiTurnJSONL,
		`{"type":"turn_end"`,
		"{\"type\":\"tool_execution_start\",\"toolCallId\":\"t1\",\"toolName\":\"read\",\"args\":{}}\n{\"type\":\"tool_execution_end\",\"toolCallId\":\"t1\",\"toolName\":\"read\",\"result\":{},\"isError\":true}\n{\"type\":\"turn_end\"", 1)
	code, _ = classifyPiTurn(nil, PiRunReport{}, []byte(toolFailure), false)
	if code != managementpi.TurnCodeToolFailed {
		t.Fatalf("tool failure class = %q", code)
	}
}
