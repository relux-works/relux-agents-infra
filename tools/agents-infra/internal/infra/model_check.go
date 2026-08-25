package infra

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ModelCheckSummarySchemaVersion = 1
	MinimumModelCheckDeadline      = time.Millisecond
	DefaultModelCheckDeadline      = 5 * time.Minute
	MaximumModelCheckDeadline      = 30 * time.Minute
	ModelCheckFinalResponseBytes   = 4096

	ModelCheckExitSuccess           = 0
	ModelCheckExitExecutionFailed   = 1
	ModelCheckExitTimeout           = 2
	ModelCheckExitMalformedStream   = 3
	ModelCheckExitExpectationFailed = 4
	ModelCheckExitToolFailure       = 5
)

const (
	modelCheckEventsArtifact  = "events.jsonl"
	modelCheckStderrArtifact  = "stderr.log"
	modelCheckJSONSummary     = "summary.json"
	modelCheckTextSummary     = "summary.txt"
	modelCheckMaxEventLine    = 16 << 20
	modelCheckMaxIdentityText = 256
)

// ModelCheckOptions describes one bounded production behavior check. Target is
// the configured canonical entrypoint (for example qwen-infra), not a provider
// executable or an unvalidated model name.
type ModelCheckOptions struct {
	ProjectDir    string
	HomeDir       string
	Target        string
	Prompt        string
	OutputDir     string
	Deadline      time.Duration
	ExpectedTools []string
	ExpectedText  []string
	Environ       []string
	Producer      ChildLaunchCompositionProducer
}

type ModelCheckTargetSummary struct {
	Entrypoint  string `json:"entrypoint"`
	Name        string `json:"name"`
	Environment string `json:"environment"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
}

type ModelCheckToolCall struct {
	Name      string `json:"name"`
	Completed bool   `json:"completed"`
	Failed    bool   `json:"failed"`
}

type ModelCheckExpectation struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Met   bool   `json:"met"`
}

type ModelCheckEventStreamSummary struct {
	Valid    bool   `json:"valid"`
	Complete bool   `json:"complete"`
	Error    string `json:"error,omitempty"`
}

type ModelCheckArtifactSummary struct {
	EventsJSONL string `json:"events_jsonl"`
	Stderr      string `json:"stderr"`
	JSONSummary string `json:"json_summary"`
	TextSummary string `json:"text_summary"`
}

type ModelCheckSummary struct {
	SchemaVersion          int                          `json:"schema_version"`
	Status                 string                       `json:"status"`
	ExitCode               int                          `json:"exit_code"`
	ProcessExitCode        *int                         `json:"process_exit_code"`
	TimedOut               bool                         `json:"timed_out"`
	DeadlineMS             int64                        `json:"deadline_ms"`
	DurationMS             int64                        `json:"duration_ms"`
	Target                 ModelCheckTargetSummary      `json:"target"`
	EventCounts            map[string]int               `json:"event_counts"`
	ToolCalls              []ModelCheckToolCall         `json:"tool_calls"`
	ToolFailures           int                          `json:"tool_failures"`
	FinalResponse          string                       `json:"final_response"`
	FinalResponseSHA256    string                       `json:"final_response_sha256"`
	FinalResponseTruncated bool                         `json:"final_response_truncated"`
	Expectations           []ModelCheckExpectation      `json:"expectations"`
	EventStream            ModelCheckEventStreamSummary `json:"event_stream"`
	ManagedRuntime         PiRunReport                  `json:"managed_runtime"`
	Artifacts              ModelCheckArtifactSummary    `json:"artifacts"`
	Errors                 []string                     `json:"errors"`
}

// ModelCheckFailure is deliberately safe to print. Detailed provider bytes
// remain in mode-0600 raw artifacts; this error names only the stable outcome.
type ModelCheckFailure struct {
	Code   int
	Reason string
}

func (e *ModelCheckFailure) Error() string {
	return "model check " + e.Reason + "; inspect the sanitized summary"
}
func (e *ModelCheckFailure) ExitCode() int { return e.Code }

type parsedModelCheckEvents struct {
	counts           map[string]int
	toolCalls        []ModelCheckToolCall
	toolFailures     int
	finalResponse    string
	assistantFailure string
	valid            bool
	complete         bool
	errText          string
}

type modelCheckToolState struct {
	name      string
	completed bool
	failed    bool
}

// RunModelCheck resolves a canonical target, runs it through the existing
// managed Pi lifecycle, and persists raw plus sanitized evidence. It never
// mirrors provider stdout/stderr to the terminal.
func RunModelCheck(opts ModelCheckOptions) (ModelCheckSummary, error) {
	summary := ModelCheckSummary{
		SchemaVersion: ModelCheckSummarySchemaVersion,
		Status:        "failed",
		EventCounts:   map[string]int{},
		EventStream:   ModelCheckEventStreamSummary{Valid: true},
		Artifacts: ModelCheckArtifactSummary{
			EventsJSONL: modelCheckEventsArtifact,
			Stderr:      modelCheckStderrArtifact,
			JSONSummary: modelCheckJSONSummary,
			TextSummary: modelCheckTextSummary,
		},
	}
	if err := validateModelCheckOptions(&opts); err != nil {
		return ModelCheckSummary{}, err
	}
	summary.DeadlineMS = opts.Deadline.Milliseconds()
	summary.Target.Entrypoint = sanitizeModelCheckText(opts.Target, modelCheckMaxIdentityText)

	outputDir, err := filepath.Abs(opts.OutputDir)
	if err != nil {
		return ModelCheckSummary{}, fmt.Errorf("resolve model-check output directory: %w", err)
	}
	if err := prepareModelCheckOutputDir(outputDir); err != nil {
		return ModelCheckSummary{}, err
	}
	eventsFile, err := createModelCheckArtifact(filepath.Join(outputDir, modelCheckEventsArtifact))
	if err != nil {
		return ModelCheckSummary{}, err
	}
	stderrFile, err := createModelCheckArtifact(filepath.Join(outputDir, modelCheckStderrArtifact))
	if err != nil {
		eventsFile.Close()
		return ModelCheckSummary{}, err
	}

	started := time.Now()
	checkArgs := []string{"--mode", "json", "--print", "--no-session", "--approve", "--", opts.Prompt}
	plan, planErr := BuildCanonicalTargetLaunchPlan(opts.Target, opts.ProjectDir, opts.HomeDir, checkArgs, opts.Producer, nil)
	if planErr == nil {
		planErr = validateModelCheckPlan(plan)
	}
	if plan.Target != nil {
		summary.Target = modelCheckTargetFromPlan(plan)
	}

	var runErr error
	if planErr == nil {
		ctx, cancel := context.WithTimeout(context.Background(), opts.Deadline)
		runErr = RunPi(RunPiOptions{
			ProjectDir: plan.ProjectDir,
			HomeDir:    opts.HomeDir,
			Args:       plan.TargetProviderArgs(),
			Environ:    opts.Environ,
			Stdout:     eventsFile,
			Stderr:     stderrFile,
			Context:    ctx,
			Report:     &summary.ManagedRuntime,
		})
		cancel()
	} else {
		runErr = planErr
	}
	closeErr := errors.Join(eventsFile.Close(), stderrFile.Close())
	if closeErr != nil && runErr == nil {
		runErr = closeErr
	}
	summary.DurationMS = time.Since(started).Milliseconds()
	summary.TimedOut = summary.ManagedRuntime.DeadlineExceeded
	summary.ProcessExitCode = modelCheckProcessExitCode(runErr)

	parsed := parseModelCheckEvents(filepath.Join(outputDir, modelCheckEventsArtifact))
	summary.EventCounts = parsed.counts
	summary.ToolCalls = parsed.toolCalls
	summary.ToolFailures = parsed.toolFailures
	summary.EventStream = ModelCheckEventStreamSummary{Valid: parsed.valid, Complete: parsed.complete, Error: parsed.errText}
	finalSum := sha256.Sum256([]byte(parsed.finalResponse))
	summary.FinalResponseSHA256 = hex.EncodeToString(finalSum[:])
	sanitizedFinal := sanitizeModelCheckText(parsed.finalResponse, 0)
	summary.FinalResponse, summary.FinalResponseTruncated = truncateModelCheckText(sanitizedFinal, ModelCheckFinalResponseBytes)
	summary.Expectations = evaluateModelCheckExpectations(opts.ExpectedTools, opts.ExpectedText, parsed)

	status, exitCode, reasons := evaluateModelCheckOutcome(summary, parsed, runErr)
	summary.Status = status
	summary.ExitCode = exitCode
	summary.Errors = reasons

	if err := writeModelCheckSummaries(outputDir, summary); err != nil {
		return summary, err
	}
	if exitCode != 0 {
		return summary, &ModelCheckFailure{Code: exitCode, Reason: modelCheckFailureReason(status, exitCode)}
	}
	return summary, nil
}

func validateModelCheckOptions(opts *ModelCheckOptions) error {
	if strings.TrimSpace(opts.Target) == "" {
		return errors.New("model-check requires a configured --target entrypoint")
	}
	if opts.Prompt == "" {
		return errors.New("model-check requires a non-empty --prompt")
	}
	if strings.TrimSpace(opts.OutputDir) == "" {
		return errors.New("model-check requires --output-dir")
	}
	if opts.Deadline < MinimumModelCheckDeadline || opts.Deadline > MaximumModelCheckDeadline {
		return fmt.Errorf("model-check deadline must be between %s and %s", MinimumModelCheckDeadline, MaximumModelCheckDeadline)
	}
	if opts.ProjectDir == "" {
		var err error
		opts.ProjectDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("resolve model-check project directory: %w", err)
		}
	}
	if opts.Environ == nil {
		opts.Environ = os.Environ()
	}
	for _, value := range append(append([]string(nil), opts.ExpectedTools...), opts.ExpectedText...) {
		if value == "" {
			return errors.New("model-check expectations must be non-empty")
		}
	}
	return nil
}

func validateModelCheckPlan(plan PrimarySessionLaunchPlan) error {
	if plan.Provider != "pi" || plan.Target == nil || plan.Target.Environment != "pi" {
		return errors.New("model-check target must resolve to the managed Pi environment")
	}
	if plan.Pi == nil || !plan.Pi.Managed || plan.Pi.Runtime == nil {
		return errors.New("model-check target must resolve to a managed local Pi profile")
	}
	return nil
}

func modelCheckTargetFromPlan(plan PrimarySessionLaunchPlan) ModelCheckTargetSummary {
	result := ModelCheckTargetSummary{
		Entrypoint:  plan.Target.Entrypoint,
		Name:        plan.Target.Name,
		Environment: plan.Target.Environment,
		Model:       plan.Target.Model,
	}
	if plan.Resolved.ProfileProvider != nil && plan.Resolved.ProfileProvider.Value != nil {
		result.Provider = *plan.Resolved.ProfileProvider.Value
	}
	result.Entrypoint = sanitizeModelCheckText(result.Entrypoint, modelCheckMaxIdentityText)
	result.Name = sanitizeModelCheckText(result.Name, modelCheckMaxIdentityText)
	result.Environment = sanitizeModelCheckText(result.Environment, modelCheckMaxIdentityText)
	result.Provider = sanitizeModelCheckText(result.Provider, modelCheckMaxIdentityText)
	result.Model = sanitizeModelCheckText(result.Model, modelCheckMaxIdentityText)
	return result
}

func prepareModelCheckOutputDir(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return fmt.Errorf("create model-check output directory: %w", err)
	}
	if err := os.Chmod(outputDir, 0o700); err != nil {
		return fmt.Errorf("secure model-check output directory: %w", err)
	}
	for _, name := range []string{modelCheckEventsArtifact, modelCheckStderrArtifact, modelCheckJSONSummary, modelCheckTextSummary} {
		path := filepath.Join(outputDir, name)
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("model-check refuses to overwrite existing artifact %s", name)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect model-check artifact %s: %w", name, err)
		}
	}
	return nil
}

func createModelCheckArtifact(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create model-check artifact %s: %w", filepath.Base(path), err)
	}
	return file, nil
}

func parseModelCheckEvents(path string) parsedModelCheckEvents {
	result := parsedModelCheckEvents{counts: map[string]int{}, valid: true}
	file, err := os.Open(path)
	if err != nil {
		result.valid = false
		result.errText = "event artifact could not be read"
		return result
	}
	defer file.Close()

	toolOrder := []string{}
	tools := map[string]*modelCheckToolState{}
	sawSession, sawAgentStart, sawAgentEnd := false, false, false
	lineNumber, eventNumber := 0, 0
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64<<10), modelCheckMaxEventLine)
	for scanner.Scan() {
		lineNumber++
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		eventNumber++
		var event struct {
			Type       string          `json:"type"`
			ToolCallID string          `json:"toolCallId"`
			ToolName   string          `json:"toolName"`
			IsError    *bool           `json:"isError"`
			Message    json.RawMessage `json:"message"`
		}
		if err := decodeOneModelCheckEvent(line, &event); err != nil {
			markModelCheckStreamInvalid(&result, fmt.Sprintf("malformed JSONL event at line %d", lineNumber))
			continue
		}
		if event.Type == "" {
			markModelCheckStreamInvalid(&result, fmt.Sprintf("event type is missing at line %d", lineNumber))
			continue
		}
		if eventNumber == 1 && event.Type != "session" {
			markModelCheckStreamInvalid(&result, "first JSONL event is not a session header")
		}
		result.counts[event.Type]++
		switch event.Type {
		case "session":
			if sawSession {
				markModelCheckStreamInvalid(&result, "event stream contains multiple session headers")
			}
			sawSession = true
		case "agent_start":
			sawAgentStart = true
		case "agent_end":
			sawAgentEnd = true
		case "tool_execution_start":
			if event.ToolCallID == "" || event.ToolName == "" {
				markModelCheckStreamInvalid(&result, fmt.Sprintf("tool start is incomplete at line %d", lineNumber))
				continue
			}
			if _, exists := tools[event.ToolCallID]; exists {
				markModelCheckStreamInvalid(&result, fmt.Sprintf("tool call is duplicated at line %d", lineNumber))
				continue
			}
			tools[event.ToolCallID] = &modelCheckToolState{name: event.ToolName}
			toolOrder = append(toolOrder, event.ToolCallID)
		case "tool_execution_end":
			if event.ToolCallID == "" || event.ToolName == "" || event.IsError == nil {
				markModelCheckStreamInvalid(&result, fmt.Sprintf("tool end is incomplete at line %d", lineNumber))
				continue
			}
			state, exists := tools[event.ToolCallID]
			if !exists || state.completed || state.name != event.ToolName {
				markModelCheckStreamInvalid(&result, fmt.Sprintf("tool end has no matching start at line %d", lineNumber))
				continue
			}
			state.completed = true
			state.failed = *event.IsError
		case "message_end":
			message, messageErr := parseModelCheckMessage(event.Message)
			if messageErr != nil {
				markModelCheckStreamInvalid(&result, fmt.Sprintf("message_end is malformed at line %d", lineNumber))
				continue
			}
			if message.role == "assistant" {
				result.finalResponse = message.text
				if message.errorText != "" {
					result.assistantFailure = message.errorText
				}
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		markModelCheckStreamInvalid(&result, "event stream contains an overlong or unreadable JSONL record")
	}
	for _, id := range toolOrder {
		state := tools[id]
		call := ModelCheckToolCall{
			Name:      sanitizeModelCheckText(state.name, modelCheckMaxIdentityText),
			Completed: state.completed,
			Failed:    state.failed,
		}
		result.toolCalls = append(result.toolCalls, call)
		if call.Failed {
			result.toolFailures++
		}
		if !call.Completed {
			markModelCheckStreamInvalid(&result, "event stream ended with an unfinished tool call")
		}
	}
	result.complete = sawSession && sawAgentStart && sawAgentEnd && result.valid
	if !result.complete && result.errText == "" && eventNumber > 0 {
		result.errText = "event stream did not reach a complete agent lifecycle"
	}
	return result
}

func decodeOneModelCheckEvent(line []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(line)))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("multiple JSON values in one JSONL record")
	}
	return nil
}

type modelCheckMessage struct {
	role      string
	text      string
	errorText string
}

func parseModelCheckMessage(raw json.RawMessage) (modelCheckMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return modelCheckMessage{}, errors.New("message is absent")
	}
	var message struct {
		Role         string          `json:"role"`
		Content      json.RawMessage `json:"content"`
		StopReason   string          `json:"stopReason"`
		ErrorMessage string          `json:"errorMessage"`
	}
	if err := json.Unmarshal(raw, &message); err != nil || message.Role == "" {
		return modelCheckMessage{}, errors.New("message shape is invalid")
	}
	text, err := modelCheckMessageText(message.Content)
	if err != nil {
		return modelCheckMessage{}, err
	}
	errorText := ""
	if message.ErrorMessage != "" || message.StopReason == "error" {
		errorText = "assistant message reported an error"
	}
	return modelCheckMessage{role: message.Role, text: text, errorText: errorText}, nil
}

func modelCheckMessageText(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct, nil
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return "", errors.New("message content is invalid")
	}
	var text strings.Builder
	for _, block := range blocks {
		if block.Type == "text" {
			text.WriteString(block.Text)
		}
	}
	return text.String(), nil
}

func markModelCheckStreamInvalid(result *parsedModelCheckEvents, message string) {
	result.valid = false
	if result.errText == "" {
		result.errText = message
	}
}

func evaluateModelCheckExpectations(expectedTools, expectedText []string, parsed parsedModelCheckEvents) []ModelCheckExpectation {
	result := make([]ModelCheckExpectation, 0, len(expectedTools)+len(expectedText))
	for _, expected := range expectedTools {
		met := false
		for _, call := range parsed.toolCalls {
			if call.Name == expected {
				met = true
				break
			}
		}
		result = append(result, ModelCheckExpectation{Kind: "tool", Value: sanitizeModelCheckText(expected, modelCheckMaxIdentityText), Met: met})
	}
	for _, expected := range expectedText {
		result = append(result, ModelCheckExpectation{Kind: "text", Value: sanitizeModelCheckText(expected, modelCheckMaxIdentityText), Met: strings.Contains(parsed.finalResponse, expected)})
	}
	return result
}

func evaluateModelCheckOutcome(summary ModelCheckSummary, parsed parsedModelCheckEvents, runErr error) (string, int, []string) {
	var reasons []string
	if summary.TimedOut {
		reasons = append(reasons, "deadline exceeded")
		if !summary.ManagedRuntime.CleanupConfirmed {
			reasons = append(reasons, "managed runtime cleanup was not confirmed")
		}
		return "timed_out", ModelCheckExitTimeout, reasons
	}
	if !parsed.valid || (runErr == nil && !parsed.complete) {
		reason := parsed.errText
		if reason == "" {
			reason = "event stream did not reach a complete agent lifecycle"
		}
		reasons = append(reasons, sanitizeModelCheckText(reason, modelCheckMaxIdentityText))
		return "failed", ModelCheckExitMalformedStream, compactModelCheckReasons(reasons)
	}
	if runErr != nil {
		reasons = append(reasons, sanitizeModelCheckText(runErr.Error(), modelCheckMaxIdentityText))
		return "failed", ModelCheckExitExecutionFailed, compactModelCheckReasons(reasons)
	}
	if parsed.assistantFailure != "" {
		reasons = append(reasons, parsed.assistantFailure)
		return "failed", ModelCheckExitExecutionFailed, compactModelCheckReasons(reasons)
	}
	if !summary.ManagedRuntime.CleanupConfirmed {
		reasons = append(reasons, "managed runtime cleanup was not confirmed")
		return "failed", ModelCheckExitExecutionFailed, reasons
	}
	if parsed.toolFailures > 0 {
		reasons = append(reasons, fmt.Sprintf("%d tool execution(s) failed", parsed.toolFailures))
		return "failed", ModelCheckExitToolFailure, reasons
	}
	for _, expectation := range summary.Expectations {
		if !expectation.Met {
			reasons = append(reasons, "expected "+expectation.Kind+" was not observed: "+expectation.Value)
		}
	}
	if len(reasons) > 0 {
		return "failed", ModelCheckExitExpectationFailed, compactModelCheckReasons(reasons)
	}
	return "passed", ModelCheckExitSuccess, nil
}

func compactModelCheckReasons(reasons []string) []string {
	result := reasons[:0]
	for _, reason := range reasons {
		if strings.TrimSpace(reason) != "" {
			result = append(result, reason)
		}
	}
	return result
}

func modelCheckFailureReason(status string, exitCode int) string {
	if status == "timed_out" {
		return "timed out"
	}
	switch exitCode {
	case ModelCheckExitMalformedStream:
		return "reported a malformed event stream"
	case ModelCheckExitExpectationFailed:
		return "failed an expectation"
	case ModelCheckExitToolFailure:
		return "observed a failed tool execution"
	default:
		return "failed"
	}
}

func modelCheckProcessExitCode(err error) *int {
	code := 0
	if err == nil {
		return &code
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
		return &code
	}
	return nil
}

func writeModelCheckSummaries(outputDir string, summary ModelCheckSummary) error {
	jsonFile, err := createModelCheckArtifact(filepath.Join(outputDir, modelCheckJSONSummary))
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(jsonFile)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(summary); err != nil {
		jsonFile.Close()
		return fmt.Errorf("write model-check JSON summary: %w", err)
	}
	if err := jsonFile.Close(); err != nil {
		return fmt.Errorf("close model-check JSON summary: %w", err)
	}
	textFile, err := createModelCheckArtifact(filepath.Join(outputDir, modelCheckTextSummary))
	if err != nil {
		return err
	}
	if _, err := io.WriteString(textFile, RenderModelCheckSummary(summary)); err != nil {
		textFile.Close()
		return fmt.Errorf("write model-check text summary: %w", err)
	}
	if err := textFile.Close(); err != nil {
		return fmt.Errorf("close model-check text summary: %w", err)
	}
	return nil
}

// RenderModelCheckSummary is deterministic and contains sanitized, bounded
// values only. Raw provider output stays in events.jsonl and stderr.log.
func RenderModelCheckSummary(summary ModelCheckSummary) string {
	var text strings.Builder
	fmt.Fprintf(&text, "status: %s\n", summary.Status)
	fmt.Fprintf(&text, "exit_code: %d\n", summary.ExitCode)
	if summary.ProcessExitCode == nil {
		text.WriteString("process_exit_code: unknown\n")
	} else {
		fmt.Fprintf(&text, "process_exit_code: %d\n", *summary.ProcessExitCode)
	}
	fmt.Fprintf(&text, "timed_out: %t\n", summary.TimedOut)
	fmt.Fprintf(&text, "deadline_ms: %d\n", summary.DeadlineMS)
	fmt.Fprintf(&text, "duration_ms: %d\n", summary.DurationMS)
	fmt.Fprintf(&text, "target: %s (%s)\n", summary.Target.Name, summary.Target.Entrypoint)
	fmt.Fprintf(&text, "provider: %s\n", summary.Target.Provider)
	fmt.Fprintf(&text, "model: %s\n", summary.Target.Model)
	text.WriteString("event_counts:\n")
	keys := make([]string, 0, len(summary.EventCounts))
	for key := range summary.EventCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&text, "  %s: %d\n", key, summary.EventCounts[key])
	}
	text.WriteString("tool_calls:\n")
	for _, call := range summary.ToolCalls {
		fmt.Fprintf(&text, "  - name=%s completed=%t failed=%t\n", call.Name, call.Completed, call.Failed)
	}
	fmt.Fprintf(&text, "tool_failures: %d\n", summary.ToolFailures)
	fmt.Fprintf(&text, "final_response: %q\n", summary.FinalResponse)
	fmt.Fprintf(&text, "final_response_truncated: %t\n", summary.FinalResponseTruncated)
	text.WriteString("expectations:\n")
	for _, expectation := range summary.Expectations {
		fmt.Fprintf(&text, "  - kind=%s value=%q met=%t\n", expectation.Kind, expectation.Value, expectation.Met)
	}
	fmt.Fprintf(&text, "event_stream_valid: %t\n", summary.EventStream.Valid)
	fmt.Fprintf(&text, "event_stream_complete: %t\n", summary.EventStream.Complete)
	if summary.EventStream.Error != "" {
		fmt.Fprintf(&text, "event_stream_error: %s\n", summary.EventStream.Error)
	}
	fmt.Fprintf(&text, "managed_runtime_cleanup: %t\n", summary.ManagedRuntime.CleanupConfirmed)
	fmt.Fprintf(&text, "pi_process_group_cleanup: %s\n", summary.ManagedRuntime.PiProcessGroupCleanup)
	fmt.Fprintf(&text, "runtime_process_group_cleanup: %s\n", summary.ManagedRuntime.RuntimeProcessGroupCleanup)
	for _, reason := range summary.Errors {
		fmt.Fprintf(&text, "error: %s\n", reason)
	}
	return text.String()
}

var (
	modelCheckAssignmentSecret = regexp.MustCompile(`(?i)\b([a-z0-9_.-]*(?:api[_-]?key|token|secret|password|passwd|pwd))\b\s*[:=]\s*([^\s,;]+)`)
	modelCheckBearerSecret     = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`)
	modelCheckHexSecret        = regexp.MustCompile(`\b[0-9a-fA-F]{32,}\b`)
	modelCheckKeylikeSecret    = regexp.MustCompile(`\b[A-Za-z0-9_-]{48,}\b`)
)

func sanitizeModelCheckText(value string, limit int) string {
	value = strings.ToValidUTF8(value, "�")
	value = modelCheckBearerSecret.ReplaceAllString(value, "Bearer [REDACTED_SECRET]")
	value = modelCheckAssignmentSecret.ReplaceAllString(value, "$1=[REDACTED_SECRET]")
	value = modelCheckHexSecret.ReplaceAllString(value, "[REDACTED_HEX_SECRET]")
	value = modelCheckKeylikeSecret.ReplaceAllString(value, "[REDACTED_KEYLIKE_SECRET]")
	if limit > 0 {
		value, _ = truncateModelCheckText(value, limit)
	}
	return value
}

func truncateModelCheckText(value string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value, false
	}
	cut := maxBytes
	for cut > 0 && !utf8.ValidString(value[:cut]) {
		cut--
	}
	return value[:cut], true
}
