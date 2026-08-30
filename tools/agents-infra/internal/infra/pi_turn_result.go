package infra

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	managementpi "github.com/relux-works/skill-agents-management/pkg/agentic/systems/pi"
)

const maxRawPiTurnBytes = 16 << 20

type PiTurnProcessAError struct {
	code managementpi.TurnResultCode
	exit int
}

func WritePiTurnRefusal(output io.Writer, code managementpi.TurnResultCode) error {
	document := piTurnResultDocument{
		Contract:      managementpi.TurnResultContract,
		SchemaVersion: managementpi.TurnResultSchemaVersion,
		Status:        "error",
		Error:         &piTurnResultError{Code: code},
	}
	if err := json.NewEncoder(output).Encode(document); err != nil {
		return &PiTurnProcessAError{code: managementpi.TurnCodeResultInvalid, exit: 3}
	}
	return &PiTurnProcessAError{code: code, exit: piTurnResultExit(code)}
}

func (e *PiTurnProcessAError) Error() string { return string(e.code) }
func (e *PiTurnProcessAError) ExitCode() int { return e.exit }

type piTurnResultDocument struct {
	Contract      string             `json:"contract"`
	SchemaVersion int                `json:"schema_version"`
	Status        string             `json:"status"`
	FinalText     string             `json:"final_text,omitempty"`
	Error         *piTurnResultError `json:"error,omitempty"`
}

type piTurnResultError struct {
	Code managementpi.TurnResultCode `json:"code"`
}

// RunPiTurnProcessA owns schema-1 production. Raw Pi JSONL is captured and
// translated here; it never crosses the Process-A stdout boundary.
func RunPiTurnProcessA(opts RunPiOptions, output io.Writer) error {
	if output == nil {
		return errors.New("agents-infra: Pi turn result output is required")
	}
	raw := newBoundedProcessAStdout(maxRawPiTurnBytes)
	report := PiRunReport{}
	opts.Stdout = raw
	opts.Stderr = io.Discard
	opts.Report = &report
	runErr := RunPi(opts)

	code, finalText := classifyPiTurn(runErr, report, raw.Bytes(), raw.Truncated())
	document := piTurnResultDocument{
		Contract:      managementpi.TurnResultContract,
		SchemaVersion: managementpi.TurnResultSchemaVersion,
	}
	exit := 0
	if code == "" {
		document.Status = "ok"
		document.FinalText = finalText
	} else {
		document.Status = "error"
		document.Error = &piTurnResultError{Code: code}
		exit = piTurnResultExit(code)
	}
	if err := json.NewEncoder(output).Encode(document); err != nil {
		return &PiTurnProcessAError{code: managementpi.TurnCodeResultInvalid, exit: 3}
	}
	if exit != 0 {
		return &PiTurnProcessAError{code: code, exit: exit}
	}
	return nil
}

func classifyPiTurn(runErr error, report PiRunReport, raw []byte, truncated bool) (managementpi.TurnResultCode, string) {
	if report.Managed && !report.CleanupConfirmed {
		return managementpi.TurnCodeCleanupFailed, ""
	}
	if errors.Is(runErr, context.Canceled) {
		return managementpi.TurnCodeCancelled, ""
	}
	if report.DeadlineExceeded || errors.Is(runErr, context.DeadlineExceeded) {
		return managementpi.TurnCodeDeadlineExceeded, ""
	}
	if runErr != nil {
		if code := piTurnPreChildCode(runErr); code != "" {
			return code, ""
		}
		return managementpi.TurnCodeChildFailed, ""
	}
	finalText, toolFailed, err := parsePiTurnJSONL(raw, truncated)
	if err != nil {
		return managementpi.TurnCodeResultInvalid, ""
	}
	if toolFailed {
		return managementpi.TurnCodeToolFailed, ""
	}
	return "", finalText
}

func piTurnPreChildCode(err error) managementpi.TurnResultCode {
	var launch *PiLaunchError
	if !errors.As(err, &launch) {
		return ""
	}
	switch launch.Code {
	case "pi_standalone_prompt_invalid", "pi_standalone_conflicting_arguments", "pi_standalone_entrypoint_invalid", "pi_standalone_deadline_invalid":
		return managementpi.TurnCodeRequestInvalid
	case "pi_profile_missing":
		return managementpi.TurnCodeProfileMissing
	case "unknown_pi_profile":
		return managementpi.TurnCodeProfileUnknown
	case "pi_profile_mismatch":
		return managementpi.TurnCodeProfileMismatch
	case "pi_execution_environment_malformed":
		return managementpi.TurnCodeEnvironmentMalformed
	case "pi_execution_environment_invalid":
		return managementpi.TurnCodeEnvironmentDenied
	case "invalid_project_configuration":
		return managementpi.TurnCodeConfigurationInvalid
	case "pi_tool_authorization_required", "pi_tool_allowlist_required", "pi_tool_allowlist_invalid":
		return managementpi.TurnCodeAuthorizationDenied
	case "provider_executable_not_found", "pi_execution_identity_invalid", "pi_execution_identity_changed":
		return managementpi.TurnCodeIdentityInvalid
	case "runtime_executable_not_found", "runtime_executable_invalid", "runtime_listener_occupied", "runtime_listener_check_failed", "runtime_start_failed", "runtime_readiness_failed", "runtime_exited_early", "shared_runtime_refused":
		return managementpi.TurnCodeRuntimeRefused
	default:
		return ""
	}
}

func piTurnResultExit(code managementpi.TurnResultCode) int {
	switch code {
	case managementpi.TurnCodeCancelled, managementpi.TurnCodeDeadlineExceeded:
		return 2
	case managementpi.TurnCodeResultInvalid:
		return 3
	default:
		return 1
	}
}

func parsePiTurnJSONL(raw []byte, truncated bool) (string, bool, error) {
	if truncated || len(raw) == 0 || !utf8.Valid(raw) {
		return "", false, errors.New("invalid bounded Pi event stream")
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 64<<10), maxRawPiTurnBytes)
	lineNumber := 0
	sawSession, sawAgentStart, sawAgentEnd := false, false, false
	sawFinalAssistantMessage := false
	turnOpen, messageOpen := false, false
	toolFailed := false
	openTools := map[string]string{}
	finalText := ""
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		lineNumber++
		if err := rejectDuplicateJSONNames(line); err != nil {
			return "", false, err
		}
		var event struct {
			Type       string          `json:"type"`
			Version    int             `json:"version"`
			ToolCallID string          `json:"toolCallId"`
			ToolName   string          `json:"toolName"`
			IsError    *bool           `json:"isError"`
			Message    json.RawMessage `json:"message"`
		}
		if err := decodeOneModelCheckEvent(line, &event); err != nil || event.Type == "" {
			return "", false, errors.New("malformed Pi event")
		}
		if lineNumber == 1 && (event.Type != "session" || event.Version != 3) {
			return "", false, errors.New("invalid Pi session header")
		}
		switch event.Type {
		case "session":
			if sawSession || lineNumber != 1 {
				return "", false, errors.New("duplicate Pi session")
			}
			sawSession = true
		case "agent_start":
			if !sawSession || sawAgentStart {
				return "", false, errors.New("invalid Pi agent start")
			}
			sawAgentStart = true
		case "agent_end":
			if !sawAgentStart || sawAgentEnd || turnOpen {
				return "", false, errors.New("invalid Pi agent end")
			}
			sawAgentEnd = true
		case "queue_update", "compaction_start", "compaction_end":
			if !sawAgentStart || sawAgentEnd {
				return "", false, errors.New("event outside Pi lifecycle")
			}
		case "turn_start":
			if !sawAgentStart || sawAgentEnd || turnOpen {
				return "", false, errors.New("invalid Pi turn start")
			}
			turnOpen = true
		case "turn_end":
			if !sawAgentStart || sawAgentEnd || !turnOpen || messageOpen {
				return "", false, errors.New("invalid Pi turn end")
			}
			turnOpen = false
		case "message_start":
			if !sawAgentStart || sawAgentEnd || !turnOpen || messageOpen {
				return "", false, errors.New("invalid Pi message start")
			}
			messageOpen = true
		case "message_update":
			if !sawAgentStart || sawAgentEnd || !turnOpen || !messageOpen {
				return "", false, errors.New("message update outside Pi lifecycle")
			}
		case "message_end":
			if !sawAgentStart || sawAgentEnd || !turnOpen || !messageOpen {
				return "", false, errors.New("invalid Pi message end")
			}
			messageOpen = false
			message, err := parseModelCheckMessage(event.Message)
			if err != nil {
				return "", false, err
			}
			if message.role == "assistant" {
				if message.errorText != "" {
					return "", false, errors.New("assistant failure")
				}
				finalText = message.text
				sawFinalAssistantMessage = true
			}
		case "tool_execution_start":
			if !sawAgentStart || sawAgentEnd {
				return "", false, errors.New("tool start outside Pi lifecycle")
			}
			if event.ToolCallID == "" || event.ToolName == "" || openTools[event.ToolCallID] != "" {
				return "", false, errors.New("invalid Pi tool start")
			}
			openTools[event.ToolCallID] = event.ToolName
		case "tool_execution_update":
			if !sawAgentStart || sawAgentEnd {
				return "", false, errors.New("tool update outside Pi lifecycle")
			}
			if openTools[event.ToolCallID] == "" {
				return "", false, errors.New("invalid Pi tool update")
			}
		case "tool_execution_end":
			if !sawAgentStart || sawAgentEnd {
				return "", false, errors.New("tool end outside Pi lifecycle")
			}
			if event.IsError == nil || event.ToolName == "" || openTools[event.ToolCallID] != event.ToolName {
				return "", false, errors.New("invalid Pi tool end")
			}
			toolFailed = toolFailed || *event.IsError
			delete(openTools, event.ToolCallID)
		default:
			return "", false, fmt.Errorf("unknown Pi event %q", event.Type)
		}
	}
	if err := scanner.Err(); err != nil || !sawSession || !sawAgentStart || !sawAgentEnd || turnOpen || messageOpen || len(openTools) != 0 {
		return "", false, errors.New("incomplete Pi event stream")
	}
	if !sawFinalAssistantMessage {
		return "", false, errors.New("missing authoritative assistant message")
	}
	return finalText, toolFailed, nil
}

func rejectDuplicateJSONNames(document []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(document))
	var walk func() error
	walk = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				nameToken, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := nameToken.(string)
				if !ok || seen[name] {
					return errors.New("duplicate JSON object member")
				}
				seen[name] = true
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := walk(); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		default:
			return errors.New("unexpected JSON delimiter")
		}
	}
	if err := walk(); err != nil {
		return err
	}
	if token, err := decoder.Token(); !errors.Is(err, io.EOF) || token != nil {
		return errors.New("trailing JSON value")
	}
	return nil
}
