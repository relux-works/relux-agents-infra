//go:build !windows

package infra

import (
	"bytes"
	"io"
	"regexp"
	"strconv"
	"sync"
)

// piKVCacheBoundExceededPattern matches the mlx-lm server's greppable
// `kv_cache_bound_exceeded` warning line (mlx_lm/server.py,
// _report_kv_cache_bound_if_exceeded), emitted once per completed request
// whose prompt plus generated tokens exceeded the active generation KV
// cache bound.
var piKVCacheBoundExceededPattern = regexp.MustCompile(
	`kv_cache_bound_exceeded request_id=(\S+) max_kv_size=(\d+) observed_tokens=(\d+)`,
)

// piKVBoundAlertMaxCarryBytes bounds the pending-partial-line buffer so a
// runtime that never emits a newline cannot grow it unboundedly.
const piKVBoundAlertMaxCarryBytes = 4096

// newPiKVBoundAlertWriter wraps a runtime child's merged stdout/stderr
// writer and reports every observed kv_cache_bound_exceeded line through
// report, so an operator sees a truncated-context request in the Pi
// session log instead of only in raw server output. It forwards every byte
// to target unmodified and unconditionally; a line it cannot parse is still
// forwarded, and report is never called on the target's own write path.
func newPiKVBoundAlertWriter(target io.Writer, report func(requestID string, maxKVSize, observedTokens int)) io.Writer {
	if target == nil {
		return nil
	}
	return &piKVBoundAlertWriter{target: target, report: report}
}

type piKVBoundAlertWriter struct {
	target io.Writer
	report func(requestID string, maxKVSize, observedTokens int)

	mu    sync.Mutex
	carry []byte
}

func (w *piKVBoundAlertWriter) Write(p []byte) (int, error) {
	n, err := w.target.Write(p)
	if n == 0 {
		return n, err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.carry = append(w.carry, p[:n]...)
	for {
		idx := bytes.IndexByte(w.carry, '\n')
		if idx < 0 {
			break
		}
		line := w.carry[:idx]
		w.carry = w.carry[idx+1:]
		w.scanLine(line)
	}
	if len(w.carry) > piKVBoundAlertMaxCarryBytes {
		w.carry = w.carry[len(w.carry)-piKVBoundAlertMaxCarryBytes:]
	}
	return n, err
}

func (w *piKVBoundAlertWriter) scanLine(line []byte) {
	match := piKVCacheBoundExceededPattern.FindSubmatch(line)
	if match == nil {
		return
	}
	maxKVSize, err := strconv.Atoi(string(match[2]))
	if err != nil {
		return
	}
	observedTokens, err := strconv.Atoi(string(match[3]))
	if err != nil {
		return
	}
	w.report(string(match[1]), maxKVSize, observedTokens)
}
