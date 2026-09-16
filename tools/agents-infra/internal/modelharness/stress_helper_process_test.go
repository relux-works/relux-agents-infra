package modelharness

// Self-re-exec fixture for the Stress process-lifecycle integration witness.
//
// TestStressRunsBoundedPrefillAndStopsRuntime spawns the test binary itself as
// the child runtime (argv: stress-helper-child <host> <port>). This keeps every
// gate property of the witness — a real separate child process, real readiness
// polling, real RSS sampling, a real stop — while staying hermetic: no
// interpreter dependency, and the listener is the same Go binary that the
// hosted runners already prove reachable on loopback.
//
// Why not an external fixture server: a Homebrew-python child listener on the
// macOS hosted runner accepts no inbound loopback connection — dials hang to
// timeout while the child stays alive and silent, then refuse again the moment
// the child dies (diagnostic run 35063667613: pre-spawn refused, polls 2-8
// i/o timeout, post-kill refused; the Go httptest control in the same process
// answered in 1 ms). The exact runner mechanism (suspected per-binary loopback
// filtering) is unproven, so the fixture no longer depends on it.

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
)

// stressHelperArgv0 is the child-mode trigger: TestMain serves instead of
// running tests when os.Args[1] equals it.
const stressHelperArgv0 = "stress-helper-child"

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == stressHelperArgv0 {
		os.Exit(runStressHelperChild(os.Args[2:]))
	}
	os.Exit(m.Run())
}

func runStressHelperChild(args []string) int {
	if len(args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <host> <port>\n", stressHelperArgv0)
		return 2
	}
	host := args[0]
	port, err := strconv.Atoi(args[1])
	if err != nil || host == "" || port < 1 || port > 65535 {
		fmt.Fprintf(os.Stderr, "invalid helper address %q:%q\n", host, args[1])
		return 2
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"Model"}]}`))
	})
	mux.HandleFunc("/v1/chat/completions", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(io.LimitReader(request.Body, 64<<20))
		if err != nil {
			http.Error(writer, "unreadable body", http.StatusBadRequest)
			return
		}
		var document struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &document); err != nil || len(document.Messages) != 1 {
			http.Error(writer, "invalid completion request", http.StatusBadRequest)
			return
		}
		// Same token model as the witness asserts: one token per "x " repeat
		// plus the 11-token fixed overhead.
		prompt := strings.Count(document.Messages[0].Content, "x ") + 11
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(fmt.Sprintf(
			`{"usage":{"prompt_tokens":%d,"completion_tokens":1,"total_tokens":%d}}`,
			prompt, prompt+1)))
	})
	server := &http.Server{Handler: mux}
	listener, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "helper listen: %v\n", err)
		return 1
	}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "helper serve: %v\n", err)
		return 1
	}
	return 0
}
