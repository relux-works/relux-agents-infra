package modelharness

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	StressContract      = "model-harness.stress-report"
	StressSchemaVersion = 1
)

type StressReport struct {
	Contract                     string  `json:"contract"`
	SchemaVersion                int     `json:"schema_version"`
	Profile                      string  `json:"profile"`
	Endpoint                     string  `json:"endpoint"`
	Model                        string  `json:"model,omitempty"`
	RequestedPromptTokens        int     `json:"requested_prompt_tokens"`
	ObservedPromptTokens         int     `json:"observed_prompt_tokens,omitempty"`
	PromptTokenDelta             int     `json:"prompt_token_delta,omitempty"`
	WithinTargetTolerance        bool    `json:"within_target_tolerance"`
	MaxOutputTokens              int     `json:"max_output_tokens"`
	PayloadBytes                 int     `json:"payload_bytes,omitempty"`
	StartupMilliseconds          int64   `json:"startup_milliseconds,omitempty"`
	PrefillMilliseconds          int64   `json:"prefill_milliseconds,omitempty"`
	BaselineRSSBytes             uint64  `json:"baseline_rss_bytes,omitempty"`
	PeakRSSBytes                 uint64  `json:"peak_rss_bytes,omitempty"`
	HostMemoryBytes              uint64  `json:"host_memory_bytes,omitempty"`
	PeakHostMemoryPercent        float64 `json:"peak_host_memory_percent,omitempty"`
	MemorySamples                int     `json:"memory_samples"`
	MemorySamplingError          string  `json:"memory_sampling_error,omitempty"`
	CalibrationSmallPromptTokens int     `json:"calibration_small_prompt_tokens,omitempty"`
	CalibrationLargePromptTokens int     `json:"calibration_large_prompt_tokens,omitempty"`
	Status                       string  `json:"status"`
	Error                        string  `json:"error,omitempty"`
}

type completionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func Stress(plan Plan, stderr io.Writer) (report StressReport, err error) {
	report = StressReport{
		Contract:      StressContract,
		SchemaVersion: StressSchemaVersion,
		Profile:       plan.Profile,
		Endpoint:      plan.Endpoint,
		Status:        "failed",
	}
	if plan.Mode != "local" {
		err = errors.New("stress currently supports only local profiles")
		report.Error = err.Error()
		return report, err
	}
	if plan.Stress == nil {
		err = fmt.Errorf("profile %q has no stress policy", plan.Profile)
		report.Error = err.Error()
		return report, err
	}
	policy := *plan.Stress
	report.RequestedPromptTokens = policy.PromptTokens
	report.MaxOutputTokens = policy.MaxOutputTokens
	if err = inspectExecutable(plan.Executable); err != nil {
		report.Error = err.Error()
		return report, err
	}
	if err = preflightStressEndpoint(plan.Endpoint); err != nil {
		report.Error = err.Error()
		return report, err
	}

	command := exec.Command(plan.Executable, plan.Argv...)
	command.Stdout = stderr
	command.Stderr = stderr
	configureStressProcess(command)
	if err = command.Start(); err != nil {
		err = fmt.Errorf("start local profile %q: %w", plan.Profile, err)
		report.Error = err.Error()
		return report, err
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- command.Wait() }()
	childExited := false
	sampler := startRSSSampler(command.Process.Pid, time.Duration(policy.SampleIntervalMilliseconds)*time.Millisecond)
	defer func() {
		memory := sampler.Stop()
		report.PeakRSSBytes = memory.PeakBytes
		report.MemorySamples = memory.Samples
		report.MemorySamplingError = memory.Error
		report.HostMemoryBytes = hostMemoryBytes()
		if report.HostMemoryBytes > 0 && report.PeakRSSBytes > 0 {
			report.PeakHostMemoryPercent = math.Round((float64(report.PeakRSSBytes)/float64(report.HostMemoryBytes))*10000) / 100
		}
		if !childExited {
			_ = stopStressProcess(command)
			select {
			case <-waitCh:
			case <-time.After(5 * time.Second):
			}
		}
		if err != nil && report.Error == "" {
			report.Error = err.Error()
		}
	}()

	started := time.Now()
	model, readinessErr := waitForStressModel(plan.Endpoint, time.Duration(policy.StartupTimeoutSeconds)*time.Second, waitCh, &childExited)
	report.StartupMilliseconds = time.Since(started).Milliseconds()
	if readinessErr != nil {
		err = readinessErr
		return report, err
	}
	report.Model = model
	if rss, sampleErr := processRSSBytes(command.Process.Pid); sampleErr == nil {
		report.BaselineRSSBytes = rss
	}

	client := &http.Client{Timeout: time.Duration(policy.RequestTimeoutSeconds) * time.Second}
	const smallRepeats = 256
	const largeRepeats = 1024
	smallUsage, _, _, requestErr := syntheticCompletion(client, plan.Endpoint, model, smallRepeats, policy.MaxOutputTokens)
	if requestErr != nil {
		err = fmt.Errorf("calibrate synthetic prompt at %d repeats: %w", smallRepeats, requestErr)
		return report, err
	}
	largeUsage, _, _, requestErr := syntheticCompletion(client, plan.Endpoint, model, largeRepeats, policy.MaxOutputTokens)
	if requestErr != nil {
		err = fmt.Errorf("calibrate synthetic prompt at %d repeats: %w", largeRepeats, requestErr)
		return report, err
	}
	report.CalibrationSmallPromptTokens = smallUsage.PromptTokens
	report.CalibrationLargePromptTokens = largeUsage.PromptTokens
	repeats, calibrationErr := calibratedRepeatCount(policy.PromptTokens, smallRepeats, smallUsage.PromptTokens, largeRepeats, largeUsage.PromptTokens)
	if calibrationErr != nil {
		err = calibrationErr
		return report, err
	}
	usage, payloadBytes, elapsed, requestErr := syntheticCompletion(client, plan.Endpoint, model, repeats, policy.MaxOutputTokens)
	report.PayloadBytes = payloadBytes
	report.PrefillMilliseconds = elapsed.Milliseconds()
	if requestErr != nil {
		err = fmt.Errorf("synthetic prefill request: %w", requestErr)
		return report, err
	}
	report.ObservedPromptTokens = usage.PromptTokens
	report.PromptTokenDelta = usage.PromptTokens - policy.PromptTokens
	tolerance := max(32, policy.PromptTokens/100)
	report.WithinTargetTolerance = abs(report.PromptTokenDelta) <= tolerance
	if !report.WithinTargetTolerance {
		err = fmt.Errorf("observed prompt tokens %d differ from target %d by more than tolerance %d", usage.PromptTokens, policy.PromptTokens, tolerance)
		return report, err
	}
	report.Status = "passed"
	return report, nil
}

func EncodeStressReport(w io.Writer, report StressReport) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func preflightStressEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("parse stress endpoint: %w", err)
	}
	listener, err := net.Listen("tcp", parsed.Host)
	if err != nil {
		return fmt.Errorf("stress endpoint %s is already occupied: %w", parsed.Host, err)
	}
	return listener.Close()
}

func waitForStressModel(endpoint string, timeout time.Duration, waitCh <-chan error, childExited *bool) (string, error) {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	modelsURL := strings.TrimSuffix(endpoint, "/") + "/models"
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case childErr := <-waitCh:
			*childExited = true
			if childErr == nil {
				childErr = errors.New("runtime exited before readiness")
			}
			return "", fmt.Errorf("local runtime exited before readiness: %w", childErr)
		default:
		}
		response, requestErr := client.Get(modelsURL)
		if requestErr == nil {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
			response.Body.Close()
			if readErr == nil && response.StatusCode == http.StatusOK {
				var document struct {
					Data []struct {
						ID string `json:"id"`
					} `json:"data"`
				}
				if decodeErr := json.Unmarshal(body, &document); decodeErr == nil && len(document.Data) > 0 && document.Data[0].ID != "" {
					return document.Data[0].ID, nil
				}
				lastErr = errors.New("models response did not contain a model id")
			} else if readErr != nil {
				lastErr = readErr
			} else {
				lastErr = fmt.Errorf("models readiness returned HTTP %d", response.StatusCode)
			}
		} else {
			lastErr = requestErr
		}
		time.Sleep(250 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = errors.New("readiness deadline elapsed")
	}
	return "", fmt.Errorf("wait for local runtime readiness: %w", lastErr)
}

func syntheticCompletion(client *http.Client, endpoint, model string, repeats, maxOutputTokens int) (completionUsage, int, time.Duration, error) {
	content := strings.Repeat("x ", repeats)
	document := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens int  `json:"max_tokens"`
		Stream    bool `json:"stream"`
	}{Model: model, MaxTokens: maxOutputTokens, Stream: false}
	document.Messages = append(document.Messages, struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{Role: "user", Content: content})
	payload, err := json.Marshal(document)
	if err != nil {
		return completionUsage{}, 0, 0, err
	}
	request, err := http.NewRequest(http.MethodPost, strings.TrimSuffix(endpoint, "/")+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return completionUsage{}, len(payload), 0, err
	}
	request.Header.Set("Content-Type", "application/json")
	started := time.Now()
	response, err := client.Do(request)
	elapsed := time.Since(started)
	if err != nil {
		return completionUsage{}, len(payload), elapsed, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return completionUsage{}, len(payload), elapsed, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return completionUsage{}, len(payload), elapsed, fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	var completion struct {
		Usage completionUsage `json:"usage"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		return completionUsage{}, len(payload), elapsed, fmt.Errorf("decode completion response: %w", err)
	}
	if completion.Usage.PromptTokens < 1 {
		return completionUsage{}, len(payload), elapsed, errors.New("completion response omitted prompt token usage")
	}
	return completion.Usage, len(payload), elapsed, nil
}

func calibratedRepeatCount(targetTokens, smallRepeats, smallTokens, largeRepeats, largeTokens int) (int, error) {
	if targetTokens < 1 || smallRepeats < 1 || largeRepeats <= smallRepeats || smallTokens < 1 || largeTokens <= smallTokens {
		return 0, errors.New("synthetic prompt calibration is not monotonic")
	}
	tokensPerRepeat := float64(largeTokens-smallTokens) / float64(largeRepeats-smallRepeats)
	intercept := float64(smallTokens) - tokensPerRepeat*float64(smallRepeats)
	repeats := int(math.Round((float64(targetTokens) - intercept) / tokensPerRepeat))
	if repeats < 1 || repeats > 2_000_000 {
		return 0, fmt.Errorf("calibrated repeat count %d is outside the safe range", repeats)
	}
	return repeats, nil
}

type rssSampleSummary struct {
	PeakBytes uint64
	Samples   int
	Error     string
}

type rssSampler struct {
	stop chan struct{}
	done chan rssSampleSummary
	once sync.Once
	last rssSampleSummary
}

func startRSSSampler(pid int, interval time.Duration) *rssSampler {
	sampler := &rssSampler{stop: make(chan struct{}), done: make(chan rssSampleSummary, 1)}
	go func() {
		result := rssSampleSummary{}
		sample := func() {
			rss, err := processRSSBytes(pid)
			if err != nil {
				if result.Error == "" {
					result.Error = err.Error()
				}
				return
			}
			result.Samples++
			if rss > result.PeakBytes {
				result.PeakBytes = rss
			}
		}
		sample()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sample()
			case <-sampler.stop:
				sample()
				sampler.done <- result
				return
			}
		}
	}()
	return sampler
}

func (sampler *rssSampler) Stop() rssSampleSummary {
	sampler.once.Do(func() {
		close(sampler.stop)
		sampler.last = <-sampler.done
	})
	return sampler.last
}

func processRSSBytes(pid int) (uint64, error) {
	ps, err := exec.LookPath("ps")
	if err != nil {
		return 0, err
	}
	output, err := exec.Command(ps, "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, err
	}
	kilobytes, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse process RSS: %w", err)
	}
	return kilobytes * 1024, nil
}

func hostMemoryBytes() uint64 {
	if runtime.GOOS == "darwin" {
		output, err := exec.Command("/usr/sbin/sysctl", "-n", "hw.memsize").Output()
		if err == nil {
			value, parseErr := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 64)
			if parseErr == nil {
				return value
			}
		}
	}
	if runtime.GOOS == "linux" {
		content, err := os.ReadFile("/proc/meminfo")
		if err == nil {
			for _, line := range strings.Split(string(content), "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 && fields[0] == "MemTotal:" {
					kilobytes, parseErr := strconv.ParseUint(fields[1], 10, 64)
					if parseErr == nil {
						return kilobytes * 1024
					}
				}
			}
		}
	}
	return 0
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
