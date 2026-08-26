package modelharness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestStressRunsBoundedPrefillAndStopsRuntime(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is required for the process-lifecycle integration witness")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	server := `
import argparse, json
from http.server import BaseHTTPRequestHandler, HTTPServer
p=argparse.ArgumentParser(); p.add_argument('--host'); p.add_argument('--port',type=int); a=p.parse_args()
class H(BaseHTTPRequestHandler):
  def log_message(self,*args): pass
  def do_GET(self):
    if self.path!='/v1/models': self.send_error(404); return
    body=json.dumps({'data':[{'id':'Model'}]}).encode(); self.send_response(200); self.send_header('Content-Type','application/json'); self.send_header('Content-Length',str(len(body))); self.end_headers(); self.wfile.write(body)
  def do_POST(self):
    if self.path!='/v1/chat/completions': self.send_error(404); return
    n=int(self.headers.get('Content-Length','0')); doc=json.loads(self.rfile.read(n)); repeats=doc['messages'][0]['content'].count('x '); prompt=repeats+11
    body=json.dumps({'usage':{'prompt_tokens':prompt,'completion_tokens':1,'total_tokens':prompt+1}}).encode(); self.send_response(200); self.send_header('Content-Type','application/json'); self.send_header('Content-Length',str(len(body))); self.end_headers(); self.wfile.write(body)
HTTPServer((a.host,a.port),H).serve_forever()
`
	plan := Plan{
		Profile:    "local",
		Mode:       "local",
		Executable: python,
		Argv:       []string{"-c", server, "--host", "127.0.0.1", "--port", strconv.Itoa(port)},
		Endpoint:   fmt.Sprintf("http://127.0.0.1:%d/v1", port),
		Stress: &StressPolicy{
			PromptTokens:               2048,
			MaxOutputTokens:            1,
			StartupTimeoutSeconds:      5,
			RequestTimeoutSeconds:      5,
			SampleIntervalMilliseconds: 50,
		},
	}
	var stderr bytes.Buffer
	report, err := Stress(plan, &stderr)
	if err != nil {
		t.Fatalf("Stress: %v stderr=%s report=%#v", err, stderr.String(), report)
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
