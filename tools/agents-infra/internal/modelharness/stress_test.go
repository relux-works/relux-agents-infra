package modelharness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStressRunsBoundedPrefillAndStopsRuntime(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// The child runtime is this test binary in helper mode
	// (stress_helper_process_test.go): a Homebrew-python child listener on
	// the macOS hosted runner accepts no inbound loopback connection — dials
	// hang to timeout while the child stays alive and silent, then refuse
	// again the moment the child dies (diagnostic run 35063667613) — while
	// the same Go binary answers on loopback in 1 ms. The retry below still
	// guards only the bind :0, close, then rebind-in-child race: each attempt
	// uses a fresh port and a fresh runtime (Stress always stops its child
	// before returning), and every assertion still runs against one real
	// pass; a systematic Stress breakage fails all attempts identically.
	const attempts = 3
	var report StressReport
	var port int
	var stderr bytes.Buffer
	for attempt := 1; ; attempt++ {
		stderr.Reset()
		listener, listenErr := net.Listen("tcp", "127.0.0.1:0")
		if listenErr != nil {
			t.Fatal(listenErr)
		}
		port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
		plan := Plan{
			Profile:    "local",
			Mode:       "local",
			Executable: self,
			Argv:       []string{stressHelperArgv0, "127.0.0.1", strconv.Itoa(port)},
			Endpoint:   fmt.Sprintf("http://127.0.0.1:%d/v1", port),
			Stress: &StressPolicy{
				PromptTokens:               2048,
				MaxOutputTokens:            1,
				StartupTimeoutSeconds:      5,
				RequestTimeoutSeconds:      5,
				SampleIntervalMilliseconds: 50,
			},
		}
		report, err = Stress(plan, &stderr)
		if err == nil {
			break
		}
		t.Logf("stress attempt %d/%d on port %d failed: %v stderr=%s report=%#v", attempt, attempts, port, err, stderr.String(), report)
		if attempt == attempts {
			t.Fatalf("Stress: %v stderr=%s report=%#v", err, stderr.String(), report)
		}
	}
	if report.Status != "passed" || report.ObservedPromptTokens != 2048 || !report.WithinTargetTolerance || report.PeakRSSBytes == 0 || report.MemorySamples == 0 {
		t.Fatalf("report=%#v", report)
	}
	connection, dialErr := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 250*time.Millisecond)
	if dialErr == nil {
		connection.Close()
		t.Fatal("stress runtime remained reachable after report")
	}
}

func TestCalibratedRepeatCountTargetsObservedPromptTokens(t *testing.T) {
	repeats, err := calibratedRepeatCount(50000, 256, 267, 1024, 1035)
	if err != nil {
		t.Fatal(err)
	}
	if got := repeats + 11; got != 50000 {
		t.Fatalf("observed prompt tokens=%d repeats=%d", got, repeats)
	}
	if _, err := calibratedRepeatCount(50000, 256, 267, 1024, 267); err == nil {
		t.Fatal("non-monotonic calibration was accepted")
	}
}

func TestSyntheticCompletionUsesOneBoundedPromptAndReadsUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/chat/completions" {
			t.Fatalf("request=%s %s", request.Method, request.URL.Path)
		}
		var document struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
			MaxTokens int  `json:"max_tokens"`
			Stream    bool `json:"stream"`
		}
		if err := json.NewDecoder(request.Body).Decode(&document); err != nil {
			t.Fatal(err)
		}
		if document.Model != "Model" || document.MaxTokens != 1 || document.Stream || len(document.Messages) != 1 || document.Messages[0].Role != "user" {
			t.Fatalf("completion request=%#v", document)
		}
		repeats := strings.Count(document.Messages[0].Content, "x ")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"usage":{"prompt_tokens":` + strconv.Itoa(repeats+11) + `,"completion_tokens":1,"total_tokens":` + strconv.Itoa(repeats+12) + `}}`))
	}))
	defer server.Close()
	usage, payloadBytes, elapsed, err := syntheticCompletion(server.Client(), server.URL+"/v1", "Model", 100, 1)
	if err != nil {
		t.Fatal(err)
	}
	if usage.PromptTokens != 111 || usage.CompletionTokens != 1 || payloadBytes < 100 || elapsed <= 0*time.Millisecond {
		t.Fatalf("usage=%#v payload=%d elapsed=%s", usage, payloadBytes, elapsed)
	}
}
